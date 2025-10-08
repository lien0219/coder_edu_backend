package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

// LearningGoalController 处理学习目标的API请求

type LearningGoalController struct {
	LearningGoalService *service.LearningGoalService
}

func NewLearningGoalController(learningGoalService *service.LearningGoalService) *LearningGoalController {
	return &LearningGoalController{LearningGoalService: learningGoalService}
}

// @Summary 获取推荐资源模块
// @Description 获取所有可用于学习目标的推荐资源模块
// @Tags 学习目标
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/learning-goals/resources [get]
func (c *LearningGoalController) GetRecommendedResourceModules(ctx *gin.Context) {
	modules, err := c.LearningGoalService.GetRecommendedResourceModules()
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, modules)
}

// @Summary 创建学习目标
// @Description 创建新的学习目标
// @Tags 学习目标
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param goal body service.CreateGoalRequest true "学习目标信息"
// @Success 201 {object} util.Response
// @Router /api/learning-goals [post]
func (c *LearningGoalController) CreateGoal(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.CreateGoalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	goal, err := c.LearningGoalService.CreateGoal(user.UserID, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, goal)
}

// @Summary 获取所有学习目标
// @Description 获取用户的所有学习目标
// @Tags 学习目标
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/learning-goals [get]
func (c *LearningGoalController) GetUserGoals(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	goals, err := c.LearningGoalService.GetUserGoals(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, goals)
}

// @Summary 获取特定类型的学习目标
// @Description 获取用户特定类型的学习目标（短期或长期）
// @Tags 学习目标
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param type query string true "目标类型" enums(short_term,long_term)
// @Success 200 {object} util.Response
// @Router /api/learning-goals/type [get]
func (c *LearningGoalController) GetUserGoalsByType(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	goalTypeStr := ctx.Query("type")
	if goalTypeStr != "short_term" && goalTypeStr != "long_term" {
		util.BadRequest(ctx, "Invalid goal type. Must be 'short_term' or 'long_term'")
		return
	}

	goalType := model.GoalType(goalTypeStr)
	goals, err := c.LearningGoalService.GetUserGoalsByType(user.UserID, goalType)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, goals)
}

// @Summary 获取特定ID的学习目标
// @Description 获取特定ID的学习目标详情
// @Tags 学习目标
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "目标ID"
// @Success 200 {object} util.Response
// @Router /api/learning-goals/{id} [get]
func (c *LearningGoalController) GetGoalByID(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	goalIDStr := ctx.Param("id")
	goalID, err := strconv.ParseUint(goalIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid goal ID")
		return
	}

	goal, err := c.LearningGoalService.GetGoalByID(user.UserID, uint(goalID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, goal)
}

// @Summary 更新学习目标
// @Description 更新学习目标信息
// @Tags 学习目标
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "目标ID"
// @Param goal body service.UpdateGoalRequest true "学习目标更新信息"
// @Success 200 {object} util.Response
// @Router /api/learning-goals/{id} [put]
func (c *LearningGoalController) UpdateGoal(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	goalIDStr := ctx.Param("id")
	goalID, err := strconv.ParseUint(goalIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid goal ID")
		return
	}

	var req service.UpdateGoalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	goal, err := c.LearningGoalService.UpdateGoal(user.UserID, uint(goalID), req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, goal)
}

// @Summary 删除学习目标
// @Description 删除学习目标
// @Tags 学习目标
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "目标ID"
// @Success 200 {object} util.Response
// @Router /api/learning-goals/{id} [delete]
func (c *LearningGoalController) DeleteGoal(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	goalIDStr := ctx.Param("id")
	goalID, err := strconv.ParseUint(goalIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid goal ID")
		return
	}

	err = c.LearningGoalService.DeleteGoal(user.UserID, uint(goalID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"message": "Goal deleted successfully"})
}

// @Summary 获取学习目标详情（含资源模块进度）
// @Description 获取学习目标详情，包括关联资源模块的详细进度
// @Tags 学习目标
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "目标ID"
// @Success 200 {object} util.Response
// @Router /api/learning-goals/{id}/details [get]
func (c *LearningGoalController) GetGoalDetails(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	goalIDStr := ctx.Param("id")
	goalID, err := strconv.ParseUint(goalIDStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "Invalid goal ID")
		return
	}

	goal, err := c.LearningGoalService.GetGoalByID(user.UserID, uint(goalID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, goal)
}
