package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/schema/services/order_iface/v1/order_ifaceconnect"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

type OrderShippedFunc cli.ActionFunc

func NewOrderShipped(
	db *gorm.DB,
	cfg *configs.AppConfig,
	defaultClientInterceptor custom_connect.DefaultClientInterceptor,
	helper *Helper,
) OrderShippedFunc {
	return func(ctx context.Context, c *cli.Command) error {

		orderService := order_ifaceconnect.NewOrderServiceClient(
			http.DefaultClient,
			cfg.OrderService.Endpoint,
			defaultClientInterceptor,
		)

		query := db.
			Model(&db_models.Order{}).
			Where("status = ?", db_models.OrdShipped).
			Where("id > ?", 1037942).
			Order("id asc")

		rows, err := query.Rows()
		if err != nil {
			return err
		}

		// setting token
		ctx, err = helper.SetAuthorization(ctx, "palingsakti")
		if err != nil {
			return err
		}

		for rows.Next() {
			var ord db_models.Order
			err = db.ScanRows(rows, &ord)
			if err != nil {
				return err
			}

			res, err := orderService.OrderTracking(ctx, &connect.Request[order_iface.OrderTrackingRequest]{
				Msg: &order_iface.OrderTrackingRequest{
					Track: &order_iface.OrderTrackingRequest_Shipped{
						Shipped: &order_iface.ShippedTrack{
							SetShipment: true,
							OrderIds: []uint64{
								uint64(ord.ID),
							},
						},
					},
				},
			})

			if err != nil {
				slog.Error(err.Error())
				time.Sleep(time.Second)
				continue
			}

			for _, item := range res.Msg.Result {
				slog.Info("track", "order_id", ord.ID, "status", item.Status)
			}

		}

		return nil
	}
}
