package model

import "encoding/json"

// swagger:model AssessmentSubmission
type AssessmentSubmission struct {
	BaseModel
	UserID           uint            `gorm:"index;type:bigint unsigned" json:"userId"`
	User             *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	AssessmentID     uint            `gorm:"index;type:bigint unsigned" json:"assessmentId"`
	Answers          json.RawMessage `gorm:"type:json" json:"answers"`
	TotalScore       int             `json:"totalScore"`
	Status           string          `gorm:"size:20;default:'pending'" json:"status"` // pending, completed
	Feedback         string          `gorm:"type:text" json:"feedback"`
	RecommendedLevel int             `json:"recommendedLevel"` // 1:基础, 2:初级, 3:中级, 4:高级
}

func (AssessmentSubmission) TableName() string {
	return "assessment_submissions"
}

type QuestionAnswer struct {
	QuestionID uint   `json:"questionId"`
	Answer     string `json:"answer"`
}
