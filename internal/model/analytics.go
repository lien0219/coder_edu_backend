package model

import (
	"time"

	"gorm.io/gorm"
)

type LearningSession struct {
	gorm.Model
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"index"`
	ModuleID  uint `gorm:"index"`
	StartTime time.Time
	EndTime   *time.Time
	Duration  int    `gorm:"default:0"`
	Activity  string `gorm:"type:text"`
}

func (LearningSession) TableName() string {
	return "learning_sessions"
}

type SkillAssessment struct {
	gorm.Model
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"index"`
	Skill      string `gorm:"size:100;not null"`
	Score      int    `gorm:"default:0"`
	AssessedAt time.Time
}

func (SkillAssessment) TableName() string {
	return "skill_assessments"
}
