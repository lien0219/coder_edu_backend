package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

// ResourceWithCompletionStatus 带用户完成状态的资源
type ResourceWithCompletionStatus struct {
	model.Resource
	IsCompleted bool `json:"isCompleted"`
}

// ResourceModuleWithProgress 带进度的资源模块
type ResourceModuleWithProgress struct {
	model.CProgrammingResource
	Videos           []ResourceWithCompletionStatus  `json:"videos"`
	Articles         []ResourceWithCompletionStatus  `json:"articles"`
	ExerciseCategory []ExerciseCategoryWithQuestions `json:"exerciseCategories"`
	Progress         float64                         `json:"progress"`
	IsCompleted      bool                            `json:"isCompleted"`
	Status           string                          `json:"status"` // 状态字段："completed", "not_started", "in_progress"
}

// ExerciseCategoryWithQuestions 带题目状态的练习分类
type ExerciseCategoryWithQuestions struct {
	model.ExerciseCategory
	Questions   []QuestionWithUserStatus `json:"questions"`
	IsCompleted bool                     `json:"isCompleted"`
	Status      string                   `json:"status"`
}

// CProgrammingResourceService 处理C语言编程资源分类模块的业务逻辑
type CProgrammingResourceService struct {
	Repo                   *repository.CProgrammingResourceRepository
	CategoryRepo           *repository.ExerciseCategoryRepository
	QuestionRepo           *repository.ExerciseQuestionRepository
	SubmissionRepo         *repository.ExerciseSubmissionRepository
	ResourceRepo           *repository.ResourceRepository
	ResourceCompletionRepo *repository.ResourceCompletionRepository
	GoalRepo               *repository.GoalRepository
	TaskRepo               *repository.TaskRepository
	TaskService            *TaskService // 添加任务服务
	DB                     *gorm.DB
}

func NewCProgrammingResourceService(
	repo *repository.CProgrammingResourceRepository,
	categoryRepo *repository.ExerciseCategoryRepository,
	questionRepo *repository.ExerciseQuestionRepository,
	submissionRepo *repository.ExerciseSubmissionRepository,
	resourceRepo *repository.ResourceRepository,
	resourceCompletionRepo *repository.ResourceCompletionRepository,
	goalRepo *repository.GoalRepository,
	taskRepo *repository.TaskRepository,
	taskService *TaskService, // 添加任务服务参数
	db *gorm.DB,
) *CProgrammingResourceService {
	return &CProgrammingResourceService{
		Repo:                   repo,
		CategoryRepo:           categoryRepo,
		QuestionRepo:           questionRepo,
		SubmissionRepo:         submissionRepo,
		ResourceRepo:           resourceRepo,
		ResourceCompletionRepo: resourceCompletionRepo,
		GoalRepo:               goalRepo,
		TaskRepo:               taskRepo,
		TaskService:            taskService,
		DB:                     db,
	}
}

// CreateResource 创建新的C语言资源分类模块
func (s *CProgrammingResourceService) CreateResource(resource *model.CProgrammingResource) error {
	return s.Repo.Create(resource)
}

// UpdateResource 更新C语言资源分类模块
func (s *CProgrammingResourceService) UpdateResource(resource *model.CProgrammingResource) error {
	return s.Repo.Update(resource)
}

// DeleteResource 删除C语言资源分类模块
func (s *CProgrammingResourceService) DeleteResource(id uint) error {
	return s.Repo.Delete(id)
}

// GetResources 获取所有C语言资源分类模块，支持分页和筛选
func (s *CProgrammingResourceService) GetResources(page, limit int, enabled *bool) ([]model.CProgrammingResource, int, error) {
	return s.Repo.FindAll(page, limit, "", enabled, "order", "asc")
}

