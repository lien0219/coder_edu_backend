package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"

	"gorm.io/gorm"
)

// CProgrammingResourceService 处理C语言编程资源分类模块的业务逻辑

type CProgrammingResourceService struct {
	Repo           *repository.CProgrammingResourceRepository
	CategoryRepo   *repository.ExerciseCategoryRepository
	QuestionRepo   *repository.ExerciseQuestionRepository
	SubmissionRepo *repository.ExerciseSubmissionRepository
	ResourceRepo   *repository.ResourceRepository
	DB             *gorm.DB
}

func NewCProgrammingResourceService(
	repo *repository.CProgrammingResourceRepository,
	categoryRepo *repository.ExerciseCategoryRepository,
	questionRepo *repository.ExerciseQuestionRepository,
	submissionRepo *repository.ExerciseSubmissionRepository,
	resourceRepo *repository.ResourceRepository,
	db *gorm.DB,
) *CProgrammingResourceService {
	return &CProgrammingResourceService{
		Repo:           repo,
		CategoryRepo:   categoryRepo,
		QuestionRepo:   questionRepo,
		SubmissionRepo: submissionRepo,
		ResourceRepo:   resourceRepo,
		DB:             db,
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
