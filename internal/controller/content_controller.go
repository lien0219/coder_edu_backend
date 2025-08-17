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
// @Summary Upload a learning resource (Admin only)
// @Description Upload a PDF, video, article, or worksheet for a specific module
// @Tags content
// @Accept  multipart/form-data
// @Produce  json
// @Security ApiKeyAuth
// @Param   title formData string true "Resource title"
// @Param   description formData string false "Resource description"
// @Param   type formData string true "Resource type (pdf, video, article, worksheet)" Enums(pdf, video, article, worksheet)
// @Param   moduleType formData string true "Module type (pre-class, in-class, post-class)" Enums(pre-class, in-class, post-class)
// @Param   file formData file true "Resource file"
// @Success 201 {object} util.Response{data=object} "Created"
// @Failure 400 {object} util.Response "Bad Request"
// @Failure 401 {object} util.Response "Unauthorized"
// @Failure 403 {object} util.Response "Forbidden"
// @Failure 500 {object} util.Response "Internal Server Error"
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
// @Summary Get resources by module type
// @Description Get a list of resources for a specific module type
// @Tags content
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   moduleType query string true "Module type (pre-class, in-class, post-class)" Enums(pre-class, in-class, post-class)
// @Success 200 {object} util.Response{data=[]model.Resource} "Success"
// @Failure 400 {object} util.Response "Bad Request"
// @Failure 500 {object} util.Response "Internal Server Error"
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
