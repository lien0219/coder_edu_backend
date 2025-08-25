package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type LearningController struct {
	LearningService *service.LearningService
}

func NewLearningController(learningService *service.LearningService) *LearningController {
	return &LearningController{LearningService: learningService}
}

// @Summary 获取课前准备内容
// @Description 获取课前准备模块的内容
// @Tags 学习模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/learning/pre-class [get]
func (c *LearningController) GetPreClass(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	preClass, err := c.LearningService.GetPreClassContent(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, preClass)
}

// @Summary 获取课中学习内容
// @Description 获取课中学习模块的内容
// @Tags 学习模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/learning/in-class [get]
func (c *LearningController) GetInClass(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	inClass, err := c.LearningService.GetInClassContent(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, inClass)
}

// @Summary 获取课后回顾内容
// @Description 获取课后回顾模块的内容
// @Tags 学习模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/learning/post-class [get]
func (c *LearningController) GetPostClass(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	postClass, err := c.LearningService.GetPostClassContent(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, postClass)
}

// @Summary 提交学习日志
// @Description 提交课后学习日志和反思
// @Tags 学习模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param log body service.LearningLogRequest true "学习日志内容"
// @Success 200 {object} util.Response
// @Router /api/learning/learning-log [post]
func (c *LearningController) SubmitLearningLog(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.LearningLogRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	err := c.LearningService.SubmitLearningLog(user.UserID, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"message": "Learning log submitted"})
}

// @Summary 提交测验答案
// @Description 提交课后测验答案
// @Tags 学习模块
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param quizId path int true "测验ID"
// @Param answers body service.QuizSubmission true "测验答案"
// @Success 200 {object} util.Response
// @Router /api/learning/quiz/{quizId} [post]
func (c *LearningController) SubmitQuiz(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	quizIDStr := ctx.Param("quizId")
	quizID, err := strconv.Atoi(quizIDStr)
	if err != nil {
		util.BadRequest(ctx, "Invalid quiz ID")
		return
	}

	var submission service.QuizSubmission
	if err := ctx.ShouldBindJSON(&submission); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.LearningService.SubmitQuiz(user.UserID, uint(quizID), submission)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, result)
}
