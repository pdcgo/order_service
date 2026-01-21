package order_mock

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/schema/services/order_iface/v1/order_ifaceconnect"
)

type orderServiceMock struct {
}

// OrderReturnArrived implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderReturnArrived(context.Context, *connect.Request[order_iface.OrderReturnArrivedRequest]) (*connect.Response[order_iface.OrderReturnArrivedResponse], error) {
	panic("unimplemented")
}

// OrderDraftList implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderDraftList(context.Context, *connect.Request[order_iface.OrderDraftListRequest]) (*connect.Response[order_iface.OrderDraftListResponse], error) {
	panic("unimplemented")
}

// OrderDraftGet implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderDraftGet(context.Context, *connect.Request[order_iface.OrderDraftGetRequest]) (*connect.Response[order_iface.OrderDraftGetResponse], error) {
	panic("unimplemented")
}

// OrderCreate implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderCreate(context.Context, *connect.Request[order_iface.OrderCreateRequest]) (*connect.Response[order_iface.OrderCreateResponse], error) {
	panic("unimplemented")
}

// OrderDraftDelete implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderDraftDelete(context.Context, *connect.Request[order_iface.OrderDraftDeleteRequest]) (*connect.Response[order_iface.OrderDraftDeleteResponse], error) {
	panic("unimplemented")
}

// OrderDraftCreate implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderDraftCreate(context.Context, *connect.Request[order_iface.OrderDraftCreateRequest]) (*connect.Response[order_iface.OrderDraftCreateResponse], error) {
	panic("unimplemented")
}

// ChangeEstRevenue implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) ChangeEstRevenue(context.Context, *connect.Request[order_iface.ChangeEstRevenueRequest]) (*connect.Response[order_iface.ChangeEstRevenueResponse], error) {
	panic("unimplemented")
}

// OrderCompleted implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderCompleted(context.Context, *connect.Request[order_iface.OrderCompletedRequest]) (*connect.Response[order_iface.OrderCompletedResponse], error) {
	panic("unimplemented")
}

// ChangeOrderRefID implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) ChangeOrderRefID(context.Context, *connect.Request[order_iface.ChangeOrderRefIDRequest]) (*connect.Response[order_iface.ChangeOrderRefIDResponse], error) {
	panic("unimplemented")
}

// MpPaymentCreate implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) MpPaymentCreate(context.Context, *connect.Request[order_iface.MpPaymentCreateRequest]) (*connect.Response[order_iface.MpPaymentCreateResponse], error) {
	panic("unimplemented")
}

// MpPaymentDelete implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) MpPaymentDelete(context.Context, *connect.Request[order_iface.MpPaymentDeleteRequest]) (*connect.Response[order_iface.MpPaymentDeleteResponse], error) {
	panic("unimplemented")
}

// MpPaymentOrderList implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) MpPaymentOrderList(context.Context, *connect.Request[order_iface.MpPaymentOrderListRequest]) (*connect.Response[order_iface.MpPaymentOrderListResponse], error) {
	panic("unimplemented")
}

// OrderTagAdd implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderTagAdd(context.Context, *connect.Request[order_iface.OrderTagAddRequest]) (*connect.Response[order_iface.OrderTagAddResponse], error) {
	panic("unimplemented")
}

// OrderFundSet implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderFundSet(context.Context, *connect.ClientStream[order_iface.OrderFundSetRequest]) (*connect.Response[order_iface.OrderFundSetResponse], error) {
	return &connect.Response[order_iface.OrderFundSetResponse]{}, nil
}

// OrderList implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderList(context.Context, *connect.Request[order_iface.OrderListRequest], *connect.ServerStream[order_iface.OrderListResponse]) error {
	panic("unimplemented")
}

// OrderOverview implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderOverview(context.Context, *connect.Request[order_iface.OrderOverviewRequest]) (*connect.Response[order_iface.OrderOverviewResponse], error) {
	panic("unimplemented")
}

// OrderTagRemove implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceMock) OrderTagRemove(context.Context, *connect.Request[order_iface.OrderTagRemoveRequest]) (*connect.Response[order_iface.OrderTagRemoveResponse], error) {
	panic("unimplemented")
}

func NewOrderServiceMock() order_ifaceconnect.OrderServiceHandler {
	return &orderServiceMock{}
}
