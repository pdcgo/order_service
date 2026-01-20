package order

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
)

// OrderDraftDelete implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderDraftDelete(
	ctx context.Context,
	req *connect.Request[order_iface.OrderDraftDeleteRequest],
) (*connect.Response[order_iface.OrderDraftDeleteResponse], error) {
	var err error

	res := order_iface.OrderDraftDeleteResponse{}
	pay := req.Msg

	indentity := o.
		auth.
		AuthIdentityFromHeader(req.Header())

	agent := indentity.Identity()

	err = indentity.Err()
	if err != nil {
		return connect.NewResponse(&res), err
	}

	err = indentity.HasPermission(authorization_iface.CheckPermissionGroup{
		&DraftOrder{}: &authorization_iface.CheckPermission{
			DomainID: uint(pay.TeamId),
			Actions:  []authorization_iface.Action{authorization_iface.Delete},
		},
	}).Err()

	if err != nil {
		return connect.NewResponse(&res), err
	}

	db := o.db.WithContext(ctx)

	err = db.
		Model(&DraftOrder{}).
		Where("team_id = ?", pay.TeamId).
		Where("user_id = ?", agent.GetUserID()).
		Where("id = ?", pay.DraftId).
		Delete(&DraftOrder{}).
		Error

	if err != nil {
		return connect.NewResponse(&res), err
	}
	return connect.NewResponse(&res), nil
}
