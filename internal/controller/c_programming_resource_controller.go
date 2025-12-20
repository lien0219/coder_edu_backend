package controller

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CProgrammingResourceController 处理C语言编程资源分类模块的API请求
type CProgrammingResourceController struct {
	Service        *service.CProgrammingResourceService
	ContentService *service.ContentService
}

func NewCProgrammingResourceController(
	service *service.CProgrammingResourceService,
	contentService *service.ContentService,
) *CProgrammingResourceController {
	return &CProgrammingResourceController{
		Service:        service,
		ContentService: contentService,
	}
}

// @Summary 创建C语言资源分类
// @Description 创建新的C语言资源分类模块（需要管理员权限）
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param resource body model.CProgrammingResource true "C语言资源分类信息"
// @Success 201 {object} util.Response
// @Router /api/admin/c-programming/resources [post]
func (c *CProgrammingResourceController) CreateResource(ctx *gin.Context) {
	var requestResource struct {
		Name        string `json:"name" binding:"required,max=255"`
		IconURL     string `json:"iconURL" binding:"required,url,max=255"`
		Description string `json:"description" binding:"max=1000"`
		Enabled     bool   `json:"enabled"`
		Order       int    `json:"order"`
	}

	if err := ctx.ShouldBindJSON(&requestResource); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	resource := &model.CProgrammingResource{
		Name:        requestResource.Name,
		IconURL:     requestResource.IconURL,
		Description: requestResource.Description,
		Enabled:     requestResource.Enabled,
		Order:       requestResource.Order,
	}

	err := c.Service.CreateResource(resource)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, resource)
}

// GetAdminResources godoc
// @Summary 管理员获取所有C语言资源分类列表（支持搜索和分页）
// @Description 管理员获取所有C语言资源分类列表，支持搜索、分页、筛选和排序
// @Tags C语言编程资源
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param page query int false "页码，从1开始" default(1)
// @Param limit query int false "每页记录数" default(10)
// @Param search query string false "搜索关键词，匹配名称或描述"
// @Param enabled query boolean false "过滤启用/禁用的资源"
// @Param sortBy query string false "排序字段：name, order, createdAt, updatedAt" default(order)
// @Param sortOrder query string false "排序方向：asc(升序), desc(降序)" default(asc)
// @Success 200 {object} util.Response{data=map[string]interface{}}
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/c-programming/resources [get]
func (c *CProgrammingResourceController) GetAdminResources(ctx *gin.Context) {
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	search := ctx.Query("search")
	enabledStr := ctx.Query("enabled")
	sortBy := ctx.DefaultQuery("sortBy", "order")
	sortOrder := ctx.DefaultQuery("sortOrder", "asc")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		util.BadRequest(ctx, "页码必须大于0")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		util.BadRequest(ctx, "每页记录数必须大于0")
		return
	}

	validSortFields := map[string]bool{
		"name":      true,
		"order":     true,
		"createdAt": true,
		"updatedAt": true,
	}
	if !validSortFields[sortBy] {
		util.BadRequest(ctx, "无效的排序字段")
		return
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		util.BadRequest(ctx, "排序方向必须是asc或desc")
		return
	}

	// 处理enabled参数
	var enabled *bool
	if enabledStr != "" {
		enabledVal, err := strconv.ParseBool(enabledStr)
		if err != nil {
			util.BadRequest(ctx, "enabled必须是布尔值")
			return
		}
		enabled = &enabledVal
	}

	result, err := c.Service.GetResourcesWithStats(page, limit, search, enabled, sortBy, sortOrder)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, result)
}

// @Summary 更新C语言资源分类
// @Description 更新C语言资源分类模块（需要管理员权限）
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源ID"
// @Param resource body model.CProgrammingResource true "C语言资源分类更新信息"
// @Success 200 {object} util.Response
// @Router /api/admin/c-programming/resources/{id} [put]
func (c *CProgrammingResourceController) UpdateResource(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	var resource model.CProgrammingResource
	if err := ctx.ShouldBindJSON(&resource); err != nil {
		util.BadRequest(ctx, "Invalid request body")
		return
	}

	resource.ID = uint(id)
	err = c.Service.UpdateResource(&resource)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, resource)
}

