package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AchievementController struct {
	AchievementService *service.AchievementService
}

func NewAchievementController(achievementService *service.AchievementService) *AchievementController {
	return &AchievementController{AchievementService: achievementService}
}

// @Summary 获取用户成就
// @Description 获取用户的成就、徽章和积分
// @Tags 成就系统
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/achievements [get]
func (c *AchievementController) GetUserAchievements(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	achievements, err := c.AchievementService.GetUserAchievements(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, achievements)
}

// @Summary 获取排行榜
// @Description 获取用户积分排行榜
// @Tags 成就系统
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "返回数量" default(10)
// @Success 200 {object} util.Response
// @Router /api/achievements/leaderboard [get]
func (c *AchievementController) GetLeaderboard(ctx *gin.Context) {
	limit := 10
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	leaderboard, err := c.AchievementService.GetLeaderboard(limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, leaderboard)
}

// @Summary 获取用户目标
// @Description 获取用户的学习目标
// @Tags 成就系统
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/achievements/goals [get]
func (c *AchievementController) GetUserGoals(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	goals, err := c.AchievementService.GetUserGoals(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, goals)
}

// @Summary 创建学习目标
// @Description 创建新的学习目标
// @Tags 成就系统
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param goal body service.GoalRequest true "目标信息"
// @Success 200 {object} util.Response
// @Router /api/achievements/goals [post]
func (c *AchievementController) CreateGoal(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.GoalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	goal, err := c.AchievementService.CreateGoal(user.UserID, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, goal)
}

// @Summary 更新目标进度
// @Description 更新学习目标的进度
// @Tags 成就系统
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param goalId path int true "目标ID"
// @Param progress body int true "进度百分比" minimum(0) maximum(100)
// @Success 200 {object} util.Response
// @Router /api/achievements/goals/{goalId} [patch]
func (c *AchievementController) UpdateGoalProgress(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	goalIDStr := ctx.Param("goalId")
	goalID, err := strconv.Atoi(goalIDStr)
	if err != nil {
		util.BadRequest(ctx, "Invalid goal ID")
		return
	}

	var req struct {
		Progress int `json:"progress" binding:"required,min=0,max=100"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	err = c.AchievementService.UpdateGoalProgress(user.UserID, uint(goalID), req.Progress)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"message": "Goal progress updated"})
}
