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

// @Summary 获取能力评估雷达图
// @Description 获取用户的六维能力评估数据（问题解决、批判性思维等）
// @Tags 分析
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/analytics/abilities [get]
func (c *AnalyticsController) GetAbilities(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	abilities, err := c.AnalyticsService.GetAbilityRadar(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, abilities)
}

// @Summary 获取关卡挑战曲线
// @Description 获取用户在特定关卡中的多次尝试得分变化趋势
// @Tags 分析
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param levelId path int false "关卡ID (不传则取最近一次尝试的关卡)"
// @Param limit query int false "返回最近几次尝试 (默认10)" default(10)
// @Success 200 {object} util.Response
// @Router /api/analytics/levels/{levelId}/curve [get]
func (c *AnalyticsController) GetLevelCurve(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	levelIDStr := ctx.Param("levelId")
	var levelID int
	if levelIDStr != "" && levelIDStr != "0" {
		levelID, _ = strconv.Atoi(levelIDStr)
	}

	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))

	curve, err := c.AnalyticsService.GetLevelLearningCurve(user.UserID, uint(levelID), limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, curve)
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

// @Summary 获取每周挑战统计
// @Description 获取用户每周的挑战平均分和完成挑战的个数，用于曲线图
// @Tags 分析
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param weeks query int false "周数 (默认8)" default(8)
// @Param week query string false "指定周 (格式: YYYY-WW, 例如 2026-02)"
// @Success 200 {object} util.Response
// @Router /api/analytics/challenges/weekly [get]
func (c *AnalyticsController) GetWeeklyChallengeStats(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	weeks, _ := strconv.Atoi(ctx.DefaultQuery("weeks", "8"))
	specificWeek := ctx.Query("week")

	stats, err := c.AnalyticsService.GetWeeklyChallengeStats(user.UserID, weeks, specificWeek)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, stats)
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
