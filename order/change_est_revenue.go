package order

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// ChangeEstRevenue implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) ChangeEstRevenue(ctx context.Context, req *connect.Request[order_iface.ChangeEstRevenueRequest]) (*connect.Response[order_iface.ChangeEstRevenueResponse], error) {
	var err error

	source, err := custom_connect.GetRequestSource(ctx)
	if err != nil {
		return nil, err
	}

	pay := req.Msg

	var domainID uint
	switch source.RequestFrom {
	case access_iface.RequestFrom_REQUEST_FROM_ADMIN:
		domainID = authorization.RootDomain
	default:
		domainID = uint(source.TeamId)
		if pay.TeamId != source.TeamId {
			return nil, fmt.Errorf("cannot cross team")
		}
	}

	identity := o.auth.
		AuthIdentityFromHeader(req.Header())

	// agent := identity.
	// 	Identity()

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.Order{}: &authorization_iface.CheckPermission{
				DomainID: domainID,
				Actions:  []authorization_iface.Action{authorization_iface.Update},
			},
		}).
		Err()

	if err != nil {
		return nil, err
	}

	db := o.db.WithContext(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		var ord db_models.Order
		err = tx.
			Model(&db_models.Order{}).
			Where("team_id = ?", pay.TeamId).
			Where("id = ?", pay.OrderId).
			First(&ord).
			Error

		if err != nil {
			return err
		}

		// filter hanya boleh diedit sekitar satu minggu
		if time.Since(ord.CreatedAt) > time.Hour*24*7 {
			return fmt.Errorf("order %s sudah lebih dari satu minggu", ord.OrderRefID)
		}

		err = tx.
			Model(&db_models.Order{}).
			Where("id = ?", pay.OrderId).
			Update("order_mp_total", pay.EstRevenueAmount).
			Error

		return err
	})

	if err != nil {
		return nil, err
	}

	// sending to edit receivable adjustment
	_, err = o.revenueService.OrderEditSellingReceivable(ctx, &connect.Request[revenue_iface.OrderEditSellingReceivableRequest]{
		Msg: &revenue_iface.OrderEditSellingReceivableRequest{
			TeamId:           pay.TeamId,
			OrderId:          pay.OrderId,
			EstRevenueAmount: pay.EstRevenueAmount,
		},
	})

	return &connect.Response[order_iface.ChangeEstRevenueResponse]{}, err
}
