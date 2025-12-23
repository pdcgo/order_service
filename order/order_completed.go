package order

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// OrderCompleted implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderCompleted(ctx context.Context, req *connect.Request[order_iface.OrderCompletedRequest]) (*connect.Response[order_iface.OrderCompletedResponse], error) {
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
	db := o.db.WithContext(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {

		err = tx.
			Model(&db_models.Order{}).
			Where("id = ?", pay.OrderId).
			Where("team_id = ?", pay.TeamId).
			Where("status NOT IN ?", []db_models.OrdStatus{
				db_models.OrdCancel,
				db_models.OrdCompleted,
				db_models.OrdReturnProblem,
			}).
			Updates(map[string]interface{}{
				"status": db_models.OrdCompleted,
			}).
			Error

		if err != nil {
			return err
		}

		// log completed
		ts := db_models.OrderTimestamp{
			OrderID:     uint(pay.OrderId),
			UserID:      agent.IdentityID(),
			OrderStatus: db_models.OrdCompleted,
			Timestamp:   time.Now(),
			From:        agent.GetAgentType(),
		}
		err = tx.Save(&ts).Error

		if err != nil {
			return err
		}

		// removing tag related
		err = tx.
			Model(&db_models.OrderTagRelation{}).
			Where("relation_from = ?", db_models.RelationFromTracking).
			Where("order_id = ?", pay.OrderId).
			Delete(&db_models.OrderTagRelation{}).
			Error

		if err != nil {
			return err
		}

		return nil

	})

	return &connect.Response[order_iface.OrderCompletedResponse]{}, err
}
