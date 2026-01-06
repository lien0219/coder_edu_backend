package model

import (
	"encoding/json"
	"time"
)

const (
	LevelDifficultyEasy   = "easy"
	LevelDifficultyMedium = "medium"
	LevelDifficultyHard   = "hard"
)

// swagger:model Level
type Level struct {
	BaseModel

	CreatorID        uint   `gorm:"index;type:bigint unsigned" json:"creatorId"`
	Title            string `gorm:"size:255;not null" json:"title"`
	Description      string `gorm:"type:text" json:"description"`
	CoverURL         string `gorm:"size:255" json:"coverUrl"`
	Difficulty       string `gorm:"type:enum('easy','medium','hard');default:'easy'" json:"difficulty"`
	EstimatedMinutes int    `gorm:"default:0" json:"estimatedMinutes"` // 预计完成时间（分钟）
	AttemptLimit     int    `gorm:"default:10" json:"attemptLimit"`
	PassingScore     int    `gorm:"default:60" json:"passingScore"`
	BasePoints       int    `gorm:"default:0" json:"basePoints"`
	AllowPause       bool   `gorm:"default:true" json:"allowPause"`

	LevelType          string          `gorm:"size:100" json:"levelType"` // 关卡类型
	IsPublished        bool            `gorm:"default:false" json:"isPublished"`
	PublishedAt        *time.Time      `json:"publishedAt,omitempty"`
	ScheduledPublishAt *time.Time      `json:"scheduledPublishAt,omitempty"`              // 定时发布时间
	VisibleScope       string          `gorm:"size:50;default:'all'" json:"visibleScope"` // all/class/specific
	VisibleTo          json.RawMessage `gorm:"type:json" json:"visibleTo"`                // 当为 specific 时，存放学生ID数组
	AvailableFrom      *time.Time      `json:"availableFrom,omitempty"`
	AvailableTo        *time.Time      `json:"availableTo,omitempty"`

	CurrentVersion uint `gorm:"default:0" json:"currentVersion"`
}

func (Level) TableName() string {
	return "levels"
}
