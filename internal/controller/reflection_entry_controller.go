package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

func (c *ReflectionController) requireStudent(ctx *gin.Context) (*util.Claims, bool) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return nil, false
	}
	if user.Role != model.Student {
		util.Forbidden(ctx)
		return nil, false
	}
	return user, true
}

func (c *ReflectionController) writeEntryError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrReflectionVersionConflict):
		util.ErrorWithCode(ctx, http.StatusConflict, "VERSION_CONFLICT", "reflection version conflict")
	case errors.Is(err, service.ErrReflectionNotFound):
		util.NotFound(ctx)
	case errors.Is(err, util.ErrPermissionDenied):
		util.Forbidden(ctx)
	case errors.Is(err, service.ErrReflectionIdempotencyConflict):
		util.ErrorWithCode(ctx, http.StatusConflict, "IDEMPOTENCY_CONFLICT", err.Error())
	case errors.Is(err, service.ErrReflectionEmpty),
		errors.Is(err, service.ErrReflectionBinding),
		errors.Is(err, service.ErrReflectionIdempotency),
		errors.Is(err, service.ErrReflectionEditUnconfirmed),
		errors.Is(err, service.ErrPlanNotConfirmed),
		errors.Is(err, service.ErrNextStepsEmpty),
		errors.Is(err, service.ErrReflectionModuleRequired),
		errors.Is(err, service.ErrReflectionLegacyReadOnly),
		errors.Is(err, service.ErrReflectionType):
		util.BadRequest(ctx, err.Error())
	default:
		util.InternalServerError(ctx)
	}
}

