package repository

import (
	"coder_edu_backend/internal/model"
	"time"

	"gorm.io/gorm"
)

// GoalRepository 处理学习目标的数据访问

type GoalRepository struct {
	DB *gorm.DB
}

func NewGoalRepository(db *gorm.DB) *GoalRepository {
	return &GoalRepository{DB: db}
}

// Create 创建新的学习目标
func (r *GoalRepository) Create(goal *model.Goal) error {
	return r.DB.Create(goal).Error
}

// Update 更新学习目标
func (r *GoalRepository) Update(goal *model.Goal) error {
	return r.DB.Model(&model.Goal{}).
		Where("id = ?", goal.ID).
		Updates(map[string]interface{}{
			"title":                goal.Title,
			"description":          goal.Description,
			"status":               goal.Status,
			"current":              goal.Current,
			"target":               goal.Target,
			"progress":             goal.Progress,
			"target_date":          goal.TargetDate,
			"goal_type":            goal.GoalType,
			"resource_module_id":   goal.ResourceModuleID,
			"resource_module_name": goal.ResourceModuleName,
			"updated_at":           time.Now(),
		}).Error
}

// Delete 删除学习目标
func (r *GoalRepository) Delete(id uint) error {
	return r.DB.Delete(&model.Goal{}, id).Error
}

// FindByID 根据ID查找学习目标
func (r *GoalRepository) FindByID(id uint) (*model.Goal, error) {
	var goal model.Goal
	err := r.DB.First(&goal, id).Error
	return &goal, err
}

// FindByUserID 获取用户的所有学习目标
func (r *GoalRepository) FindByUserID(userID uint) ([]model.Goal, error) {
	var goals []model.Goal
	err := r.DB.Where("user_id = ?", userID).Order("target_date").Find(&goals).Error
	return goals, err
}

// FindByUserIDAndStatus 获取用户特定状态的学习目标
func (r *GoalRepository) FindByUserIDAndStatus(userID uint, status model.GoalStatus) ([]model.Goal, error) {
	var goals []model.Goal
	err := r.DB.Where("user_id = ? AND status = ?", userID, status).Order("target_date").Find(&goals).Error
	return goals, err
}

// FindByUserIDAndGoalType 获取用户特定类型的学习目标
func (r *GoalRepository) FindByUserIDAndGoalType(userID uint, goalType model.GoalType) ([]model.Goal, error) {
	var goals []model.Goal
	err := r.DB.Where("user_id = ? AND goal_type = ?", userID, goalType).Order("target_date").Find(&goals).Error
	return goals, err
}

// FindByUserIDAndResourceModuleID 获取用户与特定资源模块关联的学习目标
func (r *GoalRepository) FindByUserIDAndResourceModuleID(userID, resourceModuleID uint) ([]model.Goal, error) {
	var goals []model.Goal
	err := r.DB.Where("user_id = ? AND resource_module_id = ?", userID, resourceModuleID).Order("target_date").Find(&goals).Error
	return goals, err
}

// FindByIDAndUserID 根据ID和用户ID查找学习目标
func (r *GoalRepository) FindByIDAndUserID(id, userID uint) (*model.Goal, error) {
	var goal model.Goal
	err := r.DB.Where("id = ? AND user_id = ?", id, userID).First(&goal).Error
	return &goal, err
}
