package model

import (
	"time"
)

// swagger:model MigrationTask
type MigrationTask struct {
	UUIDBase
	Title       string     `gorm:"size:255;not null" json:"title"`
	Description string     `gorm:"type:text" json:"description"`
	Difficulty  string     `gorm:"size:20;not null;default:'medium'" json:"difficulty"` // simple, medium, hard
	TimeLimit   int        `gorm:"default:0" json:"timeLimit"`
	IsPublished bool       `gorm:"default:false" json:"isPublished"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	CreatorID   uint       `gorm:"index;type:bigint unsigned" json:"creatorId"`
}

func (MigrationTask) TableName() string {
	return "migration_tasks"
}

// swagger:model MigrationQuestion
type MigrationQuestion struct {
	UUIDBase
	TaskID         string `gorm:"index;type:varchar(36)" json:"taskId"`
	Title          string `gorm:"size:255;not null" json:"title"`
	Description    string `gorm:"type:text;not null" json:"description"`
	Difficulty     string `gorm:"size:20;not null" json:"difficulty"` // simple, medium, hard
	StandardAnswer string `gorm:"type:text;not null" json:"standardAnswer"`
	Points         int    `gorm:"default:0" json:"points"`
	Order          int    `gorm:"default:0" json:"order"`
}

func (MigrationQuestion) TableName() string {
	return "migration_questions"
}

// swagger:model MigrationSubmission
type MigrationSubmission struct {
	UUIDBase
	TaskID      string     `gorm:"index;type:varchar(36)" json:"taskId"`
	UserID      uint       `gorm:"index;type:bigint unsigned" json:"userId"`
	Score       int        `gorm:"default:0" json:"score"`
	Status      string     `gorm:"size:20;default:'completed'" json:"status"`
	StartedAt   time.Time  `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt"`
}

func (MigrationSubmission) TableName() string {
	return "migration_submissions"
}

// swagger:model MigrationAnswer
type MigrationAnswer struct {
	UUIDBase
	SubmissionID        string `gorm:"index;type:varchar(36)" json:"submissionId"`
	QuestionID          string `gorm:"index;type:varchar(36)" json:"questionId"`
	QuestionTitle       string `gorm:"size:255" json:"questionTitle"`
	QuestionDescription string `gorm:"type:text" json:"questionDescription"`
	UserCode            string `gorm:"type:text" json:"userCode"`
	UserAnswer          string `gorm:"type:text" json:"userAnswer"`
	IsCorrect           bool   `gorm:"default:false" json:"isCorrect"`
	Points              int    `gorm:"default:0" json:"points"`
}

func (MigrationAnswer) TableName() string {
	return "migration_answers"
}
