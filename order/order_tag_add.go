package order

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/shared/db_models"
)

// OrderTagAdd implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderTagAdd(
	ctx context.Context,
	req *connect.Request[order_iface.OrderTagAddRequest],
) (*connect.Response[order_iface.OrderTagAddResponse], error) {
	var err error
	pay := req.Msg

	err = o.
		auth.
		AuthIdentityFromHeader(req.Header()).
		Err()

	if err != nil {
		return nil, err
	}

	db := o.db.WithContext(ctx)

	for _, tagp := range pay.Tags {
		tag := db_models.OrderTag{
			Name: tagp.Value,
		}

		err = db.
			Model(&db_models.OrderTag{}).
			Where("name = ?", tag.Name).
			Find(&tag).
			Error

		if err != nil {
			return nil, err
		}

		if tag.ID == 0 {
			err = db.Save(&tag).Error
			if err != nil {
				return nil, err
			}
		}

		rel := &db_models.OrderTagRelation{
			OrderID:      uint(pay.OrderId),
			OrderTagID:   tag.ID,
			RelationFrom: order_iface.TagType_name[int32(tagp.Type)],
		}
		err = db.Save(rel).Error
		if err != nil {
			return nil, err
		}
	}

	return &connect.Response[order_iface.OrderTagAddResponse]{}, nil

}
