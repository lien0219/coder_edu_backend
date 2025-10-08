package repository

import (
	"coder_edu_backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type ResourceCompletionRepository struct {
	DB *gorm.DB
}

func NewResourceCompletionRepository(db *gorm.DB) *ResourceCompletionRepository {
	return &ResourceCompletionRepository{DB: db}
}

// GetCompletionStatus 获取用户对指定资源的完成状态
func (r *ResourceCompletionRepository) GetCompletionStatus(userID, resourceID uint) (bool, error) {
	var completion model.ResourceCompletion
	err := r.DB.Where("user_id = ? AND resource_id = ?", userID, resourceID).First(&completion).Error
	if err != nil {
		// 如果记录不存在，则表示未完成
		return false, nil
	}
	return completion.Completed, nil
}

// UpdateCompletionStatus 更新用户对资源的完成状态
func (r *ResourceCompletionRepository) UpdateCompletionStatus(userID, resourceID uint, completed bool) error {
	tx := r.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查是否已存在记录
	var existing model.ResourceCompletion
	err := tx.Where("user_id = ? AND resource_id = ?", userID, resourceID).First(&existing).Error

	now := time.Now()

	if err != nil {
		// 创建新记录
		completion := &model.ResourceCompletion{
			UserID:      userID,
			ResourceID:  resourceID,
			Completed:   completed,
			CompletedAt: &now,
		}
		err = tx.Create(completion).Error
	} else {
		// 更新现有记录
		existing.Completed = completed
		if completed {
			existing.CompletedAt = &now
		} else {
			existing.CompletedAt = nil
		}
		err = tx.Save(&existing).Error
	}

	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// GetUserResourceCompletions 获取用户对一组资源的完成状态
func (r *ResourceCompletionRepository) GetUserResourceCompletions(userID uint, resourceIDs []uint) (map[uint]bool, error) {
	var completions []model.ResourceCompletion
	err := r.DB.Where("user_id = ? AND resource_id IN ?", userID, resourceIDs).Find(&completions).Error
	if err != nil {
		return nil, err
	}

	statusMap := make(map[uint]bool)
	for _, completion := range completions {
		statusMap[completion.ResourceID] = completion.Completed
	}

	return statusMap, nil
}
