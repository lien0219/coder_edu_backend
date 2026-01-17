package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type LearningPathController struct {
	Service *service.LearningPathService
}

func NewLearningPathController(svc *service.LearningPathService) *LearningPathController {
	return &LearningPathController{Service: svc}
}

// @Summary 创建学习路径资料
// @Tags 学习路径
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body service.CreateMaterialRequest true "资料信息"
// @Success 201 {object} util.Response
// @Router /api/teacher/learning-path/materials [post]
func (c *LearningPathController) CreateMaterial(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.CreateMaterialRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	material, err := c.Service.CreateMaterial(user.UserID, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, material)
}

// @Summary 获取学习路径资料列表
// @Tags 学习路径
// @Produce json
// @Security BearerAuth
// @Param level query int false "等级 (1:基础, 2:初级, 3:中级, 4:高级)"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} util.Response
// @Router /api/teacher/learning-path/materials [get]
func (c *LearningPathController) ListMaterials(ctx *gin.Context) {
	levelStr := ctx.Query("level")
	level := 0
	if levelStr != "" {
		level, _ = strconv.Atoi(levelStr)
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))

	ms, total, err := c.Service.ListMaterials(level, page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"items": ms, "total": total})
}

// @Summary 获取学习路径资料详情
// @Tags 学习路径
// @Produce json
// @Security BearerAuth
// @Param id path string true "资料ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/learning-path/materials/{id} [get]
func (c *LearningPathController) GetMaterial(ctx *gin.Context) {
	id := ctx.Param("id")

	m, err := c.Service.GetMaterial(id)
	if err != nil {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, m)
}

// @Summary 更新学习路径资料
// @Tags 学习路径
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "资料ID"
// @Param body body service.CreateMaterialRequest true "资料信息"
// @Success 200 {object} util.Response
// @Router /api/teacher/learning-path/materials/{id} [put]
func (c *LearningPathController) UpdateMaterial(ctx *gin.Context) {
	id := ctx.Param("id")

	var req service.CreateMaterialRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	m, err := c.Service.UpdateMaterial(id, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, m)
}

// @Summary 删除学习路径资料
// @Tags 学习路径
// @Produce json
// @Security BearerAuth
// @Param id path string true "资料ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/learning-path/materials/{id} [delete]
func (c *LearningPathController) DeleteMaterial(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.Service.DeleteMaterial(id); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"deleted": id})
}