// @Summary 删除C语言资源分类
// @Description 删除C语言资源分类模块（需要管理员权限）
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源ID"
// @Success 200 {object} util.Response
// @Router /api/admin/c-programming/resources/{id} [delete]
func (c *CProgrammingResourceController) DeleteResource(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	err = c.Service.DeleteResource(uint(id))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, nil)
}

// @Summary 获取C语言资源分类列表
// @Description 获取所有C语言资源分类模块，支持分页和筛选
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(9)
// @Param enabled query boolean false "是否启用"
// @Success 200 {object} util.Response
// @Router /api/c-programming/resources [get]
func (c *CProgrammingResourceController) GetResources(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "9"))
	enabledStr := ctx.Query("enabled")

	var enabled *bool
	if enabledStr != "" {
		val := enabledStr == "true"
		enabled = &val
	}

	resources, total, err := c.Service.GetResources(page, limit, enabled)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"resources": resources,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

// @Summary 获取C语言资源分类详情
// @Description 获取单个C语言资源分类模块
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源ID"
// @Success 200 {object} util.Response
// @Router /api/c-programming/resources/{id} [get]
func (c *CProgrammingResourceController) GetResourceByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	resource, err := c.Service.GetResourceByID(uint(id))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, resource)
}

// @Summary 创建练习题分类
// @Description 为指定C语言资源创建新的练习题分类（需要管理员权限）
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源ID"
// @Param category body model.ExerciseCategory true "练习题分类信息"
// @Success 201 {object} util.Response
// @Router /api/admin/c-programming/resources/{id}/categories [post]
func (c *CProgrammingResourceController) CreateCategory(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	var category model.ExerciseCategory
	if err := ctx.ShouldBindJSON(&category); err != nil {
		util.BadRequest(ctx, "Invalid request body")
		return
	}

	category.CProgrammingResID = uint(id)
	err = c.Service.CreateCategory(&category)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, category)
}

// @Summary 获取练习题分类列表
// @Description 获取指定C语言资源下的所有练习题分类
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源ID"
// @Success 200 {object} util.Response
// @Router /api/c-programming/resources/{id}/categories [get]
func (c *CProgrammingResourceController) GetCategoriesByResourceID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	categories, err := c.Service.GetCategoriesByResourceID(uint(id))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, categories)
}

// @Summary 创建练习题题目
// @Description 为指定练习题分类创建新的练习题题目（需要管理员权限）
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param categoryId path int true "分类ID"
// @Param question body model.ExerciseQuestion true "练习题题目信息"
// @Success 201 {object} util.Response
// @Router /api/admin/c-programming/categories/{categoryId}/questions [post]
func (c *CProgrammingResourceController) CreateQuestion(ctx *gin.Context) {
	categoryIDStr := ctx.Param("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid category ID")
		return
	}

	var question model.ExerciseQuestion
	if err := ctx.ShouldBindJSON(&question); err != nil {
		util.BadRequest(ctx, "Invalid request body")
		return
	}
	if question.Points < 0 {
		util.BadRequest(ctx, "积分不能为负数")
		return
	}

	// 验证题目类型和必填字段
	switch question.QuestionType {
	case "single_choice", "multiple_choice":
		// 选择题必须包含选项和正确答案
		if len(question.Options) == 0 {
			util.BadRequest(ctx, "选择题必须提供选项")
			return
		}
		if question.CorrectAnswer == "" {
			util.BadRequest(ctx, "选择题必须提供正确答案")
			return
		}
	case "programming":
		// 编程题可以省略选项和正确答案
		if question.SolutionCode == "" {
			util.BadRequest(ctx, "编程题必须提供解决方案代码")
			return
		}
	default:
		// 默认为编程题
		question.QuestionType = "programming"
	}

	question.CategoryID = uint(categoryID)
	err = c.Service.CreateQuestion(&question)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, question)
}

