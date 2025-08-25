package model

import (
	"time"

	"gorm.io/gorm"
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskProgress  TaskStatus = "in_progress"
	TaskCompleted TaskStatus = "completed"
)

type Task struct {
	gorm.Model
	ID          uint       `gorm:"primaryKey"`
	Title       string     `gorm:"size:255;not null"`
	Description string     `gorm:"type:text"`
	ModuleType  string     `gorm:"size:50;not null"` // pre-class, in-class, post-class
	Status      TaskStatus `gorm:"type:enum('pending','in_progress','completed');default:'pending'"`
	UserID      uint       `gorm:"index;type:int unsigned"`
	DueDate     time.Time
	Order       int    `gorm:"default:0"`
	Difficulty  string `gorm:"size:10"` // 难度字段
}

func (Task) TableName() string {
	return "tasks"
}
