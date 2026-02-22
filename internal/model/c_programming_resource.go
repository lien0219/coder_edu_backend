package model

import "encoding/json"

// CProgrammingResource 表示C语言学习资源的分类模块
// swagger:model CProgrammingResource
type CProgrammingResource struct {
	BaseModel
	Name        string `gorm:"size:255;not null"`
	IconURL     string `gorm:"size:255;not null"`
	Description string `gorm:"type:text"`
	Enabled     bool   `gorm:"default:true"`
	Order       int    `gorm:"default:0"`
}

func (CProgrammingResource) TableName() string {
	return "c_programming_resources"
}

// ExerciseCategory 表示练习题的分类
// swagger:model ExerciseCategory
type ExerciseCategory struct {
	BaseModel
	Name              string `gorm:"size:255;not null"`
	Description       string `gorm:"type:text"`
	Order             int    `gorm:"default:0"`
	CProgrammingResID uint   `gorm:"index;type:bigint unsigned"`
}

func (ExerciseCategory) TableName() string {
	return "exercise_categories"
}

// ExerciseQuestion 表示练习题题目
// swagger:model ExerciseQuestion
type ExerciseQuestion struct {
	BaseModel
	CategoryID    uint            `gorm:"index;type:bigint unsigned"`
	Title         string          `gorm:"size:255;not null"`
	Description   string          `gorm:"type:text"`
	Difficulty    string          `gorm:"size:50;default:'easy'"` // easy, medium, hard
	Hint          string          `gorm:"type:text"`
	SolutionCode  string          `gorm:"type:text"`
	QuestionType  string          `gorm:"size:50;default:'programming'"` // programming, multiple_choice, single_choice
	Options       json.RawMessage `gorm:"type:json"`                     // 存储选择题选项
	CorrectAnswer string          `gorm:"type:text"`                     // 存储正确答案
	Points        int             `gorm:"default:0"`                     // 完成此题可获得的积分
	Tags          string          `gorm:"size:500;default:''"`           // AI 自动生成的关键词标签，逗号分隔
}

func (ExerciseQuestion) TableName() string {
	return "exercise_questions"
}

// ExerciseSubmission 存储用户的练习提交记录
type ExerciseSubmission struct {
	BaseModel
	UserID          uint   `gorm:"index;type:bigint unsigned"`
	QuestionID      uint   `gorm:"index;type:bigint unsigned"`
	SubmittedAnswer string `gorm:"type:text"`
	IsCorrect       bool   `gorm:"default:false"`
}

func (ExerciseSubmission) TableName() string {
	return "exercise_submissions"
}
