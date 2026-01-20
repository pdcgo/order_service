package order

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
)

// OrderDraftGet implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderDraftGet(
	ctx context.Context,
	req *connect.Request[order_iface.OrderDraftGetRequest],
) (*connect.Response[order_iface.OrderDraftGetResponse], error) {
	var err error

	res := order_iface.OrderDraftGetResponse{
		Data: &order_iface.DraftItem{},
	}
	pay := req.Msg

	indentity := o.
		auth.
		AuthIdentityFromHeader(req.Header())

	// agent := indentity.Identity()

	err = indentity.Err()
	if err != nil {
		return connect.NewResponse(&res), err
	}

	err = indentity.HasPermission(authorization_iface.CheckPermissionGroup{
		&DraftOrder{}: &authorization_iface.CheckPermission{
			DomainID: uint(pay.TeamId),
			Actions:  []authorization_iface.Action{authorization_iface.Read},
		},
	}).Err()

	if err != nil {
		return connect.NewResponse(&res), err
	}

	db := o.db.WithContext(ctx)
	var draft DraftOrder
	err = db.
		Model(&DraftOrder{}).
		Where("id = ?", pay.Id).
		First(&draft).
		Error

	if err != nil {
		return connect.NewResponse(&res), err
	}

	if draft.DraftVersion != "proto" {
		return connect.NewResponse(&res), errors.New("draft order is older version")
	}

	res.Data = &order_iface.DraftItem{
		Id:         uint64(draft.ID),
		TeamId:     uint64(draft.TeamID),
		MpProducts: draft.MpProducts,
		Payload:    draft.OrderPayload.Data(),
	}

	return connect.NewResponse(&res), err
}
