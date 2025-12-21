package model

// swagger:model LevelQuestion
type LevelQuestion struct {
	BaseModel

	LevelID       uint   `gorm:"index;type:bigint unsigned" json:"levelId"`
	QuestionType  string `gorm:"size:50" json:"questionType"` // multiple_choice, fill_blank, essay, composite
	Content       string `gorm:"type:json" json:"content"`    // JSON: stem, images, code, formula, etc.
	Options       string `gorm:"type:json" json:"options"`    // 选择题选项（JSON array）
	CorrectAnswer string `gorm:"type:json" json:"correctAnswer"`
	Points        int    `gorm:"default:0" json:"points"`
	Weight        int    `gorm:"default:1" json:"weight"`            // 权重，默认1
	ManualGrading bool   `gorm:"default:false" json:"manualGrading"` // 是否需要人工评分
	Order         int    `gorm:"default:0" json:"order"`
	ScoringRule   string `gorm:"type:text" json:"scoringRule"` // 自定义评分规则或权重
	Explanation   string `gorm:"type:text" json:"explanation"` // 答案解析
}

func (LevelQuestion) TableName() string {
	return "level_questions"
}
