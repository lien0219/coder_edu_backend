package model

import (
	"time"

	"gorm.io/gorm"
)

// ResourceCompletion 记录用户对资源的完成状态
// swagger:model ResourceCompletion
type ResourceCompletion struct {
	gorm.Model
	ID          uint `gorm:"primaryKey"`
	UserID      uint `gorm:"index:idx_user_resource,unique"`
	ResourceID  uint `gorm:"index:idx_user_resource,unique"`
	Completed   bool `gorm:"default:false"`
	CompletedAt *time.Time
}

func (ResourceCompletion) TableName() string {
	return "resource_completions"
}
