package order

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/order_service/order/order_core"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
)

// MpPaymentDelete implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) MpPaymentDelete(ctx context.Context, req *connect.Request[order_iface.MpPaymentDeleteRequest]) (*connect.Response[order_iface.MpPaymentDeleteResponse], error) {
	var err error

	return nil, errors.New("unimplemented")

	source, err := custom_connect.GetRequestSource(ctx)
	if err != nil {
		return nil, err
	}

	pay := req.Msg

	var domainID uint
	switch source.RequestFrom {
	case access_iface.RequestFrom_REQUEST_FROM_ADMIN:
		domainID = authorization.RootDomain
	default:
		domainID = uint(pay.TeamId)
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

	var adj db_models.OrderAdjustment

	err = order_core.NewChain(
		func(next order_core.NextFunc) order_core.NextFunc { // getting adjustment
			return func() error {
				err = db.
					Model(&db_models.OrderAdjustment{}).
					First(&adj, pay.AdjId).
					Error

				if err != nil {
					return err
				}

				return next()
			}
		},
		func(next order_core.NextFunc) order_core.NextFunc {
			return func() error {

				if domainID == authorization.RootDomain {
					return next()
				}

				var teamID uint64
				err = db.
					Model(&db_models.Order{}).
					Select("team_id").
					Where("id = ?", adj.OrderID).
					Find(&teamID).
					Error

				if err != nil {
					return err
				}

				if teamID != pay.TeamId {
					return err
				}

				return next()
			}
		},
		func(next order_core.NextFunc) order_core.NextFunc { // delete
			return func() error {
				err = db.Model(&db_models.OrderAdjustment{}).Where("id = ?", adj.ID).Delete(&db_models.OrderAdjustment{}).Error
				if err != nil {
					return err
				}
				return next()
			}
		},
	)

	return &connect.Response[order_iface.MpPaymentDeleteResponse]{}, err
}
