package main

import (
	"context"
	"net/http"
	"os"

	"github.com/pdcgo/schema/services/revenue_iface/v1/revenue_ifaceconnect"
	"github.com/pdcgo/schema/services/tracking_iface/v1/tracking_ifaceconnect"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/cloud_logging"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

func NewDatabase(cfg *configs.AppConfig) (*gorm.DB, error) {
	return db_connect.NewProductionDatabase("order_service", &cfg.Database)
}

func NewAuthorization(
	cfg *configs.AppConfig,
	db *gorm.DB,
	cache ware_cache.Cache,
) authorization_iface.Authorization {
	return authorization.NewAuthorization(cache, db, cfg.JwtSecret)
}

func NewCache(cfg *configs.AppConfig) (ware_cache.Cache, error) {
	return ware_cache.NewCustomCache(cfg.CacheService.Endpoint), nil
}

func NewRevenueServiceClient(
	cfg *configs.AppConfig,
	defaultInterceptor custom_connect.DefaultClientInterceptor,
) revenue_ifaceconnect.RevenueServiceClient {
	return revenue_ifaceconnect.NewRevenueServiceClient(
		http.DefaultClient,
		cfg.AccountingService.Endpoint,
		defaultInterceptor,
	)
}

func NewTrackingServiceClient(
	cfg *configs.AppConfig,
	defaultInterceptor custom_connect.DefaultClientInterceptor,
) tracking_ifaceconnect.TrackingServiceClient {
	return tracking_ifaceconnect.NewTrackingServiceClient(
		http.DefaultClient,
		cfg.TrackingService.Endpoint,
		defaultInterceptor,
	)
}

type App *cli.Command

// type App struct {
// 	Run func() error
// }

func NewApp(
	api ApiFunc,
	orderShipped OrderShippedFunc,
) App {

	return &cli.Command{
		Commands: []*cli.Command{
			{
				Name:        "batch",
				Description: "untuk batch updater",
				Commands: []*cli.Command{
					{
						Name:        "shipped",
						Description: "check updated order shipped",
						Action:      cli.ActionFunc(orderShipped),
					},
				},
			},
		},

		Action: cli.ActionFunc(api),
	}
}

func main() {
	if os.Getenv("DISABLE_CLOUD_LOGGING") == "" {
		cloud_logging.SetCloudLoggingDefault()
	}

	app, err := InitializeApp()
	if err != nil {
		panic(err)
	}

	var run *cli.Command = app
	err = run.Run(context.Background(), os.Args)
	if err != nil {
		panic(err)
	}
}
