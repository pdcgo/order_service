package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pdcgo/order_service"
	"github.com/pdcgo/schema/services/revenue_iface/v1/revenue_ifaceconnect"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/cloud_logging"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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

type App struct {
	Run func() error
}

func NewApp(
	mux *http.ServeMux,
	orderRegister order_service.RegisterHandler,
) *App {
	return &App{
		Run: func() error {
			cancel, err := custom_connect.InitTracer("order-service")
			if err != nil {
				return err
			}

			defer cancel(context.Background())

			// register api
			orderRegister()

			port := os.Getenv("PORT")
			if port == "" {
				port = "8083"
			}

			host := os.Getenv("HOST")
			listen := fmt.Sprintf("%s:%s", host, port)
			log.Println("listening on", listen)

			http.ListenAndServe(
				listen,
				// Use h2c so we can serve HTTP/2 without TLS.
				h2c.NewHandler(
					custom_connect.WithCORS(mux),
					&http2.Server{}),
			)

			return nil
		},
	}
}

func main() {
	cloud_logging.SetCloudLoggingDefault()
	app, err := InitializeApp()
	if err != nil {
		panic(err)
	}

	err = app.Run()
	if err != nil {
		panic(err)
	}
}
