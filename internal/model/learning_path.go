package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	LearningLevelBasic        = 1 // 基础
	LearningLevelElementary   = 2 // 初级
	LearningLevelIntermediate = 3 // 中级
	LearningLevelAdvanced     = 4 // 高级
)

// swagger:model LearningPathMaterial
type LearningPathMaterial struct {
	ID            string         `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Level         int            `gorm:"not null" json:"level"` // 1: 基础, 2: 初级, 3: 中级, 4: 高级
	TotalChapters int            `gorm:"default:0" json:"totalChapters"`
	ChapterNumber int            `gorm:"default:0" json:"chapterNumber"`
	Title         string         `gorm:"size:255;not null" json:"title"`
	Content       string         `gorm:"type:longtext" json:"content"`
	Points        int            `gorm:"default:0" json:"points"`
	CreatorID     uint           `gorm:"index;type:bigint unsigned" json:"creatorId"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (LearningPathMaterial) TableName() string {
	return "learning_path_materials"
}
