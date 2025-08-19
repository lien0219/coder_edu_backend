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
