package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/pkg/logger"
	goctx "context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type QAService struct {
	db        *gorm.DB
	rdb       *redis.Client
	aiService *AIService
}

func NewQAService(db *gorm.DB, rdb *redis.Client, aiService *AIService) *QAService {
	return &QAService{
		db:        db,
		rdb:       rdb,
		aiService: aiService,
	}
}

type AskRequest struct {
	Question  string `json:"question" binding:"required"`
	SessionID string `json:"sessionId"` // 前端传入当前会话 ID
}

type AskResponse struct {
	Answer string `json:"answer"`
	Source string `json:"source"` // "knowledge_base" 或者 "llm"
}

func (s *QAService) GetDB() *gorm.DB {
	return s.db
}

// extractKeywords 简单的关键词提取逻辑
func (s *QAService) extractKeywords(question string) []string {
	// 1. 移除常见语气词和标点
	stopWords := []string{"什么是", "怎么", "如何", "为什么", "请问", "帮我", "解释", "一下", "的", "了", "吗", "呢", "？", "?", "。", "，", ","}
	cleanQuestion := question
	for _, word := range stopWords {
		cleanQuestion = strings.ReplaceAll(cleanQuestion, word, " ")
	}

	// 2. 按空格切分并过滤短词
	words := strings.Fields(cleanQuestion)
	var keywords []string
	seen := make(map[string]bool)

	for _, w := range words {
		w = strings.TrimSpace(w)
		if len(w) > 1 && !seen[w] { // 过滤单字（如“在”、“有”）
			keywords = append(keywords, w)
			seen[w] = true
		}
	}

	// 3. 如果没提取出关键词，退化为使用原句
	if len(keywords) == 0 {
		return []string{question}
	}
	return keywords
}

func (s *QAService) buildSearchQuery(db *gorm.DB, table string, fields []string, keywords []string) *gorm.DB {
	if len(keywords) == 0 {
		return db
	}

	// 定义支持全文索引的表及其对应的索引字段（必须与 database.go 中一致）
	fullTextTables := map[string]bool{
		"knowledge_points":          true,
		"exercise_questions":        true,
		"assessment_questions":      true,
		"post_class_test_questions": true,
		"posts":                     true,
		"questions":                 true,
	}

	// 如果该表支持全文索引，使用 MATCH AGAINST 语法
	if fullTextTables[table] {
		fieldsStr := strings.Join(fields, ",")
		searchStr := strings.Join(keywords, " ") // 全文搜索通常以空格分隔关键词
		return db.Where(fmt.Sprintf("MATCH(%s) AGAINST(? IN NATURAL LANGUAGE MODE)", fieldsStr), searchStr)
	}

	// 否则退化为传统的 LIKE 模式
	var conditions []string
	var values []interface{}
	for _, kw := range keywords {
		for _, field := range fields {
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", field))
			values = append(values, "%"+kw+"%")
		}
	}

	if len(conditions) > 0 {
		return db.Where(strings.Join(conditions, " OR "), values...)
	}
	return db
}

