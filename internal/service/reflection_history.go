package service

import (
	"errors"
	"strings"
	"time"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/util"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrReflectionVersionConflict     = errors.New("reflection version conflict")
	ErrReflectionNotFound            = errors.New("reflection entry not found")
	ErrReflectionEmpty               = errors.New("at least one reflection field is required")
	ErrReflectionBinding             = errors.New("module and task binding do not match")
	ErrReflectionIdempotency         = errors.New("idempotency key must not be blank")
	ErrReflectionEditUnconfirmed     = errors.New("edit must be explicitly confirmed")
	ErrPlanNotConfirmed              = errors.New("learning goal must be explicitly confirmed")
	ErrNextStepsEmpty                = errors.New("next steps are empty")
	ErrReflectionModuleRequired      = errors.New("resource module is required")
	ErrReflectionLegacyReadOnly      = errors.New("legacy snapshot cannot be edited")
	ErrReflectionType                = errors.New("reflection type is not accepted")
	ErrReflectionIdempotencyConflict = errors.New("idempotency key was reused for a different reflection")
)

type EntryInput struct {
	ReflectionType   string
	ResourceModuleID *uint
	WeeklyTaskID     *uint
	TaskItemID       *uint
	Summary          string
	Challenges       string
	Connections      string
	NextSteps        string
	IdempotencyKey   string
}

type EntryEdit struct {
	Version     int
	ConfirmEdit bool
	Summary     string
	Challenges  string
	Connections string
	NextSteps   string
}

type PlanInput struct {
	Confirmed        bool
	Title            string
	TargetDate       time.Time
	ResourceModuleID uint
}

type PlanResult struct {
	Goal     *model.Goal
	Replayed bool
}

