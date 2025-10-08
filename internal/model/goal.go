package model

import "time"

type GoalStatus string

const (
	GoalPending           GoalStatus = "pending"
	GoalInProgress        GoalStatus = "in_progress"
	GoalCompleted         GoalStatus = "completed"
	GoalPendingExpired    GoalStatus = "pending_expired"
	GoalInProgressExpired GoalStatus = "in_progress_expired"
	GoalCompletedExpired  GoalStatus = "completed_expired"
)

type GoalType string

const (
	GoalTypeShortTerm GoalType = "short_term"
	GoalTypeLongTerm  GoalType = "long_term"
)

type Goal struct {
	BaseModel
	UserID             uint       `gorm:"index;type:bigint unsigned"`
	Title              string     `gorm:"size:255;not null"`
	Description        string     `gorm:"type:text"`
	Status             GoalStatus `gorm:"type:enum('pending','in_progress','completed','pending_expired','in_progress_expired','completed_expired');default:'pending'"`
	Current            int        `gorm:"default:0"`
	Target             int        `gorm:"not null"`
	Progress           float64    `gorm:"default:0"`
	TargetDate         time.Time  `gorm:"type:datetime"`
	GoalType           GoalType   `gorm:"type:enum('short_term','long_term');default:'short_term'"`
	ResourceModuleID   uint       `gorm:"index;type:bigint unsigned"`
	ResourceModuleName string     `gorm:"size:255"`
}

func (Goal) TableName() string {
	return "goals"
}
