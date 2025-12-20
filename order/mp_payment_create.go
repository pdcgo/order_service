package order

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// MpPaymentCreate implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) MpPaymentCreate(ctx context.Context, req *connect.Request[order_iface.MpPaymentCreateRequest]) (*connect.Response[order_iface.MpPaymentCreateResponse], error) {
	var err error

	source, err := custom_connect.GetRequestSource(ctx)
	if err != nil {
		return nil, err
	}
	var domainID uint
	switch source.RequestFrom {
	case access_iface.RequestFrom_REQUEST_FROM_ADMIN:
		domainID = authorization.RootDomain
	default:
		domainID = uint(source.TeamId)
	}

	identity := o.auth.
		AuthIdentityFromHeader(req.Header())

	// agent := identity.
	// 	Identity()

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.Order{}: &authorization_iface.CheckPermission{
				DomainID: domainID,
				Actions:  []authorization_iface.Action{authorization_iface.Update},
			},
		}).
		Err()

	if err != nil {
		return nil, err
	}

	db := o.db.WithContext(ctx)
	pay := req.Msg
	result := order_iface.MpPaymentCreateResponse{}

	if pay.Amount == 0 {
		return &connect.Response[order_iface.MpPaymentCreateResponse]{}, errors.New("amount is zero")
	}

	var adj db_models.OrderAdjustment
	var teamID uint64
	var adjExist bool

	err = db.Transaction(func(tx *gorm.DB) error {

		err = tx.
			Model(&db_models.Order{}).
			Select("team_id").
			Where("id = ?", pay.OrderId).
			Find(&teamID).
			Error

		if err != nil {
			return err
		}

		// checking team id
		if teamID != pay.TeamId {
			return fmt.Errorf("order id %d not in team id %d", pay.OrderId, pay.TeamId)
		}

		err = tx.
			Model(&db_models.OrderAdjustment{}).
			Where("order_id = ?", pay.OrderId).
			Where("at = ?", pay.At.AsTime()).
			Where("type = ?", pay.Type).
			Find(&adj).
			Error

		if err != nil {
			return err
		}

		if adj.ID == 0 { // jika baru
			adj = db_models.OrderAdjustment{
				OrderID: uint(pay.OrderId),
				MpID:    uint(pay.ShopId),
				At:      pay.At.AsTime(),
				FundAt:  pay.WdAt.AsTime(),
				Type:    db_models.AdjustmentType(pay.Type),
				Amount:  pay.Amount,
				Source:  pay.Source,
				Desc:    pay.Desc,
			}

			err = tx.Save(&adj).Error
			if err != nil {
				return err
			}
			result.Id = uint64(adj.ID)

			return nil
		}

		adjExist = true
		adj.Amount = pay.Amount
		adj.Desc = pay.Desc
		adj.FundAt = pay.WdAt.AsTime()
		adj.Source = pay.Source

		err = tx.Save(&adj).Error

		if err != nil {
			return err
		}

		var revType revenue_iface.ReceivableAdjustmentType
		switch adj.Type {
		case db_models.AdjReturn:
			revType = revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_RETURN
		default:
			return errors.New("unimplemented")
		}

		var desc string
		if adjExist {
			desc = fmt.Sprintf("edit %s sebelumnya", adj.Desc)
		} else {
			desc = pay.Desc
		}

		// send to accounting revenue adjustment
		_, err = o.revenueService.SellingReceivableAdjustment(ctx, &connect.Request[revenue_iface.SellingReceivableAdjustmentRequest]{
			Msg: &revenue_iface.SellingReceivableAdjustmentRequest{
				OrderId:  uint64(adj.OrderID),
				AdjRefId: fmt.Sprintf("%d", adj.ID),
				TeamId:   teamID,
				Amount:   adj.Amount,
				Desc:     desc,
				Type:     revType,
			},
		})

		if err != nil {
			return err
		}

		result.Id = uint64(adj.ID)
		return nil
	})

	return &connect.Response[order_iface.MpPaymentCreateResponse]{}, nil

}
