package model

import (
	"time"

	"gorm.io/gorm"
)

type KnowledgePointType string

const (
	KPConcept     KnowledgePointType = "concept"
	KPRule        KnowledgePointType = "rule"
	KPSyntax      KnowledgePointType = "syntax"
	KPProcess     KnowledgePointType = "process"
	KPStrategy    KnowledgePointType = "strategy"
	KPApplication KnowledgePointType = "application"
)

type KnowledgePoint struct {
	ID              string                   `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Title           string                   `gorm:"size:255;not null" json:"title"`
	Description     string                   `gorm:"type:text" json:"description"`
	Type            KnowledgePointType       `gorm:"size:50;not null" json:"type"`
	ArticleContent  string                   `gorm:"type:longtext" json:"articleContent"`
	TimeLimit       int                      `gorm:"default:0" json:"timeLimit"`
	Order           int                      `gorm:"default:0" json:"order"`
	CompletionScore int                      `gorm:"default:0" json:"completionScore"`
	Videos          []KnowledgePointVideo    `gorm:"foreignKey:KnowledgePointID" json:"videos"`
	Exercises       []KnowledgePointExercise `gorm:"foreignKey:KnowledgePointID" json:"exercises"`
	CreatedAt       time.Time                `json:"createdAt"`
	UpdatedAt       time.Time                `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt           `gorm:"index" json:"-"`
}

func (KnowledgePoint) TableName() string {
	return "knowledge_points"
}

type KnowledgePointVideo struct {
	ID               string         `gorm:"primaryKey;type:varchar(36)" json:"id"`
	KnowledgePointID string         `gorm:"index;type:varchar(36)" json:"knowledgePointId"`
	Title            string         `gorm:"size:255;not null" json:"title"`
	URL              string         `gorm:"size:500;not null" json:"url"`
	Description      string         `gorm:"type:text" json:"description"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (KnowledgePointVideo) TableName() string {
	return "knowledge_point_videos"
}

type ExerciseType string

const (
	ExSingleChoice   ExerciseType = "single_choice"
	ExMultipleChoice ExerciseType = "multiple_choice"
	ExFillIn         ExerciseType = "fill_in"
	ExTrueFalse      ExerciseType = "true_false"
	ExProgramming    ExerciseType = "programming"
)

type KnowledgePointExercise struct {
	ID               string         `gorm:"primaryKey;type:varchar(36)" json:"id"`
	KnowledgePointID string         `gorm:"index;type:varchar(36)" json:"knowledgePointId"`
	Type             ExerciseType   `gorm:"size:50;not null" json:"type"`
	Question         string         `gorm:"type:text;not null" json:"question"`
	Options          string         `gorm:"type:json" json:"options"` // string array JSON: ["A", "B"]
	Answer           string         `gorm:"type:text;not null" json:"answer"`
	Explanation      string         `gorm:"type:text" json:"explanation"`
	Points           int            `gorm:"default:0" json:"points"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (KnowledgePointExercise) TableName() string {
	return "knowledge_point_exercises"
}
