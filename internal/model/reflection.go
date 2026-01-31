package model

// Reflection 有效反思策略
// swagger:model
type Reflection struct {
	UUIDBase
	UserID      uint   `gorm:"index;comment:用户ID" json:"userId"`
	Summary     string `gorm:"type:text;comment:总结关键知识点" json:"summary"`
	Challenges  string `gorm:"type:text;comment:识别挑战" json:"challenges"`
	Connections string `gorm:"type:text;comment:连接已有知识" json:"connections"`
	NextSteps   string `gorm:"type:text;comment:规划下一步" json:"nextSteps"`

	// 关联用户
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Reflection) TableName() string {
	return "reflections"
}
