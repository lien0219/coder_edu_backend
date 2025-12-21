package model

// Ability 能力分类（如问题解决、编程基础等）
type Ability struct {
	BaseModel
	Code        string `gorm:"size:100;uniqueIndex" json:"code"`
	Name        string `gorm:"size:255;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Order       int    `gorm:"default:0" json:"order"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
}

func (Ability) TableName() string {
	return "abilities"
}