func (s *ReflectionService) CreateEntry(userID uint, in EntryInput) (*model.ReflectionEntry, bool, error) {
	if userID == 0 {
		return nil, false, util.ErrPermissionDenied
	}
	summary, challenges, connections, nextSteps, err := normalizeFields(in.Summary, in.Challenges, in.Connections, in.NextSteps)
	if err != nil {
		return nil, false, err
	}
	key := strings.TrimSpace(in.IdempotencyKey)
	if key == "" {
		return nil, false, ErrReflectionIdempotency
	}
	idem := &key
	reflectionType, err := acceptedNewReflectionType(in.ReflectionType)
	if err != nil {
		return nil, false, err
	}
	if err := bindingShape(reflectionType, in.ResourceModuleID, in.WeeklyTaskID, in.TaskItemID); err != nil {
		return nil, false, err
	}

	now := time.Now().Truncate(time.Millisecond)
	entry := &model.ReflectionEntry{
		ID:             repository.NewEntryID(),
		UserID:         userID,
		ReflectionType: reflectionType,
		Summary:        summary,
		Challenges:     challenges,
		Connections:    connections,
		NextSteps:      nextSteps,
		Version:        1,
		Source:         model.ReflectionSourceStudent,
		IdempotencyKey: idem,
		CreatedAt:      now,
		UpdatedAt:      now,
		HistoryKind:    model.HistoryKindAppend,
	}

	replayed := false
	err = s.db.Transaction(func(tx *gorm.DB) error {
		moduleID, weeklyID, taskID, err := resolveBinding(tx, reflectionType, in.ResourceModuleID, in.WeeklyTaskID, in.TaskItemID, now)
		if err != nil {
			return err
		}
		entry.ResourceModuleID = moduleID
		entry.WeeklyTaskID = weeklyID
		entry.TaskItemID = taskID
		if err := tx.Create(entry).Error; err != nil {
			if !isDuplicate(err) || idem == nil {
				return err
			}
			var existing model.ReflectionEntry
			if err := tx.Where("user_id = ? AND idempotency_key = ?", userID, *idem).First(&existing).Error; err != nil {
				return err
			}
			if !idempotentPayloadMatches(existing, reflectionType, moduleID, weeklyID, taskID, summary, challenges, connections, nextSteps) {
				return ErrReflectionIdempotencyConflict
			}
			*entry = existing
			replayed = true
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if entry.Source == model.ReflectionSourceLegacy {
		entry.HistoryKind = model.HistoryKindLegacy
	} else {
		entry.HistoryKind = model.HistoryKindAppend
	}
	return entry, replayed, nil
}

func (s *ReflectionService) ListEntries(userID uint, f repository.HistoryFilter) ([]model.ReflectionEntry, error) {
	if userID == 0 {
		return nil, util.ErrPermissionDenied
	}
	f.UserID = userID
	return s.entries.ListHistory(f)
}

func (s *ReflectionService) LatestEntry(userID uint) (*model.ReflectionEntry, error) {
	rows, err := s.ListEntries(userID, repository.HistoryFilter{Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (s *ReflectionService) UpdateEntry(userID uint, id string, edit EntryEdit) (*model.ReflectionEntry, error) {
	if !edit.ConfirmEdit {
		return nil, ErrReflectionEditUnconfirmed
	}
	summary, challenges, connections, nextSteps, err := normalizeFields(edit.Summary, edit.Challenges, edit.Connections, edit.NextSteps)
	if err != nil {
		return nil, err
	}
	var updated model.ReflectionEntry
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var current model.ReflectionEntry
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", id, userID).
			First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrReflectionNotFound
		}
		if err != nil {
			return err
		}
		if current.DeletedAt != nil || current.Version != edit.Version {
			return ErrReflectionVersionConflict
		}
		if err := rejectLegacyEdit(current.Source); err != nil {
			return err
		}
		rev := model.ReflectionEntryRevision{
			EntryID:           current.ID,
			VersionSuperseded: current.Version,
			Summary:           current.Summary,
			Challenges:        current.Challenges,
			Connections:       current.Connections,
			NextSteps:         current.NextSteps,
			EditorUserID:      userID,
			CreatedAt:         time.Now().Truncate(time.Millisecond),
		}
		if err := tx.Create(&rev).Error; err != nil {
			return err
		}
		now := time.Now().Truncate(time.Millisecond)
		res := tx.Exec(reflectionEntryTextUpdate,
			summary, challenges, connections, nextSteps, now, id, userID, edit.Version)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrReflectionVersionConflict
		}
		return tx.Where("id = ? AND user_id = ?", id, userID).First(&updated).Error
	})
	if err != nil {
		return nil, err
	}
	updated.HistoryKind = historyKind(updated.Source)
	return &updated, nil
}

func (s *ReflectionService) HideEntry(userID uint, id string, version int) error {
	return s.setHidden(userID, id, version, true)
}

func (s *ReflectionService) RestoreEntry(userID uint, id string, version int) error {
	return s.setHidden(userID, id, version, false)
}

func (s *ReflectionService) setHidden(userID uint, id string, version int, hide bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var current model.ReflectionEntry
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", id, userID).
			First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrReflectionNotFound
		}
		if err != nil {
			return err
		}
		if current.Version != version {
			return ErrReflectionVersionConflict
		}
		now := time.Now().Truncate(time.Millisecond)
		var res *gorm.DB
		if hide {
			if current.DeletedAt != nil {
				return nil
			}
			res = tx.Exec(`UPDATE reflection_entries SET deleted_at = ?, version = version + 1, updated_at = ? WHERE id = ? AND user_id = ? AND version = ? AND deleted_at IS NULL`,
				now, now, id, userID, version)
		} else {
			if current.DeletedAt == nil {
				return nil
			}
			res = tx.Exec(`UPDATE reflection_entries SET deleted_at = NULL, version = version + 1, updated_at = ? WHERE id = ? AND user_id = ? AND version = ? AND deleted_at IS NOT NULL`,
				now, id, userID, version)
		}
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrReflectionVersionConflict
		}
		return nil
	})
}

