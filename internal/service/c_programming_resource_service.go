package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
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
	DB                     *gorm.DB
}

func NewCProgrammingResourceService(
	repo *repository.CProgrammingResourceRepository,
	categoryRepo *repository.ExerciseCategoryRepository,
	questionRepo *repository.ExerciseQuestionRepository,
	submissionRepo *repository.ExerciseSubmissionRepository,
	resourceRepo *repository.ResourceRepository,
	resourceCompletionRepo *repository.ResourceCompletionRepository,
	db *gorm.DB,
) *CProgrammingResourceService {
	return &CProgrammingResourceService{
		Repo:                   repo,
		CategoryRepo:           categoryRepo,
		QuestionRepo:           questionRepo,
		SubmissionRepo:         submissionRepo,
		ResourceRepo:           resourceRepo,
		ResourceCompletionRepo: resourceCompletionRepo,
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

	query := s.DB.Where("module_id = ? AND type = ?", resourceID, model.Video)
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

	query := s.DB.Where("module_id = ? AND type = ?", resourceID, model.Article)
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
	err := s.DB.Where("module_id = ? AND type = ?", resourceID, model.Video).Find(&videos).Error
	return videos, err
}

// 根据资源ID获取所有文章（不分页）
func (s *CProgrammingResourceService) GetAllArticlesByResourceID(resourceID uint) ([]model.Resource, error) {
	var articles []model.Resource
	err := s.DB.Where("module_id = ? AND type = ?", resourceID, model.Article).Find(&articles).Error
	return articles, err
}

// 更新视频
func (s *CProgrammingResourceService) UpdateVideo(videoID uint, updates map[string]interface{}) error {
	return s.DB.Model(&model.Resource{}).
		Where("id = ? AND type = ?", videoID, model.Video).
		Updates(updates).Error
}

// 更新文章
func (s *CProgrammingResourceService) UpdateArticle(articleID uint, updates map[string]interface{}) error {
	return s.DB.Model(&model.Resource{}).
		Where("id = ? AND type = ?", articleID, model.Article).
		Updates(updates).Error
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
func (s *CProgrammingResourceService) GetResourcesWithAllContent(enabled *bool, page, limit int) ([]map[string]interface{}, int, error) {
	// 获取分页的资源分类
	resources, total, err := s.Repo.FindAll(page, limit, "", enabled, "order", "asc")
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
	UserID uint   `json:"user_id" binding:"required"`
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
