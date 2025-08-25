package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AnalyticsController struct {
	AnalyticsService *service.AnalyticsService
}

func NewAnalyticsController(analyticsService *service.AnalyticsService) *AnalyticsController {
	return &AnalyticsController{AnalyticsService: analyticsService}
}

// @Summary 获取学习分析概览
// @Description 获取用户的学习分析概览数据
// @Tags 分析
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/analytics/overview [get]
func (c *AnalyticsController) GetOverview(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	overview, err := c.AnalyticsService.GetLearningOverview(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, overview)
}

// @Summary 获取学习进度
// @Description 获取用户的学习进度数据
// @Tags 分析
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param weeks query int false "周数" default(6)
// @Success 200 {object} util.Response
// @Router /api/analytics/progress [get]
func (c *AnalyticsController) GetProgress(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	weeks, _ := strconv.Atoi(ctx.DefaultQuery("weeks", "6"))

	progress, err := c.AnalyticsService.GetLearningProgress(user.UserID, weeks)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, progress)
}

// @Summary 获取技能评估
// @Description 获取用户的技能评估数据
// @Tags 分析
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/analytics/skills [get]
func (c *AnalyticsController) GetSkills(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	skills, err := c.AnalyticsService.GetSkillAssessments(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, skills)
}

// @Summary 获取个性化建议
// @Description 获取基于用户学习数据的个性化建议
// @Tags 分析
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/analytics/recommendations [get]
func (c *AnalyticsController) GetRecommendations(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	recommendations, err := c.AnalyticsService.GetPersonalizedRecommendations(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, recommendations)
}

// @Summary 记录学习会话
// @Description 记录用户的学习会话开始
// @Tags 分析
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param moduleId query int true "模块ID"
// @Success 200 {object} util.Response
// @Router /api/analytics/session/start [post]
func (c *AnalyticsController) StartSession(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	moduleID, _ := strconv.Atoi(ctx.Query("moduleId"))

	sessionID, err := c.AnalyticsService.StartLearningSession(user.UserID, uint(moduleID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"sessionId": sessionID})
}

// @Summary 结束学习会话
// @Description 记录用户的学习会话结束
// @Tags 分析
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param sessionId path int true "会话ID"
// @Param activity body string false "活动数据"
// @Success 200 {object} util.Response
// @Router /api/analytics/session/{sessionId}/end [post]
func (c *AnalyticsController) EndSession(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	sessionIDStr := ctx.Param("sessionId")
	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		util.BadRequest(ctx, "Invalid session ID")
		return
	}

	var req struct {
		Activity string `json:"activity"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	err = c.AnalyticsService.EndLearningSession(user.UserID, uint(sessionID), req.Activity)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"message": "Session ended"})
}