func (s *ReflectionService) ConfirmPlan(userID uint, entryID string, in PlanInput) (*PlanResult, error) {
	if !in.Confirmed {
		return nil, ErrPlanNotConfirmed
	}
	title := strings.TrimSpace(in.Title)
	if title == "" || len(title) > 255 || in.TargetDate.IsZero() {
		return nil, ErrPlanNotConfirmed
	}
	var result PlanResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var entry model.ReflectionEntry
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ? AND deleted_at IS NULL", entryID, userID).
			First(&entry).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrReflectionNotFound
		}
		if err != nil {
			return err
		}
		var link model.GoalReflectionLink
		err = tx.Where("entry_id = ?", entry.ID).First(&link).Error
		if err == nil {
			var goal model.Goal
			if err := tx.First(&goal, link.GoalID).Error; err != nil {
				return err
			}
			if goal.UserID != entry.UserID || link.UserID != entry.UserID {
				return util.ErrPermissionDenied
			}
			result.Goal = &goal
			result.Replayed = true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if strings.TrimSpace(entry.NextSteps) == "" {
			return ErrNextStepsEmpty
		}
		moduleID := uint(0)
		if entry.ResourceModuleID != nil {
			moduleID = *entry.ResourceModuleID
		}
		if moduleID == 0 {
			moduleID = in.ResourceModuleID
		}
		if moduleID == 0 {
			return ErrReflectionModuleRequired
		}
		var moduleName string
		if err := tx.Raw(`SELECT name FROM c_programming_resources WHERE id = ? AND deleted_at IS NULL`, moduleID).Scan(&moduleName).Error; err != nil {
			return err
		}
		if moduleName == "" {
			return ErrReflectionBinding
		}
		goal := &model.Goal{
			UserID:             entry.UserID,
			Title:              title,
			Description:        entry.NextSteps,
			Status:             model.GoalPending,
			Current:            0,
			Target:             100,
			Progress:           0,
			TargetDate:         in.TargetDate,
			GoalType:           model.GoalTypeShortTerm,
			ResourceModuleID:   moduleID,
			ResourceModuleName: moduleName,
		}
		if err := tx.Create(goal).Error; err != nil {
			return err
		}
		link = model.GoalReflectionLink{
			EntryID:           entry.ID,
			GoalID:            goal.ID,
			UserID:            entry.UserID,
			NextStepsSnapshot: entry.NextSteps,
			CreatedAt:         time.Now().Truncate(time.Millisecond),
		}
		if err := tx.Create(&link).Error; err != nil {
			return err
		}
		var stored model.ReflectionEntry
		if err := tx.Where("id = ?", entry.ID).First(&stored).Error; err != nil {
			return err
		}
		if stored.NextSteps != entry.NextSteps || stored.Summary != entry.Summary {
			return errors.New("reflection text changed during goal confirmation")
		}
		result.Goal = goal
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ReflectionService) BackfillLegacy() (int64, error) {
	return s.entries.BackfillLegacy()
}

func (s *ReflectionService) VerifyLegacy() (repository.LegacyVerify, error) {
	return s.entries.VerifyLegacy()
}

func (s *ReflectionService) ListRevisions(userID uint, id string) (*model.ReflectionEntry, []model.ReflectionEntryRevision, error) {
	if userID == 0 || strings.TrimSpace(id) == "" {
		return nil, nil, util.ErrPermissionDenied
	}
	var entry model.ReflectionEntry
	err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrReflectionNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	entry.HistoryKind = historyKind(entry.Source)
	rows, err := s.entries.ListRevisions(id)
	return &entry, rows, err
}

func (s *ReflectionService) ListOpenedTasks(moduleID uint, now time.Time) ([]repository.OpenedTaskOption, error) {
	return s.entries.ListOpenedTasks(moduleID, now.Format("2006-01-02"), 0, false)
}

