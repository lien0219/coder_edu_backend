package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type LearningPathRepository struct {
	DB *gorm.DB
}

func NewLearningPathRepository(db *gorm.DB) *LearningPathRepository {
	return &LearningPathRepository{DB: db}
}

func (r *LearningPathRepository) CreateMaterial(material *model.LearningPathMaterial) error {
	return r.DB.Create(material).Error
}

func (r *LearningPathRepository) FindMaterialByID(id string) (*model.LearningPathMaterial, error) {
	var m model.LearningPathMaterial
	err := r.DB.Where("id = ?", id).First(&m).Error
	return &m, err
}

func (r *LearningPathRepository) ListMaterials(level int, page, limit int) ([]model.LearningPathMaterial, int64, error) {
	var ms []model.LearningPathMaterial
	var total int64
	query := r.DB.Model(&model.LearningPathMaterial{})
	if level > 0 {
		query = query.Where("level = ?", level)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * limit
	err := query.Order("chapter_number asc, created_at desc").Offset(offset).Limit(limit).Find(&ms).Error
	return ms, total, err
}

func (r *LearningPathRepository) UpdateMaterial(material *model.LearningPathMaterial) error {
	return r.DB.Save(material).Error
}

func (r *LearningPathRepository) DeleteMaterial(id string) error {
	return r.DB.Where("id = ?", id).Delete(&model.LearningPathMaterial{}).Error
}
