package model

import "time"

// LevelAttemptQuestionScore 表示对单题的人工评分记录
type LevelAttemptQuestionScore struct {
	BaseModel
	AttemptID  uint       `gorm:"index;type:bigint unsigned" json:"attemptId"`
	QuestionID uint       `gorm:"index;type:bigint unsigned" json:"questionId"`
	Score      int        `json:"score"` // 教师给的分数
	GraderID   uint       `gorm:"index;type:bigint unsigned" json:"graderId"`
	Comment    string     `gorm:"type:text" json:"comment"`
	GradedAt   *time.Time `json:"gradedAt,omitempty"`
}

func (LevelAttemptQuestionScore) TableName() string {
	return "level_attempt_question_scores"
}