// @Summary 获取练习题题目列表
// @Description 获取指定练习题分类下的所有练习题题目，支持分页
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param categoryId path int true "分类ID"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(5)
// @Success 200 {object} util.Response
// @Router /api/c-programming/categories/{categoryId}/questions [get]
func (c *CProgrammingResourceController) GetQuestionsByCategoryID(ctx *gin.Context) {
	categoryIDStr := ctx.Param("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid category ID")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "5"))

	questions, total, err := c.Service.GetQuestionsByCategoryID(uint(categoryID), page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"questions": questions,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

// @Summary 获取C语言视频资源列表
// @Description 获取指定C语言资源下的所有视频，支持分页
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源ID"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(5)
// @Success 200 {object} util.Response
// @Router /api/c-programming/resources/{id}/videos [get]
func (c *CProgrammingResourceController) GetVideosByResourceID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "5"))

	videos, total, err := c.Service.GetVideosByResourceID(uint(id), page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"videos": videos,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// @Summary 获取C语言文章资源列表
// @Description 获取指定C语言资源下的所有文章，支持分页
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源ID"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(5)
// @Success 200 {object} util.Response
// @Router /api/c-programming/resources/{id}/articles [get]
func (c *CProgrammingResourceController) GetArticlesByResourceID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "5"))

	articles, total, err := c.Service.GetArticlesByResourceID(uint(id), page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"articles": articles,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// UploadResource 上传C语言编程资源文件
// @Summary 上传C语言编程资源文件
// @Description 上传视频、文章等资源文件到指定的C语言资源模块
// @Tags C语言编程资源
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param resource_id formData uint true "资源模块ID"
// @Param type formData string true "资源类型(video/article/pdf/worksheet)"
// @Param title formData string true "资源标题"
// @Param description formData string false "资源描述"
// @Param file formData file true "资源文件"
// @Success 201 {object} util.Response
// @Failure 400 {object} util.Response
// @Failure 401 {object} util.Response
// @Failure 500 {object} util.Response
// @Router /api/c-programming/resources/upload [post]
func (c *CProgrammingResourceController) UploadResource(ctx *gin.Context) {
	resourceID := ctx.PostForm("resource_id")
	resourceType := ctx.PostForm("type")
	title := ctx.PostForm("title")
	description := ctx.PostForm("description")

	if resourceID == "" || resourceType == "" || title == "" {
		util.BadRequest(ctx, "resource_id, type, and title are required")
		return
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		util.BadRequest(ctx, "file is required")
		return
	}

	resource := &model.Resource{
		ModuleID:    mustParseUint(resourceID),
		Type:        model.ResourceType(resourceType),
		Title:       title,
		Description: description,
	}

	if err := c.ContentService.UploadResource(ctx, file, resource); err != nil {
		util.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.Created(ctx, resource)
}

// 将字符串转换为uint
func mustParseUint(s string) uint {
	var id uint
	fmt.Sscanf(s, "%d", &id)
	return id
}

// GetResourceCompleteContent godoc
// @Summary 获取资源分类的完整内容
// @Description 获取指定资源分类的完整内容，包括视频、文章和练习题类目
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源ID"
// @Success 200 {object} util.Response
// @Router /api/admin/resources/{id}/content [get]
func (c *CProgrammingResourceController) GetResourceCompleteContent(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	// 获取资源基本信息
	resource, err := c.Service.GetResourceByID(uint(id))
	if err != nil {
		util.NotFound(ctx)
		return
	}

	// 获取视频列表
	videos, _, err := c.Service.GetVideosByResourceID(uint(id), 1, 1000) // 获取所有视频
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	// 获取文章列表
	articles, _, err := c.Service.GetArticlesByResourceID(uint(id), 1, 1000) // 获取所有文章
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	// 获取练习题类目及题目
	categories, err := c.Service.GetCategoriesByResourceID(uint(id))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	// 为每个类目获取题目
	categoriesWithQuestions := make([]map[string]interface{}, 0, len(categories))
	for _, category := range categories {
		questions, _, _ := c.Service.GetQuestionsByCategoryID(category.ID, 1, 1000)
		categoryMap := map[string]interface{}{
			"id":          category.ID,
			"name":        category.Name,
			"description": category.Description,
			"order":       category.Order,
			"createdAt":   category.CreatedAt,
			"questions":   questions,
		}
		categoriesWithQuestions = append(categoriesWithQuestions, categoryMap)
	}

	result := map[string]interface{}{
		"resource":           resource,
		"videos":             videos,
		"articles":           articles,
		"exerciseCategories": categoriesWithQuestions,
	}

	util.Success(ctx, result)
}

// AddVideoToResource godoc
// @Summary 添加视频到资源分类
// @Description 为指定资源分类添加新的视频
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源ID"
// @Param video body model.Resource true "视频信息"
// @Success 201 {object} util.Response
// @Router /api/admin/resources/{id}/videos [post]
func (c *CProgrammingResourceController) AddVideoToResource(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	var video struct {
		Title       string  `json:"title" binding:"required"`
		Description string  `json:"description"`
		URL         string  `json:"url" binding:"required"`
		Duration    float64 `json:"duration"`
		Order       int     `json:"order"`
		Points      int     `json:"points" binding:"gte=0"`
		Thumbnail   string  `json:"thumbnail"`
	}

	if err := ctx.ShouldBindJSON(&video); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	resource := &model.Resource{
		ModuleID:    uint(id),
		ModuleType:  "c_programming",
		Type:        model.Video,
		Title:       video.Title,
		Description: video.Description,
		URL:         video.URL,
		Duration:    video.Duration,
		Points:      video.Points,
		Thumbnail:   video.Thumbnail,
		// Duration和Order可以存储在额外字段中，这里使用description扩展
	}

	if err := c.ContentService.ResourceRepo.Create(resource); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, resource)
}

// AddArticleToResource godoc
// @Summary 添加文章到资源分类
// @Description 为指定资源分类添加新的文章
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源ID"
// @Param article body model.Resource true "文章信息"
// @Success 201 {object} util.Response
// @Router /api/admin/resources/{id}/articles [post]
func (c *CProgrammingResourceController) AddArticleToResource(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	var article struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Order   int    `json:"order"`
		Points  int    `json:"points" binding:"gte=0"`
	}

	if err := ctx.ShouldBindJSON(&article); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	// 获取当前登录用户信息
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	resource := &model.Resource{
		ModuleID:    uint(id),
		ModuleType:  "c_programming",
		Type:        model.Article,
		Title:       article.Title,
		Description: article.Content, // 存储文章内容
		URL:         "",              // 文章没有URL，使用本地内容
		UploaderID:  user.UserID,
		Points:      article.Points, // 积分字段
	}

	if err := c.ContentService.ResourceRepo.Create(resource); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, resource)
}

