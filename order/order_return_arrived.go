package order

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrderReturnArrived implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderReturnArrived(
	ctx context.Context,
	req *connect.Request[order_iface.OrderReturnArrivedRequest],
) (*connect.Response[order_iface.OrderReturnArrivedResponse], error) {
	// dipanggil setelah stock di accept
	var err error
	pay := req.Msg

	identity := o.
		auth.
		AuthIdentityFromHeader(req.Header())

	// agent := identity.
	// 	Identity()

	err = identity.
		Err()

	if err != nil {
		return nil, err
	}

	db := o.db.WithContext(ctx)
	var ord db_models.Order
	var aftertx []func() error

	err = db.Transaction(func(tx *gorm.DB) error {
		// getting and lock order

		err = tx.
			Clauses(clause.Locking{
				Strength: "UPDATE",
			}).
			Model(&db_models.Order{}).
			Where("invertory_return_tx_id  = ?", pay.TxId).
			Find(&ord).
			Error
		if err != nil {
			return err
		}

		if ord.ID == 0 {
			return errors.New("order is not return")
		}

		// getting and lock payment
		var payment *db_models.OrderPayment
		payment, err = o.getOrderPaymentMeta(tx, ord.ID, true)
		if err != nil {
			return err
		}

		if !payment.IsReceivableAdjusted {
			// call mp adjustment
			aftertx = append(aftertx, func() error {
				_, err = o.revenueService.SellingReceivableAdjustment(ctx, connect.NewRequest(
					&revenue_iface.SellingReceivableAdjustmentRequest{
						OrderId: uint64(ord.ID),
						TeamId:  uint64(ord.TeamID),
						ShopId:  uint64(ord.OrderMpID),
						Amount:  float64(ord.OrderMpTotal),
						Desc:    "accept return to warehouse",
						Type:    revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_CANCEL_RECEIVE,
						At:      timestamppb.Now(),
						WdAt:    timestamppb.Now(),
					},
				))
				return err

			})

			payment.IsReceivableAdjusted = true
			err = tx.Save(payment).Error
			if err != nil {
				return err
			}
		}

		// change status
		err = tx.
			Model(&db_models.Order{}).
			Where("id = ?", ord.ID).
			Updates(map[string]interface{}{
				"status": db_models.OrdReturnCompleted,
			}).
			Error

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	for _, after := range aftertx {
		err = after()
		if err != nil {
			return nil, err
		}
	}

	return &connect.Response[order_iface.OrderReturnArrivedResponse]{}, nil
}

func (o *orderServiceImpl) getOrderPaymentMeta(tx *gorm.DB, orderId uint, lock bool) (*db_models.OrderPayment, error) {
	var err error
	var meta db_models.OrderPayment

	if lock {
		tx = tx.Clauses(clause.Locking{
			Strength: "UPDATE",
		})

	}

	err = tx.
		Clauses(clause.Locking{
			Strength: "UPDATE",
		}).
		Model(&db_models.OrderPayment{}).
		Where("order_id = ?", orderId).
		Find(&meta).
		Error

	if err != nil {
		return &meta, err
	}

	if meta.ID == 0 {
		meta = db_models.OrderPayment{
			OrderID: orderId,
		}

		err = tx.
			Save(&meta).
			Error

		if err != nil {
			return &meta, err
		}
	}

	return &meta, nil

}
