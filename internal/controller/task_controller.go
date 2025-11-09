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

// TaskController 处理任务相关的API请求
type TaskController struct {
	TaskService *service.TaskService
}

func NewTaskController(taskService *service.TaskService) *TaskController {
	return &TaskController{TaskService: taskService}
}

// SetWeeklyTaskRequest 定义周任务设置请求模型
// swagger:model SetWeeklyTaskRequest
type SetWeeklyTaskRequest struct {
	ResourceModuleID uint             `json:"resourceModuleId" binding:"required"`
	TaskItems        []model.TaskItem `json:"taskItems" binding:"required"`
}

// GetWeeklyTasksRequest 定义获取周任务列表请求参数
// swagger:model GetWeeklyTasksRequest
type GetWeeklyTasksRequest struct {
	Page   int    `form:"page" binding:"omitempty,min=1"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Search string `form:"search"`
}

// SetWeeklyTask godoc
// @Summary 老师设置周任务
// @Description 老师为特定资源分类设置一周的学习任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SetWeeklyTaskRequest true "周任务设置请求"
// @Success 200 {object} util.Response{data=map[string]interface{}} "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/teacher/tasks/weekly [post]
func (c *TaskController) SetWeeklyTask(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil || (user.Role != model.Teacher && user.Role != model.Admin) {
		util.Forbidden(ctx)
		return
	}

	var request SetWeeklyTaskRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	weeklyTask, err := c.TaskService.SetWeeklyTask(user.UserID, request.ResourceModuleID, request.TaskItems)
	if err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	util.Success(ctx, gin.H{
		"task": weeklyTask,
	})
}

// GetTodayTasks godoc
// @Summary 获取今天的学习任务
// @Description 获取当前用户今天需要完成的学习任务列表
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param resourceModuleId query int true "资源分类ID"
// @Success 200 {object} util.Response{data=map[string]interface{}} "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/tasks/today [get]
func (c *TaskController) GetTodayTasks(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	resourceModuleIDStr := ctx.Query("resourceModuleId")
	resourceModuleID, err := strconv.ParseUint(resourceModuleIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "资源分类ID无效")
		return
	}

	tasks, err := c.TaskService.GetTodayTasks(user.UserID, uint(resourceModuleID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"tasks": tasks,
	})
}

// UpdateTaskCompletionRequest 定义任务完成状态更新请求模型
// swagger:model UpdateTaskCompletionRequest
type UpdateTaskCompletionRequest struct {
	IsCompleted       bool    `json:"isCompleted"`
	Progress          float64 `json:"progress"`
	ResourceCompleted bool    `json:"resourceCompleted"`
}

// UpdateTaskCompletion godoc
// @Summary 更新任务完成状态
// @Description 更新指定任务的完成状态和进度
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param taskItemId path int true "任务项ID"
// @Param request body UpdateTaskCompletionRequest true "任务完成状态更新请求"
// @Success 200 {object} util.Response{data=map[string]interface{}} "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/tasks/{taskItemId}/completion [post]
func (c *TaskController) UpdateTaskCompletion(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	taskItemIDStr := ctx.Param("taskItemId")
	taskItemID, err := strconv.ParseUint(taskItemIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "任务项ID无效")
		return
	}

	var request UpdateTaskCompletionRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.TaskService.UpdateTaskCompletion(user.UserID, uint(taskItemID),
		request.IsCompleted, request.Progress, request.ResourceCompleted); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	util.Success(ctx, gin.H{
		"message": "任务完成状态已更新",
	})
}

// GetWeeklyTasks godoc
// @Summary 获取老师的周任务列表
// @Description 获取当前老师的历史和当前所有周任务列表，支持分页和搜索
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认1" default(1)
// @Param limit query int false "每页数量，默认10，最大100" default(10)
// @Param search query string false "搜索关键词（按资源模块名称）"
// @Success 200 {object} util.Response{data=map[string]interface{}} "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/teacher/tasks/weekly [get]
func (c *TaskController) GetWeeklyTasks(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil || (user.Role != model.Teacher && user.Role != model.Admin) {
		util.Forbidden(ctx)
		return
	}

	var request GetWeeklyTasksRequest
	// 设置默认值
	request.Page = 1
	request.Limit = 10

	if err := ctx.ShouldBindQuery(&request); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	tasks, total, err := c.TaskService.GetWeeklyTasks(user.UserID, request.Page, request.Limit, request.Search)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"tasks": tasks,
		"total": total,
		"page":  request.Page,
		"limit": request.Limit,
	})
}

// GetCurrentWeekTask godoc
// @Summary 获取当前周任务
// @Description 获取当前老师本周的任务，可选择指定资源分类ID
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param resourceModuleId query int false "资源分类ID"
// @Success 200 {object} util.Response{data=map[string]interface{}} "成功"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 404 {object} util.Response "当前周任务不存在"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/teacher/tasks/weekly/current [get]
func (c *TaskController) GetCurrentWeekTask(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil || (user.Role != model.Teacher && user.Role != model.Admin) {
		util.Forbidden(ctx)
		return
	}

	// 获取资源分类ID查询参数
	var resourceModuleID uint
	resourceModuleIDStr := ctx.Query("resourceModuleId")
	if resourceModuleIDStr != "" {
		id, err := strconv.ParseUint(resourceModuleIDStr, 10, 32)
		if err != nil {
			util.BadRequest(ctx, "资源分类ID无效")
			return
		}
		resourceModuleID = uint(id)
	}

	task, err := c.TaskService.GetCurrentWeekTask(user.UserID, resourceModuleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			util.Error(ctx, http.StatusNotFound, "当前周任务不存在")
			return
		}
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"task": task,
	})
}

// DeleteWeeklyTask godoc
// @Summary 删除周任务
// @Description 删除指定的周任务及其所有任务项
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param taskId path int true "周任务ID"
// @Success 200 {object} util.Response{data=map[string]interface{}} "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 404 {object} util.Response "任务不存在或无权删除"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/teacher/tasks/weekly/{taskId} [delete]
func (c *TaskController) DeleteWeeklyTask(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil || (user.Role != model.Teacher && user.Role != model.Admin) {
		util.Forbidden(ctx)
		return
	}

	taskIDStr := ctx.Param("taskId")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "任务ID无效")
		return
	}

	err = c.TaskService.DeleteWeeklyTask(uint(taskID), user.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			util.Error(ctx, http.StatusNotFound, "任务不存在或无权删除")
			return
		}
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"message": "周任务删除成功",
	})
}
