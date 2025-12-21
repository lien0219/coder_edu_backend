package model

// LevelKnowledge 关联表：关卡 <-> 知识点标签
type LevelKnowledge struct {
	BaseModel
	LevelID        uint `gorm:"index;type:bigint unsigned" json:"levelId"`
	KnowledgeTagID uint `gorm:"index;type:bigint unsigned" json:"knowledgeTagId"`
}

func (LevelKnowledge) TableName() string {
	return "level_knowledge_tags"
}
