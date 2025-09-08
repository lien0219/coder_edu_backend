package model

import (
	"time"

	"gorm.io/gorm"
)

type ModuleType string

const (
	PreClass  ModuleType = "pre_class"
	InClass   ModuleType = "in_class"
	PostClass ModuleType = "post_class"
)

type LearningModule struct {
	gorm.Model
	// ID          uint       `gorm:"primaryKey"`
	Title       string     `gorm:"size:255;not null"`
	Description string     `gorm:"type:text"`
	Type        ModuleType `gorm:"type:enum('pre_class','in_class','post_class');not null"`
	Order       int        `gorm:"default:0"`
	Tasks       []Task     `gorm:"foreignKey:ModuleID"`
	Resources   []Resource `gorm:"foreignKey:ModuleID"`
}

func (LearningModule) TableName() string {
	return "learning_modules"
}

type UserProgress struct {
	gorm.Model
	ID          uint `gorm:"primaryKey"`
	UserID      uint `gorm:"index;type:bigint unsigned"`
	ModuleID    uint `gorm:"index;type:bigint unsigned"`
	Completed   bool `gorm:"default:false"`
	Score       int  `gorm:"default:0"`
	TimeSpent   int  `gorm:"default:0"`
	StartedAt   time.Time
	CompletedAt *time.Time
}

func (UserProgress) TableName() string {
	return "user_progress"
}
