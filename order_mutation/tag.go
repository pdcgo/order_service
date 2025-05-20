package order_mutation

import (
	"errors"

	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/order_iface"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tagMutationImpl struct {
	db *gorm.DB
}

// Add implements order_iface.OrderTagMutation.
func (t *tagMutationImpl) Add(from db_models.RelationFrom, orderIDs []uint, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	err := t.validateTask(tags)
	if err != nil {
		return err
	}

	tagIds, err := t.getTagIds(tags)
	if err != nil {
		return err
	}

	relates := []*db_models.OrderTagRelation{}
	for _, ordID := range orderIDs {
		for _, tagID := range tagIds {
			relate := db_models.OrderTagRelation{
				OrderID:      ordID,
				OrderTagID:   tagID,
				RelationFrom: db_models.RelationFromTracking,
			}
			relates = append(relates, &relate)
		}
	}

	err = t.db.
		Clauses(
			clause.OnConflict{
				DoNothing: true,
			},
		).
		Save(&relates).
		Error

	if err != nil {
		return err
	}

	return nil
}

// Remove implements order_iface.OrderTagMutation.
func (t *tagMutationImpl) Remove(from db_models.RelationFrom, orderIDs []uint, tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	tagIds, err := t.getTagIds(tags)
	if err != nil {
		return err
	}

	err = t.db.
		Model(&db_models.OrderTagRelation{}).
		Where("relation_from = ?", from).
		Where("order_id in ?", orderIDs).
		Where("order_tag_id in ?", tagIds).
		Delete(&db_models.OrderTagRelation{}).
		Error

	return err
}

// RemoveAllFrom implements order_iface.OrderTagMutation.
func (t *tagMutationImpl) RemoveAllFrom(from db_models.RelationFrom, orderIDs []uint) error {
	err := t.db.
		Model(&db_models.OrderTagRelation{}).
		Where("relation_from = ?", from).
		Where("order_id in ?", orderIDs).
		Delete(&db_models.OrderTagRelation{}).
		Error

	return err
}

func (t *tagMutationImpl) getTagIds(tags []string) ([]uint, error) {
	hasil := []uint{}
	maptags := map[string]*db_models.OrderTag{}
	otags := []*db_models.OrderTag{}

	for _, t := range tags {
		maptags[t] = nil
	}

	err := t.db.Model(&db_models.OrderTag{}).Where("name in ?", tags).Find(&otags).Error
	if err != nil {
		return hasil, err
	}

	for _, dd := range otags {
		item := dd
		maptags[item.Name] = item
	}

	for key, tag := range maptags {
		if maptags[key] != nil {
			hasil = append(hasil, tag.ID)
			continue
		}

		tt := &db_models.OrderTag{
			Name: key,
		}
		err = t.db.Save(tt).Error
		if err != nil {
			return hasil, err
		}

		hasil = append(hasil, tt.ID)
	}
	return hasil, nil
}

func (t *tagMutationImpl) validateTask(tags []string) error {
	for _, tag := range tags {
		if len(tag) > 300 {
			return errors.New("tag kepanjangan")
		}
	}
	return nil
}

func NewTagMutation(db *gorm.DB) order_iface.OrderTagMutation {
	return &tagMutationImpl{
		db: db,
	}
}
