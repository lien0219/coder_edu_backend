package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"coder_edu_backend/pkg/logger"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LevelController struct {
	LevelService   *service.LevelService
	ContentService *service.ContentService
}

func NewLevelController(levelService *service.LevelService, contentService *service.ContentService) *LevelController {
	return &LevelController{LevelService: levelService, ContentService: contentService}
}

// @Summary 创建关卡
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param level body service.LevelCreateRequest true "关卡信息"
// @Success 201 {object} util.Response
// @Router /api/teacher/levels [post]
func (c *LevelController) CreateLevel(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.LevelCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	level, err := c.LevelService.CreateLevel(user.UserID, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, level)
}

// @Summary 获取关卡详情
// @Tags 关卡管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "关卡ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id} [get]
func (c *LevelController) GetLevel(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}
	level, err := c.LevelService.LevelRepo.FindByID(uint(id))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, level)
}

// @Summary 列表关卡
// @Tags 关卡管理
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} util.Response
// @Router /api/teacher/levels [get]
func (c *LevelController) ListLevels(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}
	page := 1
	limit := 20
	if p := ctx.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := ctx.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	levels, total, err := c.LevelService.ListLevelsFull(user.UserID, page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"items": levels, "total": total})
}

// @Summary 更新关卡
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Param level body service.LevelCreateRequest true "关卡信息"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id} [put]
func (c *LevelController) UpdateLevel(ctx *gin.Context) {
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
	var req service.LevelCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}
	level, err := c.LevelService.UpdateLevel(user.UserID, uint(id), req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, level)
}

// @Summary 发布/下架关卡
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Param action body object true "publish:bool"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id}/publish [post]
func (c *LevelController) PublishLevel(ctx *gin.Context) {
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
	var body struct {
		Publish bool `json:"publish"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}
	if err := c.LevelService.PublishLevel(user.UserID, uint(id), body.Publish); err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"published": body.Publish})
}

// @Summary 批量更新关卡字段（上限/积分/发布等）
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "ids, updates"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/bulk [post]
func (c *LevelController) BulkUpdate(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}
	var body struct {
		IDs     []uint                 `json:"ids" binding:"required"`
		Updates map[string]interface{} `json:"updates" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}
	if err := c.LevelService.BulkUpdateLevels(user.UserID, body.IDs, body.Updates); err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"updated": len(body.IDs)})
}

// @Summary 获取关卡版本列表
// @Tags 关卡管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id}/versions [get]
func (c *LevelController) GetVersions(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}
	versions, err := c.LevelService.GetVersions(uint(id))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, versions)
}

// @Summary 回滚到某个版本
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Param versionId path int true "版本ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id}/versions/{versionId}/rollback [post]
func (c *LevelController) RollbackVersion(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}
	idStr := ctx.Param("id")
	_, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}
	verStr := ctx.Param("versionId")
	verID, err := strconv.Atoi(verStr)
	if err != nil {
		util.BadRequest(ctx, "invalid version id")
		return
	}
	levelID, _ := strconv.ParseUint(idStr, 10, 32)
	if err := c.LevelService.RollbackToVersion(user.UserID, uint(levelID), uint(verID)); err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"rolled_back_to": verID})
}

// @Summary 上传关卡封面（教师）
// @Tags 关卡管理
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param cover formData file true "封面文件"
// @Param id path int true "关卡ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id}/upload/cover [post]
func (c *LevelController) UploadCover(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}
	idStr := ctx.Param("id")
	if _, err := strconv.Atoi(idStr); err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}
	file, err := ctx.FormFile("cover")
	if err != nil {
		util.BadRequest(ctx, "cover file is required")
		return
	}

	// validate extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true}
	if !allowed[ext] {
		util.BadRequest(ctx, "unsupported file type")
		return
	}
	// upload via ContentService to create a Resource record
	resource := &model.Resource{
		Title:      fmt.Sprintf("Level %s Cover", idStr),
		Type:       model.Article,
		ModuleType: "level_cover",
	}
	if err := c.ContentService.UploadResource(ctx, file, resource); err != nil {
		util.InternalServerError(ctx)
		return
	}
	// attach to level
	levelID, _ := strconv.ParseUint(idStr, 10, 32)
	level, err := c.LevelService.LevelRepo.FindByID(uint(levelID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	level.CoverURL = resource.URL
	if err := c.LevelService.LevelRepo.Update(level); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"url": resource.URL, "resourceId": resource.ID})
}

