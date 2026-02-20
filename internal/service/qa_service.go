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
	ctx := goctx.Background()
	var cursor uint64
	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, "qa:context:cache:*", 100).Result()
		if err != nil {
			break
		}
		if len(keys) > 0 {
			rdb.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return &QAService{
		db:        db,
		rdb:       rdb,
		aiService: aiService,
	}
}

type AskRequest struct {
	Question  string `json:"question" binding:"required"`
	SessionID string `json:"sessionId"`
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
	// 0. 优先提取引号/书名号内的精确名称（如 "测试添加编程题" 或 《指针入门》）
	var quoted []string
	for _, pair := range [][2]string{
		{"\u201c", "\u201d"}, // ""
		{"\"", "\""},
		{"\u2018", "\u2019"}, // ''
		{"\u300a", "\u300b"}, // 《》
		{"\u300c", "\u300d"}, // 「」
	} {
		parts := strings.Split(question, pair[0])
		for i := 1; i < len(parts); i++ {
			end := strings.Index(parts[i], pair[1])
			if end > 0 {
				quoted = append(quoted, parts[i][:end])
			}
		}
	}
	if len(quoted) > 0 {
		return quoted // 引号内的名称就是最精准的关键词
	}

	// 1. 移除常见语气词、意图触发词和标点
	stopWords := []string{"什么是", "怎么做", "怎么样", "怎么", "如何", "为什么", "请问", "帮我", "解释", "一下",
		"这道叫", "这道", "这个", "有没有", "叫做", "叫",
		"什么", "哪些", "哪个", "哪里", "可以", "推荐", "有什么", "还有", "应该",
		"接下来", "学到哪", "学到", "进度", "计划", "关卡", "课程",
		"学习", "继续", "开始", "需要", "适合", "建议",
		"的", "了", "吗", "呢", "是", "做", "在", "和", "或", "去", "到", "学", "看",
		"？", "?", "。", "，", ",", "、", "！", "!", "：", ":"}
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
		if len(w) > 1 && !seen[w] { // 过滤单字（如"在"、"有"）
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

	query := db.Where("deleted_at IS NULL")

	// 定义支持全文索引的表及其对应的索引字段
	fullTextTables := map[string]bool{
		"knowledge_points":          true,
		"exercise_questions":        true,
		"assessment_questions":      true,
		"post_class_test_questions": true,
		"posts":                     true,
		"questions":                 true,
	}

	if fullTextTables[table] {
		fieldsStr := strings.Join(fields, ",")
		searchStr := strings.Join(keywords, " ")
		return query.Where(fmt.Sprintf("MATCH(%s) AGAINST(? IN NATURAL LANGUAGE MODE)", fieldsStr), searchStr)
	}

	var conditions []string
	var values []interface{}
	for _, kw := range keywords {
		for _, field := range fields {
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", field))
			values = append(values, "%"+kw+"%")
		}
	}

	if len(conditions) > 0 {
		return query.Where(strings.Join(conditions, " OR "), values...)
	}
	return query
}

// Intent 意图类型
type Intent string

const (
	IntentKnowledge Intent = "knowledge" // 知识查询
	IntentPractice  Intent = "practice"  // 练习/错题分析
	IntentCommunity Intent = "community" // 社区/互动
	IntentProgress  Intent = "progress"  // 进度/计划
	IntentGeneral   Intent = "general"   // 通用聊天
)

// classifyIntent 简单的意图识别逻辑（支持多意图，返回主意图）
func (s *QAService) classifyIntent(question string) Intent {
	q := strings.ToLower(question)
	// 1. 练习/错题分析意图
	if strings.Contains(q, "练习") || strings.Contains(q, "编程题") || strings.Contains(q, "题目") ||
		strings.Contains(q, "错") || strings.Contains(q, "报错") || strings.Contains(q, "没过") || strings.Contains(q, "作业") {
		return IntentPractice
	}
	// 2. 进度/计划意图
	if strings.Contains(q, "进度") || strings.Contains(q, "学到哪") || strings.Contains(q, "计划") || strings.Contains(q, "接下来") || strings.Contains(q, "关卡") {
		return IntentProgress
	}
	// 3. 社区/互动意图
	if strings.Contains(q, "帖子") || strings.Contains(q, "有人问") || strings.Contains(q, "讨论") || strings.Contains(q, "社区") {
		return IntentCommunity
	}
	// 4. 知识查询意图 (默认)
	if strings.Contains(q, "什么是") || strings.Contains(q, "怎么") || strings.Contains(q, "原理") || strings.Contains(q, "知识") || len(q) > 4 {
		return IntentKnowledge
	}
	return IntentGeneral
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

		// 历史记录Token/长度控制 (简单字数检查)
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

	// 2. 意图识别与关键词提取
	intent := s.classifyIntent(question)
	keywords := s.extractKeywords(question)
	context := fmt.Sprintf("【当前对话意图: %s】\n", intent)
	source := "llm"

	// 预生成引用链接列表
	var citations []string

	// 3. 检查Redis缓存(针对高频问题，同时缓存citations)
	cacheKey := fmt.Sprintf("qa:context:cache:%s", strings.Join(keywords, "_"))
	citationCacheKey := cacheKey + ":citations"
	if cachedContext, err := s.rdb.Get(goctx.Background(), cacheKey).Result(); err == nil {
		context = cachedContext
		source = "knowledge_base"
		// 同步恢复缓存的citations
		if cachedCitations, err := s.rdb.Get(goctx.Background(), citationCacheKey).Result(); err == nil && cachedCitations != "" {
			citations = strings.Split(cachedCitations, "|||")
		}
	} else {
		// 4. 根据意图按需检索，减小数据库压力
		// ==================== 引用链接与前端handleMessageClick对齐说明（后续也可以拓展） ====================
		// 前端路由映射：
		//   /knowledge/detail/:id → /community/resources/detail?id=:id  需要 community_resources UUID ← 与 knowledge_points ID 不匹配，不生成链接
		//   /practice/:id         → /levels/detail?id=:id               需要 levels uint ID          ← 与 exercise_questions ID 不匹配，不生成链接
		//   /community/post/:id   → /community/discussion?id=:id        需要 posts UUID              ← 匹配
		//   /courses/:id          → /levels/detail?id=:id               需要 levels uint ID          ← 匹配
		// =======================================================================================

		// 4.1 知识类检索 (仅在知识或通用意图下执行)
		if intent == IntentKnowledge || intent == IntentGeneral {
			var kps []model.KnowledgePoint
			s.buildSearchQuery(s.db.Model(&model.KnowledgePoint{}), "knowledge_points", []string{"title", "article_content"}, keywords).
				Limit(2).Find(&kps)
			for _, kp := range kps {
				source = "knowledge_base"
				context += fmt.Sprintf("标题: %s\n内容: %s\n\n", kp.Title, kp.ArticleContent)
				// 不生成链接：因为前端将 /knowledge/detail/:id 跳转到 community_resources 页面，但此 ID 来自 knowledge_points 表，不匹配
			}
		}

		// 4.2 练习类检索 (仅在练习或通用意图下执行)
		if intent == IntentPractice || intent == IntentGeneral {
			var exercises []model.ExerciseQuestion
			s.buildSearchQuery(s.db.Model(&model.ExerciseQuestion{}), "exercise_questions", []string{"title", "description"}, keywords).
				Limit(2).Find(&exercises)
			for _, ex := range exercises {
				source = "knowledge_base"
				context += fmt.Sprintf("题目: %s\n描述: %s\n提示: %s\n\n", ex.Title, ex.Description, ex.Hint)
				// 不生成链接：因为前端将 /practice/:id 跳转到 /levels/detail，但 exercise ID ≠ level ID，不匹配
			}

			var lastSubmissions []model.ExerciseSubmission
			s.db.Where("user_id = ?", userID).Order("created_at desc").Limit(3).Find(&lastSubmissions)
			if len(lastSubmissions) > 0 {
				context += "【用户最近练习记录】\n"
				for _, sub := range lastSubmissions {
					resStr := "错误"
					if sub.IsCorrect {
						resStr = "正确"
					}
					context += fmt.Sprintf("- 题目ID: %d, 结果: %s, 时间: %s\n", sub.QuestionID, resStr, sub.CreatedAt.Format("15:04"))
				}
			}
		}

		// 4.3 进度类检索
		if intent == IntentProgress || intent == IntentGeneral {
			var progress []model.UserProgress
			s.db.Where("user_id = ?", userID).Order("updated_at desc").Limit(2).Find(&progress)
			for _, p := range progress {
				status := "进行中"
				if p.Completed {
					status = "已完成"
				}
				context += fmt.Sprintf("学习进度: 模块ID: %d, 状态: %s, 积分: %d\n", p.ModuleID, status, p.Score)
			}

			// 先尝试用关键词匹配已发布关卡
			var levels []model.Level
			hasKeywords := false
			for _, kw := range keywords {
				if len(kw) > 1 {
					hasKeywords = true
					break
				}
			}
			if hasKeywords {
				s.buildSearchQuery(s.db.Where("is_published = ?", true).Model(&model.Level{}), "levels", []string{"title", "description"}, keywords).
					Limit(5).Find(&levels)
			}

			// 兜底：如果关键词未匹配到关卡，返回最近发布的关卡
			if len(levels) == 0 {
				s.db.Where("is_published = ?", true).Order("published_at desc").Limit(5).Find(&levels)
			}

			if len(levels) > 0 {
				source = "knowledge_base"
				context += "【已发布的关卡列表】\n"
				for _, lv := range levels {
					context += fmt.Sprintf("关卡: %s（难度: %s）\n描述: %s\n\n", lv.Title, lv.Difficulty, lv.Description)
					citations = append(citations, fmt.Sprintf("- [%s](/courses/%d)", lv.Title, lv.ID))
				}
			}
		}

		// 4.4 社区类检索
		if intent == IntentCommunity || intent == IntentGeneral {
			var posts []model.Post
			s.buildSearchQuery(s.db.Model(&model.Post{}), "posts", []string{"title", "content"}, keywords).
				Limit(1).Find(&posts)
			for _, p := range posts {
				source = "knowledge_base"
				context += fmt.Sprintf("帖子: %s\n内容摘要: %s\n\n", p.Title, p.Content)
				citations = append(citations, fmt.Sprintf("- [%s](/community/post/%s)", p.Title, p.ID))
			}
		}

		// 缓存检索结果和 citations (5分钟)
		if source == "knowledge_base" {
			s.rdb.Set(goctx.Background(), cacheKey, context, 5*time.Minute)
			if len(citations) > 0 {
				s.rdb.Set(goctx.Background(), citationCacheKey, strings.Join(citations, "|||"), 5*time.Minute)
			}
		}
	}

	// 5. 后端预生成引用链接（不再交给 AI 输出，由后端代码在流结束后自动追加）
	citationBlock := ""
	if len(citations) > 0 {
		citationBlock = "\n\n**相关资源：**\n" + strings.Join(citations, "\n")
	}

	// 6. 调用AI Service获取流式回答（不再在 Prompt 中要求 AI 输出链接）
	stream, aiErrChan := s.aiService.ChatStream(question, context, historyMessages)

	// 4. 创建一个包装后的 channel
	wrappedOut := make(chan string)
	wrappedErr := make(chan error, 1)

	tempKey := fmt.Sprintf("qa:stream:temp:%d:%s:%d", userID, sessionID, time.Now().Unix())

	go func() {
		defer close(wrappedErr)
		defer close(wrappedOut)

		var fullAnswer string
		streamCompleted := false // 标记流是否正常完成

		for content := range stream {
			fullAnswer += content
			s.rdb.Set(goctx.Background(), tempKey, fullAnswer, 10*time.Minute)
			wrappedOut <- content
		}

		if err := <-aiErrChan; err != nil {
			logger.Log.Error("AI stream error", zap.Error(err))
			wrappedErr <- err
		} else {
			streamCompleted = true
		}

		// AI 流正常结束后，由后端代码追加引用链接（100% 确定性，不依赖 AI）
		if streamCompleted && citationBlock != "" {
			fullAnswer += citationBlock
			wrappedOut <- citationBlock
		}

		// 清理 Redis 暂存
		s.rdb.Del(goctx.Background(), tempKey)

		// 保存对话历史
		if fullAnswer != "" {
			finalAnswer := strings.TrimSpace(fullAnswer)
			answerSuffix := ""
			if !streamCompleted {
				answerSuffix = "... [已停止生成]"
			}

			var count int64
			s.db.Model(&model.AIQAHistory{}).Where("user_id = ? AND session_id = ? AND question = ? AND answer = ?",
				userID, sessionID, question, finalAnswer+answerSuffix).Count(&count)

			if count == 0 {
				history := model.AIQAHistory{
					UserID:    userID,
					SessionID: sessionID,
					Question:  question,
					Answer:    finalAnswer + answerSuffix,
					Source:    source,
				}
				if err := s.db.Create(&history).Error; err != nil {
					logger.Log.Error("Failed to save QA history", zap.Error(err))
				}
			}
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

// DeleteSession 删除指定会话的所有历史记录
func (s *QAService) DeleteSession(userID uint, sessionID string) error {
	result := s.db.Where("user_id = ? AND session_id = ?", userID, sessionID).Delete(&model.AIQAHistory{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("会话不存在或无权删除")
	}
	return nil
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

// DiagnoseCode 自动代码诊断
func (s *QAService) DiagnoseCode(userID uint, questionID uint, code string, compilerError string) (<-chan string, <-chan error) {
	// 1. 获取题目背景
	var exercise model.ExerciseQuestion
	s.db.First(&exercise, questionID)

	// 2. 构造诊断 Context
	context := fmt.Sprintf("【题目信息】\n标题: %s\n描述: %s\n提示: %s\n\n",
		exercise.Title, exercise.Description, exercise.Hint)
	context += fmt.Sprintf("【用户提交的代码】\n```c\n%s\n```\n\n", code)
	context += fmt.Sprintf("【编译器/判题报错】\n%s\n", compilerError)

	systemPrompt := "你是一个资深的编程导师。请分析用户的代码和报错信息，指出逻辑错误或语法错误。要求：1. 不要直接给出完整正确答案；2. 采用启发式引导，指出错误行号和原因；3. 给出修改建议。严格遵守 Markdown 渲染指令。"

	// 3. 调用 AI
	return s.aiService.ChatStream(systemPrompt, context, nil)
}
