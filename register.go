package order_service

import (
	"net/http"

	"github.com/pdcgo/order_service/order"
	"github.com/pdcgo/schema/services/order_iface/v1/order_ifaceconnect"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type RegisterHandler func()

func NewRegister(
	mux *http.ServeMux,
	db *gorm.DB,
	auth authorization_iface.Authorization,
	// trackingService tracking_ifaceconnect.TrackingServiceClient,
	defaultInterceptor custom_connect.DefaultInterceptor,
) RegisterHandler {
	return func() {
		path, handler := order_ifaceconnect.NewOrderServiceHandler(order.NewOrderService(
			auth,
			db,
			// trackingService,
		), defaultInterceptor)
		mux.Handle(path, handler)
	}
}