// @Summary 上传关卡题目附件（教师）
// @Tags 关卡管理
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "附件文件"
// @Param id path int true "关卡ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id}/upload/attachment [post]
func (c *LevelController) UploadAttachment(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}
	idStr := ctx.Param("id")
	if _, err := strconv.Atoi(idStr); err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		util.BadRequest(ctx, "file is required")
		return
	}
	// reuse ContentService to upload and create resource record
	resource := &model.Resource{
		Title:      file.Filename,
		Type:       model.Article,
		ModuleType: "level_attachment",
	}
	if err := c.ContentService.UploadResource(ctx, file, resource); err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"url": resource.URL, "resourceId": resource.ID})
}

// @Summary 新增题目到关卡
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Param body body service.LevelQuestionRequest true "题目信息"
// @Success 201 {object} util.Response
// @Router /api/teacher/levels/{id}/questions [post]
func (c *LevelController) CreateQuestion(ctx *gin.Context) {
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
	var req service.LevelQuestionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}
	q, err := c.LevelService.AddQuestion(user.UserID, uint(id), req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Created(ctx, q)
}

// @Summary 更新关卡题目
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Param qid path int true "题目ID"
// @Param body body service.LevelQuestionRequest true "题目信息"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{levelId}/questions/{qid} [put]
func (c *LevelController) UpdateQuestion(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}
	levelStr := ctx.Param("id")
	levelID, err := strconv.Atoi(levelStr)
	if err != nil {
		util.BadRequest(ctx, "invalid level id")
		return
	}
	qStr := ctx.Param("qid")
	qid, err := strconv.Atoi(qStr)
	if err != nil {
		util.BadRequest(ctx, "invalid question id")
		return
	}
	var req service.LevelQuestionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}
	q, err := c.LevelService.UpdateQuestion(user.UserID, uint(levelID), uint(qid), req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, q)
}

// @Summary 删除题目
// @Tags 关卡管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Param qid path int true "题目ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{levelId}/questions/{qid} [delete]
func (c *LevelController) DeleteQuestion(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}
	levelStr := ctx.Param("id")
	levelID, err := strconv.Atoi(levelStr)
	if err != nil {
		util.BadRequest(ctx, "invalid level id")
		return
	}
	qStr := ctx.Param("qid")
	qid, err := strconv.Atoi(qStr)
	if err != nil {
		util.BadRequest(ctx, "invalid question id")
		return
	}
	if err := c.LevelService.DeleteQuestion(uint(levelID), uint(qid)); err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"deleted": qid})
}

// @Summary 删除关卡
// @Tags 关卡管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id} [delete]
func (c *LevelController) DeleteLevel(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}
	levelStr := ctx.Param("id")
	levelID, err := strconv.Atoi(levelStr)
	if err != nil {
		util.BadRequest(ctx, "invalid level id")
		return
	}
	if err := c.LevelService.DeleteLevel(user.UserID, uint(levelID)); err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"deleted": levelID})
}

