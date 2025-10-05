package repository

import (
	"coder_edu_backend/internal/model"
	"fmt"

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

	fmt.Printf("Creating resource: ModuleID=%d, ModuleType=%s, UploaderID=%d\n",
		resource.ModuleID, resource.ModuleType, resource.UploaderID)

	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET FOREIGN_KEY_CHECKS=0").Error; err != nil {
			fmt.Printf("Failed to disable foreign key checks: %v\n", err)
			return err
		}

		if err := tx.Create(resource).Error; err != nil {
			fmt.Printf("Error creating resource: %v\n", err)
			return err
		}

		if err := tx.Exec("SET FOREIGN_KEY_CHECKS=1").Error; err != nil {
			fmt.Printf("Failed to enable foreign key checks: %v\n", err)
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