func (s *ReflectionService) ListStaffTaskOptions(role model.UserRole, actorID, moduleID uint, now time.Time) ([]repository.OpenedTaskOption, error) {
	if actorID == 0 || (role != model.Teacher && role != model.Admin) {
		return nil, util.ErrPermissionDenied
	}
	restrict := role == model.Teacher
	return s.entries.ListOpenedTasks(moduleID, now.Format("2006-01-02"), actorID, restrict)
}

func staffFilter(role model.UserRole, actorID uint, f repository.StaffHistoryFilter) (repository.StaffHistoryFilter, error) {
	if actorID == 0 || (role != model.Teacher && role != model.Admin) {
		return f, util.ErrPermissionDenied
	}
	f.ActorID = actorID
	f.AdminScope = role == model.Admin
	return f, nil
}

func (s *ReflectionService) ListStaffReflections(role model.UserRole, actorID uint, f repository.StaffHistoryFilter) ([]repository.StaffReflectionRow, error) {
	scoped, err := staffFilter(role, actorID, f)
	if err != nil {
		return nil, err
	}
	return s.entries.ListStaffReflections(scoped)
}

func (s *ReflectionService) GetStaffReflection(role model.UserRole, actorID uint, entryID string) (*repository.StaffReflectionRow, error) {
	scoped, err := staffFilter(role, actorID, repository.StaffHistoryFilter{})
	if err != nil {
		return nil, err
	}
	row, err := s.entries.GetStaffReflection(scoped, entryID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrReflectionNotFound
	}
	return row, err
}

func (s *ReflectionService) ListStaffRevisions(role model.UserRole, actorID uint, entryID string) (*repository.StaffReflectionRow, []model.ReflectionEntryRevision, error) {
	row, err := s.GetStaffReflection(role, actorID, entryID)
	if err != nil {
		return nil, nil, err
	}
	revs, err := s.entries.ListRevisions(entryID)
	return row, revs, err
}

func normalizeFields(summary, challenges, connections, nextSteps string) (string, string, string, string, error) {
	summary = strings.TrimSpace(summary)
	challenges = strings.TrimSpace(challenges)
	connections = strings.TrimSpace(connections)
	nextSteps = strings.TrimSpace(nextSteps)
	if summary == "" && challenges == "" && connections == "" && nextSteps == "" {
		return "", "", "", "", ErrReflectionEmpty
	}
	return summary, challenges, connections, nextSteps, nil
}

// weekHasOpened reports whether a weekly task has started by the given instant.
// Future weeks stay closed. Past weeks remain available.
func weekHasOpened(weekStart, now time.Time) bool {
	if weekStart.IsZero() {
		return false
	}
	sy, sm, sd := weekStart.Date()
	ny, nm, nd := now.Date()
	start := time.Date(sy, sm, sd, 0, 0, 0, 0, time.UTC)
	today := time.Date(ny, nm, nd, 0, 0, 0, 0, time.UTC)
	return !start.After(today)
}

const reflectionEntryTextUpdate = `
UPDATE reflection_entries
SET summary = ?, challenges = ?, connections = ?, next_steps = ?, version = version + 1, updated_at = ?
WHERE id = ? AND user_id = ? AND version = ? AND deleted_at IS NULL`

func rejectLegacyEdit(source string) error {
	if source == model.ReflectionSourceLegacy {
		return ErrReflectionLegacyReadOnly
	}
	return nil
}

func acceptedNewReflectionType(raw string) (string, error) {
	switch strings.TrimSpace(raw) {
	case model.ReflectionTypeTask, model.ReflectionTypeModule:
		return strings.TrimSpace(raw), nil
	default:
		return "", ErrReflectionType
	}
}

