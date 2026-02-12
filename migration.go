package order_service

import "gorm.io/gorm"

type MigrationFunc func(db *gorm.DB) error

func NewMigration() MigrationFunc {
	return func(db *gorm.DB) error {
		return nil
	}
}
