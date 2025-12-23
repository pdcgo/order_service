package order

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/schema/services/revenue_iface/v1/revenue_ifaceconnect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type orderServiceImpl struct {
	auth authorization_iface.Authorization
	db   *gorm.DB
	// revenueClient revenue_ifaceconnect.RevenueServiceClient
	// trackingService tracking_ifaceconnect.TrackingServiceClient
	revenueService revenue_ifaceconnect.RevenueServiceClient
}

// ChangeOrderRefID implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) ChangeOrderRefID(context.Context, *connect.Request[order_iface.ChangeOrderRefIDRequest]) (*connect.Response[order_iface.ChangeOrderRefIDResponse], error) {
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
	revenueService revenue_ifaceconnect.RevenueServiceClient,
	// trackingService tracking_ifaceconnect.TrackingServiceClient,
) *orderServiceImpl {
	return &orderServiceImpl{
		auth,
		db,
		revenueService,
		// trackingService: trackingService,
	}
}