// GetResourcesWithStats 获取所有C语言资源分类模块（带统计信息），支持分页、筛选、搜索和排序
func (s *CProgrammingResourceService) GetResourcesWithStats(page, limit int, search string, enabled *bool, sortBy, sortOrder string) (map[string]interface{}, error) {
	// 获取资源列表
	resources, total, err := s.Repo.FindAll(page, limit, search, enabled, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}

	// 为每个资源添加统计信息
	resourcesWithStats := make([]map[string]interface{}, 0, len(resources))
	for _, resource := range resources {
		resourceMap := map[string]interface{}{
			"id":          resource.ID,
			"name":        resource.Name,
			"iconURL":     resource.IconURL,
			"description": resource.Description,
			"enabled":     resource.Enabled,
			"order":       resource.Order,
			"createdAt":   resource.CreatedAt,
			"updatedAt":   resource.UpdatedAt,
		}

		// 获取视频数量
		videos, videoCount, _ := s.GetVideosByResourceID(resource.ID, 1, 1)
		if len(videos) > 0 || videoCount > 0 {
			resourceMap["videoCount"] = videoCount
		} else {
			resourceMap["videoCount"] = 0
		}

		// 获取文章数量
		articles, articleCount, _ := s.GetArticlesByResourceID(resource.ID, 1, 1)
		if len(articles) > 0 || articleCount > 0 {
			resourceMap["articleCount"] = articleCount
		} else {
			resourceMap["articleCount"] = 0
		}

		// 获取练习题分类数量
		categories, _ := s.GetCategoriesByResourceID(resource.ID)
		resourceMap["exerciseCategoryCount"] = len(categories)

		resourcesWithStats = append(resourcesWithStats, resourceMap)
	}

	// 计算分页信息
	totalPages := (total + limit - 1) / limit
	pagination := map[string]interface{}{
		"currentPage": page,
		"pageSize":    limit,
		"totalItems":  total,
		"totalPages":  totalPages,
		"hasNext":     page < totalPages,
		"hasPrev":     page > 1,
	}

	result := map[string]interface{}{
		"resources":  resourcesWithStats,
		"pagination": pagination,
	}

	return result, nil
}

// GetResourceByID 根据ID获取C语言资源分类模块
func (s *CProgrammingResourceService) GetResourceByID(id uint) (*model.CProgrammingResource, error) {
	return s.Repo.FindByID(id)
}

// CreateCategory 创建新的练习题分类
func (s *CProgrammingResourceService) CreateCategory(category *model.ExerciseCategory) error {
	return s.CategoryRepo.Create(category)
}

// GetCategoriesByResourceID 根据资源ID获取练习题分类
func (s *CProgrammingResourceService) GetCategoriesByResourceID(resourceID uint) ([]model.ExerciseCategory, error) {
	return s.CategoryRepo.FindByResourceID(resourceID)
}

// CreateQuestion 创建新的练习题题目
func (s *CProgrammingResourceService) CreateQuestion(question *model.ExerciseQuestion) error {
	return s.QuestionRepo.Create(question)
}

// GetQuestionsByCategoryID 根据分类ID获取练习题题目，支持分页
func (s *CProgrammingResourceService) GetQuestionsByCategoryID(categoryID uint, page, limit int) ([]model.ExerciseQuestion, int, error) {
	return s.QuestionRepo.FindByCategoryID(categoryID, page, limit)
}