func (c *ReflectionController) CreateEntry(ctx *gin.Context) {
	user, ok := c.requireStudent(ctx)
	if !ok {
		return
	}
	var body struct {
		ReflectionType   string `json:"reflectionType"`
		ResourceModuleID *uint  `json:"resourceModuleId"`
		WeeklyTaskID     *uint  `json:"weeklyTaskId"`
		TaskItemID       *uint  `json:"taskItemId"`
		Summary          string `json:"summary"`
		Challenges       string `json:"challenges"`
		Connections      string `json:"connections"`
		NextSteps        string `json:"nextSteps"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		util.BadRequest(ctx, "invalid request")
		return
	}
	entry, replayed, err := c.service.CreateEntry(user.UserID, service.EntryInput{
		ReflectionType:   body.ReflectionType,
		ResourceModuleID: body.ResourceModuleID,
		WeeklyTaskID:     body.WeeklyTaskID,
		TaskItemID:       body.TaskItemID,
		Summary:          body.Summary,
		Challenges:       body.Challenges,
		Connections:      body.Connections,
		NextSteps:        body.NextSteps,
		IdempotencyKey:   ctx.GetHeader("Idempotency-Key"),
	})
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	util.Success(ctx, gin.H{"entry": entry, "replayed": replayed})
}

func (c *ReflectionController) ListEntries(ctx *gin.Context) {
	user, ok := c.requireStudent(ctx)
	if !ok {
		return
	}
	moduleID, _ := strconv.ParseUint(ctx.Query("moduleId"), 10, 64)
	taskID, _ := strconv.ParseUint(ctx.Query("taskItemId"), 10, 64)
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	cursorTime, ok := optionalTime(ctx, "cursorTime")
	if !ok {
		return
	}
	fromTime, ok := optionalTime(ctx, "from")
	if !ok {
		return
	}
	toTime, ok := optionalTime(ctx, "to")
	if !ok {
		return
	}
	reflectionType := strings.TrimSpace(ctx.Query("type"))
	switch reflectionType {
	case "", model.ReflectionTypeTask, model.ReflectionTypeModule, model.ReflectionTypePostClass:
	default:
		util.BadRequest(ctx, "reflection type is not accepted")
		return
	}
	rows, err := c.service.ListEntries(user.UserID, repository.HistoryFilter{
		ModuleID:       uint(moduleID),
		TaskItemID:     uint(taskID),
		ReflectionType: reflectionType,
		IncludeHidden:  ctx.Query("includeHidden") == "1",
		CursorTime:     cursorTime,
		CursorID:       ctx.Query("cursorId"),
		From:           fromTime,
		To:             toTime,
		Limit:          limit,
	})
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	var nextCursor gin.H
	if len(rows) == limit {
		last := rows[len(rows)-1]
		nextCursor = gin.H{"cursorTime": last.CreatedAt.Format(time.RFC3339Nano), "cursorId": last.ID}
	}
	util.Success(ctx, gin.H{"items": rows, "nextCursor": nextCursor})
}

func optionalTime(ctx *gin.Context, key string) (*time.Time, bool) {
	raw := strings.TrimSpace(ctx.Query(key))
	if raw == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, raw)
	}
	if err != nil {
		util.BadRequest(ctx, "invalid "+key)
		return nil, false
	}
	return &parsed, true
}

func (c *ReflectionController) TaskOptions(ctx *gin.Context) {
	if _, ok := c.requireStudent(ctx); !ok {
		return
	}
	moduleID, _ := strconv.ParseUint(ctx.Query("moduleId"), 10, 64)
	rows, err := c.service.ListOpenedTasks(uint(moduleID), time.Now())
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	util.Success(ctx, gin.H{"items": rows})
}

func (c *ReflectionController) ListRevisions(ctx *gin.Context) {
	user, ok := c.requireStudent(ctx)
	if !ok {
		return
	}
	entry, rows, err := c.service.ListRevisions(user.UserID, ctx.Param("id"))
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	util.Success(ctx, gin.H{"entry": entry, "revisions": rows})
}

func (c *ReflectionController) requireStaff(ctx *gin.Context) (*util.Claims, bool) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return nil, false
	}
	if user.Role != model.Teacher && user.Role != model.Admin {
		util.Forbidden(ctx)
		return nil, false
	}
	return user, true
}

func staffFilterFromQuery(ctx *gin.Context) (repository.StaffHistoryFilter, bool) {
	moduleID, _ := strconv.ParseUint(ctx.Query("moduleId"), 10, 64)
	taskID, _ := strconv.ParseUint(ctx.Query("taskItemId"), 10, 64)
	studentID, _ := strconv.ParseUint(ctx.Query("studentId"), 10, 64)
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	cursorTime, ok := optionalTime(ctx, "cursorTime")
	if !ok {
		return repository.StaffHistoryFilter{}, false
	}
	fromTime, ok := optionalTime(ctx, "from")
	if !ok {
		return repository.StaffHistoryFilter{}, false
	}
	toTime, ok := optionalTime(ctx, "to")
	if !ok {
		return repository.StaffHistoryFilter{}, false
	}
	return repository.StaffHistoryFilter{
		StudentID:  uint(studentID),
		Name:       ctx.Query("name"),
		ModuleID:   uint(moduleID),
		TaskItemID: uint(taskID),
		CursorTime: cursorTime,
		CursorID:   ctx.Query("cursorId"),
		From:       fromTime,
		To:         toTime,
		Limit:      limit,
	}, true
}

func (c *ReflectionController) StaffTaskOptions(ctx *gin.Context) {
	user, ok := c.requireStaff(ctx)
	if !ok {
		return
	}
	moduleID, _ := strconv.ParseUint(ctx.Query("moduleId"), 10, 64)
	rows, err := c.service.ListStaffTaskOptions(user.Role, user.UserID, uint(moduleID), time.Now())
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	util.Success(ctx, gin.H{"items": rows})
}

func (c *ReflectionController) ListStaffEntries(ctx *gin.Context) {
	user, ok := c.requireStaff(ctx)
	if !ok {
		return
	}
	filter, ok := staffFilterFromQuery(ctx)
	if !ok {
		return
	}
	rows, err := c.service.ListStaffReflections(user.Role, user.UserID, filter)
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var nextCursor gin.H
	if len(rows) == limit {
		last := rows[len(rows)-1]
		nextCursor = gin.H{"cursorTime": last.CreatedAt.Format(time.RFC3339Nano), "cursorId": last.ID}
	}
	util.Success(ctx, gin.H{"items": rows, "nextCursor": nextCursor})
}

func (c *ReflectionController) GetStaffEntry(ctx *gin.Context) {
	user, ok := c.requireStaff(ctx)
	if !ok {
		return
	}
	row, err := c.service.GetStaffReflection(user.Role, user.UserID, ctx.Param("id"))
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	util.Success(ctx, gin.H{"entry": row})
}

func (c *ReflectionController) ListStaffRevisions(ctx *gin.Context) {
	user, ok := c.requireStaff(ctx)
	if !ok {
		return
	}
	entry, rows, err := c.service.ListStaffRevisions(user.Role, user.UserID, ctx.Param("id"))
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	util.Success(ctx, gin.H{"entry": entry, "revisions": rows})
}

func (c *ReflectionController) LatestEntry(ctx *gin.Context) {
	user, ok := c.requireStudent(ctx)
	if !ok {
		return
	}
	entry, err := c.service.LatestEntry(user.UserID)
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	util.Success(ctx, gin.H{"entry": entry})
}

func (c *ReflectionController) UpdateEntry(ctx *gin.Context) {
	user, ok := c.requireStudent(ctx)
	if !ok {
		return
	}
	var body struct {
		Version     int    `json:"version"`
		ConfirmEdit bool   `json:"confirmEdit"`
		Summary     string `json:"summary"`
		Challenges  string `json:"challenges"`
		Connections string `json:"connections"`
		NextSteps   string `json:"nextSteps"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		util.BadRequest(ctx, "invalid request")
		return
	}
	entry, err := c.service.UpdateEntry(user.UserID, ctx.Param("id"), service.EntryEdit{
		Version:     body.Version,
		ConfirmEdit: body.ConfirmEdit,
		Summary:     body.Summary,
		Challenges:  body.Challenges,
		Connections: body.Connections,
		NextSteps:   body.NextSteps,
	})
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	util.Success(ctx, gin.H{"entry": entry})
}

