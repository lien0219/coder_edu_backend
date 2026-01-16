package model

import "encoding/json"

// AssessmentQuestion represents a question within a pre-school assessment
// swagger:model AssessmentQuestion
type AssessmentQuestion struct {
	BaseModel
	AssessmentID uint            `gorm:"index;type:bigint unsigned" json:"assessmentId"`
	QuestionType string          `gorm:"size:50;not null" json:"questionType"` // single_choice, multiple_choice, true_false, fill_blank, essay, code
	Title        string          `gorm:"size:255" json:"title"`
	Content      string          `gorm:"type:text;not null" json:"content"` // Stem
	Options      json.RawMessage `gorm:"type:json" json:"options"`          // JSON: []Option
	Answer       string          `gorm:"type:text" json:"answer"`           // Correct answer
	Points       int             `gorm:"default:0" json:"points"`
	Order        int             `gorm:"default:0" json:"order"`
	Explanation  string          `gorm:"type:text" json:"explanation"`
}

func (AssessmentQuestion) TableName() string {
	return "assessment_questions"
}
