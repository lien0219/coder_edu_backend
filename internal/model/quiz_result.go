package model

import (
	"time"

	"gorm.io/gorm"
)

// QuizResult 存储用户的测验结果
type QuizResult struct {
	gorm.Model
	ID          uint         `gorm:"primaryKey"`
	UserID      uint         `gorm:"index"`
	QuizID      uint         `gorm:"index"`
	Score       int          `gorm:"not null"`
	Total       int          `gorm:"not null"`
	Answers     map[uint]int `gorm:"type:json"`     // 答案字段
	Completed   bool         `gorm:"default:false"` // 完成状态字段
	CompletedAt time.Time
}

func (QuizResult) TableName() string {
	return "quiz_results"
}
