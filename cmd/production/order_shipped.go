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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
		cancel, err := custom_connect.InitTracer("order-service-shipped-worker")
		if err != nil {
			return err
		}

		defer cancel(ctx)

		orderService := order_ifaceconnect.NewOrderServiceClient(
			http.DefaultClient,
			cfg.OrderService.Endpoint,
			defaultClientInterceptor,
		)

		query := db.
			Model(&db_models.Order{}).
			Where("status = ?", db_models.OrdShipped).
			Where("created_at > ?", time.Now().AddDate(0, 0, -15)).
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

		// initiating tracert
		tracer := otel.GetTracerProvider().Tracer("")

		for rows.Next() {
			var ord db_models.Order
			err = db.ScanRows(rows, &ord)
			if err != nil {
				return err
			}

			var span trace.Span
			ctx, span = tracer.Start(ctx, "check_shipped")

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
				span.SetStatus(codes.Error, err.Error())
				span.End()

				slog.Error(err.Error())
				time.Sleep(time.Second)
				continue
			}

			for _, item := range res.Msg.Result {
				slog.Info("track", "order_id", ord.ID, "status", item.Status)
			}

			span.End()

		}

		return nil
	}
}
