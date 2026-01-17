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

// @Summary 学生端：获取自我评估学习路径
// @Tags 学习路径
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/learning-path/student [get]
func (c *LearningPathController) GetStudentPath(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	path, err := c.Service.GetStudentPath(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, path)
}

// @Summary 学生端：获取指定等级的所有资料
// @Tags 学习路径
// @Produce json
// @Security BearerAuth
// @Param level path int true "等级 (1:基础, 2:初级, 3:中级, 4:高级)"
// @Success 200 {object} util.Response
// @Router /api/learning-path/levels/{level}/materials [get]
func (c *LearningPathController) GetMaterialsByLevel(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	level, _ := strconv.Atoi(ctx.Param("level"))
	materials, err := c.Service.GetMaterialsByLevel(user.UserID, level)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	if materials == nil {
		util.Error(ctx, 403, "该等级资料尚未解锁")
		return
	}

	util.Success(ctx, materials)
}

// @Summary 学生端：记录学习路径资料的学习时长
// @Tags 学习路径
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "资料ID"
// @Param body body service.RecordLearningTimeRequest true "学习时长信息"
// @Success 200 {object} util.Response
// @Router /api/learning-path/materials/{id}/learning-time [post]
func (c *LearningPathController) RecordLearningTime(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	id := ctx.Param("id")
	var req service.RecordLearningTimeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.Service.RecordLearningTime(user.UserID, id, req.Duration); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, "学习时长记录成功")
}

// @Summary 学生端：标记资料为已完成
// @Tags 学习路径
// @Produce json
// @Security BearerAuth
// @Param id path string true "资料ID"
// @Success 200 {object} util.Response
// @Router /api/learning-path/materials/{id}/complete [post]
func (c *LearningPathController) CompleteMaterial(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	id := ctx.Param("id")
	if err := c.Service.CompleteMaterial(user.UserID, id); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, "标记完成成功")
}
