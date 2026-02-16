package report

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/order_iface/v1"
	"github.com/pdcgo/shared/db_connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// DailyFundList implements order_ifaceconnect.OrderReportServiceHandler.
func (o *orderReportServiceImpl) DailyFundList(
	ctx context.Context,
	req *connect.Request[order_iface.DailyFundListRequest],
) (*connect.Response[order_iface.DailyFundListResponse], error) {
	var err error

	db := o.db.WithContext(ctx)

	pay := req.Msg
	result := order_iface.DailyFundListResponse{
		Data:     []*order_iface.HoldFundValue{},
		PageInfo: &common.PageInfo{},
	}

	_, err = db_connect.NewQueryChain(db,
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // selecting table
				switch kind := pay.Filter.(type) {
				case *order_iface.DailyFundListRequest_TeamFilter:
					fteam := kind.TeamFilter

					query = query.
						Table("stats.daily_team_holds d").
						Where("d.team_id = ?", fteam.TeamId)

					return next(query)

				case *order_iface.DailyFundListRequest_ShopFilter:
					fshop := kind.ShopFilter

					query := query.
						Table("stats.daily_shop_holds d").
						Where("d.shop_id = ?", fshop.ShopId)

					return next(query)

				default:
					return query, fmt.Errorf("filter not supported")
				}
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filtering timerange
				if pay.TimeRange == nil {
					return next(query)
				}

				trange := pay.TimeRange

				if trange.EndDate.IsValid() {
					query = query.Where("d.day <= ?",
						trange.EndDate.AsTime(),
					)
				}

				if trange.StartDate.IsValid() {
					query = query.Where("d.day >= ?",
						trange.StartDate.AsTime(),
					)
				}

				return next(query)

			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // day sort

				if pay.Sort == order_iface.DaySort_DAY_SORT_DESC {
					query = query.Order("d.day DESC")
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // set pagination
				var err error
				var paginated *gorm.DB

				paginated, result.PageInfo, err = db_connect.SetPaginationQuery(db, func() (*gorm.DB, error) {
					return query.Session(&gorm.Session{}), nil
				}, pay.Page)

				if err != nil {
					return nil, err

				}

				return next(paginated)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) {
				var err error

				switch pay.Filter.(type) {
				case *order_iface.DailyFundListRequest_TeamFilter:
					tempdata := []struct {
						SyncAt     time.Time
						Day        time.Time
						TeamID     uint64
						HoldCount  int64
						HoldAmount float64
					}{}

					err = query.
						Find(&tempdata).
						Error

					if err != nil {
						return query, err
					}
					for _, data := range tempdata {
						result.Data = append(result.Data, &order_iface.HoldFundValue{
							SyncAt:     timestamppb.New(data.SyncAt),
							Day:        timestamppb.New(data.Day),
							HoldCount:  data.HoldCount,
							HoldAmount: data.HoldAmount,
							Label: &order_iface.HoldFundValue_TeamId{
								TeamId: data.TeamID,
							},
						})
					}

					return next(query)

				case *order_iface.DailyFundListRequest_ShopFilter:
					tempdata := []struct {
						SyncAt     time.Time
						Day        time.Time
						ShopID     uint64
						HoldCount  int64
						HoldAmount float64
					}{}

					err = query.
						Find(&tempdata).
						Error

					if err != nil {
						return query, err
					}
					for _, data := range tempdata {
						result.Data = append(result.Data, &order_iface.HoldFundValue{
							SyncAt:     timestamppb.New(data.SyncAt),
							Day:        timestamppb.New(data.Day),
							HoldCount:  data.HoldCount,
							HoldAmount: data.HoldAmount,
							Label: &order_iface.HoldFundValue_ShopId{
								ShopId: data.ShopID,
							},
						})
					}

					return next(query)

				default:
					return query, fmt.Errorf("data model not supported")
				}
			}
		},
	)

	return connect.NewResponse(&result), err
}
