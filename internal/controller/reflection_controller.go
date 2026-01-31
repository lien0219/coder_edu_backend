package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ReflectionController struct {
	service *service.ReflectionService
}

func NewReflectionController(s *service.ReflectionService) *ReflectionController {
	return &ReflectionController{service: s}
}

type SaveReflectionRequest struct {
	Summary     string `json:"summary"`
	Challenges  string `json:"challenges"`
	Connections string `json:"connections"`
	NextSteps   string `json:"nextSteps"`
}

// SaveReflection godoc
// @Summary 学生保存或更新有效反思
// @Description 学生填写总结关键知识点、识别挑战、连接已有知识、规划下一步
// @Tags 有效反思
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body SaveReflectionRequest true "反思内容"
// @Success 200 {object} util.Response{data=model.Reflection}
// @Router /api/reflections/my [post]
func (c *ReflectionController) SaveMyReflection(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req SaveReflectionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	reflection, err := c.service.SaveReflection(user.UserID, req.Summary, req.Challenges, req.Connections, req.NextSteps)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, reflection)
}

// GetMyReflection godoc
// @Summary 获取我的有效反思
// @Tags 有效反思
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} util.Response{data=model.Reflection}
// @Router /api/reflections/my [get]
func (c *ReflectionController) GetMyReflection(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	reflection, err := c.service.GetReflectionByUserID(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, reflection)
}

// ListAllReflections godoc
// @Summary 老师/管理员列出所有学生的有效反思
// @Tags 有效反思
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param name query string false "学生姓名筛选"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Success 200 {object} util.Response{data=map[string]interface{}}
// @Router /api/teacher/reflections [get]
func (c *ReflectionController) ListAllReflections(ctx *gin.Context) {
	name := ctx.Query("name")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))

	reflections, total, err := c.service.ListAllReflections(name, page, pageSize)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"items": reflections,
		"total": total,
		"page":  page,
		"pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// UpdateReflection godoc
// @Summary 老师/管理员修改有效反思数据
// @Tags 有效反思
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param userId path int true "用户ID"
// @Param body body SaveReflectionRequest true "反思内容"
// @Success 200 {object} util.Response{data=model.Reflection}
// @Router /api/teacher/reflections/user/{userId} [put]
func (c *ReflectionController) UpdateReflection(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		util.BadRequest(ctx, "无效的用户ID")
		return
	}

	var req SaveReflectionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	reflection, err := c.service.UpdateReflectionByUserID(uint(userID), req.Summary, req.Challenges, req.Connections, req.NextSteps)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, reflection)
}