// @Summary 获取学生端关卡列表
// @Description 获取学生可访问的关卡列表，支持搜索、筛选和分页
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词（关卡名称或描述）"
// @Param difficulty query string false "难度筛选" Enums(easy,medium,hard,all)
// @Param status query string false "状态筛选" Enums(not_started,in_progress,completed,all)
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(9)
// @Success 200 {object} util.Response
// @Router /api/levels/student [get]
func (c *LevelController) GetStudentLevels(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	// 获取查询参数
	search := ctx.Query("search")
	difficulty := ctx.Query("difficulty")
	status := ctx.Query("status")

	page := 1
	limit := 9 // 默认一页9条数据

	if p := ctx.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	if l := ctx.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	// 获取关卡列表
	levels, total, err := c.LevelService.ListLevelsForStudent(user.UserID, search, difficulty, status, page, limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"items": levels,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// @Summary 获取学生端关卡详情
// @Description 获取学生可访问的关卡详细信息，包括题目信息（不含答案）
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Success 200 {object} util.Response
// @Router /api/levels/student/{id} [get]
func (c *LevelController) GetStudentLevelDetail(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	levelStr := ctx.Param("id")
	levelID, err := strconv.Atoi(levelStr)
	if err != nil {
		util.BadRequest(ctx, "invalid level id")
		return
	}

	levelDetail, err := c.LevelService.GetStudentLevelDetail(user.UserID, uint(levelID))
	if err != nil {
		if err.Error() == "level not found" || err.Error() == "level not accessible" ||
			err.Error() == "level not yet available" || err.Error() == "level no longer available" ||
			err.Error() == "record not found" {
			util.NotFound(ctx)
		} else {
			util.InternalServerError(ctx)
		}
		return
	}

	util.Success(ctx, levelDetail)
}

// @Summary 获取学生端关卡题目列表
// @Description 获取学生可访问的关卡所有题目列表
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Success 200 {object} util.Response
// @Router /api/levels/student/{id}/questions [get]
func (c *LevelController) GetStudentLevelQuestions(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	levelStr := ctx.Param("id")
	levelID, err := strconv.Atoi(levelStr)
	if err != nil {
		util.BadRequest(ctx, "invalid level id")
		return
	}

	questions, err := c.LevelService.GetStudentLevelQuestions(user.UserID, uint(levelID))
	if err != nil {
		if err.Error() == "level not found" || err.Error() == "level not accessible" ||
			err.Error() == "level not yet available" || err.Error() == "level no longer available" ||
			err.Error() == "record not found" {
			util.NotFound(ctx)
		} else {
			util.InternalServerError(ctx)
		}
		return
	}

	util.Success(ctx, gin.H{
		"items": questions,
		"total": len(questions),
	})
}

// @Summary 批量提交关卡答案
// @Description 一次性提交关卡的所有或部分问题答案，支持部分提交
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param levelId path int true "关卡ID"
// @Param attemptId path int true "尝试ID"
// @Param body body map[string]interface{} true "答案提交请求" "{"answers": [{"questionId": 1, "answer": "答案"}]}"
// @Success 200 {object} util.Response
// @Router /api/levels/{levelId}/attempts/{attemptId}/submit [post]
func (c *LevelController) BatchSubmitAnswers(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	levelStr := ctx.Param("id")
	levelID, err := strconv.Atoi(levelStr)
	if err != nil {
		util.BadRequest(ctx, "invalid level id")
		return
	}

	attemptStr := ctx.Param("attemptId")
	attemptID, err := strconv.Atoi(attemptStr)
	if err != nil {
		util.BadRequest(ctx, "invalid attempt id")
		return
	}

	// 使用map接收JSON数据
	var req map[string]interface{}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.LevelService.BatchSubmitAnswers(user.UserID, uint(levelID), uint(attemptID), req)
	if err != nil {
		if err.Error() == "level not found" || err.Error() == "level not accessible" ||
			err.Error() == "level not yet available" || err.Error() == "level no longer available" ||
			err.Error() == "attempt not found" {
			util.NotFound(ctx)
		} else {
			util.InternalServerError(ctx)
		}
		return
	}

	util.Success(ctx, result)
}

// @Summary 获取关卡尝试统计
// @Tags 关卡管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Param start query string false "开始时间 RFC3339"
// @Param end query string false "结束时间 RFC3339"
// @Param studentId query int false "学生ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id}/attempts/stats [get]
func (c *LevelController) GetAttemptStats(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}
	var startPtr *time.Time
	var endPtr *time.Time
	if s := ctx.Query("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startPtr = &t
		} else {
			util.BadRequest(ctx, "invalid start time")
			return
		}
	}
	if e := ctx.Query("end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			endPtr = &t
		} else {
			util.BadRequest(ctx, "invalid end time")
			return
		}
	}
	studentID := uint(0)
	if sid := ctx.Query("studentId"); sid != "" {
		if v, err := strconv.Atoi(sid); err == nil {
			studentID = uint(v)
		}
	}
	stats, err := c.LevelService.GetAttemptStats(uint(id), startPtr, endPtr, studentID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, stats)
}

// @Summary 批量发布/下架关卡
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "ids, publish"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/bulk/publish [post]
func (c *LevelController) BulkPublish(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	if user.Role != model.Teacher && user.Role != model.Admin {
		util.Error(ctx, 403, "only teachers and admins can bulk publish levels")
		return
	}

	var body struct {
		IDs     []uint `json:"ids" binding:"required"`
		Publish bool   `json:"publish"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.LevelService.BulkPublish(user.UserID, body.IDs, body.Publish); err != nil {
		logger.Log.Error("Bulk publish error", zap.Error(err))
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"updated": len(body.IDs), "published": body.Publish})
}

// @Summary 设置定时发布
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Param body body object true "scheduledAt RFC3339 or null to cancel"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id}/schedule_publish [post]
func (c *LevelController) SchedulePublish(ctx *gin.Context) {
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
	var body struct {
		ScheduledAt *string `json:"scheduledAt"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}
	var tPtr *time.Time
	if body.ScheduledAt != nil && *body.ScheduledAt != "" {
		t, err := time.Parse(time.RFC3339, *body.ScheduledAt)
		if err != nil {
			util.BadRequest(ctx, "invalid time format")
			return
		}
		tPtr = &t
	}
	if err := c.LevelService.SchedulePublish(user.UserID, uint(id), tPtr); err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"scheduledAt": tPtr})
}

