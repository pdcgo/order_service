package order_core

import (
	"fmt"

	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderPaymentManage struct {
	tx      *gorm.DB
	orderID uint
	userID  uint

	meta    *db_models.OrderPayment
	pay     *order_iface.MpPaymentCreateRequest
	mpTotal float64

	CreatedReceivableAdjustmentAmount float64
	Adj                               *db_models.OrderAdjustment
	IsReceivableCreatedAdjustment     bool
	IsEdited                          bool
	IsSendReceivableAdjustment        bool
}

func (o *OrderPaymentManage) Create(nexts ...NextHandler) error {
	err := NewChain(
		o.checkTeamID,
		o.getOrderPaymentMeta,
		o.checkMustReceivableAdjusted,
		o.createOrderAdjustment,
		o.getMpTotal,
		o.calculateMpAdjustment,
		o.setOrderPaymentInfoLegacy,
		// nexts...,
	)
	return err
}

func (o *OrderPaymentManage) setOrderPaymentInfoLegacy(next NextFunc) NextFunc {
	return func() error {
		if !o.IsReceivableCreatedAdjustment {
			return next()
		}

		err := o.
			tx.
			Model(&db_models.Order{}).
			Where("id = ?", o.orderID).
			Updates(map[string]interface{}{
				"wd_total":   o.Adj.Amount,
				"wd_fund":    true,
				"wd_fund_at": o.pay.WdAt.AsTime(),
			}).
			Error

		if err != nil {
			return err
		}

		return next()
	}
}

func (o *OrderPaymentManage) getMpTotal(next NextFunc) NextFunc {
	return func() error {
		err := o.
			tx.
			Model(&db_models.Order{}).
			Where("id = ?", o.orderID).
			Select("order_mp_total").
			Find(&o.mpTotal).
			Error
		if err != nil {
			return err
		}

		return next()
	}
}

func (o *OrderPaymentManage) calculateMpAdjustment(next NextFunc) NextFunc {
	return func() error {
		o.CreatedReceivableAdjustmentAmount = o.mpTotal - o.Adj.Amount
		return next()
	}
}

func (o *OrderPaymentManage) createOrderAdjustment(next NextFunc) NextFunc {
	return func() error {
		var err error
		var adj db_models.OrderAdjustment
		pay := o.pay

		err = o.
			tx.
			Model(&db_models.OrderAdjustment{}).
			Where("order_id = ?", pay.OrderId).
			Where("at = ?", pay.At.AsTime()).
			Where("type = ?", pay.Type).
			Find(&adj).
			Error

		if err != nil {
			return err
		}

		if adj.ID == 0 {
			adj = db_models.OrderAdjustment{
				OrderID:       uint(pay.OrderId),
				MpID:          uint(pay.ShopId),
				IsMultiRegion: pay.IsMultiRegion,
				At:            pay.At.AsTime(),
				FundAt:        pay.WdAt.AsTime(),
				Type:          db_models.AdjustmentType(pay.Type),
				Amount:        pay.Amount,
				Source:        pay.Source,
				Desc:          pay.Desc,
			}

			err = o.
				tx.
				Save(&adj).
				Error

			if err != nil {
				return err
			}

			o.IsSendReceivableAdjustment = true
		} else {
			if adj.Amount == pay.Amount &&
				adj.FundAt.Equal(pay.WdAt.AsTime()) &&
				adj.At.Equal(pay.At.AsTime()) {

				o.Adj = &adj
				return nil
			}

			adj.Amount = pay.Amount
			adj.Desc = pay.Desc
			adj.FundAt = pay.WdAt.AsTime()
			adj.At = pay.At.AsTime()
			adj.Source = pay.Source

			err = o.tx.Save(&adj).Error

			if err != nil {
				return err
			}

			o.IsSendReceivableAdjustment = true
			o.IsEdited = true
		}

		o.Adj = &adj
		return next()
	}
}

func (o *OrderPaymentManage) checkTeamID(next NextFunc) NextFunc {
	return func() error {
		var teamID uint64

		err := o.tx.
			Model(&db_models.Order{}).
			Select("team_id").
			Where("id = ?", o.orderID).
			Find(&teamID).
			Error

		if err != nil {
			return err
		}

		// checking team id
		if teamID != o.pay.TeamId {
			return fmt.Errorf("order id %d not in team id %d", o.pay.OrderId, o.pay.TeamId)
		}

		return next()
	}
}

func (o *OrderPaymentManage) checkMustReceivableAdjusted(next NextFunc) NextFunc {
	return func() error {
		if o.meta.IsReceivableAdjusted {
			return next()
		}

		tipe := db_models.AdjustmentType(o.pay.Type)
		switch tipe {
		case db_models.AdjOrderFund,
			db_models.AdjLostCompensation:
			o.IsReceivableCreatedAdjustment = true
		}

		o.meta.IsReceivableAdjusted = true
		err := o.tx.Save(o.meta).Error

		if err != nil {
			return err
		}

		return next()
	}
}

func (o *OrderPaymentManage) getOrderPaymentMeta(next NextFunc) NextFunc {
	return func() error {
		var err error
		o.meta = &db_models.OrderPayment{}

		err = o.
			tx.
			Clauses(clause.Locking{
				Strength: "UPDATE",
			}).
			Model(&db_models.OrderPayment{}).
			Where("order_id = ?", o.orderID).
			Find(o.meta).
			Error

		if err != nil {
			return err
		}

		if o.meta.ID == 0 {
			o.meta = &db_models.OrderPayment{
				OrderID: o.orderID,
			}
			err = o.
				tx.
				Save(o.meta).
				Error

			if err != nil {
				return err
			}
		}

		return next()
	}
}

func NewOrderPaymentManage(
	tx *gorm.DB,
	orderID,
	userID uint,
	pay *order_iface.MpPaymentCreateRequest,
) *OrderPaymentManage {
	return &OrderPaymentManage{
		tx:      tx,
		orderID: orderID,
		userID:  userID,
		pay:     pay,
	}
}
