package model

import (
	"time"

	"gorm.io/gorm"
)

// Motivation 每日激励短句
type Motivation struct {
	gorm.Model
	ID              uint      `gorm:"primarykey" json:"id"`
	Content         string    `gorm:"type:text;not null" json:"content"`
	IsEnabled       bool      `gorm:"default:true" json:"is_enabled"`
	IsCurrentlyUsed bool      `gorm:"default:false" json:"is_currently_used"`
	LastUsedAt      time.Time `gorm:"autoCreateTime" json:"last_used_at"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Motivation) TableName() string {
	return "motivations"
}
