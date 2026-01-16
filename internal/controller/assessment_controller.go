package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AssessmentController struct {
	Service *service.AssessmentService
}

func NewAssessmentController(svc *service.AssessmentService) *AssessmentController {
	return &AssessmentController{Service: svc}
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
// @Success 200 {object} util.Response
// @Router /api/teacher/assessments/questions [get]
func (c *AssessmentController) ListQuestions(ctx *gin.Context) {
	assessmentID := uint(0)
	if idStr := ctx.Query("assessmentId"); idStr != "" {
		if id, err := strconv.Atoi(idStr); err == nil {
			assessmentID = uint(id)
		}
	}

	qs, err := c.Service.ListQuestions(assessmentID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, qs)
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
		util.NotFound(ctx)
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

	// 检查学生是否有权进行测试
	canTake, err := c.Service.GetUserAssessmentStatus(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	if !canTake {
		util.Error(ctx, 403, "您已完成测试，暂不可重测")
		return
	}

	qs, err := c.Service.ListStudentQuestions()
	if err != nil {
		util.InternalServerError(ctx)
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

	// 检查学生是否有权进行测试
	canTake, err := c.Service.GetUserAssessmentStatus(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	if !canTake {
		util.Error(ctx, 403, "您已完成测试，暂不可重测")
		return
	}

	var req service.AssessmentSubmissionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	_, err = c.Service.SubmitAssessment(user.UserID, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, "提交成功")
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