// 辅助函数：将map中的驼峰命名键转换为蛇形命名
func convertMapKeysToSnakeCase(input map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range input {
		// 对于viewCount特殊处理
		if key == "viewCount" {
			result["view_count"] = value
			continue
		}
		result[key] = value
	}

	return result
}

// 更新资源分类内容项的通用方法
func (c *CProgrammingResourceController) UpdateContentItem(ctx *gin.Context, contentType string, itemID uint, updateData interface{}) error {
	switch contentType {
	case "video":
		var videoData map[string]interface{}
		if v, ok := updateData.(map[string]interface{}); ok {
			videoData = convertMapKeysToSnakeCase(v)

			// 过滤不需要更新的字段，但保留duration和thumbnail等视频相关字段
			filteredData := make(map[string]interface{})
			for key, value := range videoData {
				if key != "created_at" && key != "updated_at" && key != "order" {
					filteredData[key] = value
				}
			}
			videoData = filteredData
		}

		return c.ContentService.ResourceRepo.DB.Model(&model.Resource{}).
			Where("id = ? AND type = ?", itemID, model.Video).
			Updates(videoData).Error

	case "article":
		var articleData map[string]interface{}
		if a, ok := updateData.(map[string]interface{}); ok {
			articleData = convertMapKeysToSnakeCase(a)
			// 过滤不需要更新的字段
			filteredData := make(map[string]interface{})
			for key, value := range articleData {
				if key != "created_at" && key != "updated_at" {
					filteredData[key] = value
				}
			}
			articleData = filteredData
		}

		return c.ContentService.ResourceRepo.DB.Model(&model.Resource{}).
			Where("id = ? AND type = ?", itemID, model.Article).
			Updates(articleData).Error

	case "exercise-category":
		return c.Service.CategoryRepo.DB.Model(&model.ExerciseCategory{}).
			Where("id = ?", itemID).
			Updates(updateData).Error

	case "question":
		return c.Service.QuestionRepo.DB.Model(&model.ExerciseQuestion{}).
			Where("id = ?", itemID).
			Updates(updateData).Error

	default:
		return fmt.Errorf("unsupported content type")
	}
	// switch contentType {
	// case "video", "article":
	// 	if contentType == "video" {
	// 		var videoData map[string]interface{}
	// 		if v, ok := updateData.(map[string]interface{}); ok {
	// 			filteredData := make(map[string]interface{})
	// 			for key, value := range v {
	// 				if key != "duration" && key != "order" && key != "thumbnail" && key != "createdAt" && key != "updatedAt" {
	// 					filteredData[key] = value
	// 				}
	// 			}
	// 			videoData = filteredData
	// 		}
	// 		return c.ContentService.ResourceRepo.DB.Model(&model.Resource{}).
	// 			Where("id = ? AND type = ?", itemID, model.Video).
	// 			Updates(videoData).Error
	// 	} else {
	// 		// var articleData map[string]interface{}
	// 		// if a, ok := updateData.(map[string]interface{}); ok {
	// 		// 	articleData = a
	// 		// }
	// 		// return c.ContentService.ResourceRepo.DB.Model(&model.Resource{}).
	// 		// 	Where("id = ? AND type = ?", itemID, model.Article).
	// 		// 	Updates(articleData).Error
	// 		var articleData map[string]interface{}
	// 		if a, ok := updateData.(map[string]interface{}); ok {
	// 			articleData = convertMapKeysToSnakeCase(a)
	// 		}

	// 		return c.ContentService.ResourceRepo.DB.Model(&model.Resource{}).
	// 			Where("id = ? AND type = ?", itemID, model.Article).
	// 			Updates(articleData).Error
	// 	}
	// case "exercise-category":
	// 	return c.Service.CategoryRepo.DB.Model(&model.ExerciseCategory{}).
	// 		Where("id = ?", itemID).
	// 		Updates(updateData).Error
	// case "question":
	// 	return c.Service.QuestionRepo.DB.Model(&model.ExerciseQuestion{}).
	// 		Where("id = ?", itemID).
	// 		Updates(updateData).Error
	// default:
	// 	return fmt.Errorf("unsupported content type")
	// }
}

