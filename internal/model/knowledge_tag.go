package model

// KnowledgeTag 知识点标签
type KnowledgeTag struct {
	BaseModel
	Code        string `gorm:"size:100;uniqueIndex" json:"code"`
	Name        string `gorm:"size:255;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Order       int    `gorm:"default:0" json:"order"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
}

func (KnowledgeTag) TableName() string {
	return "knowledge_tags"
}
