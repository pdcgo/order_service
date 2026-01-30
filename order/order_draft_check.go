package order

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/order_iface/v1"
)

// OrderDraftCheck implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderDraftCheck(
	ctx context.Context,
	req *connect.Request[order_iface.OrderDraftCheckRequest],
) (*connect.Response[order_iface.OrderDraftCheckResponse], error) {
	var err error

	pay := req.Msg
	db := o.db.WithContext(ctx)

	result := order_iface.OrderDraftCheckResponse{
		Data: map[string]*order_iface.DraftCheckItem{},
	}

	for _, p := range pay.OrderRefIds {
		result.Data[p] = &order_iface.DraftCheckItem{
			OrderRefId: p,
			IsExist:    false,
		}
	}

	draftList := []*DraftOrder{}
	err = db.
		Model(&DraftOrder{}).
		Select([]string{
			"id",
			"order_ref_id",
			"team_id",
		}).
		Where("team_id = ?", pay.TeamId).
		Where("order_ref_id IN ?", pay.OrderRefIds).
		Find(&draftList).
		Error

	if err != nil {
		return nil, err
	}

	for _, draft := range draftList {
		result.Data[draft.OrderRefID].IsExist = true
	}

	return connect.NewResponse(&result), nil

}
