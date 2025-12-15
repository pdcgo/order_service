package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/schema/services/tracking_iface/v1/tracking_ifaceconnect"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type orderServiceImpl struct {
	auth authorization_iface.Authorization
	db   *gorm.DB
	// revenueClient revenue_ifaceconnect.RevenueServiceClient
	trackingService tracking_ifaceconnect.TrackingServiceClient
}

// ChangeOrderRefID implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) ChangeOrderRefID(context.Context, *connect.Request[order_iface.ChangeOrderRefIDRequest]) (*connect.Response[order_iface.ChangeOrderRefIDResponse], error) {
	panic("unimplemented")
}

// MpPaymentCreate implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) MpPaymentCreate(context.Context, *connect.Request[order_iface.MpPaymentCreateRequest]) (*connect.Response[order_iface.MpPaymentCreateResponse], error) {
	panic("unimplemented")
}

// MpPaymentDelete implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) MpPaymentDelete(context.Context, *connect.Request[order_iface.MpPaymentDeleteRequest]) (*connect.Response[order_iface.MpPaymentDeleteResponse], error) {
	panic("unimplemented")
}

// MpPaymentOrderList implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) MpPaymentOrderList(context.Context, *connect.Request[order_iface.MpPaymentOrderListRequest]) (*connect.Response[order_iface.MpPaymentOrderListResponse], error) {
	panic("unimplemented")
}

// OrderList implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderList(context.Context, *connect.Request[order_iface.OrderListRequest], *connect.ServerStream[order_iface.OrderListResponse]) error {
	panic("unimplemented")
}

// OrderOverview implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderOverview(context.Context, *connect.Request[order_iface.OrderOverviewRequest]) (*connect.Response[order_iface.OrderOverviewResponse], error) {
	panic("unimplemented")
}

// OrderTagRemove implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderTagRemove(context.Context, *connect.Request[order_iface.OrderTagRemoveRequest]) (*connect.Response[order_iface.OrderTagRemoveResponse], error) {
	panic("unimplemented")
}

func NewOrderService(
	auth authorization_iface.Authorization,
	db *gorm.DB,
	// trackingService tracking_ifaceconnect.TrackingServiceClient,
) *orderServiceImpl {
	return &orderServiceImpl{
		auth: auth,
		db:   db,
		// trackingService: trackingService,
	}
}

func (o *orderServiceImpl) getOrder(
	tx *gorm.DB,
	teamID uint64,
	orderID uint64,
	ordRefID string,
	lock bool,
) (*db_models.Order, error) {
	if orderID == 0 && ordRefID == "" {
		return nil, errors.New("empty orderid or ref id")
	}

	if lock {
		tx = tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "NOWAIT",
		})
	}

	ord := db_models.Order{}
	err := tx.
		Model(&db_models.Order{}).
		Where("team_id = ?", teamID).
		Where("(id = ?) or (order_ref_id = ?)", orderID, ordRefID).
		Where("status NOT IN ?", []db_models.OrdStatus{
			db_models.OrdCancel,
		}).
		Find(&ord).
		Error

	if err != nil {
		return nil, err
	}

	if ord.OrderMpID == 0 {
		return &ord, fmt.Errorf("refmode order with id %s marketplace not set %d with receipt %s with ref %s id %d", ord.OrderRefID, ord.ID, ord.Receipt, ordRefID, orderID)
	}

	return &ord, err
}

// OrderFundSet implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderFundSet(
	ctx context.Context,
	stream *connect.ClientStream[order_iface.OrderFundSetRequest],
) (*connect.Response[order_iface.OrderFundSetResponse], error) {
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
		AuthIdentityFromHeader(stream.RequestHeader())

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
	err = db.Transaction(func(tx *gorm.DB) error {
		for stream.Receive() {
			msg := stream.Msg()

			switch event := msg.Kind.(type) {
			case *order_iface.OrderFundSetRequest_OrderFundRollback:
				return fmt.Errorf("error orderfund stream %s", event.OrderFundRollback.Message)
			case *order_iface.OrderFundSetRequest_OrderFundSet:
				fundset := event.OrderFundSet
				var ord *db_models.Order
				switch value := fundset.OrderIdentifier.(type) {
				case *order_iface.OrderFundSet_OrderId:
					ord, err = o.getOrder(tx, fundset.TeamId, value.OrderId, "", false)
				case *order_iface.OrderFundSet_OrderRefId:
					ord, err = o.getOrder(tx, fundset.TeamId, 0, value.OrderRefId, false)
				default:
					return errors.New("unknown identifier orderfund")

				}

				if err != nil {
					return err
				}

				// Log Adjustment
				var ordAdjust db_models.OrderAdjustment
				tipe := db_models.AdjOrderFund

				err = tx.
					Model(&db_models.OrderAdjustment{}).
					Where("order_id = ?", ord.ID).
					Where("type = ?", tipe).
					Find(&ordAdjust).
					Error

				if err != nil {
					return err
				}

				if ordAdjust.ID == 0 {
					ordAdjust = db_models.OrderAdjustment{
						OrderID: ord.ID,
						MpID:    ord.OrderMpID,
						At:      event.OrderFundSet.At.AsTime(),
						Type:    tipe,
						Amount:  event.OrderFundSet.Amount,
						Desc:    event.OrderFundSet.Desc,
					}

					err = tx.Save(&ordAdjust).Error
					if err != nil {
						return err
					}
				} else {
					ordAdjust.Amount = event.OrderFundSet.Amount
					ordAdjust.At = event.OrderFundSet.At.AsTime()
					err = tx.
						Save(&ordAdjust).
						Error
					if err != nil {
						return err
					}
				}

				// log.Println("send to revenue")
			case *order_iface.OrderFundSetRequest_OrderCompletedSet:
				completedSet := event.OrderCompletedSet

				var ord *db_models.Order
				switch value := completedSet.OrderIdentifier.(type) {
				case *order_iface.OrderCompletedSet_OrderId:
					ord, err = o.getOrder(tx, completedSet.TeamId, value.OrderId, "", true)
				case *order_iface.OrderCompletedSet_OrderRefId:
					ord, err = o.getOrder(tx, completedSet.TeamId, 0, value.OrderRefId, true)
				default:
					return errors.New("unknown identifier orderfund")

				}

				if err != nil {
					return err
				}

				err = tx.
					Model(&db_models.Order{}).
					Where("id = ?", ord.ID).
					Updates(map[string]interface{}{
						"wd_total":   completedSet.Amount,
						"wd_fund":    true,
						"wd_fund_at": completedSet.WdAt.AsTime(),
						"status":     db_models.OrdCompleted,
					}).
					Error

				if err != nil {
					return err
				}

				// log adjustment change
				err = tx.
					Model(&db_models.OrderAdjustment{}).
					Where("order_id = ?", ord.ID).
					Where("type = ?", db_models.AdjOrderFund).
					Updates(map[string]interface{}{
						"fund_at": completedSet.WdAt.AsTime(),
					}).
					Error

				if err != nil {
					return err
				}

				// log completed
				ts := db_models.OrderTimestamp{
					OrderID:     ord.ID,
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
					Where("order_id = ?", ord.ID).
					Delete(&db_models.OrderTagRelation{}).
					Error

				if err != nil {
					return err
				}

			default:
				return errors.New("unknown event orderfund")

			}

		}

		return stream.Err()
	})

	return &connect.Response[order_iface.OrderFundSetResponse]{}, err
}
