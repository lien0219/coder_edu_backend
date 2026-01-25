package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PostClassTestController struct {
	Service *service.PostClassTestService
}

func NewPostClassTestController(svc *service.PostClassTestService) *PostClassTestController {
	return &PostClassTestController{Service: svc}
}

// @Summary 创建课后测试试卷
// @Tags 课后测试模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body service.PostClassTestReq true "试卷信息"
// @Success 201 {object} util.Response
// @Router /api/teacher/post-class-tests [post]
func (c *PostClassTestController) CreateTest(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.PostClassTestReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	test, err := c.Service.CreateTest(user.UserID, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, test)
}

// @Summary 获取课后测试试卷列表
// @Tags 课后测试模块
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} util.Response
// @Router /api/teacher/post-class-tests [get]
func (c *PostClassTestController) ListTests(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))

	tests, total, err := c.Service.ListTests(page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"items": tests, "total": total})
}

// @Summary 学生获取已发布的课后测试
// @Tags 课后测试模块
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/student/post-class-tests/published [get]
func (c *PostClassTestController) GetPublishedTest(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	test, err := c.Service.GetPublishedTestForStudent(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, test)
}

// @Summary 学生获取已发布的课后测试详情（包含题目）
// @Tags 课后测试模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "试卷ID"
// @Success 200 {object} util.Response
// @Router /api/student/post-class-tests/{id} [get]
func (c *PostClassTestController) GetStudentTestDetail(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	id := ctx.Param("id")

	detail, err := c.Service.GetStudentTestDetail(user.UserID, id)
	if err != nil {
		if err.Error() == "test not published or not accessible" {
			util.Error(ctx, 403, err.Error())
		} else {
			util.NotFound(ctx)
		}
		return
	}

	util.Success(ctx, detail)
}

// @Summary 学生开始课后测试答题
// @Tags 课后测试模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "试卷ID"
// @Success 200 {object} util.Response
// @Router /api/student/post-class-tests/{id}/start [post]
func (c *PostClassTestController) StartTest(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	id := ctx.Param("id")
	submission, err := c.Service.StartTest(user.UserID, id)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, submission)
}

// @Summary 学生记录课后测试学习时长
// @Tags 课后测试模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "试卷ID"
// @Param body body service.RecordLearningTimeRequest true "学习时长信息"
// @Success 200 {object} util.Response
// @Router /api/student/post-class-tests/{id}/learning-time [post]
func (c *PostClassTestController) RecordLearningTime(ctx *gin.Context) {
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

	util.Success(ctx, "记录成功")
}

// @Summary 学生提交课后测试答案
// @Tags 课后测试模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "试卷ID"
// @Param body body service.PostClassTestSubmissionReq true "提交信息"
// @Success 200 {object} util.Response
// @Router /api/student/post-class-tests/{id}/submit [post]
func (c *PostClassTestController) SubmitTest(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	id := ctx.Param("id")
	var req service.PostClassTestSubmissionReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	submission, err := c.Service.SubmitTest(user.UserID, id, req)
	if err != nil {
		if err.Error() == "test already submitted" {
			util.Error(ctx, 403, err.Error())
		} else {
			util.InternalServerError(ctx)
		}
		return
	}

	util.Success(ctx, submission)
}

// @Summary 获取课后测试试卷详情
// @Tags 课后测试模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "试卷ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/post-class-tests/{id} [get]
func (c *PostClassTestController) GetTest(ctx *gin.Context) {
	id := ctx.Param("id")

	test, qs, err := c.Service.GetTest(id)
	if err != nil {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, gin.H{"test": test, "questions": qs})
}

// @Summary 更新课后测试试卷
// @Tags 课后测试模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "试卷ID"
// @Param body body service.PostClassTestReq true "试卷信息"
// @Success 200 {object} util.Response
// @Router /api/teacher/post-class-tests/{id} [put]
func (c *PostClassTestController) UpdateTest(ctx *gin.Context) {
	id := ctx.Param("id")

	var req service.PostClassTestReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	test, err := c.Service.UpdateTest(id, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, test)
}

// @Summary 删除课后测试试卷
// @Tags 课后测试模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "试卷ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/post-class-tests/{id} [delete]
func (c *PostClassTestController) DeleteTest(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.Service.DeleteTest(id); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"deleted": id})
}

// @Summary 获取试卷答题情况列表
// @Tags 课后测试模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "试卷ID"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param name query string false "学生姓名"
// @Success 200 {object} util.Response
// @Router /api/teacher/post-class-tests/{id}/submissions [get]
func (c *PostClassTestController) ListSubmissions(ctx *gin.Context) {
	id := ctx.Query("testId")
	if id == "" {
		id = ctx.Param("id")
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	name := ctx.Query("name")
	status := ctx.Query("status")

	ss, total, err := c.Service.ListSubmissions(id, page, limit, name, status)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"items": ss, "total": total})
}

// @Summary 获取学生答题详情
// @Tags 课后测试模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "提交ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/post-class-tests/submissions/{id} [get]
func (c *PostClassTestController) GetSubmissionDetail(ctx *gin.Context) {
	id := ctx.Param("id")

	detail, err := c.Service.GetSubmissionDetail(id)
	if err != nil {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, detail)
}

// @Summary 重置学生测试（支持单人或批量）
// @Tags 课后测试模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string][]string true "提交记录ID列表 {'ids': ['uuid1', 'uuid2']}"
// @Success 200 {object} util.Response
// @Router /api/teacher/post-class-tests/submissions/reset [post]
func (c *PostClassTestController) ResetStudentTests(ctx *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.Service.BatchResetStudentTests(req.IDs); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, "已重置选中的学生测试状态")
}
