package model

// AssessmentQuestionKnowledgePoint 学前题与 KnowledgePoint 的多对多关联。
// 不加入 AutoMigrate；由独立 SQL 迁移建表。
type AssessmentQuestionKnowledgePoint struct {
	AssessmentQuestionID uint   `gorm:"uniqueIndex:idx_aq_kp;index;not null" json:"assessmentQuestionId"`
	KnowledgePointID     string `gorm:"uniqueIndex:idx_aq_kp;index;size:36;not null" json:"knowledgePointId"`
}

func (AssessmentQuestionKnowledgePoint) TableName() string {
	return "assessment_question_knowledge_points"
}

// LearningPathMaterialKnowledgePoint 学习资料与 KnowledgePoint 的多对多关联。
type LearningPathMaterialKnowledgePoint struct {
	MaterialID       string `gorm:"uniqueIndex:idx_lpm_kp;index;size:36;not null" json:"materialId"`
	KnowledgePointID string `gorm:"uniqueIndex:idx_lpm_kp;index;size:36;not null" json:"knowledgePointId"`
}

func (LearningPathMaterialKnowledgePoint) TableName() string {
	return "learning_path_material_knowledge_points"
}