// GetVideosByResourceID 根据资源ID获取视频列表，支持分页
func (s *CProgrammingResourceService) GetVideosByResourceID(resourceID uint, page, limit int) ([]model.Resource, int, error) {
	offset := (page - 1) * limit
	var videos []model.Resource
	var total int64

	query := s.ResourceRepo.DB.Where("module_id = ? AND type = ?", resourceID, model.Video)
	err := query.Model(&model.Resource{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&videos).Error

	return videos, int(total), err
}

// GetArticlesByResourceID 根据资源ID获取文章列表，支持分页
func (s *CProgrammingResourceService) GetArticlesByResourceID(resourceID uint, page, limit int) ([]model.Resource, int, error) {
	offset := (page - 1) * limit
	var articles []model.Resource
	var total int64

	query := s.ResourceRepo.DB.Where("module_id = ? AND type = ?", resourceID, model.Article)
	err := query.Model(&model.Resource{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&articles).Error

	return articles, int(total), err
}

// 根据资源ID获取所有视频（不分页）
func (s *CProgrammingResourceService) GetAllVideosByResourceID(resourceID uint) ([]model.Resource, error) {
	var videos []model.Resource
	err := s.ResourceRepo.DB.Where("module_id = ? AND type = ?", resourceID, model.Video).Find(&videos).Error
	return videos, err
}

// 根据资源ID获取所有文章（不分页）
func (s *CProgrammingResourceService) GetAllArticlesByResourceID(resourceID uint) ([]model.Resource, error) {
	var articles []model.Resource
	err := s.ResourceRepo.DB.Where("module_id = ? AND type = ?", resourceID, model.Article).Find(&articles).Error
	return articles, err
}

// UpdateVideo 更新视频
func (s *CProgrammingResourceService) UpdateVideo(videoID uint, updates map[string]interface{}) error {
	return s.ResourceRepo.UpdateFields(videoID, model.Video, updates)
}

// UpdateArticle 更新文章
func (s *CProgrammingResourceService) UpdateArticle(articleID uint, updates map[string]interface{}) error {
	return s.ResourceRepo.UpdateFields(articleID, model.Article, updates)
}

// UpdateExerciseCategory 更新练习分类
func (s *CProgrammingResourceService) UpdateExerciseCategory(id uint, updates map[string]interface{}) error {
	return s.CategoryRepo.UpdateFields(id, updates)
}

// UpdateExerciseQuestionFields 更新练习题目字段
func (s *CProgrammingResourceService) UpdateExerciseQuestionFields(id uint, updates map[string]interface{}) error {
	return s.QuestionRepo.UpdateFields(id, updates)
}

// DeleteContentItem 删除内容项
func (s *CProgrammingResourceService) DeleteContentItem(itemType string, itemID uint) error {
	switch itemType {
	case "videos":
		return s.ResourceRepo.DeleteByType(itemID, model.Video)
	case "articles":
		return s.ResourceRepo.DeleteByType(itemID, model.Article)
	case "exercise-categories":
		return s.CategoryRepo.Delete(itemID)
	case "questions":
		return s.QuestionRepo.Delete(itemID)
	default:
		return fmt.Errorf("unsupported item type: %s", itemType)
	}
}

// UpdateQuestion 更新练习题题目信息
func (s *CProgrammingResourceService) UpdateQuestion(question *model.ExerciseQuestion) error {
	return s.QuestionRepo.UpdateQuestion(question)
}

// GetAllQuestionsByCategoryID 获取指定分类下的所有练习题题目
func (s *CProgrammingResourceService) GetAllQuestionsByCategoryID(categoryID uint) ([]model.ExerciseQuestion, int, error) {
	questions, err := s.QuestionRepo.FindAllByCategoryID(categoryID)
	if err != nil {
		return nil, 0, err
	}
	return questions, len(questions), nil
}

// GetResourcesWithAllContent 获取所有资源分类及其完整内容（支持分页）
func (s *CProgrammingResourceService) GetResourcesWithAllContent(enabled *bool, page, limit int, userID uint) ([]map[string]interface{}, int, error) {
	fmt.Printf("[DEBUG] === GetResourcesWithAllContent 开始 ===\n")
	fmt.Printf("[DEBUG] 参数: page=%d, limit=%d, userID=%d, enabled=%v\n", page, limit, userID, enabled)

	// 获取分页的资源分类
	resources, total, err := s.Repo.FindAll(page, limit, "", enabled, "order", "asc")
	fmt.Printf("[DEBUG] FindAll返回: 资源数量=%d, 错误=%v\n", len(resources), err)

	// 显示所有资源的ID和名称
	for i, res := range resources {
		fmt.Printf("[DEBUG] 资源[%d] - ID: %d, 名称: %s\n", i, res.ID, res.Name)
	}

	// 特别检查资源模块24
	fmt.Printf("[DEBUG] === 特别检查资源模块24 ===\n")
	for _, res := range resources {
		if res.ID == 24 {
			fmt.Printf("[DEBUG] 找到资源模块24: 名称=%s, 描述=%s\n", res.Name, res.Description)

			// 检查资源模块24的内容
			videos24, _ := s.GetAllVideosByResourceID(24)
			articles24, _ := s.GetAllArticlesByResourceID(24)
			categories24, _ := s.GetCategoriesByResourceID(24)

			fmt.Printf("[DEBUG] 资源模块24内容统计: 视频=%d, 文章=%d, 练习分类=%d\n",
				len(videos24), len(articles24), len(categories24))
			break
		}
	}
	if err != nil {
		return nil, 0, err
	}

	result := make([]map[string]interface{}, 0, len(resources))

	for _, resource := range resources {
		resourceMap := map[string]interface{}{
			"id":          resource.ID,
			"name":        resource.Name,
			"iconURL":     resource.IconURL,
			"description": resource.Description,
			"enabled":     resource.Enabled,
			"order":       resource.Order,
			"createdAt":   resource.CreatedAt,
			"updatedAt":   resource.UpdatedAt,
		}

		// 检查当前用户是否有与该资源模块关联的学习目标或当天老师布置的周任务
		hasLearningGoal := false
		if userID > 0 {
			// fmt.Printf("[DEBUG] 检查资源模块ID: %d, 用户ID: %d\n", resource.ID, userID)

			// 检查个人学习目标
			if s.GoalRepo != nil {
				goals, err := s.GoalRepo.FindByUserIDAndResourceModuleID(userID, resource.ID)
				// fmt.Printf("[DEBUG] 个人学习目标检查结果 - 错误: %v, 目标数量: %d\n", err, len(goals))
				if err == nil && len(goals) > 0 {
					hasLearningGoal = true
					// fmt.Printf("[DEBUG] 发现个人学习目标，hasLearningGoal设置为true\n")
				}
			}

			// 检查用户当天在该资源模块下的周任务（无论是否有个人学习目标都要检查）
			if s.TaskService != nil {
				// fmt.Printf("[DEBUG] 开始检查老师布置的当天任务\n")
				// 获取用户今天的所有任务
				todayTasks, err := s.TaskService.GetTodayTasks(userID, resource.ID)
				// fmt.Printf("[DEBUG] 获取今天任务结果 - 错误: %v, 任务数量: %d\n", err, len(todayTasks))
				if err == nil && len(todayTasks) > 0 {
					// 检查是否有任务属于当前资源模块
					// fmt.Printf("[DEBUG] 开始遍历今天任务，查找资源模块ID: %d (资源名称: %s)\n", resource.ID, resource.Name)
					for i, task := range todayTasks {
						if taskResourceModuleID, ok := task["resourceModuleId"].(uint); ok {
							// fmt.Printf("[DEBUG] 任务[%d] - resourceModuleId: %d, 当前资源模块ID: %d, 匹配: %v\n",
							// 	i, taskResourceModuleID, resource.ID, taskResourceModuleID == resource.ID)
							if taskResourceModuleID == resource.ID {
								hasLearningGoal = true
								// fmt.Printf("[DEBUG] 发现匹配的当天任务，hasLearningGoal设置为true\n")
								break
							}
						} else {
							fmt.Printf("[DEBUG] 任务[%d] - resourceModuleId类型断言失败或不存在\n", i)
						}
					}
				} else {
					fmt.Printf("[DEBUG] 没有获取到今天的任务或发生错误\n")
				}
			}
			// fmt.Printf("[DEBUG] 最终hasLearningGoal结果: %v\n", hasLearningGoal)
		}
		resourceMap["hasLearningGoal"] = hasLearningGoal

		// 获取所有视频
		videos, err := s.GetAllVideosByResourceID(resource.ID)
		if err != nil {
			return nil, 0, err
		}
		resourceMap["videos"] = videos

		// 获取所有文章
		articles, err := s.GetAllArticlesByResourceID(resource.ID)
		if err != nil {
			return nil, 0, err
		}
		resourceMap["articles"] = articles

		// 获取所有练习题分类及题目
		categories, err := s.GetCategoriesByResourceID(resource.ID)
		if err != nil {
			return nil, 0, err
		}

		categoriesWithQuestions := make([]map[string]interface{}, 0, len(categories))
		for _, category := range categories {
			categoryMap := map[string]interface{}{
				"id":          category.ID,
				"name":        category.Name,
				"description": category.Description,
				"order":       category.Order,
				"createdAt":   category.CreatedAt,
				"updatedAt":   category.UpdatedAt,
			}

			// 获取当前分类下的所有题目
			questions, _, _ := s.GetAllQuestionsByCategoryID(category.ID)
			categoryMap["questions"] = questions

			categoriesWithQuestions = append(categoriesWithQuestions, categoryMap)
		}

		resourceMap["exerciseCategories"] = categoriesWithQuestions
		result = append(result, resourceMap)
	}

	return result, total, nil
}

// GetQuestionsByCategoryIDWithUserStatus 根据分类ID获取练习题题目并添加用户提交状态
type QuestionWithUserStatus struct {
	model.ExerciseQuestion
	IsSubmitted bool `json:"isSubmitted"`
}

func (s *CProgrammingResourceService) GetQuestionsByCategoryIDWithUserStatus(categoryID, userID uint, page, limit int) ([]QuestionWithUserStatus, int, error) {
	// 获取题目列表
	questions, total, err := s.QuestionRepo.FindQuestionsByCategoryIDWithPagination(categoryID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	// 为每个题目添加用户提交状态
	questionsWithStatus := make([]QuestionWithUserStatus, 0, len(questions))
	for _, question := range questions {
		isSubmitted := false
		// 检查用户是否提交过该题目
		if userID > 0 {
			submission, err := s.SubmissionRepo.FindByUserAndQuestion(userID, question.ID)
			if err == nil && submission.IsCorrect {
				isSubmitted = true
			}
		}

		questionsWithStatus = append(questionsWithStatus, QuestionWithUserStatus{
			ExerciseQuestion: question,
			IsSubmitted:      isSubmitted,
		})
	}

	return questionsWithStatus, total, nil
}

// SubmitExerciseAnswer 提交练习答案
type SubmitExerciseAnswerRequest struct {
	UserID uint   `json:"user_id"`
	Answer string `json:"answer" binding:"required"`
}

func (s *CProgrammingResourceService) SubmitExerciseAnswer(questionID uint, req SubmitExerciseAnswerRequest) (bool, error) {
	// 事务处理
	tx := s.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取题目信息
	question, err := s.QuestionRepo.FindByID(questionID)
	if err != nil {
		tx.Rollback()
		return false, err
	}

	// 检查答案是否正确
	isCorrect := question.CorrectAnswer == req.Answer

	// 检查是否已经有提交记录
	submission, err := s.SubmissionRepo.FindByUserAndQuestion(req.UserID, questionID)
	if err != nil {
		// 创建新的提交记录
		submission = &model.ExerciseSubmission{
			UserID:          req.UserID,
			QuestionID:      questionID,
			SubmittedAnswer: req.Answer,
			IsCorrect:       isCorrect,
		}
		if err := tx.Create(submission).Error; err != nil {
			tx.Rollback()
			return false, err
		}
	} else {
		// 更新现有提交记录
		submission.SubmittedAnswer = req.Answer
		submission.IsCorrect = isCorrect
		if err := tx.Save(submission).Error; err != nil {
			tx.Rollback()
			return false, err
		}
	}

	// 提交事务
	tx.Commit()

	// 如果答案正确且任务服务可用，尝试将对应的今日任务标记为已完成
	if isCorrect && s.TaskService != nil {
		// 计算本周的开始和结束日期
		today := time.Now()
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		weekStart := time.Date(today.Year(), today.Month(), today.Day()-weekday+1, 0, 0, 0, 0, today.Location())
		weekEnd := weekStart.AddDate(0, 0, 6)

		// 计算今天对应的 dayOfWeek 字符串（与 model.Weekday 常量一致，小写）
		var dayOfWeek model.Weekday
		switch time.Now().Weekday() {
		case time.Monday:
			dayOfWeek = model.Monday
		case time.Tuesday:
			dayOfWeek = model.Tuesday
		case time.Wednesday:
			dayOfWeek = model.Wednesday
		case time.Thursday:
			dayOfWeek = model.Thursday
		case time.Friday:
			dayOfWeek = model.Friday
		case time.Saturday:
			dayOfWeek = model.Saturday
		case time.Sunday:
			dayOfWeek = model.Sunday
		}

		// 在当前周中查找与该题目对应的 task_item（exercise_id）
		if taskItem, err := s.TaskRepo.FindTaskItemByExerciseAndWeek(questionID, dayOfWeek, weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02")); err == nil {
			// 标记为已完成（进度 100）
			_ = s.TaskService.UpdateTaskCompletion(req.UserID, taskItem.ID, true, 100.0, true)
		}
	}

	return isCorrect, nil
}

// CheckUserSubmittedQuestion 检查用户是否提交过特定题目
func (s *CProgrammingResourceService) CheckUserSubmittedQuestion(userID, questionID uint) (bool, error) {
	submission, err := s.SubmissionRepo.FindByUserAndQuestion(userID, questionID)
	if err != nil {
		// 如果记录不存在，返回false
		if err.Error() == "record not found" {
			return false, nil
		}
		return false, err
	}
	// 如果记录存在且答案正确，则表示已提交
	return submission.IsCorrect, nil
}

//获取带进度的资源模块

func (s *CProgrammingResourceService) GetResourceModuleWithProgress(resourceID, userID uint) (*ResourceModuleWithProgress, error) {
	// 获取资源模块信息
	resource, err := s.Repo.FindByID(resourceID)
	if err != nil {
		return nil, err
	}

	// 构建结果对象
	result := &ResourceModuleWithProgress{
		CProgrammingResource: *resource,
		Videos:               []ResourceWithCompletionStatus{},
		Articles:             []ResourceWithCompletionStatus{},
		ExerciseCategory:     []ExerciseCategoryWithQuestions{},
	}

	// 计算总进度
	totalItems := 0
	completedItems := 0

	// 获取并添加视频状态
	videos, err := s.GetAllVideosByResourceID(resourceID)
	if err == nil {
		videoIDs := make([]uint, len(videos))
		for i, video := range videos {
			videoIDs[i] = video.ID
		}

		// 获取视频完成状态
		videoCompletions, _ := s.ResourceCompletionRepo.GetUserResourceCompletions(userID, videoIDs)

		// 添加带状态的视频
		for _, video := range videos {
			isCompleted := videoCompletions[video.ID]
			result.Videos = append(result.Videos, ResourceWithCompletionStatus{
				Resource:    video,
				IsCompleted: isCompleted,
			})

			totalItems++
			if isCompleted {
				completedItems++
			}
		}
	}

	// 获取并添加文章状态
	articles, err := s.GetAllArticlesByResourceID(resourceID)
	if err == nil {
		articleIDs := make([]uint, len(articles))
		for i, article := range articles {
			articleIDs[i] = article.ID
		}

		// 获取文章完成状态
		articleCompletions, _ := s.ResourceCompletionRepo.GetUserResourceCompletions(userID, articleIDs)

		// 添加带状态的文章
		for _, article := range articles {
			isCompleted := articleCompletions[article.ID]
			result.Articles = append(result.Articles, ResourceWithCompletionStatus{
				Resource:    article,
				IsCompleted: isCompleted,
			})

			totalItems++
			if isCompleted {
				completedItems++
			}
		}
	}

	// 获取并添加练习分类和题目状态
	categories, err := s.GetCategoriesByResourceID(resourceID)
	if err == nil {
		for _, category := range categories {
			// 获取分类下的所有题目
			questions, _, err := s.QuestionRepo.FindQuestionsByCategoryIDWithPagination(category.ID, 1, 1000) // 获取所有题目
			if err != nil {
				continue
			}

			categoryWithQuestions := ExerciseCategoryWithQuestions{
				ExerciseCategory: category,
				Questions:        []QuestionWithUserStatus{},
				IsCompleted:      true, // 默认为已完成，如有未完成的题目则设为false
				Status:           "completed",
			}

			var categoryCompletedItems = 0

			// 添加题目状态
			for _, question := range questions {
				isSubmitted := false
				if userID > 0 {
					submission, err := s.SubmissionRepo.FindByUserAndQuestion(userID, question.ID)
					if err == nil && submission.IsCorrect {
						isSubmitted = true
					} else if err != nil || !submission.IsCorrect {
						// 只有在确定题目未完成时才设置分类未完成
						categoryWithQuestions.IsCompleted = false
					}
				} else {
					categoryWithQuestions.IsCompleted = false
				}

				categoryWithQuestions.Questions = append(categoryWithQuestions.Questions, QuestionWithUserStatus{
					ExerciseQuestion: question,
					IsSubmitted:      isSubmitted,
				})

				if isSubmitted {
					categoryCompletedItems++
				}

				totalItems++

				if isSubmitted {
					completedItems++
				}
			}

			if len(questions) == 0 {
				categoryWithQuestions.Status = "not_started"
			} else if categoryCompletedItems == len(questions) {
				categoryWithQuestions.Status = "completed"
			} else if categoryCompletedItems == 0 {
				categoryWithQuestions.Status = "not_started"
			} else {
				categoryWithQuestions.Status = "in_progress"
			}

			result.ExerciseCategory = append(result.ExerciseCategory, categoryWithQuestions)
		}
	}

	// 计算进度百分比
	if totalItems > 0 {
		result.Progress = float64(completedItems) / float64(totalItems) * 100
	} else {
		result.Progress = 0
	}

	// 判断模块是否完全完成
	result.IsCompleted = totalItems > 0 && completedItems == totalItems

	// 设置三种状态
	if totalItems == 0 {
		result.Status = "not_started" // 没有项目时视为未开始
	} else if completedItems == totalItems {
		result.Status = "completed" // 所有项目都已完成
	} else if completedItems == 0 {
		result.Status = "not_started" // 有项目但没有完成的，视为未开始
	} else {
		result.Status = "in_progress" // 部分完成，视为进行中
	}

	return result, nil
}

// 更新资源完成状态
func (s *CProgrammingResourceService) UpdateResourceCompletionStatus(userID, resourceID uint, completed bool) error {
	return s.ResourceCompletionRepo.UpdateCompletionStatus(userID, resourceID, completed)
}

// GetUnfinishedResourceModules 获取未完成的资源模块列表（带进度）
func (s *CProgrammingResourceService) GetUnfinishedResourceModules(userID uint, limit int) ([]*ResourceModuleWithProgress, error) {
	// 1. 获取所有资源模块
	allResources, _, err := s.GetResources(1, 1000, nil) // 获取所有启用的资源模块
	if err != nil {
		return nil, err
	}

	// 2. 筛选出未完成的资源模块
	var unfinishedModules []*ResourceModuleWithProgress

	for _, resource := range allResources {
		module, err := s.GetResourceModuleWithProgress(resource.ID, userID)
		if err != nil {
			continue
		}

		// 检查是否有未完成的内容（视频、文章或练习题）
		hasUnfinishedVideos := false
		for _, video := range module.Videos {
			if !video.IsCompleted {
				hasUnfinishedVideos = true
				break
			}
		}

		hasUnfinishedArticles := false
		for _, article := range module.Articles {
			if !article.IsCompleted {
				hasUnfinishedArticles = true
				break
			}
		}

		hasUnfinishedExercises := false
		for _, category := range module.ExerciseCategory {
			if !category.IsCompleted {
				hasUnfinishedExercises = true
				break
			}
		}

		// 只要有任一类型的未完成内容，就加入结果集
		if hasUnfinishedVideos || hasUnfinishedArticles || hasUnfinishedExercises {
			unfinishedModules = append(unfinishedModules, module)
		}
	}

	// 3. 随机选择指定数量的模块（最多3个）
	if len(unfinishedModules) > limit {
		// 使用随机数打乱顺序
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(unfinishedModules), func(i, j int) {
			unfinishedModules[i], unfinishedModules[j] = unfinishedModules[j], unfinishedModules[i]
		})
		// 返回前limit个结果
		unfinishedModules = unfinishedModules[:limit]
	}

	return unfinishedModules, nil
}

// GetAllResourceModulesWithProgress 获取所有带进度的资源模块
func (s *CProgrammingResourceService) GetAllResourceModulesWithProgress(userID uint, enabled *bool) ([]*ResourceModuleWithProgress, error) {
	// 获取所有资源模块
	resources, _, err := s.Repo.FindAll(1, 1000, "", enabled, "order", "asc")
	if err != nil {
		return nil, err
	}

	// 为每个资源模块获取进度信息
	var result []*ResourceModuleWithProgress
	for _, resource := range resources {
		module, err := s.GetResourceModuleWithProgress(resource.ID, userID)
		if err != nil {
			continue
		}
		result = append(result, module)
	}

	return result, nil
}