// DeleteContentItem godoc
// @Summary 删除资源分类内容项
// @Description 删除视频、文章、练习题类目或题目
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param itemType path string true "内容类型(videos/articles/exercise-categories/questions)"
// @Param itemId path int true "内容项ID"
// @Success 200 {object} util.Response
// @Router /api/admin/{itemType}/{itemId} [delete]
func (c *CProgrammingResourceController) DeleteContentItem(ctx *gin.Context) {
	itemType := ctx.Param("itemType")
	itemIDStr := ctx.Param("itemId")
	itemID, err := strconv.ParseUint(itemIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid item ID")
		return
	}

	var errResult error
	switch itemType {
	case "videos":
		errResult = c.ContentService.ResourceRepo.DB.Where("id = ? AND type = ?", itemID, model.Video).Delete(&model.Resource{}).Error
	case "articles":
		errResult = c.ContentService.ResourceRepo.DB.Where("id = ? AND type = ?", itemID, model.Article).Delete(&model.Resource{}).Error
	case "exercise-categories":
		errResult = c.Service.CategoryRepo.Delete(uint(itemID))
	case "questions":
		errResult = c.Service.QuestionRepo.Delete(uint(itemID))
	default:
		util.BadRequest(ctx, "Unsupported content type")
		return
	}

	if errResult != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, nil)
}

