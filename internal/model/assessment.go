package model

import (
	"encoding/json"
	"time"
)

// swagger:model Assessment
type Assessment struct {
	BaseModel
	Title       string     `gorm:"size:255;not null" json:"title"`
	Description string     `gorm:"type:text" json:"description"`
	TimeLimit   int        `gorm:"default:0" json:"timeLimit"` // Minutes
	IsPublished bool       `gorm:"default:false" json:"isPublished"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
}

func (Assessment) TableName() string {
	return "assessments"
}

type PostClassTest struct {
	UUIDBase
	Title       string     `gorm:"size:255;not null" json:"title"`
	Description string     `gorm:"type:text" json:"description"`
	TimeLimit   int        `gorm:"default:0" json:"timeLimit"` // Minutes
	IsPublished bool       `gorm:"default:false" json:"isPublished"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	CreatorID   uint       `gorm:"index;type:bigint unsigned" json:"creatorId"`
}

func (PostClassTest) TableName() string {
	return "post_class_tests"
}

type PostClassTestQuestion struct {
	UUIDBase
	TestID       string          `gorm:"index;type:varchar(36)" json:"testId"`
	QuestionType string          `gorm:"size:50;not null" json:"questionType"`
	Content      string          `gorm:"type:text;not null" json:"content"`
	Options      json.RawMessage `gorm:"type:json" json:"options,omitempty"`
	Answer       string          `gorm:"type:text" json:"answer"`
	Points       int             `gorm:"default:0" json:"points"`
	RewardXP     int             `gorm:"default:0" json:"rewardXp"`
	Explanation  string          `gorm:"type:text" json:"explanation"`
	Order        int             `gorm:"default:0" json:"order"`
}

func (PostClassTestQuestion) TableName() string {
	return "post_class_test_questions"
}

type PostClassTestSubmission struct {
	UUIDBase
	TestID      string     `gorm:"index;type:varchar(36)" json:"testId"`
	UserID      uint       `gorm:"index;type:bigint unsigned" json:"userId"`
	Score       int        `gorm:"default:0" json:"score"`
	RewardXP    int        `gorm:"default:0" json:"rewardXp"`
	Status      string     `gorm:"size:20;default:'completed'" json:"status"`
	IsRetest    bool       `gorm:"default:false" json:"isRetest"`
	StartedAt   time.Time  `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt"`
}

func (PostClassTestSubmission) TableName() string {
	return "post_class_test_submissions"
}

type PostClassTestAnswer struct {
	UUIDBase
	SubmissionID string `gorm:"index;type:varchar(36)" json:"submissionId"`
	QuestionID   string `gorm:"index;type:varchar(36)" json:"questionId"`
	UserAnswer   string `gorm:"type:text" json:"userAnswer"`
	IsCorrect    bool   `gorm:"default:false" json:"isCorrect"`
	Score        int    `gorm:"default:0" json:"score"`
}

func (PostClassTestAnswer) TableName() string {
	return "post_class_test_answers"
}
