package model

import (
	"time"

	"gorm.io/gorm"
)

// Checkin 记录用户的学习签到信息
// swagger:model Checkin
type Checkin struct {
	gorm.Model
	ID         uint      `gorm:"primaryKey"`
	UserID     uint      `gorm:"index;type:bigint unsigned;not null"`
	CheckinAt  time.Time `gorm:"not null;index:idx_user_checkin_date,unique"`
	StreakDays int       `gorm:"default:1"` // 连续签到天数
}

func (Checkin) TableName() string {
	return "checkins"
}
