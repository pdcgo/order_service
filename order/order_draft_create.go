package order

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// OrderDraftCreate implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderDraftCreate(
	ctx context.Context,
	req *connect.Request[order_iface.OrderDraftCreateRequest],
) (*connect.Response[order_iface.OrderDraftCreateResponse], error) {
	var err error

	res := order_iface.OrderDraftCreateResponse{}
	pay := req.Msg
	createPay := pay.Payload

	indentity := o.
		auth.
		AuthIdentityFromHeader(req.Header())

	agent := indentity.Identity()

	err = indentity.Err()
	if err != nil {
		return connect.NewResponse(&res), err
	}

	err = indentity.HasPermission(authorization_iface.CheckPermissionGroup{
		&db_models.Order{}: &authorization_iface.CheckPermission{
			DomainID: uint(createPay.TeamId),
			Actions:  []authorization_iface.Action{authorization_iface.Create},
		},
	}).Err()

	if err != nil {
		return connect.NewResponse(&res), err
	}

	db := o.db.WithContext(ctx)

	// check order sudah ada
	var ord db_models.Order
	err = db.
		Model(&db_models.Order{}).
		Select("id").
		Where("team_id = ?", createPay.TeamId).
		Where("order_ref_id = ?", createPay.OrderRefId).
		Where("status != ?", db_models.OrdCancel).
		Find(&ord).
		Error

	if err != nil {
		return connect.NewResponse(&res), err
	}

	if ord.ID != 0 {
		return connect.NewResponse(&res), fmt.Errorf("cannot create draft because order %s is exists", createPay.OrderRefId)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		// getting marketplace
		var shop db_models.Marketplace
		err = tx.
			Model(&db_models.Marketplace{}).
			Where("team_id = ?", createPay.TeamId).
			Where("id = ?", createPay.OrderMpId).
			First(&shop).
			Error

		if err != nil {
			return err
		}

		// creating draft
		draft := &DraftOrder{
			TeamID:       uint(createPay.TeamId),
			UserID:       agent.GetUserID(),
			OrderRefID:   createPay.OrderRefId,
			DraftVersion: "proto",
			OrderMpID:    uint(createPay.OrderMpId),
			OrderTotal:   int(createPay.OrderTotal),
			OrderFrom:    db_models.OrderMpType(shop.MpType),
			Created:      time.Now(),
			MpProducts:   pay.MpProducts,
		}

		// deleting previous draft
		err = tx.
			Model(&DraftOrder{}).
			Where("order_ref_id = ?", draft.OrderRefID).
			Where("order_from = ?", shop.MpType).
			Delete(&DraftOrder{}).
			Error

		if err != nil {
			return err
		}

		err = tx.Save(&draft).Error
		if err != nil {
			return err
		}

		createPay.DraftId = uint64(draft.ID)
		if createPay.OrderDeadline.IsValid() {
			deadline := createPay.OrderDeadline.AsTime().Add(time.Minute * -1)
			createPay.OrderDeadline = timestamppb.New(deadline)
		}

		draft.OrderPayload = db_models.NewJSONType(createPay)

		err = tx.Save(&draft).Error
		if err != nil {
			return err
		}

		res.Id = uint64(draft.ID)

		return nil
	})

	return connect.NewResponse(&res), err
}

type MpProductItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type DraftOrder struct { // bakalan ada di database
	ID uint `json:"id"`

	TeamID       uint   `json:"team_id"`
	UserID       uint   `json:"user_id"`
	DraftVersion string `json:"draft_version"`

	OrderRefID   string                                          `json:"order_ref_id" gorm:"index"`
	OrderMpID    uint                                            `json:"order_mp_id"`
	OrderTotal   int                                             `json:"order_total"`
	OrderFrom    db_models.OrderMpType                           `json:"order_from"`
	OrderPayload db_models.JSONType[*order_iface.DraftOrderData] `json:"order_payload"`
	MpProducts   datatypes.JSONSlice[*order_iface.MpProductItem] `json:"mp_products"`
	Created      time.Time                                       `json:"created"`

	OrderMp *db_models.Marketplace `json:"order_mp"`
	Team    *db_models.Team        `json:"team"`
	User    *db_models.User        `json:"user"`
}

func (d *DraftOrder) GetEntityID() string {
	return "draft_order"
}