// UpdateVideo godoc
// @Summary 更新视频内容（仅管理员）
// @Description 更新指定ID的视频内容信息
// @Tags C语言编程资源
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path int true "视频ID"
// @Param   body body map[string]interface{} true "要更新的视频数据（JSON格式）"
// @Success 200 {object} util.Response{data=nil} "更新成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/videos/{id} [put]
func (c *CProgrammingResourceController) UpdateVideo(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid video ID")
		return
	}

	var updateData map[string]interface{}
	if err := ctx.ShouldBindJSON(&updateData); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if points, ok := updateData["points"].(float64); ok {
		if points < 0 {
			util.BadRequest(ctx, "积分不能为负数")
			return
		}
	}

	if err := c.UpdateContentItem(ctx, "video", uint(id), updateData); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, nil)
}

// UpdateArticle godoc
// @Summary 更新文章内容（仅管理员）
// @Description 更新指定ID的文章内容信息
// @Tags C语言编程资源
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path int true "文章ID"
// @Param   body body map[string]interface{} true "要更新的文章数据（JSON格式）"
// @Success 200 {object} util.Response{data=nil} "更新成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/articles/{id} [put]
func (c *CProgrammingResourceController) UpdateArticle(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid article ID")
		return
	}

	var updateData map[string]interface{}
	if err := ctx.ShouldBindJSON(&updateData); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if points, ok := updateData["points"].(float64); ok {
		if points < 0 {
			util.BadRequest(ctx, "积分不能为负数")
			return
		}
	}

	if err := c.UpdateContentItem(ctx, "article", uint(id), updateData); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, nil)
}

// UpdateExerciseCategory godoc
// @Summary 更新练习分类（仅管理员）
// @Description 更新指定ID的练习分类信息
// @Tags C语言编程资源
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path int true "练习分类ID"
// @Param   body body map[string]interface{} true "要更新的练习分类数据（JSON格式）"
// @Success 200 {object} util.Response{data=nil} "更新成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/exercise-categories/{id} [put]
func (c *CProgrammingResourceController) UpdateExerciseCategory(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid exercise category ID")
		return
	}

	var updateData map[string]interface{}
	if err := ctx.ShouldBindJSON(&updateData); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.UpdateContentItem(ctx, "exercise-category", uint(id), updateData); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, nil)
}

// @Summary 更新练习题题目
// @Description 更新指定的练习题题目（需要管理员权限）
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "题目ID"
// @Param question body model.ExerciseQuestion true "练习题题目更新信息"
// @Success 200 {object} util.Response
// @Router /api/admin/questions/{id} [put]
func (c *CProgrammingResourceController) UpdateQuestion(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid question ID")
		return
	}

	var question model.ExerciseQuestion
	if err := ctx.ShouldBindJSON(&question); err != nil {
		util.BadRequest(ctx, "Invalid request body")
		return
	}

	if question.Points < 0 {
		util.BadRequest(ctx, "积分不能为负数")
		return
	}

	// 验证题目类型和必填字段
	switch question.QuestionType {
	case "single_choice", "multiple_choice":
		// 选择题必须包含选项和正确答案
		if len(question.Options) == 0 {
			util.BadRequest(ctx, "选择题必须提供选项")
			return
		}
		if question.CorrectAnswer == "" {
			util.BadRequest(ctx, "选择题必须提供正确答案")
			return
		}
	case "programming":
		// 编程题可以省略选项和正确答案
		if question.SolutionCode == "" {
			util.BadRequest(ctx, "编程题必须提供解决方案代码")
			return
		}
	}

	question.ID = uint(id)

	// 直接调用Service层的UpdateQuestion方法
	err = c.Service.UpdateQuestion(&question)
	if err != nil {
		// 如果Service层的UpdateQuestion方法不存在或有问题，可以使用UpdateContentItem方法
		// 转换question为map格式
		updateData := make(map[string]interface{})
		updateData["title"] = question.Title
		updateData["description"] = question.Description
		updateData["difficulty"] = question.Difficulty
		updateData["hint"] = question.Hint
		updateData["solution_code"] = question.SolutionCode
		updateData["question_type"] = question.QuestionType
		updateData["options"] = question.Options
		updateData["correct_answer"] = question.CorrectAnswer

		if err := c.UpdateContentItem(ctx, "question", uint(id), updateData); err != nil {
			util.InternalServerError(ctx)
			return
		}
	}

	util.Success(ctx, question)
}

