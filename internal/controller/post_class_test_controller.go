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
	id := ctx.Param("id")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	name := ctx.Query("name")

	ss, total, err := c.Service.ListSubmissions(id, page, limit, name)
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

// @Summary 重设学生测试（允许重测）
// @Tags 课后测试模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "提交ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/post-class-tests/submissions/{id}/reset [post]
func (c *PostClassTestController) ResetStudentTest(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.Service.ResetStudentTest(id); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, "已重置学生测试状态")
}

// @Summary 批量重设学生测试
// @Tags 课后测试模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string][]string true "提交ID列表 {'submissionIds': ['uuid1','uuid2']}"
// @Success 200 {object} util.Response
// @Router /api/teacher/post-class-tests/submissions/batch-reset [post]
func (c *PostClassTestController) BatchResetStudentTests(ctx *gin.Context) {
	var req struct {
		SubmissionIDs []string `json:"submissionIds" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.Service.BatchResetStudentTests(req.SubmissionIDs); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, "已批量重置学生测试状态")
}