func bindingShape(reflectionType string, moduleID, weeklyID, taskID *uint) error {
	switch reflectionType {
	case model.ReflectionTypeTask:
		if moduleID == nil || *moduleID == 0 || taskID == nil || *taskID == 0 {
			return ErrReflectionBinding
		}
	case model.ReflectionTypeModule:
		if moduleID == nil || *moduleID == 0 {
			return ErrReflectionModuleRequired
		}
		if idPresent(weeklyID) || idPresent(taskID) {
			return ErrReflectionBinding
		}
	default:
		return ErrReflectionType
	}
	return nil
}

func idPresent(id *uint) bool {
	return id != nil && *id != 0
}

func sameOptionalID(left, right *uint) bool {
	var l, r uint
	if left != nil {
		l = *left
	}
	if right != nil {
		r = *right
	}
	return l == r
}

func idempotentPayloadMatches(existing model.ReflectionEntry, reflectionType string, moduleID, weeklyID, taskID *uint, summary, challenges, connections, nextSteps string) bool {
	return existing.ReflectionType == reflectionType &&
		sameOptionalID(existing.ResourceModuleID, moduleID) &&
		sameOptionalID(existing.WeeklyTaskID, weeklyID) &&
		sameOptionalID(existing.TaskItemID, taskID) &&
		existing.Summary == summary &&
		existing.Challenges == challenges &&
		existing.Connections == connections &&
		existing.NextSteps == nextSteps
}

func resolveBinding(tx *gorm.DB, reflectionType string, moduleID, weeklyID, taskID *uint, now time.Time) (*uint, *uint, *uint, error) {
	if err := bindingShape(reflectionType, moduleID, weeklyID, taskID); err != nil {
		return nil, nil, nil, err
	}
	if reflectionType == model.ReflectionTypeModule {
		var count int64
		if err := tx.Raw(`SELECT COUNT(*) FROM c_programming_resources WHERE id = ? AND deleted_at IS NULL AND enabled = 1`, *moduleID).Scan(&count).Error; err != nil {
			return nil, nil, nil, err
		}
		if count != 1 {
			return nil, nil, nil, ErrReflectionBinding
		}
		module := *moduleID
		return &module, nil, nil, nil
	}
	if taskID == nil || *taskID == 0 {
		return nil, nil, nil, ErrReflectionBinding
	}
	var row struct {
		WeeklyTaskID     uint
		ResourceModuleID uint
		WeekStart        time.Time
		Enabled          int
	}
	err := tx.Raw(`
SELECT ti.weekly_task_id AS weekly_task_id,
       tw.resource_module_id AS resource_module_id,
       tw.week_start_date AS week_start,
       m.enabled AS enabled
FROM task_items ti
JOIN teacher_weekly_tasks tw ON tw.id = ti.weekly_task_id AND tw.deleted_at IS NULL
JOIN c_programming_resources m ON m.id = tw.resource_module_id AND m.deleted_at IS NULL
WHERE ti.id = ? AND ti.deleted_at IS NULL`, *taskID).Scan(&row).Error
	if err != nil {
		return nil, nil, nil, err
	}
	if row.WeeklyTaskID == 0 || row.ResourceModuleID == 0 || row.Enabled != 1 || !weekHasOpened(row.WeekStart, now) {
		return nil, nil, nil, ErrReflectionBinding
	}
	if *moduleID != row.ResourceModuleID {
		return nil, nil, nil, ErrReflectionBinding
	}
	if weeklyID != nil && *weeklyID != 0 && *weeklyID != row.WeeklyTaskID {
		return nil, nil, nil, ErrReflectionBinding
	}
	module := row.ResourceModuleID
	weekly := row.WeeklyTaskID
	task := *taskID
	return &module, &weekly, &task, nil
}

func isDuplicate(err error) bool {
	var mysqlErr *mysqldriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func historyKind(source string) string {
	if source == model.ReflectionSourceLegacy {
		return model.HistoryKindLegacy
	}
	return model.HistoryKindAppend
}
