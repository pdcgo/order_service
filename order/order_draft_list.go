package order

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// OrderDraftList implements order_ifaceconnect.OrderServiceHandler.
func (o *orderServiceImpl) OrderDraftList(
	ctx context.Context,
	req *connect.Request[order_iface.OrderDraftListRequest],
) (*connect.Response[order_iface.OrderDraftListResponse], error) {
	var err error

	res := order_iface.OrderDraftListResponse{}
	pay := req.Msg

	indentity := o.
		auth.
		AuthIdentityFromHeader(req.Header())

	// agent := indentity.Identity()

	err = indentity.Err()
	if err != nil {
		return connect.NewResponse(&res), err
	}

	source, err := custom_connect.GetRequestSource(ctx)
	if err != nil {
		return nil, err
	}

	var domainId uint

	switch source.RequestFrom {
	case access_iface.RequestFrom_REQUEST_FROM_SELLING:
		domainId = uint(pay.TeamId)
	case access_iface.RequestFrom_REQUEST_FROM_ADMIN:
		domainId = uint(source.TeamId)
	default:
		domainId = uint(pay.TeamId)
	}

	err = indentity.HasPermission(authorization_iface.CheckPermissionGroup{
		&DraftOrder{}: &authorization_iface.CheckPermission{
			DomainID: domainId,
			Actions:  []authorization_iface.Action{authorization_iface.Read},
		},
	}).
		Err()

	if err != nil {
		return connect.NewResponse(&res), err
	}

	db := o.db.WithContext(ctx)

	_, err = db_connect.NewQueryChain(db,
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter user id dan teamid
				query = query.
					Model(&DraftOrder{})

				if pay.TeamId != 0 {
					query = query.Where("team_id = ?", pay.TeamId)
				}

				if pay.UserId != 0 {
					query = query.Where("user_id = ?", pay.UserId)
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // filter time range
			return func(query *gorm.DB) (*gorm.DB, error) {

				if pay.TimeRange == nil {
					return next(query)
				}

				trange := pay.TimeRange
				if trange.StartDate.IsValid() {
					query = query.Where("created >= ?", trange.StartDate.AsTime())

				}

				if trange.EndDate.IsValid() {
					query = query.Where("created <= ?", trange.EndDate.AsTime())
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // filter marketplace or shop
			return func(query *gorm.DB) (*gorm.DB, error) {
				if pay.ShopId != 0 {
					query = query.Where("order_mp_id = ?", pay.ShopId)
					return next(query)
				}

				switch pay.Marketplace {
				case common.MarketplaceType_MARKETPLACE_TYPE_CUSTOM:
					query = query.Where("order_from = ?", "custom")
				case common.MarketplaceType_MARKETPLACE_TYPE_LAZADA:
					query = query.Where("order_from = ?", "lazada")
				case common.MarketplaceType_MARKETPLACE_TYPE_MENGANTAR:
					query = query.Where("order_from = ?", "mengantar")
				case common.MarketplaceType_MARKETPLACE_TYPE_SHOPEE:
					query = query.Where("order_from = ?", "shopee")
				case common.MarketplaceType_MARKETPLACE_TYPE_TIKTOK:
					query = query.Where("order_from = ?", "tiktok")
				case common.MarketplaceType_MARKETPLACE_TYPE_TOKOPEDIA:
					query = query.Where("order_from = ?", "tokopedia")
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // filter query
			return func(query *gorm.DB) (*gorm.DB, error) {
				if pay.Search == nil {
					return next(query)
				}

				search := pay.Search
				keyword := search.Q
				switch search.SearchType {
				case order_iface.DraftSearchType_DRAFT_SEARCH_TYPE_ORDER_REFID:
					query = query.Where("order_ref_id LIKE ?", "%"+keyword+"%")
				case order_iface.DraftSearchType_DRAFT_SEARCH_TYPE_RECEIPT:
					query = query.Where("order_payload->>'receipt' LIKE ?", "%"+keyword+"%")
				}

				return next(query)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // set paginated
			return func(query *gorm.DB) (*gorm.DB, error) {
				var queryPaginated *gorm.DB
				queryPaginated, res.PageInfo, err = db_connect.SetPaginationQuery(db, func() (*gorm.DB, error) {
					return query.Session(&gorm.Session{}), nil

				}, pay.Page)

				if err != nil {
					return query, err
				}

				return next(
					queryPaginated,
				)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) {

				var datas []*DraftOrder
				err = query.Find(&datas).Error

				if err != nil {
					return nil, err
				}

				res.Items = make([]*order_iface.DraftItem, len(datas))
				for i, data := range datas {
					res.Items[i] = &order_iface.DraftItem{
						Id:         uint64(data.ID),
						UserId:     uint64(data.UserID),
						TeamId:     uint64(data.TeamID),
						MpProducts: data.MpProducts,
						Payload:    data.OrderPayload.Data(),
						Created:    timestamppb.New(data.Created),
					}
				}

				return nil, nil
			}
		},
	)

	return connect.NewResponse(&res), err
}
