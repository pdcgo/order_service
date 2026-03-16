package order

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// OrderChangeStatus implements [order_ifaceconnect.OrderServiceHandler].
func (o *orderServiceImpl) OrderChangeStatus(
	ctx context.Context,
	req *connect.Request[order_iface.OrderChangeStatusRequest],
) (*connect.Response[order_iface.OrderChangeStatusResponse], error) {
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

	pay := req.Msg
	result := order_iface.OrderChangeStatusResponse{}
	db := o.db.WithContext(ctx)

	err = db.Transaction(func(tx *gorm.DB) error {
		switch change := pay.Status.(type) {
		case *order_iface.OrderChangeStatusRequest_Shipped:
			err = changeShipped(tx, agent, change.Shipped)
		}

		return err

	})

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&result), nil
}

func changeShipped(tx *gorm.DB, agent authorization_iface.Identity, change *order_iface.ShippedStatus) error {
	var err error

	// find in database
	var ord db_models.Order
	query := tx.
		Model(&db_models.Order{}).
		Where("team_id = ?", change.TeamId).
		Where("status = ?", db_models.OrdShipped)

	switch change.By.(type) {
	case *order_iface.ShippedStatus_OrderId:
		query = query.
			Where("id = ?", change.GetOrderId())
	case *order_iface.ShippedStatus_RefId:
		query = query.
			Where("order_ref_id = ?", change.GetRefId())
	}

	err = query.First(&ord).Error
	if err != nil {
		return err
	}

	if !ord.IsGiveToCourrier() {
		return fmt.Errorf("order %s is not given to courier", ord.OrderRefID)
	}

	// change status
	err = tx.Model(&ord).Update("status", db_models.OrdShipped).Error
	if err != nil {
		return err
	}

	return fmt.Errorf("unimplemented")
}
