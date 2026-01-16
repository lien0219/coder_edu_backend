package model

import "time"

// Assessment represents a pre-school assessment test
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
