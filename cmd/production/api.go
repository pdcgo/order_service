package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pdcgo/order_service"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/urfave/cli/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type ApiFunc cli.ActionFunc

func NewApi(
	mux *http.ServeMux,
	orderRegister order_service.RegisterHandler,
	reflectRegister custom_connect.RegisterReflectFunc,
) ApiFunc {
	return func(ctx context.Context, c *cli.Command) error {
		cancel, err := custom_connect.InitTracer("order-service")
		if err != nil {
			return err
		}

		defer cancel(context.Background())

		// register api

		var grpcReflectNames []string
		grpcReflectNames = append(grpcReflectNames, orderRegister()...)

		reflectRegister(grpcReflectNames)

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
	}
}
