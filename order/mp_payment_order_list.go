package order

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderAdjustmentList []*db_models.OrderAdjustment

func (l OrderAdjustmentList) ToProto() []*order_iface.PaymentOrderItem {
	result := make([]*order_iface.PaymentOrderItem, len(l))
	for i, item := range l {
		result[i] = &order_iface.PaymentOrderItem{
			Id:            uint64(item.ID),
			OrderId:       uint64(item.OrderID),
			ShopId:        uint64(item.MpID),
			IsMultiRegion: item.IsMultiRegion,
			Type:          string(item.Type),
			Amount:        item.Amount,
			Desc:          item.Desc,
			Source:        item.Source,
			At:            timestamppb.New(item.At),
			FundAt:        timestamppb.New(item.FundAt),
		}
	}
	return result
}

// MpPaymentOrderList implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) MpPaymentOrderList(ctx context.Context, req *connect.Request[order_iface.MpPaymentOrderListRequest]) (*connect.Response[order_iface.MpPaymentOrderListResponse], error) {
	var err error
	pay := req.Msg
	db := o.db.WithContext(ctx)

	result := order_iface.MpPaymentOrderListResponse{
		Items: []*order_iface.PaymentOrderItem{},
	}

	list := OrderAdjustmentList{}

	err = db.
		Model(&db_models.OrderAdjustment{}).
		Where("order_id = ?", pay.OrderId).
		Find(&list).
		Error

	if err != nil {
		return nil, err
	}

	result.Items = list.ToProto()

	return connect.NewResponse(&result), nil

}
