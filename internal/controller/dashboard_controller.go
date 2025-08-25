package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DashboardController struct {
	DashboardService *service.DashboardService
}

func NewDashboardController(dashboardService *service.DashboardService) *DashboardController {
	return &DashboardController{DashboardService: dashboardService}
}

// @Summary 获取仪表盘数据
// @Description 获取用户仪表盘数据，包括今日任务、进度、资源等
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/dashboard [get]
func (c *DashboardController) GetDashboard(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	dashboard, err := c.DashboardService.GetUserDashboard(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, dashboard)
}

// @Summary 获取今日任务
// @Description 获取用户今日需要完成的学习任务
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/dashboard/today-tasks [get]
func (c *DashboardController) GetTodayTasks(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	tasks, err := c.DashboardService.GetTodayTasks(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, tasks)
}

// @Summary 更新任务状态
// @Description 标记任务为完成或进行中
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param taskId path int true "任务ID"
// @Param status body string true "任务状态" Enums(completed, in_progress)
// @Success 200 {object} util.Response
// @Router /api/dashboard/tasks/{taskId} [patch]
func (c *DashboardController) UpdateTaskStatus(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req struct {
		Status model.TaskStatus `json:"status" binding:"required,oneof=completed in_progress"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	taskIDStr := ctx.Param("taskId")
	if taskIDStr == "" {
		util.BadRequest(ctx, "taskId is required")
		return
	}

	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "invalid taskId")
		return
	}

	err = c.DashboardService.UpdateTaskStatus(uint(taskID), req.Status)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"message": "Task status updated"})
}