// @Summary 管理员获取指定分类下的所有练习题题目
// @Description 管理员获取指定练习题分类下的所有题目（不需要分页）
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param categoryId path int true "分类ID"
// @Success 200 {object} util.Response
// @Router /api/admin/c-programming/categories/{categoryId}/questions/all [get]
func (c *CProgrammingResourceController) AdminGetAllQuestionsByCategoryID(ctx *gin.Context) {
	categoryIDStr := ctx.Param("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid category ID")
		return
	}

	questions, total, err := c.Service.GetAllQuestionsByCategoryID(uint(categoryID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"questions": questions,
		"total":     total,
	})
}

// GetResourcesWithAllContent godoc
// @Summary 获取所有C语言资源分类及其完整内容
// @Description 获取所有C语言资源分类模块，包括每个分类下的视频、文章和练习题类目
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param enabled query boolean false "是否只获取启用的资源分类"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(9)
// @Success 200 {object} util.Response
// @Router /api/c-programming/resources/full [get]
func (c *CProgrammingResourceController) GetResourcesWithAllContent(ctx *gin.Context) {
	enabledStr := ctx.Query("enabled")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "9"))

	var enabled *bool
	if enabledStr != "" {
		val := enabledStr == "true"
		enabled = &val
	}

	// 获取当前用户ID
	user := util.GetUserFromContext(ctx)
	var userID uint = 0
	if user != nil {
		userID = user.UserID
	}

	resourcesWithContent, total, err := c.Service.GetResourcesWithAllContent(enabled, page, limit, userID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"resources": resourcesWithContent,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

// @Summary 获取带用户状态的练习题题目列表
// @Description 获取指定练习题分类下的所有练习题题目，并包含当前用户是否已提交的状态
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Param categoryId path int true "分类ID"
// @Param user_id query int true "用户ID"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(5)
// @Success 200 {object} util.Response
// @Router /api/c-programming/categories/{categoryId}/questions-with-status [get]
func (c *CProgrammingResourceController) GetQuestionsByCategoryIDWithUserStatus(ctx *gin.Context) {
	categoryIDStr := ctx.Param("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid category ID")
		return
	}

	userIDStr := ctx.Query("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid user ID")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "5"))

	questions, total, err := c.Service.GetQuestionsByCategoryIDWithUserStatus(uint(categoryID), uint(userID), page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"questions": questions,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

// @Summary 提交练习题答案（公开接口）
// @Description 提交练习题答案，无需权限验证
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Param questionId path int true "题目ID"
// @Param answer body service.SubmitExerciseAnswerRequest true "答案内容"
// @Success 200 {object} util.Response
// @Router /api/public/c-programming/questions/{questionId}/submit [post]
func (c *CProgrammingResourceController) SubmitExerciseAnswerPublic(ctx *gin.Context) {
	questionIDStr := ctx.Param("questionId")
	questionID, err := strconv.ParseUint(questionIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid question ID")
		return
	}

	var req service.SubmitExerciseAnswerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if req.UserID == 0 {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader != "" {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			cfg := ctx.MustGet("config").(*config.Config)
			if claims, err := util.ParseJWT(tokenString, cfg.JWT.Secret); err == nil && claims != nil {
				req.UserID = claims.UserID
			}
		}
	}

	isCorrect, err := c.Service.SubmitExerciseAnswer(uint(questionID), req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"isCorrect": isCorrect,
		"message":   "Answer submitted successfully",
	})
}

