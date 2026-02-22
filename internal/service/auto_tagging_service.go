package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/pkg/logger"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AutoTaggingService 自动标签生成服务
// 定时扫描数据库中 Tags 为空的知识点和练习题，调用 AI 生成关键词标签并写回数据库
type AutoTaggingService struct {
	db *gorm.DB
	ai *AIService
}

func NewAutoTaggingService(db *gorm.DB, ai *AIService) *AutoTaggingService {
	return &AutoTaggingService{db: db, ai: ai}
}

// RunAutoTagging 执行一次自动打标签任务
func (s *AutoTaggingService) RunAutoTagging() {
	s.tagKnowledgePoints()
	s.tagExerciseQuestions()
}

// tagKnowledgePoints 为未打标签的知识点生成标签
func (s *AutoTaggingService) tagKnowledgePoints() {
	var kps []model.KnowledgePoint
	result := s.db.Where("tags = '' OR tags IS NULL").Find(&kps)
	if result.Error != nil {
		logger.Log.Error("查询未标签知识点失败", zap.Error(result.Error))
		return
	}

	if len(kps) == 0 {
		return
	}

	logger.Log.Info("开始为知识点自动生成标签", zap.Int("count", len(kps)))

	tagged := 0
	for _, kp := range kps {
		prompt := fmt.Sprintf(
			"请为以下编程知识点提取 3-5 个核心关键词标签，仅返回标签，用逗号分隔，不要有多余文字。\n标题: %s\n内容: %s",
			kp.Title, truncate(kp.ArticleContent, 500),
		)

		tags, err := s.ai.Chat(prompt, "")
		if err != nil {
			logger.Log.Warn("AI生成标签失败", zap.String("title", kp.Title), zap.Error(err))
			continue
		}

		tags = cleanTags(tags)
		if tags == "" {
			continue
		}

		if err := s.db.Model(&model.KnowledgePoint{}).Where("id = ?", kp.ID).Update("tags", tags).Error; err != nil {
			logger.Log.Warn("更新知识点标签失败", zap.String("id", kp.ID), zap.Error(err))
			continue
		}

		tagged++
		logger.Log.Info("知识点标签生成完成", zap.String("title", kp.Title), zap.String("tags", tags))

		// 避免 AI API 限流，间隔 1 秒
		time.Sleep(time.Second)
	}

	logger.Log.Info("知识点自动打标签完成", zap.Int("tagged", tagged), zap.Int("total", len(kps)))
}

// tagExerciseQuestions 为未打标签的练习题生成标签
func (s *AutoTaggingService) tagExerciseQuestions() {
	var exercises []model.ExerciseQuestion
	result := s.db.Where("tags = '' OR tags IS NULL").Find(&exercises)
	if result.Error != nil {
		logger.Log.Error("查询未标签练习题失败", zap.Error(result.Error))
		return
	}

	if len(exercises) == 0 {
		return
	}

	logger.Log.Info("开始为练习题自动生成标签", zap.Int("count", len(exercises)))

	tagged := 0
	for _, ex := range exercises {
		prompt := fmt.Sprintf(
			"请为以下编程练习题提取 3-5 个核心关键词标签，仅返回标签，用逗号分隔，不要有多余文字。\n标题: %s\n描述: %s",
			ex.Title, truncate(ex.Description, 500),
		)

		tags, err := s.ai.Chat(prompt, "")
		if err != nil {
			logger.Log.Warn("AI生成标签失败", zap.String("title", ex.Title), zap.Error(err))
			continue
		}

		tags = cleanTags(tags)
		if tags == "" {
			continue
		}

		if err := s.db.Model(&model.ExerciseQuestion{}).Where("id = ?", ex.ID).Update("tags", tags).Error; err != nil {
			logger.Log.Warn("更新练习题标签失败", zap.Uint("id", ex.ID), zap.Error(err))
			continue
		}

		tagged++
		logger.Log.Info("练习题标签生成完成", zap.String("title", ex.Title), zap.String("tags", tags))

		// 避免 AI API 限流，间隔 1 秒
		time.Sleep(time.Second)
	}

	logger.Log.Info("练习题自动打标签完成", zap.Int("tagged", tagged), zap.Int("total", len(exercises)))
}

// cleanTags 清理 AI 返回的标签文本
func cleanTags(raw string) string {
	raw = strings.TrimSpace(raw)
	// 去除可能的 markdown 格式
	raw = strings.Trim(raw, "`")
	raw = strings.ReplaceAll(raw, "、", ",")
	raw = strings.ReplaceAll(raw, "，", ",")
	raw = strings.ReplaceAll(raw, " ", "")

	// 过滤空标签
	parts := strings.Split(raw, ",")
	var cleaned []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}

	return strings.Join(cleaned, ",")
}

// truncate 截取文本前 n 个字符
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
