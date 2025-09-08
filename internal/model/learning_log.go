package model

import (
	"time"

	"gorm.io/gorm"
)

// LearningLog 记录用户的学习活动
type LearningLog struct {
	gorm.Model
	ID         uint     `gorm:"primaryKey"`
	UserID     uint     `gorm:"index;type:bigint unsigned"`
	ModuleID   uint     `gorm:"index;type:bigint unsigned"`
	Activity   string   `gorm:"type:text"`
	Content    string   `gorm:"type:text"` //内容字段
	Tags       []string `gorm:"type:json"` //标签字段
	Insights   []string `gorm:"type:json"` //见解字段
	Challenges []string `gorm:"type:json"` //挑战字段
	NextSteps  []string `gorm:"type:json"` //下一步字段
	Duration   int      `gorm:"default:0"`
	Completed  bool     `gorm:"default:false"`
	Score      int      `gorm:"default:0"`
	CreatedAt  time.Time
}

func (LearningLog) TableName() string {
	return "learning_logs"
}