// CheckUserSubmittedQuestion godoc
// @Summary 检查用户是否答过特定题目
// @Description 查询指定用户是否已经提交过特定题目的答案
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userID path uint true "用户ID"
// @Param questionID path uint true "题目ID"
// @Success 200 {object} util.Response
// @Router /api/c-programming/exercises/users/{userID}/questions/{questionID}/submission [get]
func (c *CProgrammingResourceController) CheckUserSubmittedQuestion(ctx *gin.Context) {
	// 从路径参数中获取用户ID和题目ID
	userIDStr := ctx.Param("userID")
	questionIDStr := ctx.Param("questionID")

	// 转换参数类型
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid user ID")
		return
	}

	questionID, err := strconv.ParseUint(questionIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid question ID")
		return
	}

	// 调用服务层方法检查用户是否提交过题目
	isSubmitted, err := c.Service.CheckUserSubmittedQuestion(uint(userID), uint(questionID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	// 返回结果
	util.Success(ctx, gin.H{
		"userID":      userID,
		"questionID":  questionID,
		"isSubmitted": isSubmitted,
	})
}

// @Summary 获取带进度的资源模块
// @Description 获取指定资源模块的详细信息，包括视频、文章、练习题的完成状态和进度
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param resourceId path int true "资源模块ID"
// @Success 200 {object} util.Response
// @Router /api/c-programming/resource-progress/{resourceId} [get]
func (c *CProgrammingResourceController) GetResourceModuleWithProgress(ctx *gin.Context) {
	// 获取当前用户
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	// 获取资源ID
	resourceIDStr := ctx.Param("resourceId")
	resourceID, err := strconv.ParseUint(resourceIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	// 获取带进度的资源模块
	resourceModule, err := c.Service.GetResourceModuleWithProgress(uint(resourceID), user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, resourceModule)
}

// @Summary 获取未完成的资源模块列表
// @Description 获取指定数量的未完成资源模块数据（有视频/文章未看完或练习题未做完）
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "要获取的资源模块数量，默认3个，最多3个"
// @Success 200 {object} util.Response
// @Router /api/c-programming/resource-progress/unfinished [get]
func (c *CProgrammingResourceController) GetUnfinishedResourceModules(ctx *gin.Context) {
	// 获取当前用户
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	// 获取查询参数limit，默认为3，最多3个
	limitStr := ctx.DefaultQuery("limit", "3")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 3
	} else if limit > 3 {
		limit = 3
	}

	// 调用服务层获取未完成的资源模块
	modules, err := c.Service.GetUnfinishedResourceModules(user.UserID, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, modules)
}

// @Summary 更新资源完成状态
// @Description 更新用户对特定资源的完成状态
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param resourceId path int true "资源ID"
// @Param completion body object true "完成状态对象" schema: {"completed": "boolean"}
// @Success 200 {object} util.Response
// @Router /api/c-programming/resource-progress/{resourceId}/completion [post]
func (c *CProgrammingResourceController) UpdateResourceCompletionStatus(ctx *gin.Context) {
	// 获取当前用户
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	// 获取资源ID
	resourceIDStr := ctx.Param("resourceId")
	resourceID, err := strconv.ParseUint(resourceIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid resource ID")
		return
	}

	// 解析请求体
	var req struct {
		Completed bool `json:"completed" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	// 更新资源完成状态
	err = c.Service.UpdateResourceCompletionStatus(user.UserID, uint(resourceID), req.Completed)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"message": "Resource completion status updated"})
}

// @Summary 获取所有带进度的资源模块
// @Description 获取所有资源模块的详细信息，包括视频、文章、练习题的完成状态和进度
// @Tags C语言编程资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param enabled query boolean false "是否只获取启用的资源分类"
// @Success 200 {object} util.Response
// @Router /api/c-programming/resource-progress/all [get]
func (c *CProgrammingResourceController) GetAllResourceModulesWithProgress(ctx *gin.Context) {
	// 获取当前用户
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	// 获取查询参数
	enabledStr := ctx.Query("enabled")
	var enabled *bool
	if enabledStr != "" {
		val := enabledStr == "true"
		enabled = &val
	}

	// 调用服务层方法
	resourceModules, err := c.Service.GetAllResourceModulesWithProgress(user.UserID, enabled)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, resourceModules)
}