// @Summary 更新关卡可见范围（全班/class/specific）及特定学生列表
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Param body body object true "visibleScope, visibleTo"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id}/visibility [put]
func (c *LevelController) UpdateVisibility(ctx *gin.Context) {
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
	var body struct {
		VisibleScope string `json:"visibleScope"`
		VisibleTo    []uint `json:"visibleTo"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}
	if err := c.LevelService.UpdateVisibility(user.UserID, uint(id), body.VisibleScope, body.VisibleTo); err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, gin.H{"updated": true})
}

// @Summary 开始关卡挑战
// @Description 开始关卡挑战，创建尝试记录
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Success 200 {object} util.Response
// @Router /api/levels/{id}/attempts/start [post]
// @Router /api/teacher/levels/{id}/attempts/start [post]
func (c *LevelController) StartAttempt(ctx *gin.Context) {
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
	attempt, err := c.LevelService.StartAttempt(user.UserID, uint(id))
	if err != nil {
		if errors.Is(err, util.ErrAttemptLimitReached) {
			util.Error(ctx, http.StatusOK, err.Error())
			return
		}
		util.BadRequest(ctx, err.Error())
		return
	}
	util.Created(ctx, attempt)
}

// @Summary 提交关卡挑战
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "关卡ID"
// @Param attemptId path int true "尝试ID"
// @Param body body object true "answers and perQuestionTimes"
// @Success 200 {object} util.Response
// @Router /api/teacher/levels/{id}/attempts/{attemptId}/submit [post]
func (c *LevelController) SubmitAttempt(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}
	idStr := ctx.Param("id")
	_, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "invalid id")
		return
	}
	attStr := ctx.Param("attemptId")
	attID, err := strconv.Atoi(attStr)
	if err != nil {
		util.BadRequest(ctx, "invalid attempt id")
		return
	}
	var body struct {
		Answers []service.SubmitAnswer    `json:"answers"`
		Times   []service.PerQuestionTime `json:"times"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}
	levelID, _ := strconv.ParseUint(idStr, 10, 32)
	attempt, err := c.LevelService.SubmitAttempt(user.UserID, uint(levelID), uint(attID), body.Answers, body.Times)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, attempt)
}

// @Summary 获取关卡挑战排行榜
// @Description 获取学生关卡挑战总得分排行榜，所有角色都可以访问
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "限制返回数量，默认返回全部"
// @Success 200 {object} util.Response
// @Router /api/levels/ranking [get]
func (c *LevelController) GetLevelRanking(ctx *gin.Context) {
	// 获取查询参数
	limitStr := ctx.DefaultQuery("limit", "0")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		limit = 0 // 0表示不限制
	}

	rankings, err := c.LevelService.GetLevelRanking(limit)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, rankings)
}

// @Summary 获取所有关卡的基础信息(ID和名称)
// @Description 获取系统中所有关卡的ID和标题，通常用于下拉选择器等场景
// @Tags 关卡管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/levels/basic-info [get]
func (c *LevelController) GetAllLevelsBasicInfo(ctx *gin.Context) {
	infos, err := c.LevelService.GetAllLevelsBasicInfo()
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, infos)
}

// @Summary 获取用户关卡挑战总积分
// @Description 获取单个用户的关卡挑战获得的总积分，所有角色都可以访问
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "用户ID"
// @Success 200 {object} util.Response
// @Router /api/users/{userId}/level-total-score [get]
func (c *LevelController) GetUserLevelTotalScore(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		util.BadRequest(ctx, "invalid user id")
		return
	}

	totalScore, err := c.LevelService.GetUserLevelTotalScore(uint(userID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"userId": userID, "totalScore": totalScore})
}

// @Summary 获取用户关卡挑战统计数据
// @Description 获取用户的关卡挑战综合统计数据，包括本周时长、成功率、解决挑战数和总积分
// @Tags 关卡管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "用户ID"
// @Success 200 {object} util.Response
// @Router /api/users/{userId}/level-stats [get]
func (c *LevelController) GetUserLevelStats(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		util.BadRequest(ctx, "invalid user id")
		return
	}

	stats, err := c.LevelService.GetUserLevelStats(uint(userID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, stats)
}
