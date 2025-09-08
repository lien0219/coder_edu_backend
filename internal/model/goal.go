package model

import "time"

type GoalStatus string

const (
	GoalPending    GoalStatus = "pending"
	GoalInProgress GoalStatus = "in_progress"
	GoalCompleted  GoalStatus = "completed"
)

type Goal struct {
	BaseModel
	UserID      uint       `gorm:"index;type:bigint unsigned"`
	Title       string     `gorm:"size:255;not null"`
	Description string     `gorm:"type:text"`
	Status      GoalStatus `gorm:"type:enum('pending','in_progress','completed');default:'pending'"`
	Current     int        `gorm:"default:0"`
	Target      int        `gorm:"not null"`
	Progress    float64    `gorm:"default:0"`
	TargetDate  time.Time  `gorm:"type:datetime"`
}

func (Goal) TableName() string {
	return "goals"
}