func (s *QAService) AskStream(userID uint, question string, sessionID string) (<-chan string, string, <-chan error) {
	sensitiveWords := []string{"政治", "暴力", "色情"}
	for _, word := range sensitiveWords {
		if strings.Contains(question, word) {
			errChan := make(chan error, 1)
			errChan <- fmt.Errorf("您的提问包含敏感词，请修改后重新提问")
			return nil, "llm", errChan
		}
	}

	// 1. 获取历史对话记录 (仅限当前 SessionID，且最近 5 轮)
	var historyMessages []AIChatMessage
	if sessionID != "" {
		var histories []model.AIQAHistory
		s.db.Where("user_id = ? AND session_id = ?", userID, sessionID).
			Order("created_at desc").Limit(5).Find(&histories)

		// 历史记录 Token/长度控制 (简单字数检查)
		totalChars := 0
		maxChars := 4000 // 限制历史记录总长度

		for i := len(histories) - 1; i >= 0; i-- {
			if totalChars+len(histories[i].Question)+len(histories[i].Answer) > maxChars {
				continue
			}
			historyMessages = append(historyMessages, AIChatMessage{
				Role:    "user",
				Content: histories[i].Question,
			})
			historyMessages = append(historyMessages, AIChatMessage{
				Role:    "assistant",
				Content: histories[i].Answer,
			})
			totalChars += len(histories[i].Question) + len(histories[i].Answer)
		}
	}

	// 2. 尝试从多源知识库检索相关内容
	var context string
	source := "llm"

	// 提取关键词
	keywords := s.extractKeywords(question)

	// 检索知识点 (knowledge_points) - 增加标题匹配权重
	var kps []model.KnowledgePoint
	s.buildSearchQuery(s.db.Model(&model.KnowledgePoint{}), "knowledge_points", []string{"title", "article_content"}, keywords).
		Limit(2).Find(&kps)
	if len(kps) > 0 {
		source = "knowledge_base"
		for _, kp := range kps {
			context += fmt.Sprintf("[知识点] 标题: %s\n内容: %s\n\n", kp.Title, kp.ArticleContent)
		}
	}

	// 检索课程资源 (resources)
	var resources []model.Resource
	s.buildSearchQuery(s.db.Model(&model.Resource{}), "resources", []string{"title", "description"}, keywords).
		Limit(2).Find(&resources)
	if len(resources) > 0 {
		source = "knowledge_base"
		for _, res := range resources {
			context += fmt.Sprintf("[课程资源] 标题: %s\n描述: %s\n\n", res.Title, res.Description)
		}
	}

	// 检索C语言分类模块 (c_programming_resources)
	var cRes []model.CProgrammingResource
	s.buildSearchQuery(s.db.Model(&model.CProgrammingResource{}), "c_programming_resources", []string{"name", "description"}, keywords).
		Limit(2).Find(&cRes)
	if len(cRes) > 0 {
		source = "knowledge_base"
		for _, cr := range cRes {
			context += fmt.Sprintf("[C语言模块] 名称: %s\n描述: %s\n\n", cr.Name, cr.Description)
		}
	}

	// 检索练习题库 (exercise_questions)
	var exercises []model.ExerciseQuestion
	s.buildSearchQuery(s.db.Model(&model.ExerciseQuestion{}), "exercise_questions", []string{"title", "description"}, keywords).
		Limit(2).Find(&exercises)
	if len(exercises) > 0 {
		source = "knowledge_base"
		for _, ex := range exercises {
			context += fmt.Sprintf("[练习题] 题目: %s\n描述: %s\n提示: %s\n\n", ex.Title, ex.Description, ex.Hint)
		}
	}

	// 检索关卡信息 (levels)
	var levels []model.Level
	s.buildSearchQuery(s.db.Model(&model.Level{}), "levels", []string{"title", "description"}, keywords).
		Limit(2).Find(&levels)
	if len(levels) > 0 {
		source = "knowledge_base"
		for _, lv := range levels {
			context += fmt.Sprintf("[关卡] 名称: %s\n描述: %s\n\n", lv.Title, lv.Description)
		}
	}

	// 检索社区帖子 (posts)
	var posts []model.Post
	s.buildSearchQuery(s.db.Model(&model.Post{}), "posts", []string{"title", "content"}, keywords).
		Limit(2).Find(&posts)
	if len(posts) > 0 {
		source = "knowledge_base"
		for _, p := range posts {
			context += fmt.Sprintf("[社区帖子] 标题: %s\n内容摘要: %s\n\n", p.Title, p.Content)
		}
	}

	// 检索测评题库 (assessment_questions)
	var assessmentQuestions []model.AssessmentQuestion
	s.buildSearchQuery(s.db.Model(&model.AssessmentQuestion{}), "assessment_questions", []string{"content", "explanation"}, keywords).
		Limit(2).Find(&assessmentQuestions)
	if len(assessmentQuestions) > 0 {
		source = "knowledge_base"
		for _, aq := range assessmentQuestions {
			context += fmt.Sprintf("[测评题] 内容: %s\n解析: %s\n\n", aq.Content, aq.Explanation)
		}
	}

	// 检索课后测试 (post_class_test_questions)
	var postClassQuestions []model.PostClassTestQuestion
	s.buildSearchQuery(s.db.Model(&model.PostClassTestQuestion{}), "post_class_test_questions", []string{"content", "explanation"}, keywords).
		Limit(2).Find(&postClassQuestions)
	if len(postClassQuestions) > 0 {
		source = "knowledge_base"
		for _, pq := range postClassQuestions {
			context += fmt.Sprintf("[课后测试题] 内容: %s\n解析: %s\n\n", pq.Content, pq.Explanation)
		}
	}

	// 检索社区提问 (questions)
	var communityQuestions []model.Question
	s.buildSearchQuery(s.db.Model(&model.Question{}), "questions", []string{"title", "content"}, keywords).
		Limit(2).Find(&communityQuestions)
	if len(communityQuestions) > 0 {
		source = "knowledge_base"
		for _, cq := range communityQuestions {
			context += fmt.Sprintf("[社区提问] 标题: %s\n内容: %s\n\n", cq.Title, cq.Content)
		}
	}

	// 检索社区资源 (community_resources)
	var communityRes []model.CommunityResource
	s.buildSearchQuery(s.db.Model(&model.CommunityResource{}), "community_resources", []string{"title", "description", "content"}, keywords).
		Limit(2).Find(&communityRes)
	if len(communityRes) > 0 {
		source = "knowledge_base"
		for _, cr := range communityRes {
			context += fmt.Sprintf("[社区资源] 标题: %s\n描述: %s\n内容摘要: %s\n\n", cr.Title, cr.Description, cr.Content)
		}
	}

	// 检索当前用户的学习进度 (UserProgress)
	var progress []model.UserProgress
	s.db.Where("user_id = ?", userID).Order("updated_at desc").Limit(3).Find(&progress)
	if len(progress) > 0 {
		context += "【用户当前学习进度】\n"
		for _, p := range progress {
			status := "进行中"
			if p.Completed {
				status = "已完成"
			}
			context += fmt.Sprintf("- 模块ID: %d, 状态: %s, 积分: %d, 最后学习时间: %s\n",
				p.ModuleID, status, p.Score, p.UpdatedAt.Format("2006-01-02 15:04"))
		}
		context += "\n"
	}

	// 检索用户最近的练习记录 (ExerciseSubmission)
	var lastSubmissions []model.ExerciseSubmission
	s.db.Where("user_id = ?", userID).Order("created_at desc").Limit(3).Find(&lastSubmissions)
	if len(lastSubmissions) > 0 {
		context += "【用户最近练习记录】\n"
		for _, sub := range lastSubmissions {
			result := "错误"
			if sub.IsCorrect {
				result = "正确"
			}
			context += fmt.Sprintf("- 题目ID: %d, 结果: %s, 提交时间: %s\n",
				sub.QuestionID, result, sub.CreatedAt.Format("2006-01-02 15:04"))
		}
		context += "\n"
	}

	// 3. 调用AI Service获取流式回答
	stream, aiErrChan := s.aiService.ChatStream(question, context, historyMessages)

	// 4. 创建一个包装后的 channel
	wrappedOut := make(chan string)
	wrappedErr := make(chan error, 1)

	tempKey := fmt.Sprintf("qa:stream:temp:%d:%s:%d", userID, sessionID, time.Now().Unix())

	go func() {
		defer close(wrappedOut)
		defer close(wrappedErr)

		var fullAnswer string
		// 结束后清理 Redis 暂存，并确保残缺对话被保存
		defer func() {
			s.rdb.Del(goctx.Background(), tempKey)

			if fullAnswer != "" {
				// 检查是否已经存过（防止重复保存）
				var count int64
				s.db.Model(&model.AIQAHistory{}).Where("user_id = ? AND session_id = ? AND question = ? AND answer = ?",
					userID, sessionID, question, strings.TrimSpace(fullAnswer)).Count(&count)

				if count == 0 {
					history := model.AIQAHistory{
						UserID:    userID,
						SessionID: sessionID,
						Question:  question,
						Answer:    strings.TrimSpace(fullAnswer) + "... [已停止生成]",
						Source:    source,
					}
					if err := s.db.Create(&history).Error; err != nil {
						logger.Log.Error("Failed to save partial QA history", zap.Error(err))
					}
				}
			}
		}()

		for content := range stream {
			fullAnswer += content
			s.rdb.Set(goctx.Background(), tempKey, fullAnswer, 10*time.Minute)
			wrappedOut <- content
		}

		if err := <-aiErrChan; err != nil {
			logger.Log.Error("AI stream error", zap.Error(err))
			wrappedErr <- err
			return
		}
	}()

	return wrappedOut, source, wrappedErr
}

