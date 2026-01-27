package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MigrationTaskController struct {
	Service *service.MigrationTaskService
}

func NewMigrationTaskController(svc *service.MigrationTaskService) *MigrationTaskController {
	return &MigrationTaskController{Service: svc}
}

// @Summary 创建迁移任务
// @Tags 迁移任务模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body service.MigrationTaskReq true "任务信息"
// @Success 201 {object} util.Response
// @Router /api/teacher/migration-tasks [post]
func (c *MigrationTaskController) CreateTask(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.MigrationTaskReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	task, err := c.Service.CreateTask(user.UserID, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, task)
}

// @Summary 获取迁移任务列表
// @Tags 迁移任务模块
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} util.Response
// @Router /api/teacher/migration-tasks [get]
func (c *MigrationTaskController) ListTasks(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))

	tasks, total, err := c.Service.ListTasks(page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"items": tasks, "total": total})
}

// @Summary 获取迁移任务详情
// @Tags 迁移任务模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/migration-tasks/{id} [get]
func (c *MigrationTaskController) GetTask(ctx *gin.Context) {
	id := ctx.Param("id")

	task, qs, err := c.Service.GetTask(id)
	if err != nil {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, gin.H{"task": task, "questions": qs})
}

// @Summary 更新迁移任务
// @Tags 迁移任务模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Param body body service.MigrationTaskReq true "任务信息"
// @Success 200 {object} util.Response
// @Router /api/teacher/migration-tasks/{id} [put]
func (c *MigrationTaskController) UpdateTask(ctx *gin.Context) {
	id := ctx.Param("id")

	var req service.MigrationTaskReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	task, err := c.Service.UpdateTask(id, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, task)
}

// @Summary 删除迁移任务
// @Tags 迁移任务模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/migration-tasks/{id} [delete]
func (c *MigrationTaskController) DeleteTask(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.Service.DeleteTask(id); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"deleted": id})
}

// @Summary 获取迁移任务提交情况列表
// @Tags 迁移任务模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param name query string false "学生姓名"
// @Param status query string false "状态"
// @Success 200 {object} util.Response
// @Router /api/teacher/migration-tasks/{id}/submissions [get]
func (c *MigrationTaskController) ListSubmissions(ctx *gin.Context) {
	// 优先从查询参数获取 taskId，如果为空则从路径参数获取
	taskId := ctx.Query("taskId")
	if taskId == "" {
		taskId = ctx.Param("id")
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10")) // 默认 10 条
	name := ctx.Query("name")
	status := ctx.Query("status")

	// 调用服务层，taskId 为 "" 或 "all" 时会查询所有任务
	ss, total, err := c.Service.ListSubmissions(taskId, page, limit, name, status)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"items": ss, "total": total})
}

// @Summary 获取迁移任务学生答题详情
// @Tags 迁移任务模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "提交ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/migration-tasks/submissions/{id} [get]
func (c *MigrationTaskController) GetSubmissionDetail(ctx *gin.Context) {
	id := ctx.Param("id")

	detail, err := c.Service.GetSubmissionDetail(id)
	if err != nil {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, detail)
}

// --- 学生端接口 ---

// @Summary 学生获取已发布的迁移任务列表
// @Tags 迁移任务模块
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/student/migration-tasks/published [get]
func (c *MigrationTaskController) GetPublishedTasks(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	tasks, err := c.Service.ListPublishedTasksForStudent(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, tasks)
}

// @Summary 学生获取迁移任务详情（包含题目）
// @Tags 迁移任务模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Success 200 {object} util.Response
// @Router /api/student/migration-tasks/{id} [get]
func (c *MigrationTaskController) GetStudentTaskDetail(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	id := ctx.Param("id")

	detail, err := c.Service.GetStudentTaskDetail(user.UserID, id)
	if err != nil {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, detail)
}

// @Summary 学生开始迁移任务答题
// @Tags 迁移任务模块
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Success 200 {object} util.Response
// @Router /api/student/migration-tasks/{id}/start [post]
func (c *MigrationTaskController) StartTask(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	id := ctx.Param("id")
	submission, err := c.Service.StartTask(user.UserID, id)
	if err != nil {
		util.Error(ctx, 403, err.Error())
		return
	}

	util.Success(ctx, submission)
}

// @Summary 学生提交迁移任务答案
// @Tags 迁移任务模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Param body body service.MigrationSubmissionReq true "提交信息"
// @Success 200 {object} util.Response
// @Router /api/student/migration-tasks/{id}/submit [post]
func (c *MigrationTaskController) SubmitTask(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	id := ctx.Param("id")
	var req service.MigrationSubmissionReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	submission, err := c.Service.SubmitTask(user.UserID, id, req)
	if err != nil {
		util.Error(ctx, 403, err.Error())
		return
	}

	util.Success(ctx, submission)
}

// @Summary 学生记录迁移任务学习时长
// @Tags 迁移任务模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Param body body service.RecordLearningTimeRequest true "学习时长信息"
// @Success 200 {object} util.Response
// @Router /api/student/migration-tasks/{id}/learning-time [post]
func (c *MigrationTaskController) RecordLearningTime(ctx *gin.Context) {
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
