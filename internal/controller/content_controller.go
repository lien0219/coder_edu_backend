package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

type ContentController struct {
	ContentService *service.ContentService
}

func NewContentController(contentService *service.ContentService) *ContentController {
	return &ContentController{ContentService: contentService}
}

// UploadResourceRequest defines model for resource upload
// swagger:model UploadResourceRequest
type UploadResourceRequest struct {
	Title       string `form:"title" binding:"required"`
	Description string `form:"description"`
	Type        string `form:"type" binding:"required,oneof=pdf video article worksheet"`
	ModuleType  string `form:"moduleType" binding:"required,oneof=pre-class in-class post-class"`
}

// UploadResource godoc
// @Summary 上传学习资源（仅管理员）
// @Description 上传PDF、视频、文章或工作表到特定模块
// @Tags 内容
// @Accept  multipart/form-data
// @Produce  json
// @Security ApiKeyAuth
// @Param   title formData string true "资源标题"
// @Param   description formData string false "资源描述"
// @Param   type formData string true "资源类型（pdf, video, article, worksheet)" Enums(pdf, video, article, worksheet)
// @Param   moduleType formData string true "模块类型（pre-class, in-class, post-class)" Enums(pre-class, in-class, post-class)
// @Param   file formData file true "资源文件"
// @Success 201 {object} util.Response{data=object} "创建成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/resources [post]
func (c *ContentController) UploadResource(ctx *gin.Context) {
	var req UploadResourceRequest
	if err := ctx.ShouldBind(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		util.BadRequest(ctx, "File is required")
		return
	}

	resource := &model.Resource{
		Title:       req.Title,
		Description: req.Description,
		Type:        model.ResourceType(req.Type),
		ModuleType:  req.ModuleType,
	}

	if err := c.ContentService.UploadResource(ctx, file, resource); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, gin.H{"id": resource.ID, "url": resource.URL})
}

// GetResources godoc
// @Summary 按模块类型获取资源
// @Description 获取特定模块类型的资源列表
// @Tags 内容
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   moduleType query string true "模块类型（pre-class, in-class, post-class)" Enums(pre-class, in-class, post-class)
// @Success 200 {object} util.Response{data=[]model.Resource} "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/resources [get]
func (c *ContentController) GetResources(ctx *gin.Context) {
	moduleType := ctx.Query("moduleType")
	if moduleType == "" {
		util.BadRequest(ctx, "moduleType parameter is required")
		return
	}

	resources, err := c.ContentService.ResourceRepo.FindByModule(moduleType)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, resources)
}

// UploadIcon godoc
// @Summary 上传模块图标（仅管理员）
// @Description 专门用于上传C语言编程模块的图标
// @Tags 内容
// @Accept  multipart/form-data
// @Produce  json
// @Security ApiKeyAuth
// @Param   icon formData file true "图标文件（PNG或SVG格式）"
// @Success 200 {object} util.Response{data=map[string]string} "上传成功"
// @Failure 400 {object} util.Response "请求参数错误或文件格式不支持"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/upload/icon [post]
func (c *ContentController) UploadIcon(ctx *gin.Context) {
	file, err := ctx.FormFile("icon")
	if err != nil {
		util.BadRequest(ctx, "图标文件是必需的")
		return
	}

	url, err := c.ContentService.UploadIcon(ctx, file)
	if err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	util.Success(ctx, gin.H{
		"success": true,
		"message": "图标上传成功",
		"url":     url,
	})
}

// 视频上传相关内容
// VideoUploadRequest 视频上传请求参数
type VideoUploadRequest struct {
	Title       string `form:"title"`
	Description string `form:"description"`
}

// VideoChunkUploadRequest 视频分块上传请求参数
type VideoChunkUploadRequest struct {
	ChunkNumber int    `form:"chunkNumber" binding:"required,min=1"`
	TotalChunks int    `form:"totalChunks" binding:"required,min=1"`
	Identifier  string `form:"identifier" binding:"required,max=100"`
	Filename    string `form:"filename" binding:"required,max=255"`
	Title       string `form:"title"`
	Description string `form:"description"`
}

