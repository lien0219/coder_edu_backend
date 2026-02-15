package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// TaskModuleGroup 定义单个资源模块的任务组
type TaskModuleGroup struct {
	ResourceModuleID uint             `json:"resourceModuleId" binding:"required"`
	TaskItems        []model.TaskItem `json:"taskItems" binding:"required"`
}

// SetWeeklyTaskRequest 定义周任务设置请求模型
// swagger:model SetWeeklyTaskRequest
type SetWeeklyTaskRequest struct {
	WeeklyTasks []TaskModuleGroup `json:"weekly_tasks" binding:"required"`
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

	// 先按资源模块ID分组，合并同一模块的所有任务项
	moduleTaskMap := make(map[uint][]model.TaskItem)
	for _, taskGroup := range request.WeeklyTasks {
		moduleTaskMap[taskGroup.ResourceModuleID] = append(moduleTaskMap[taskGroup.ResourceModuleID], taskGroup.TaskItems...)
	}

	// 处理多个资源模块的周任务
	results := make([]interface{}, 0, len(moduleTaskMap))
	errors := make([]string, 0)
	successCount := 0

	for resourceModuleID, allTaskItems := range moduleTaskMap {
		weeklyTask, err := c.TaskService.SetWeeklyTask(user.UserID, resourceModuleID, allTaskItems)
		if err != nil {
			errors = append(errors, fmt.Sprintf("模块%d: %s", resourceModuleID, err.Error()))
			continue // 继续处理其他模块，不中断整个流程
		}
		results = append(results, weeklyTask)
		successCount++
	}

	// 如果全部失败，返回错误
	if successCount == 0 && len(errors) > 0 {
		util.BadRequest(ctx, fmt.Sprintf("所有模块创建失败: %s", strings.Join(errors, "; ")))
		return
	}

	// 部分成功的情况
	if len(errors) > 0 {
		util.Success(ctx, gin.H{
			"tasks":        results,
			"warning":      fmt.Sprintf("部分模块创建失败: %s", strings.Join(errors, "; ")),
			"successCount": successCount,
			"totalCount":   len(request.WeeklyTasks),
		})
		return
	}

	util.Success(ctx, gin.H{
		"tasks": results,
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
	var resourceModuleID uint
	if resourceModuleIDStr != "" {
		parsedID, err := strconv.ParseUint(resourceModuleIDStr, 10, 32)
		if err != nil {
			util.BadRequest(ctx, "资源分类ID无效")
			return
		}
		resourceModuleID = uint(parsedID)
	}

	tasks, err := c.TaskService.GetTodayTasks(user.UserID, resourceModuleID)
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
// @Description 获取本周的任务，可选择指定资源分类ID和日期。如果不指定资源分类ID，返回所有模块的一周任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param resourceModuleId query int false "资源分类ID"
// @Param date query string false "目标日期 (YYYY-MM-DD)，默认为今天"
// @Success 200 {object} util.Response{data=map[string]interface{}} "成功"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 404 {object} util.Response "周任务不存在"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/teacher/tasks/weekly/current [get]
func (c *TaskController) GetCurrentWeekTask(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
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

	// 获取日期查询参数
	var targetDate time.Time
	dateStr := ctx.Query("date")
	if dateStr != "" {
		var err error
		targetDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			util.BadRequest(ctx, "日期格式无效，请使用 YYYY-MM-DD")
			return
		}
	} else {
		targetDate = time.Now()
	}

	result, err := c.TaskService.GetCurrentWeekTask(user.UserID, user.Role, resourceModuleID, targetDate)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			util.Success(ctx, gin.H{
				"task": nil,
			})
			return
		}
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"task": result,
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
