package order

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/schema/services/tracking_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"gorm.io/gorm"
)

// OrderTracking implements [order_ifaceconnect.OrderServiceHandler].
func (o *orderServiceImpl) OrderTracking(
	ctx context.Context,
	req *connect.Request[order_iface.OrderTrackingRequest],
) (*connect.Response[order_iface.OrderTrackingResponse], error) {
	var err error
	pay := req.Msg

	identity := o.auth.AuthIdentityFromHeader(req.Header())

	agent := identity.Identity()

	err = identity.
		Err()

	if err != nil {
		return nil, err
	}

	db := o.db

	result := order_iface.OrderTrackingResponse{
		Result: map[uint64]*tracking_iface.TrackInfo{},
	}

	switch track := pay.Track.(type) {
	case *order_iface.OrderTrackingRequest_Shipped:
		datas := []struct {
			ShippingId uint64
			Receipt    string
			TxId       uint64
			OrderId    uint64
			Status     db_models.OrdStatus
		}{}
		err = db.
			Table("orders o").
			Joins("left join inv_transactions it on it.id = o.invertory_tx_id").
			Where("o.id in ?", track.Shipped.OrderIds).
			Select([]string{
				"o.id as order_id",
				"it.id as tx_id",
				"it.shipping_id",
				"o.receipt",
				"o.status",
			}).
			Find(&datas).
			Error

		if err != nil {
			return connect.NewResponse(&result), err
		}

		// checking tracking
	DataLoop:
		for _, data := range datas {
			res, err := o.trackService.TrackingGet(ctx, &connect.Request[tracking_iface.TrackingGetRequest]{
				Msg: &tracking_iface.TrackingGetRequest{
					Payload: &tracking_iface.TrackingPayload{
						ShippingId: data.ShippingId,
						Receipt:    data.Receipt,
					},
				},
			})

			if err != nil {
				return connect.NewResponse(&result), err
			}
			result.Result[data.OrderId] = res.Msg.TrackInfo

			switch res.Msg.TrackInfo.Status {
			case tracking_iface.Status_STATUS_CREATED,
				tracking_iface.Status_STATUS_CANCEL,
				tracking_iface.Status_STATUS_UNSPECIFIED:
				continue DataLoop
			}

			if track.Shipped.SetShipment {
				switch data.Status {
				case db_models.OrdShipped:
					// change status
					err = db.Transaction(func(tx *gorm.DB) error {

						err = tx.
							Model(&db_models.Order{}).
							Where("id = ?", data.OrderId).
							Update("status", db_models.OrdCourrierShipped).
							Error

						if err != nil {
							return err
						}

						ts := db_models.OrderTimestamp{
							OrderID:     uint(data.OrderId),
							UserID:      agent.GetUserID(),
							From:        agent.GetAgentType(),
							OrderStatus: db_models.OrdCourrierShipped,
							Timestamp:   time.Now(),
						}

						err = tx.Save(&ts).Error
						if err != nil {
							return err
						}

						err = tx.
							Model(&db_models.InvTransaction{}).
							Where("id = ?", data.TxId).
							Update("is_shipped", true).
							Error

						if err != nil {
							return err
						}

						return nil

					})

					if err != nil {
						return nil, err
					}

				}

			}

		}

	}

	return connect.NewResponse(&result), err
}
