package controller

import (
	"strconv"
	"time"

	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

type GradeController struct {
	LevelService *service.LevelService
}

func NewGradeController(levelService *service.LevelService) *GradeController {
	return &GradeController{LevelService: levelService}
}

// @Summary 列出需人工评分的尝试（按关卡）
// @Tags 评分
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id}/attempts/pending-grading [get]
func (c *GradeController) ListPendingGrading(ctx *gin.Context) {
	levelStr := ctx.Param("id")
	levelID, err := strconv.Atoi(levelStr)
	if err != nil {
		util.BadRequest(ctx, "invalid level id")
		return
	}
	attempts, err := c.LevelService.ListAttemptsNeedingManual(uint(levelID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, attempts)
}

// @Summary 教师对尝试进行人工评分
// @Tags 评分
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Param attemptId path int true "尝试ID"
// @Param body body object true "scores [{questionId, score, comment}]"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id}/attempts/{attemptId}/grade [post]
func (c *GradeController) GradeAttempt(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}
	levelStr := ctx.Param("id")
	_, err := strconv.Atoi(levelStr)
	if err != nil {
		util.BadRequest(ctx, "invalid level id")
		return
	}
	attemptStr := ctx.Param("attemptId")
	aid, err := strconv.Atoi(attemptStr)
	if err != nil {
		util.BadRequest(ctx, "invalid attempt id")
		return
	}
	var body struct {
		Scores []struct {
			QuestionID uint   `json:"questionId"`
			Score      int    `json:"score"`
			Comment    string `json:"comment"`
		} `json:"scores"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}
	// build score entities
	var scores []service.QuestionScore
	now := time.Now()
	for _, s := range body.Scores {
		scores = append(scores, service.QuestionScore{
			QuestionID: s.QuestionID,
			Score:      s.Score,
			Comment:    s.Comment,
			GraderID:   user.UserID,
			GradedAt:   &now,
		})
	}

	if err := c.LevelService.ManualGradeAttempt(user.UserID, uint(aid), scores); err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"graded": true})
}
