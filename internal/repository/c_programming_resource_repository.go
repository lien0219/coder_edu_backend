package repository

import (
	"coder_edu_backend/internal/model"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CProgrammingResourceRepository 处理C语言编程资源分类模块的数据访问

type CProgrammingResourceRepository struct {
	DB *gorm.DB
}

func NewCProgrammingResourceRepository(db *gorm.DB) *CProgrammingResourceRepository {
	return &CProgrammingResourceRepository{DB: db}
}

// Create 创建新的C语言资源分类模块
func (r *CProgrammingResourceRepository) Create(resource *model.CProgrammingResource) error {
	return r.DB.Create(resource).Error
}

// Update 更新C语言资源分类模块
func (r *CProgrammingResourceRepository) Update(resource *model.CProgrammingResource) error {
	return r.DB.Model(&model.CProgrammingResource{}).
		Where("id = ?", resource.ID).
		Updates(map[string]interface{}{
			"name":        resource.Name,
			"icon_url":    resource.IconURL,
			"description": resource.Description,
			"enabled":     resource.Enabled,
			"order":       resource.Order,
			"updated_at":  time.Now(),
		}).Error
}

// Delete 删除C语言资源分类模块
func (r *CProgrammingResourceRepository) Delete(id uint) error {
	return r.DB.Delete(&model.CProgrammingResource{}, id).Error
}

// FindByID 根据ID查找C语言资源分类模块
func (r *CProgrammingResourceRepository) FindByID(id uint) (*model.CProgrammingResource, error) {
	var resource model.CProgrammingResource
	err := r.DB.First(&resource, id).Error
	return &resource, err
}

// FindAll 查找所有C语言资源分类模块，支持分页、筛选、搜索和排序
func (r *CProgrammingResourceRepository) FindAll(page, limit int, search string, enabled *bool, sortBy, sortOrder string) ([]model.CProgrammingResource, int, error) {
	var resources []model.CProgrammingResource
	var total int64

	query := r.DB.Model(&model.CProgrammingResource{})

	// 搜索功能
	if search != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 筛选功能
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	// 计算总数
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	orderField := "`order`"
	fieldMap := map[string]string{
		"name":      "name",
		"order":     "`order`",
		"createdAt": "created_at",
		"updatedAt": "updated_at",
	}

	if dbField, exists := fieldMap[sortBy]; exists {
		orderField = dbField
	}

	orderDirection := "ASC"
	if sortOrder == "desc" {
		orderDirection = "DESC"
	}

	query = query.Order(orderField + " " + orderDirection)

	// 分页功能
	offset := (page - 1) * limit
	err = query.Offset(offset).Limit(limit).Find(&resources).Error

	// 添加调试信息
	fmt.Printf("[DEBUG] CProgrammingResourceRepository.FindAll - 找到 %d 个资源模块:\n", len(resources))
	for i, res := range resources {
		fmt.Printf("[DEBUG] 资源[%d] - ID: %d, 名称: %s\n", i, res.ID, res.Name)
	}

	return resources, int(total), err
}

// ExerciseCategoryRepository 处理练习题分类的数据访问

type ExerciseCategoryRepository struct {
	DB *gorm.DB
}

func NewExerciseCategoryRepository(db *gorm.DB) *ExerciseCategoryRepository {
	return &ExerciseCategoryRepository{DB: db}
}

// Create 创建新的练习题分类
func (r *ExerciseCategoryRepository) Create(category *model.ExerciseCategory) error {
	return r.DB.Create(category).Error
}

// FindByResourceID 根据资源ID查找练习题分类
func (r *ExerciseCategoryRepository) FindByResourceID(resourceID uint) ([]model.ExerciseCategory, error) {
	var categories []model.ExerciseCategory
	err := r.DB.Where("c_programming_res_id = ?", resourceID).Find(&categories).Error
	return categories, err
}

func (r *ExerciseCategoryRepository) UpdateFields(id uint, updates map[string]interface{}) error {
	return r.DB.Model(&model.ExerciseCategory{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除练习题分类
func (r *ExerciseCategoryRepository) Delete(id uint) error {
	return r.DB.Delete(&model.ExerciseCategory{}, id).Error
}

// ExerciseQuestionRepository 处理练习题题目的数据访问

type ExerciseQuestionRepository struct {
	DB *gorm.DB
}

func NewExerciseQuestionRepository(db *gorm.DB) *ExerciseQuestionRepository {
	return &ExerciseQuestionRepository{DB: db}
}

// Create 创建新的练习题题目
func (r *ExerciseQuestionRepository) Create(question *model.ExerciseQuestion) error {
	return r.DB.Create(question).Error
}

// Delete 删除练习题题目
func (r *ExerciseQuestionRepository) Delete(id uint) error {
	return r.DB.Delete(&model.ExerciseQuestion{}, id).Error
}

// FindByCategoryID 根据分类ID查找练习题题目，支持分页
func (r *ExerciseQuestionRepository) FindByCategoryID(categoryID uint, page, limit int) ([]model.ExerciseQuestion, int, error) {
	var questions []model.ExerciseQuestion
	var total int64

	err := r.DB.Model(&model.ExerciseQuestion{}).Where("category_id = ?", categoryID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err = r.DB.Where("category_id = ?", categoryID).Offset(offset).Limit(limit).Find(&questions).Error

	return questions, int(total), err
}

func (r *ExerciseQuestionRepository) UpdateFields(id uint, updates map[string]interface{}) error {
	return r.DB.Model(&model.ExerciseQuestion{}).Where("id = ?", id).Updates(updates).Error
}

func (r *ExerciseQuestionRepository) FindByID(id uint) (*model.ExerciseQuestion, error) {
	var question model.ExerciseQuestion
	err := r.DB.First(&question, id).Error
	return &question, err
}

func (r *ExerciseQuestionRepository) UpdateQuestion(question *model.ExerciseQuestion) error {
	return r.DB.Save(question).Error
}

func (r *ExerciseQuestionRepository) FindAllByCategoryID(categoryID uint) ([]model.ExerciseQuestion, error) {
	var questions []model.ExerciseQuestion
	err := r.DB.Where("category_id = ?", categoryID).Find(&questions).Error
	return questions, err
}

func (r *ExerciseQuestionRepository) FindQuestionsByCategoryIDWithPagination(categoryID uint, page, limit int) ([]model.ExerciseQuestion, int, error) {
	return r.FindByCategoryID(categoryID, page, limit)
}
