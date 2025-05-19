package order_mutation_test

import (
	"testing"

	"github.com/pdcgo/order_service/order_mutation"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestTagMutation(t *testing.T) {
	var db gorm.DB
	var migration moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&db_models.Order{},
			&db_models.OrderTag{},
			&db_models.OrderTagRelation{},
		)
		assert.Nil(t, err)
		return nil
	}

	var seed moretest.SetupFunc = func(t *testing.T) func() error {
		orders := []*db_models.Order{
			{
				ID: 1,
			},
		}

		err := db.Save(&orders).Error
		assert.Nil(t, err)
		return nil
	}

	moretest.Suite(t, "testing tagging",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migration,
			seed,
		},
		func(t *testing.T) {
			tagmut := order_mutation.NewTagMutation(&db)

			t.Run("testing add tag", func(t *testing.T) {
				err := tagmut.Add(db_models.RelationFromTracking, []uint{1}, []string{"ditolak_pembeli", "selesai", "return"})
				assert.Nil(t, err)

				t.Run("testing harus ada 3 relasi", func(t *testing.T) {
					relates := []*db_models.OrderTagRelation{}
					err := db.Model(&db_models.OrderTagRelation{}).Find(&relates).Error
					assert.Nil(t, err)

					assert.Equal(t, 3, len(relates))
				})

				t.Run("coba salah satu tag conflict", func(t *testing.T) {
					err := tagmut.Add(db_models.RelationFromTracking, []uint{1}, []string{"ditolak_pembeli", "test"})
					assert.Nil(t, err)

					t.Run("testing harus ada 4 relasi", func(t *testing.T) {
						relates := []*db_models.OrderTagRelation{}
						err := db.Model(&db_models.OrderTagRelation{}).Find(&relates).Error
						assert.Nil(t, err)

						assert.Equal(t, 4, len(relates))
					})
				})

			})

			t.Run("testing delete tag", func(t *testing.T) {
				tagmut.Remove(db_models.RelationFromTracking, []uint{1}, []string{"test"})
				t.Run("testing harus ada 3 relasi", func(t *testing.T) {
					relates := []*db_models.OrderTagRelation{}
					err := db.Model(&db_models.OrderTagRelation{}).Find(&relates).Error
					assert.Nil(t, err)

					assert.Equal(t, 3, len(relates))
				})
			})

			t.Run("delete all", func(t *testing.T) {
				err := tagmut.RemoveAllFrom(db_models.RelationFromTracking, []uint{1})
				assert.Nil(t, err)

				t.Run("testing harus ada 0 relasi", func(t *testing.T) {
					relates := []*db_models.OrderTagRelation{}
					err := db.Model(&db_models.OrderTagRelation{}).Find(&relates).Error
					assert.Nil(t, err)

					assert.Equal(t, 0, len(relates))
				})
			})

		},
	)
}
