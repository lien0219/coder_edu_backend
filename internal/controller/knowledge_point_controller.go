package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

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

// @Summary 学生端：开始答题 (启动计时)
// @Tags 知识点
// @Produce json
// @Security BearerAuth
// @Param id path string true "知识点ID"
// @Success 200 {object} util.Response
// @Router /api/knowledge-points/student/{id}/start [post]
func (c *KnowledgePointController) StartExercises(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	id := ctx.Param("id")
	startTime, err := c.Service.StartExercises(user.UserID, id)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"startTime": startTime,
	})
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

// @Summary 获取所有学生提交的知识点测试 (老师/管理员)
// @Tags 知识点
// @Produce json
// @Security BearerAuth
// @Param knowledgePointId query string false "知识点ID"
// @Param status query string false "审核状态 (pending, approved, rejected, unsubmitted)"
// @Param studentName query string false "学生姓名搜索"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Success 200 {object} util.Response
// @Router /api/teacher/knowledge-points/submissions [get]
func (c *KnowledgePointController) ListSubmissions(ctx *gin.Context) {
	kpID := ctx.Query("knowledgePointId")
	status := ctx.Query("status")
	studentName := ctx.Query("studentName")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))

	submissions, total, err := c.Service.ListSubmissions(kpID, status, studentName, page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"items": submissions,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// @Summary 获取学生提交的知识点测试详情 (老师/管理员)
// @Tags 知识点
// @Produce json
// @Security BearerAuth
// @Param id path string true "提交ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/knowledge-points/submissions/{id} [get]
func (c *KnowledgePointController) GetSubmissionDetail(ctx *gin.Context) {
	id := ctx.Param("id")
	submission, err := c.Service.GetSubmissionDetail(id)
	if err != nil {
		util.NotFound(ctx)
		return
	}
	util.Success(ctx, submission)
}

// @Summary 审核学生提交的知识点测试 (老师/管理员)
// @Tags 知识点
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "提交ID"
// @Param body body map[string]interface{} true "状态 (status: approved 或 rejected, 可选 score: int 手动评分)"
// @Success 200 {object} util.Response
// @Router /api/teacher/knowledge-points/submissions/{id}/audit [post]
func (c *KnowledgePointController) AuditSubmission(ctx *gin.Context) {
	id := ctx.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
		Score  *int   `json:"score"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.Service.AuditSubmission(id, req.Status, req.Score); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, "审核操作成功")
}
