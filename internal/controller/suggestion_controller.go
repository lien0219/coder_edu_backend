package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SuggestionController struct {
	SuggestionService *service.SuggestionService
}

func NewSuggestionController(suggestionService *service.SuggestionService) *SuggestionController {
	return &SuggestionController{SuggestionService: suggestionService}
}

// @Summary 教师发布建议
// @Tags 教师建议
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param suggestion body model.Suggestion true "建议内容"
// @Success 201 {object} util.Response
// @Router /api/teacher/suggestions [post]
func (c *SuggestionController) CreateSuggestion(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var suggestion model.Suggestion
	if err := ctx.ShouldBindJSON(&suggestion); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	suggestion.TeacherID = user.UserID
	if err := c.SuggestionService.CreateSuggestion(&suggestion); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, suggestion)
}

// @Summary 教师编辑建议
// @Tags 教师建议
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "建议ID"
// @Param suggestion body model.Suggestion true "建议内容"
// @Success 200 {object} util.Response
// @Router /api/teacher/suggestions/{id} [put]
func (c *SuggestionController) UpdateSuggestion(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}

	var suggestion model.Suggestion
	if err := ctx.ShouldBindJSON(&suggestion); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.SuggestionService.UpdateSuggestion(uint(id), user.UserID, &suggestion); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, nil)
}

// @Summary 教师获取已发布建议列表
// @Tags 教师建议
// @Security BearerAuth
// @Produce json
// @Success 200 {object} util.Response
// @Router /api/teacher/suggestions [get]
func (c *SuggestionController) ListTeacherSuggestions(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	suggestions, err := c.SuggestionService.GetTeacherSuggestions(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, suggestions)
}

// @Summary 教师删除建议
// @Tags 教师建议
// @Security BearerAuth
// @Produce json
// @Param id path int true "建议ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/suggestions/{id} [delete]
func (c *SuggestionController) DeleteSuggestion(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}

	if err := c.SuggestionService.DeleteSuggestion(uint(id), user.UserID); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, nil)
}

// @Summary 学生获取建议列表
// @Tags 教师建议
// @Security BearerAuth
// @Produce json
// @Success 200 {object} util.Response
// @Router /api/student/suggestions [get]
func (c *SuggestionController) ListStudentSuggestions(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	suggestions, err := c.SuggestionService.GetStudentSuggestions(user.UserID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, suggestions)
}

// @Summary 学生完成建议
// @Tags 教师建议
// @Security BearerAuth
// @Produce json
// @Param id path int true "建议ID"
// @Success 200 {object} util.Response
// @Router /api/student/suggestions/{id}/complete [post]
func (c *SuggestionController) CompleteSuggestion(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}

	if err := c.SuggestionService.CompleteSuggestion(uint(id), user.UserID); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, nil)
}

// @Summary 教师获取学生学习进度汇总
// @Tags 教师建议
// @Security BearerAuth
// @Produce json
// @Param id path int true "学生ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/students/{id}/progress [get]
func (c *SuggestionController) GetStudentProgress(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	studentIDStr := ctx.Param("id")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		util.BadRequest(ctx, "invalid student id")
		return
	}

	progress, err := c.SuggestionService.GetStudentProgressForTeacher(uint(studentID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, progress)
}

// @Summary 教师获取所有学生学习进度列表
// @Tags 教师建议
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param search query string false "搜索关键词"
// @Success 200 {object} util.Response
// @Router /api/teacher/students/progress [get]
func (c *SuggestionController) ListStudentsProgress(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))
	search := ctx.Query("search")

	items, total, err := c.SuggestionService.ListStudentsProgress(page, pageSize, search)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"items": items,
		"total": total,
		"page":  page,
	})
}
