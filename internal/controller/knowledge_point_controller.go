package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

type KnowledgePointController struct {
	Service *service.KnowledgePointService
}

func NewKnowledgePointController(svc *service.KnowledgePointService) *KnowledgePointController {
	return &KnowledgePointController{Service: svc}
}

// @Summary 创建知识点 (老师/管理员)
// @Tags 知识点
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body service.CreateKnowledgePointRequest true "知识点信息"
// @Success 201 {object} util.Response
// @Router /api/teacher/knowledge-points [post]
func (c *KnowledgePointController) Create(ctx *gin.Context) {
	var req service.CreateKnowledgePointRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	kp, err := c.Service.CreateKnowledgePoint(req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, kp)
}

// @Summary 获取知识点列表 (老师/管理员)
// @Tags 知识点
// @Produce json
// @Security BearerAuth
// @Param title query string false "标题筛选"
// @Success 200 {object} util.Response
// @Router /api/teacher/knowledge-points [get]
func (c *KnowledgePointController) List(ctx *gin.Context) {
	title := ctx.Query("title")

	kps, err := c.Service.ListKnowledgePoints(title)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, kps)
}

// @Summary 获取知识点列表 (学生)
// @Tags 知识点
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/knowledge-points/student [get]
func (c *KnowledgePointController) ListForStudent(ctx *gin.Context) {
	claims := util.GetUserFromContext(ctx)
	if claims == nil {
		util.Unauthorized(ctx)
		return
	}

	kps, err := c.Service.ListKnowledgePointsForStudent(claims.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, kps)
}

// @Summary 获取知识点详情 (学生)
// @Tags 知识点
// @Produce json
// @Security BearerAuth
// @Param id path string true "知识点ID"
// @Success 200 {object} util.Response
// @Router /api/knowledge-points/student/{id} [get]
func (c *KnowledgePointController) GetDetailForStudent(ctx *gin.Context) {
	id := ctx.Param("id")
	claims := util.GetUserFromContext(ctx)
	if claims == nil {
		util.Unauthorized(ctx)
		return
	}

	resp, err := c.Service.GetKnowledgePointForStudent(id, claims.UserID)
	if err != nil {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, resp)
}

// @Summary 学生端：提交知识点测试结果 (包含题目、代码、执行结果)
// @Tags 知识点
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "知识点ID"
// @Param body body service.SubmitKnowledgePointExercisesRequest true "提交测试内容"
// @Success 200 {object} util.Response
// @Router /api/knowledge-points/student/submit [post]
func (c *KnowledgePointController) SubmitExercises(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.SubmitKnowledgePointExercisesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.Service.SubmitExercises(user.UserID, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, result)
}

// @Summary 学生端：记录知识点学习时长
// @Tags 知识点
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "知识点ID"
// @Param body body service.RecordLearningTimeRequest true "学习时长信息"
// @Success 200 {object} util.Response
// @Router /api/knowledge-points/student/{id}/learning-time [post]
func (c *KnowledgePointController) RecordLearningTime(ctx *gin.Context) {
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

// @Summary 更新知识点 (老师/管理员)
// @Tags 知识点
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "知识点ID"
// @Param body body service.CreateKnowledgePointRequest true "知识点信息"
// @Success 200 {object} util.Response
// @Router /api/teacher/knowledge-points/{id} [put]
func (c *KnowledgePointController) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	var req service.CreateKnowledgePointRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	kp, err := c.Service.UpdateKnowledgePoint(id, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, kp)
}

// @Summary 删除知识点 (老师/管理员)
// @Tags 知识点
// @Produce json
// @Security BearerAuth
// @Param id path string true "知识点ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/knowledge-points/{id} [delete]
func (c *KnowledgePointController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.Service.DeleteKnowledgePoint(id); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"deleted": id})
}
