package report

import (
	"gorm.io/gorm"
)

type orderReportServiceImpl struct {
	db *gorm.DB
}

func NewOrderReportService(db *gorm.DB) *orderReportServiceImpl {
	return &orderReportServiceImpl{
		db,
	}
}
