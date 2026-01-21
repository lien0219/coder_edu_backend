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

type KnowledgePointCompletion struct {
	UserID           uint      `gorm:"primaryKey;index:idx_user_kp" json:"userId"`
	KnowledgePointID string    `gorm:"primaryKey;type:varchar(36);index:idx_user_kp" json:"knowledgePointId"`
	IsCompleted      bool      `gorm:"default:false" json:"isCompleted"`
	CompletedAt      time.Time `json:"completedAt"`
}

func (KnowledgePointCompletion) TableName() string {
	return "knowledge_point_completions"
}

// KnowledgePointSubmission 记录学生提交的详细测试信息
type KnowledgePointSubmission struct {
	ID               string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID           uint   `gorm:"index" json:"userId"`
	KnowledgePointID string `gorm:"index;type:varchar(36)" json:"knowledgePointId"`
	// Details 存储 JSON 数组，包含每题的题目、类型、学生答案、代码内容、执行结果及系统初步判断
	Details      string    `gorm:"type:longtext" json:"details"`
	Score        int       `gorm:"default:0" json:"score"`                  // 系统初步计算的得分
	Status       string    `gorm:"size:20;default:'pending'" json:"status"` // pending, approved, rejected
	IsAutoSubmit bool      `gorm:"default:false" json:"isAutoSubmit"`       // 是否为自动提交
	Duration     int       `gorm:"default:0" json:"duration"`               // 答题耗时（秒）
	StartedAt    time.Time `json:"startedAt"`                               // 开始答题时间
	CreatedAt    time.Time `json:"createdAt"`
}

func (KnowledgePointSubmission) TableName() string {
	return "knowledge_point_submissions"
}
