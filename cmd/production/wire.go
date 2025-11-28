//go:build wireinject
// +build wireinject

package main

import (
	"net/http"

	"github.com/google/wire"
	"github.com/pdcgo/order_service"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
)

func InitializeApp() (*App, error) {
	wire.Build(
		http.NewServeMux,
		configs.NewProductionConfig,
		NewDatabase,
		NewAuthorization,
		NewCache,
		// client
		custom_connect.NewDefaultInterceptor,

		order_service.NewRegister,
		NewApp,
	)
	return &App{}, nil
}
