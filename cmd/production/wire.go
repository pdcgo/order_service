//go:build wireinject
// +build wireinject

package main

import (
	"net/http"

	"github.com/google/wire"
	"github.com/pdcgo/order_service"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/urfave/cli/v3"
)

func InitializeApp() (App, error) {
	wire.Build(
		http.NewServeMux,
		configs.NewProductionConfig,
		NewDatabase,
		NewAuthorization,
		NewCache,
		// client
		custom_connect.NewDefaultInterceptor,

		// external service
		custom_connect.NewDefaultClientInterceptor,
		custom_connect.NewRegisterReflect,
		NewRevenueServiceClient,
		NewTrackingServiceClient,

		order_service.NewRegister,

		// cli laen
		NewSetAuthorization,
		NewCreateTokenFromUsername,
		NewHelper,
		NewOrderShipped,

		NewApi,
		NewApp,
	)
	return &cli.Command{}, nil
}
