package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type ResourceRepository struct {
	DB *gorm.DB
}

func NewResourceRepository(db *gorm.DB) *ResourceRepository {
	return &ResourceRepository{DB: db}
}

func (r *ResourceRepository) Create(resource *model.Resource) error {
	return r.DB.Create(resource).Error
}

func (r *ResourceRepository) FindByID(id uint) (*model.Resource, error) {
	var resource model.Resource
	err := r.DB.First(&resource, id).Error
	return &resource, err
}

func (r *ResourceRepository) FindByModule(moduleType string) ([]model.Resource, error) {
	var resources []model.Resource
	err := r.DB.Where("module_type = ?", moduleType).Find(&resources).Error
	return resources, err
}
func (r *ResourceRepository) FindByModuleType(moduleType string) ([]model.Resource, error) {
	return r.FindByModule(moduleType)
}
func (r *ResourceRepository) IncrementViewCount(id uint) error {
	return r.DB.Model(&model.Resource{}).
		Where("id = ?", id).
		Update("view_count", gorm.Expr("view_count + 1")).
		Error
}

// 获取推荐资源
func (r *ResourceRepository) FindRecommended(userID uint, limit int) ([]model.Resource, error) {
	var resources []model.Resource
	// 基于用户兴趣或热门资源
	err := r.DB.Order("view_count DESC").Limit(limit).Find(&resources).Error
	return resources, err
}
