package service

import (
	"coder_edu_backend/internal/model"
	"fmt"

	"gorm.io/gorm"
)

type QAService struct {
	db        *gorm.DB
	aiService *AIService
}

func NewQAService(db *gorm.DB, aiService *AIService) *QAService {
	return &QAService{
		db:        db,
		aiService: aiService,
	}
}

type AskRequest struct {
	Question string `json:"question" binding:"required"`
}

type AskResponse struct {
	Answer string `json:"answer"`
	Source string `json:"source"` // knowledge_base或者llm
}

func (s *QAService) AskStream(question string) (<-chan string, string, <-chan error) {
	// 从多源知识库检索相关内容
	var context string
	source := "llm"

	// 检索知识点 (knowledge_points)
	var kps []model.KnowledgePoint
	s.db.Where("title LIKE ? OR article_content LIKE ?", "%"+question+"%", "%"+question+"%").
		Limit(2).Find(&kps)
	if len(kps) > 0 {
		source = "knowledge_base"
		for _, kp := range kps {
			context += fmt.Sprintf("[知识点] 标题: %s\n内容: %s\n\n", kp.Title, kp.ArticleContent)
		}
	}

	// 检索课程资源 (resources)
	var resources []model.Resource
	s.db.Where("title LIKE ? OR description LIKE ?", "%"+question+"%", "%"+question+"%").
		Limit(2).Find(&resources)
	if len(resources) > 0 {
		source = "knowledge_base"
		for _, res := range resources {
			context += fmt.Sprintf("[课程资源] 标题: %s\n描述: %s\n\n", res.Title, res.Description)
		}
	}

	// 检索C语言分类模块 (c_programming_resources)
	var cRes []model.CProgrammingResource
	s.db.Where("name LIKE ? OR description LIKE ?", "%"+question+"%", "%"+question+"%").
		Limit(2).Find(&cRes)
	if len(cRes) > 0 {
		source = "knowledge_base"
		for _, cr := range cRes {
			context += fmt.Sprintf("[C语言模块] 名称: %s\n描述: %s\n\n", cr.Name, cr.Description)
		}
	}

	// 调用AI Service获取流式回答
	stream, errChan := s.aiService.ChatStream(question, context)

	return stream, source, errChan
}

func (s *QAService) Ask(question string) (*AskResponse, error) {
	// 尝试从知识库检索相关内容
	// 这里使用简单的LIKE搜索作为基础架子，后续可以升级为向量检索
	var kps []model.KnowledgePoint
	err := s.db.Where("title LIKE ? OR description LIKE ? OR article_content LIKE ?",
		"%"+question+"%", "%"+question+"%", "%"+question+"%").
		Limit(3).
		Find(&kps).Error

	var context string
	source := "llm"

	if err == nil && len(kps) > 0 {
		source = "knowledge_base"
		for _, kp := range kps {
			context += fmt.Sprintf("标题: %s\n内容: %s\n\n", kp.Title, kp.ArticleContent)
		}
	}

	// 调用AI Service获取回答
	answer, err := s.aiService.Chat(question, context)
	if err != nil {
		return nil, err
	}

	return &AskResponse{
		Answer: answer,
		Source: source,
	}, nil
}