func (c *ReflectionController) ConfirmPlan(ctx *gin.Context) {
	user, ok := c.requireStudent(ctx)
	if !ok {
		return
	}
	var body struct {
		Confirmed        bool   `json:"confirmed"`
		Title            string `json:"title"`
		TargetDate       string `json:"targetDate"`
		ResourceModuleID uint   `json:"resourceModuleId"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		util.BadRequest(ctx, "invalid request")
		return
	}
	when, err := time.Parse(time.RFC3339, body.TargetDate)
	if body.TargetDate != "" && err != nil {
		util.BadRequest(ctx, "invalid target date")
		return
	}
	result, err := c.service.ConfirmPlan(user.UserID, ctx.Param("id"), service.PlanInput{
		Confirmed:        body.Confirmed,
		Title:            body.Title,
		TargetDate:       when,
		ResourceModuleID: body.ResourceModuleID,
	})
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	util.Success(ctx, gin.H{"goal": result.Goal, "replayed": result.Replayed})
}

func (c *ReflectionController) HideEntry(ctx *gin.Context) {
	c.changeVisibility(ctx, true)
}

func (c *ReflectionController) RestoreEntry(ctx *gin.Context) {
	c.changeVisibility(ctx, false)
}

func (c *ReflectionController) changeVisibility(ctx *gin.Context, hide bool) {
	user, ok := c.requireStudent(ctx)
	if !ok {
		return
	}
	var body struct {
		Version int `json:"version"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		util.BadRequest(ctx, "invalid request")
		return
	}
	var err error
	if hide {
		err = c.service.HideEntry(user.UserID, ctx.Param("id"), body.Version)
	} else {
		err = c.service.RestoreEntry(user.UserID, ctx.Param("id"), body.Version)
	}
	if err != nil {
		c.writeEntryError(ctx, err)
		return
	}
	util.Success(ctx, gin.H{"hidden": hide})
}
