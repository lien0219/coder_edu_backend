package model

import "time"

// swagger:model LevelAttempt
type LevelAttempt struct {
	BaseModel

	LevelID          uint       `gorm:"index;type:bigint unsigned" json:"levelId"`
	UserID           uint       `gorm:"index;type:bigint unsigned" json:"userId"`
	Score            int        `json:"score"`
	Success          bool       `gorm:"default:false" json:"success"`
	AttemptsUsed     int        `json:"attemptsUsed"`
	StartedAt        time.Time  `json:"startedAt"`
	EndedAt          *time.Time `json:"endedAt,omitempty"`
	TotalTimeSeconds int        `json:"totalTimeSeconds"`
	PerQuestionTimes string     `gorm:"type:json" json:"perQuestionTimes"`
}

func (LevelAttempt) TableName() string {
	return "level_attempts"
}
