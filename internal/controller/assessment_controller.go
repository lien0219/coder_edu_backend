package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AssessmentController struct {
	Service *service.AssessmentService
}

func NewAssessmentController(svc *service.AssessmentService) *AssessmentController {
	return &AssessmentController{Service: svc}
}

func requireTeacherOrAdmin(ctx *gin.Context) bool {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return false
	}
	if user.Role != model.Teacher && user.Role != model.Admin {
		util.Forbidden(ctx)
		return false
	}
	return true
}

func TeacherOrAdminMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !requireTeacherOrAdmin(ctx) {
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

func (c *AssessmentController) TeacherOrAdmin() gin.HandlerFunc {
	return TeacherOrAdminMiddleware()
}

func respondAssessmentError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, util.ErrNoPublishedAssessment):
		util.Error(ctx, http.StatusNotFound, util.ErrNoPublishedAssessment.Error())
	case errors.Is(err, util.ErrAssessmentPaperStale):
		util.ErrorWithCode(ctx, http.StatusConflict, util.ErrCodeAssessmentPaperStale, util.ErrAssessmentPaperStale.Error())
	case errors.Is(err, util.ErrAssessmentIdempotencyConflict):
		util.ErrorWithCode(ctx, http.StatusConflict, util.ErrCodeAssessmentIdempotencyConflict, util.ErrAssessmentIdempotencyConflict.Error())
	case errors.Is(err, util.ErrAssessmentPaperUnavailable):
		util.ErrorWithCode(ctx, http.StatusConflict, util.ErrCodeAssessmentPaperUnavailable, util.ErrAssessmentPaperUnavailable.Error())
	case errors.Is(err, util.ErrAssessmentSubmitInvalid), errors.Is(err, util.ErrAssessmentSubmitMissingMeta):
		util.BadRequest(ctx, err.Error())
	case errors.Is(err, util.ErrAssessmentRetestDenied):
		util.Error(ctx, http.StatusForbidden, util.ErrAssessmentRetestDenied.Error())
	default:
		util.LogInternalError(ctx, err)
	}
}

// @Summary 创建测试题
// @Tags 学前测试评估
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body service.AssessmentQuestionRequest true "题目信息"
// @Success 201 {object} util.Response
// @Router /api/teacher/assessments/questions [post]
func (c *AssessmentController) CreateQuestion(ctx *gin.Context) {
	var req service.AssessmentQuestionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	q, err := c.Service.CreateQuestion(req)
	if err != nil {
		if errors.Is(err, util.ErrInvalidKnowledgePoint) {
			util.BadRequest(ctx, err.Error())
			return
		}
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, q)
}

// @Summary 获取测试题列表
// @Tags 学前测试评估
// @Produce json
// @Security BearerAuth
// @Param assessmentId query int false "评估ID"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/questions [get]
func (c *AssessmentController) ListQuestions(ctx *gin.Context) {
	assessmentID := uint(0)
	if idStr := ctx.Query("assessmentId"); idStr != "" {
		if id, err := strconv.Atoi(idStr); err == nil {
			assessmentID = uint(id)
		}
	}

	page := 1
	if p := ctx.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	limit := 10
	if l := ctx.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	qs, total, err := c.Service.ListQuestions(assessmentID, page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"items": qs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// @Summary 获取测试题详情
// @Tags 学前测试评估
// @Produce json
// @Security BearerAuth
// @Param id path int true "题目ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/questions/{id} [get]
func (c *AssessmentController) GetQuestion(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}

	q, err := c.Service.GetQuestion(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			util.NotFound(ctx)
			return
		}
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, q)
}

// @Summary 学生端：获取学前测试题目列表
// @Tags 学前测试评估
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/assessments/questions [get]
func (c *AssessmentController) GetStudentQuestions(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	qs, err := c.Service.GetStudentQuestionsIfEligible(user.UserID)
	if err != nil {
		respondAssessmentError(ctx, err)
		return
	}

	util.Success(ctx, qs)
}

// @Summary 学生端：提交学前测试答案
// @Tags 学前测试评估
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body service.AssessmentSubmissionRequest true "答案信息"
// @Success 200 {object} util.Response
// @Router /api/assessments/submit [post]
func (c *AssessmentController) SubmitAssessment(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.AssessmentSubmissionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	submission, err := c.Service.SubmitAssessment(user.UserID, req)
	if err != nil {
		respondAssessmentError(ctx, err)
		return
	}

	util.Success(ctx, submission)
}

// @Summary 学生端：获取自己的评估状态和结果
// @Tags 学前测试评估
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/assessments/result [get]
func (c *AssessmentController) GetMyResult(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	res, err := c.Service.GetStudentAssessmentStatus(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, res)
}

