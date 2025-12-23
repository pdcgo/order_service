package order

import (
	"context"
	"errors"
	"fmt"
	"math"

	"connectrpc.com/connect"
	"github.com/pdcgo/order_service/order/order_core"
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

	agent := identity.
		Identity()

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

	err = db.Transaction(func(tx *gorm.DB) error {

		ordPayment := order_core.
			NewOrderPaymentManage(tx, uint(pay.OrderId), agent.IdentityID(), pay)

		err = ordPayment.
			Create()

		if err != nil {
			return err
		}

		// setting result
		result.IsEdited = ordPayment.IsEdited
		result.IsSendReceivableAdjustment = ordPayment.IsSendReceivableAdjustment
		result.IsReceivableCreatedAdjustment = ordPayment.IsReceivableCreatedAdjustment

		var desc string
		if ordPayment.IsEdited {
			desc = fmt.Sprintf("edit %s sebelumnya", ordPayment.Adj.Desc)
		} else {
			desc = pay.Desc
		}

		if ordPayment.IsReceivableCreatedAdjustment {
			// send to accounting revenue adjustment
			_, err = o.revenueService.SellingReceivableAdjustment(ctx, &connect.Request[revenue_iface.SellingReceivableAdjustmentRequest]{
				Msg: &revenue_iface.SellingReceivableAdjustmentRequest{
					ShopId:   pay.ShopId,
					OrderId:  uint64(ordPayment.Adj.OrderID),
					AdjRefId: fmt.Sprintf("%s-%d", pay.Type, ordPayment.Adj.ID),
					TeamId:   pay.TeamId,
					Amount:   ordPayment.Adj.Amount,
					Desc:     desc,
					Type:     revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_CREATED_REVENUE,
					At:       pay.At,
					WdAt:     pay.WdAt,
				},
			})

			if err != nil {
				return err
			}
		}

		if ordPayment.IsSendReceivableAdjustment {

			revType, err := o.getType(ordPayment.Adj)
			if err != nil {
				return err
			}

			amount := ordPayment.Adj.Amount
			switch revType {
			case revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_OTHER_COST,
				revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_OTHER_REVENUE:
				amount = math.Abs(amount)
			}

			// send to accounting revenue adjustment
			_, err = o.revenueService.SellingReceivableAdjustment(ctx, &connect.Request[revenue_iface.SellingReceivableAdjustmentRequest]{
				Msg: &revenue_iface.SellingReceivableAdjustmentRequest{
					ShopId:   pay.ShopId,
					OrderId:  uint64(ordPayment.Adj.OrderID),
					AdjRefId: fmt.Sprintf("%d", ordPayment.Adj.ID),
					TeamId:   pay.TeamId,
					Amount:   amount,
					Desc:     desc,
					Type:     revType,
					At:       pay.At,
					WdAt:     pay.WdAt,
				},
			})

			if err != nil {
				return err
			}
		}

		result.Id = uint64(ordPayment.Adj.ID)
		return nil
	})

	return connect.NewResponse(&result), err

}

func (o *orderServiceImpl) getType(adj *db_models.OrderAdjustment) (revenue_iface.ReceivableAdjustmentType, error) {
	var revType revenue_iface.ReceivableAdjustmentType
	switch adj.Type {
	case db_models.AdjReturn:
		revType = revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_RETURN_COST

	case db_models.AdjLostCompensation:
		revType = revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_REFUND_LOST

	case db_models.AdjOrderFund:
		revType = revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_ORDER_FUND

	case db_models.AdjPremi:
		if adj.Amount < 0 {
			revType = revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_OTHER_COST
		} else {
			revType = revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_OTHER_REVENUE
		}

	default:
		return revType, fmt.Errorf("%s revtype not mapped", adj.Type)
	}

	return revType, nil
}