func (s *QAService) GetHistory(userID uint, limit, offset int) ([]model.AIQAHistory, int64, error) {
	var histories []model.AIQAHistory
	var total int64

	db := s.db.Model(&model.AIQAHistory{}).Where("user_id = ?", userID)
	db.Count(&total)

	err := db.Order("created_at desc").Offset(offset).Limit(limit).Find(&histories).Error
	return histories, total, err
}

func (s *QAService) CheckRateLimit(userID uint) (bool, error) {
	ctx := goctx.Background()
	key := fmt.Sprintf("qa:ratelimit:%d", userID)
	limit := 10           // 每分钟 10 次
	window := time.Minute // 1分钟窗口

	count, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		s.rdb.Expire(ctx, key, window)
	}

	if count > int64(limit) {
		return false, nil
	}

	return true, nil
}

// GenerateWeeklyReport 生成学习周报
func (s *QAService) GenerateWeeklyReport(userID uint) (<-chan string, <-chan error) {
	// 1. 获取过去一周的数据
	oneWeekAgo := time.Now().AddDate(0, 0, -7)

	// 1.1 学习进度
	var progress []model.UserProgress
	s.db.Where("user_id = ? AND updated_at > ?", userID, oneWeekAgo).Find(&progress)

	// 1.2 练习记录
	var submissions []model.ExerciseSubmission
	s.db.Where("user_id = ? AND created_at > ?", userID, oneWeekAgo).Find(&submissions)

	// 1.3 社区活跃
	var posts []model.Post
	s.db.Where("author_id = ? AND created_at > ?", userID, oneWeekAgo).Find(&posts)

	// 2. 构造 Prompt
	reportContext := fmt.Sprintf("用户ID: %d\n过去一周学习数据:\n", userID)
	reportContext += fmt.Sprintf("- 完成/更新模块数: %d\n", len(progress))

	correctCount := 0
	for _, sub := range submissions {
		if sub.IsCorrect {
			correctCount++
		}
	}
	reportContext += fmt.Sprintf("- 练习提交次数: %d, 正确次数: %d\n", len(submissions), correctCount)
	reportContext += fmt.Sprintf("- 社区发帖数: %d\n", len(posts))

	systemPrompt := "你是一个专业的编程教育导师。请根据提供的用户过去一周的学习数据，生成一份鼓励性的、专业的学习周报。周报应包含：1. 学习概况总结；2. 技术亮点分析；3. 薄弱环节建议；4. 下周学习规划。请使用 Markdown 格式，并严格遵守之前的 Markdown 渲染指令。"

	// 3. 调用AI生成
	return s.aiService.ChatStream(systemPrompt, reportContext, nil)
}

func (s *QAService) Ask(question string) (*AskResponse, error) {
	// 尝试从知识库检索相关内容
	// 暂时使用简单的LIKE搜索作为基础架子，后续可以升级为向量检索
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
