package repository

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ResourceRepository struct {
	DB *gorm.DB
}

func NewResourceRepository(db *gorm.DB) *ResourceRepository {
	return &ResourceRepository{DB: db}
}

func (r *ResourceRepository) Create(resource *model.Resource) error {

	// return r.DB.Create(resource).Error

	logger.Log.Info("Creating resource",
		zap.Uint("ModuleID", resource.ModuleID),
		zap.String("ModuleType", resource.ModuleType),
		zap.Uint("UploaderID", resource.UploaderID))

	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET FOREIGN_KEY_CHECKS=0").Error; err != nil {
			logger.Log.Error("Failed to disable foreign key checks", zap.Error(err))
			return err
		}

		if err := tx.Create(resource).Error; err != nil {
			logger.Log.Error("Error creating resource", zap.Error(err))
			return err
		}

		if err := tx.Exec("SET FOREIGN_KEY_CHECKS=1").Error; err != nil {
			logger.Log.Error("Failed to enable foreign key checks", zap.Error(err))
			return err
		}

		return nil
	})
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

func (r *ResourceRepository) UpdateFields(id uint, resourceType model.ResourceType, updates map[string]interface{}) error {
	return r.DB.Model(&model.Resource{}).
		Where("id = ? AND type = ?", id, resourceType).
		Updates(updates).Error
}

func (r *ResourceRepository) DeleteByType(id uint, resourceType model.ResourceType) error {
	return r.DB.Where("id = ? AND type = ?", id, resourceType).Delete(&model.Resource{}).Error
}