// UploadVideo godoc
// @Summary 上传视频文件
// @Description 专门用于上传视频文件
// @Tags 内容
// @Accept  multipart/form-data
// @Produce  json
// @Security ApiKeyAuth
// @Param   video formData file true "视频文件"
// @Param   title formData string false "视频标题"
// @Param   description formData string false "视频描述"
// @Success 200 {object} util.Response{data=object} "上传成功"
// @Failure 400 {object} util.Response "请求参数错误或文件格式不支持"
// @Failure 401 {object} util.Response "未授权"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/upload/video [post]
func (c *ContentController) UploadVideo(ctx *gin.Context) {
	var req VideoUploadRequest
	if err := ctx.ShouldBind(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	file, err := ctx.FormFile("video")
	if err != nil {
		util.BadRequest(ctx, "视频文件是必需的")
		return
	}

	resource, err := c.ContentService.UploadVideo(ctx, file, req.Title, req.Description)
	if err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	util.Success(ctx, gin.H{
		"id":          resource.ID,
		"url":         resource.URL,
		"title":       resource.Title,
		"description": resource.Description,
		"duration":    resource.Duration,
		"size":        resource.Size,
		"format":      resource.Format,
		"thumbnail":   resource.Thumbnail,
	})
}

// UploadVideoChunk godoc
// @Summary 上传视频文件分块
// @Description 支持大视频文件的分块上传
// @Tags 内容
// @Accept  multipart/form-data
// @Produce  json
// @Security ApiKeyAuth
// @Param   chunk formData file true "文件分块数据"
// @Param   chunkNumber formData int true "当前块序号"
// @Param   totalChunks formData int true "总块数"
// @Param   identifier formData string true "文件唯一标识符"
// @Param   filename formData string true "原始文件名"
// @Param   title formData string false "视频标题"
// @Param   description formData string false "视频描述"
// @Success 200 {object} util.Response{data=object} "上传成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/upload/video/chunk [post]
func (c *ContentController) UploadVideoChunk(ctx *gin.Context) {
	var req VideoChunkUploadRequest
	if err := ctx.ShouldBind(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	chunkFile, err := ctx.FormFile("chunk")
	if err != nil {
		util.BadRequest(ctx, "分块文件是必需的")
		return
	}

	progress, resource, err := c.ContentService.UploadVideoChunk(ctx, chunkFile, req.ChunkNumber, req.TotalChunks, req.Identifier, req.Filename, req.Title, req.Description)
	if err != nil {
		util.LogInternalError(ctx, err)
		return
	}

	isComplete := progress.UploadedChunks == progress.TotalChunks
	responseData := gin.H{
		"identifier":     req.Identifier,
		"chunkNumber":    req.ChunkNumber,
		"totalChunks":    req.TotalChunks,
		"uploadedChunks": progress.UploadedChunks,
		"isComplete":     isComplete,
		"progress":       float64(progress.UploadedChunks) / float64(progress.TotalChunks) * 100,
	}

	if isComplete && resource != nil {
		responseData["id"] = resource.ID
		responseData["finalURL"] = resource.URL
		responseData["title"] = resource.Title
		responseData["description"] = resource.Description
		responseData["duration"] = resource.Duration
		responseData["size"] = resource.Size
		responseData["format"] = resource.Format
		responseData["thumbnail"] = resource.Thumbnail
	}

	util.Success(ctx, responseData)
}

// GetUploadProgress godoc
// @Summary 查询视频上传进度
// @Description 查询文件上传进度
// @Tags 内容
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   uploadId path string true "上传标识符"
// @Success 200 {object} util.Response{data=object} "查询成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 404 {object} util.Response "上传记录不存在"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/upload/video/progress/{uploadId} [get]
func (c *ContentController) GetUploadProgress(ctx *gin.Context) {
	uploadId := ctx.Param("uploadId")
	if uploadId == "" {
		util.BadRequest(ctx, "上传标识符不能为空")
		return
	}

	progress, err := c.ContentService.GetUploadProgress(uploadId)
	if err != nil {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"identifier":     progress.Identifier,
		"filename":       progress.Filename,
		"totalChunks":    progress.TotalChunks,
		"uploadedChunks": progress.UploadedChunks,
		"isComplete":     progress.UploadedChunks == progress.TotalChunks,
		"progress":       float64(progress.UploadedChunks) / float64(progress.TotalChunks) * 100,
		"createdAt":      progress.CreatedAt,
	})
}
