package model

// LevelAttemptAnswer 存储用户在一次尝试中的每题答案（用于人工评分或回放）
type LevelAttemptAnswer struct {
	BaseModel
	AttemptID  uint   `gorm:"index;type:bigint unsigned" json:"attemptId"`
	QuestionID uint   `gorm:"index;type:bigint unsigned" json:"questionId"`
	Answer     string `gorm:"type:json" json:"answer"` // JSON 存储学生答案
}

func (LevelAttemptAnswer) TableName() string {
	return "level_attempt_answers"
}
