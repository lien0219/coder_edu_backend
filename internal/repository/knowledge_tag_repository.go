package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type KnowledgeTagRepository struct {
	DB *gorm.DB
}

func NewKnowledgeTagRepository(db *gorm.DB) *KnowledgeTagRepository {
	return &KnowledgeTagRepository{DB: db}
}

func (r *KnowledgeTagRepository) FindAll() ([]model.KnowledgeTag, error) {
	var tags []model.KnowledgeTag
	err := r.DB.Order("`order` asc").Find(&tags).Error
	return tags, err
}

func (r *KnowledgeTagRepository) FindByIDs(ids []uint) ([]model.KnowledgeTag, error) {
	var tags []model.KnowledgeTag
	if len(ids) == 0 {
		return tags, nil
	}
	err := r.DB.Where("id IN ?", ids).Find(&tags).Error
	return tags, err
}
