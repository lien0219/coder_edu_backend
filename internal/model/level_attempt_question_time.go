package model

// LevelAttemptQuestionTime 单次挑战中每题耗时，便于统计与回放
type LevelAttemptQuestionTime struct {
	BaseModel
	AttemptID   uint `gorm:"index;type:bigint unsigned" json:"attemptId"`
	QuestionID  uint `gorm:"index;type:bigint unsigned" json:"questionId"`
	TimeSeconds int  `gorm:"default:0" json:"timeSeconds"`
}

func (LevelAttemptQuestionTime) TableName() string {
	return "level_attempt_question_times"
}