// @Summary 更新测试题
// @Tags 学前测试评估
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "题目ID"
// @Param body body service.AssessmentQuestionRequest true "题目信息"
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/questions/{id} [put]
func (c *AssessmentController) UpdateQuestion(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}

	var req service.AssessmentQuestionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	q, err := c.Service.UpdateQuestion(uint(id), req)
	if err != nil {
		if errors.Is(err, util.ErrInvalidKnowledgePoint) {
			util.BadRequest(ctx, err.Error())
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			util.NotFound(ctx)
			return
		}
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, q)
}

// @Summary 删除测试题
// @Tags 学前测试评估
// @Produce json
// @Security BearerAuth
// @Param id path int true "题目ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/questions/{id} [delete]
func (c *AssessmentController) DeleteQuestion(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}

	if err := c.Service.DeleteQuestion(uint(id)); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"deleted": id})
}

// @Summary 创建评估
// @Tags 学前测试评估
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body service.AssessmentRequest true "评估信息"
// @Success 201 {object} util.Response
// @Router /api/teacher/assessments [post]
func (c *AssessmentController) CreateAssessment(ctx *gin.Context) {
	var req service.AssessmentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}
	a, err := c.Service.CreateAssessment(req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Created(ctx, a)
}

// @Summary 获取评估列表
// @Tags 学前测试评估
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments [get]
func (c *AssessmentController) ListAssessments(ctx *gin.Context) {
	page := 1
	limit := 20
	if p := ctx.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := ctx.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	as, total, err := c.Service.ListAssessments(page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"items": as, "total": total})
}

// @Summary 获取评估详情
// @Tags 学前测试评估
// @Produce json
// @Security BearerAuth
// @Param id path int true "评估ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/{id} [get]
func (c *AssessmentController) GetAssessment(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}

	a, err := c.Service.GetAssessment(uint(id))
	if err != nil {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, a)
}

// @Summary 发布或取消发布评估
// @Tags 学前测试评估
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "评估ID"
// @Param body body service.PublishAssessmentRequest true "发布状态"
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/{id}/publish [post]
func (c *AssessmentController) PublishAssessment(ctx *gin.Context) {
	if !requireTeacherOrAdmin(ctx) {
		return
	}
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}
	var req service.PublishAssessmentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}
	a, err := c.Service.SetAssessmentPublished(uint(id), req.IsPublished)
	if err != nil {
		util.NotFound(ctx)
		return
	}
	util.Success(ctx, a)
}

// @Summary 教师端：获取提交列表
// @Tags 学前测试评估
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param status query string false "状态"
// @Param name query string false "学生姓名"
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/submissions [get]
func (c *AssessmentController) ListSubmissions(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	status := ctx.Query("status")
	name := ctx.Query("name")

	ss, total, err := c.Service.ListSubmissions(page, limit, status, name)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"items": ss, "total": total})
}

// @Summary 教师端：获取提交详情（用于审核）
// @Tags 学前测试评估
// @Produce json
// @Security BearerAuth
// @Param id path int true "提交ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/submissions/{id} [get]
func (c *AssessmentController) GetSubmissionDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}

	detail, err := c.Service.GetSubmissionDetail(uint(id))
	if err != nil {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, detail)
}

// @Summary 教师端：评分提交
// @Tags 学前测试评估
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "提交ID"
// @Param body body service.GradeSubmissionRequest true "评分信息 (recommendedLevel: 1-基础, 2-初级, 3-中级, 4-高级)"
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/submissions/{id}/grade [post]
func (c *AssessmentController) GradeSubmission(ctx *gin.Context) {
	if !requireTeacherOrAdmin(ctx) {
		return
	}
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}

	var req service.GradeSubmissionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.Service.GradeSubmission(uint(id), req); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, "评分成功")
}

// @Summary 教师端：删除提交记录
// @Tags 学前测试评估
// @Produce json
// @Security BearerAuth
// @Param id path int true "提交ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/submissions/{id} [delete]
func (c *AssessmentController) DeleteSubmission(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}

	if err := c.Service.DeleteSubmission(uint(id)); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"deleted": id})
}

type SetRetestRequest struct {
	UserIDs []uint `json:"userIds" binding:"required"`
	CanTake bool   `json:"canTake"`
}

// @Summary 教师端：设置学生是否可以重测
// @Tags 学前测试评估
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body SetRetestRequest true "重测设置信息"
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/retest [post]
func (c *AssessmentController) SetUserRetest(ctx *gin.Context) {
	var req SetRetestRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.Service.SetUserCanRetest(req.UserIDs, req.CanTake); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, "设置成功")
}
