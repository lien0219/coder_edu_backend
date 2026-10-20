package repository

import (
	"strings"
	"time"

	"coder_edu_backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReflectionEntryRepository struct {
	DB *gorm.DB
}

func NewReflectionEntryRepository(db *gorm.DB) *ReflectionEntryRepository {
	return &ReflectionEntryRepository{DB: db}
}

type HistoryFilter struct {
	UserID         uint
	ModuleID       uint
	TaskItemID     uint
	ReflectionType string
	IncludeHidden  bool
	CursorTime     *time.Time
	CursorID       string
	From           *time.Time
	To             *time.Time
	Limit          int
}

type StaffHistoryFilter struct {
	ActorID    uint
	AdminScope bool
	StudentID  uint
	Name       string
	ModuleID   uint
	TaskItemID uint
	CursorTime *time.Time
	CursorID   string
	From       *time.Time
	To         *time.Time
	Limit      int
}

type StaffReflectionRow struct {
	model.ReflectionEntry
	StudentName string `json:"studentName"`
	ModuleName  string `json:"moduleName"`
	TaskTitle   string `json:"taskTitle"`
}

type OpenedTaskOption struct {
	ModuleID     uint      `json:"moduleId"`
	ModuleName   string    `json:"moduleName"`
	WeeklyTaskID uint      `json:"weeklyTaskId"`
	WeekStart    time.Time `json:"weekStartDate"`
	WeekEnd      time.Time `json:"weekEndDate"`
	TaskItemID   uint      `json:"taskItemId"`
	TaskTitle    string    `json:"taskTitle"`
}

func (r *ReflectionEntryRepository) ListHistory(f HistoryFilter) ([]model.ReflectionEntry, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	includeHidden := 0
	if f.IncludeHidden {
		includeHidden = 1
	}
	hasCursor := 0
	var cursor time.Time
	if f.CursorTime != nil {
		hasCursor = 1
		cursor = *f.CursorTime
	}
	hasFrom, fromAt := boundTime(f.From)
	hasTo, toAt := boundTime(f.To)
	// Both arms use one collation. Production reflections strings are
	// utf8mb4_0900_ai_ci and reflection_entries strings are utf8mb4_unicode_ci.
	// UNION rejects that mix. COLLATE changes only this result, not stored rows.
	query := `
SELECT id, user_id, resource_module_id, weekly_task_id, task_item_id,
       reflection_type, summary, challenges, connections, next_steps,
       version, source, legacy_reflection_id, created_at, updated_at, deleted_at,
       history_kind
FROM (
  SELECT e.id COLLATE utf8mb4_unicode_ci AS id, e.user_id, e.resource_module_id, e.weekly_task_id, e.task_item_id,
         e.reflection_type COLLATE utf8mb4_unicode_ci AS reflection_type,
         e.summary COLLATE utf8mb4_unicode_ci AS summary,
         e.challenges COLLATE utf8mb4_unicode_ci AS challenges,
         e.connections COLLATE utf8mb4_unicode_ci AS connections,
         e.next_steps COLLATE utf8mb4_unicode_ci AS next_steps,
         e.version, e.source COLLATE utf8mb4_unicode_ci AS source,
         e.legacy_reflection_id COLLATE utf8mb4_unicode_ci AS legacy_reflection_id,
         e.created_at, e.updated_at, e.deleted_at,
         (IF(e.source = 'legacy', 'legacy_snapshot', 'append')) COLLATE utf8mb4_unicode_ci AS history_kind
  FROM reflection_entries e
  WHERE e.user_id = ?
    AND (? = 1 OR e.deleted_at IS NULL)
    AND (? = 0 OR e.resource_module_id = ?)
    AND (? = 0 OR e.task_item_id = ?)
    AND (? = '' OR e.reflection_type = ?)
  UNION ALL
  SELECT r.id COLLATE utf8mb4_unicode_ci, r.user_id, NULL, NULL, NULL,
         'post_class' COLLATE utf8mb4_unicode_ci,
         IFNULL(r.summary, '') COLLATE utf8mb4_unicode_ci,
         IFNULL(r.challenges, '') COLLATE utf8mb4_unicode_ci,
         IFNULL(r.connections, '') COLLATE utf8mb4_unicode_ci,
         IFNULL(r.next_steps, '') COLLATE utf8mb4_unicode_ci,
         1, 'legacy' COLLATE utf8mb4_unicode_ci, r.id COLLATE utf8mb4_unicode_ci,
         r.created_at, r.updated_at, r.deleted_at,
         'legacy_snapshot' COLLATE utf8mb4_unicode_ci
  FROM reflections r
  WHERE r.user_id = ?
    AND r.deleted_at IS NULL
    AND ? = 0
    AND ? = 0
    AND (? = '' OR ? = 'post_class')
    AND NOT EXISTS (
      SELECT 1 FROM reflection_entries e2 WHERE e2.legacy_reflection_id = r.id
    )
) h
WHERE (? = 0 OR created_at < ? OR (created_at = ? AND id < CONVERT(? USING utf8mb4) COLLATE utf8mb4_unicode_ci))
  AND (? = 0 OR created_at >= ?)
  AND (? = 0 OR created_at <= ?)
ORDER BY created_at DESC, id DESC
LIMIT ?`
	var rows []model.ReflectionEntry
	err := r.DB.Raw(query,
		f.UserID,
		includeHidden,
		f.ModuleID, f.ModuleID,
		f.TaskItemID, f.TaskItemID,
		f.ReflectionType, f.ReflectionType,
		f.UserID,
		f.ModuleID,
		f.TaskItemID,
		f.ReflectionType, f.ReflectionType,
		hasCursor, cursor, cursor, f.CursorID,
		hasFrom, fromAt,
		hasTo, toAt,
		f.Limit,
	).Scan(&rows).Error
	for i := range rows {
		if rows[i].HistoryKind == "" {
			if rows[i].Source == model.ReflectionSourceLegacy {
				rows[i].HistoryKind = model.HistoryKindLegacy
			} else {
				rows[i].HistoryKind = model.HistoryKindAppend
			}
		}
	}
	return rows, err
}

func (r *ReflectionEntryRepository) BackfillLegacy() (int64, error) {
	var inserted int64
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(`
INSERT INTO reflection_entries (
  id, user_id, reflection_type, summary, challenges, connections, next_steps,
  version, source, legacy_reflection_id, created_at, updated_at
)
SELECT UUID(), r.user_id, 'post_class', IFNULL(r.summary, ''), IFNULL(r.challenges, ''), IFNULL(r.connections, ''), IFNULL(r.next_steps, ''),
       1, 'legacy', r.id, r.created_at, r.updated_at
FROM reflections r
WHERE r.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM reflection_entries e WHERE e.legacy_reflection_id = r.id
  )`)
		if res.Error != nil {
			return res.Error
		}
		inserted = res.RowsAffected
		return nil
	})
	return inserted, err
}

type LegacyVerify struct {
	ReflectionRows int64
	LegacyEntries  int64
	Mismatches     int64
}

func (r *ReflectionEntryRepository) VerifyLegacy() (LegacyVerify, error) {
	var out LegacyVerify
	if err := r.DB.Raw(`SELECT COUNT(*) FROM reflections WHERE deleted_at IS NULL`).Scan(&out.ReflectionRows).Error; err != nil {
		return out, err
	}
	if err := r.DB.Raw(`SELECT COUNT(*) FROM reflection_entries WHERE source = 'legacy'`).Scan(&out.LegacyEntries).Error; err != nil {
		return out, err
	}
	err := r.DB.Raw(`
SELECT COUNT(*) FROM reflections r
JOIN reflection_entries e ON e.legacy_reflection_id = r.id
WHERE SHA2(CONCAT_WS(CHAR(10), IFNULL(r.summary,''), IFNULL(r.challenges,''), IFNULL(r.connections,''), IFNULL(r.next_steps,'')), 256)
   <> SHA2(CONCAT_WS(CHAR(10), IFNULL(e.summary,''), IFNULL(e.challenges,''), IFNULL(e.connections,''), IFNULL(e.next_steps,'')), 256)
`).Scan(&out.Mismatches).Error
	return out, err
}

func NewEntryID() string {
	return uuid.New().String()
}

func boundTime(value *time.Time) (int, time.Time) {
	if value == nil {
		return 0, time.Time{}
	}
	return 1, *value
}

func (r *ReflectionEntryRepository) ListRevisions(entryID string) ([]model.ReflectionEntryRevision, error) {
	var rows []model.ReflectionEntryRevision
	err := r.DB.Raw(`
SELECT id, entry_id, version_superseded, summary, challenges, connections, next_steps, editor_user_id, created_at
FROM reflection_entry_revisions
WHERE entry_id = ?
ORDER BY version_superseded DESC, id DESC`, entryID).Scan(&rows).Error
	return rows, err
}

func (r *ReflectionEntryRepository) ListOpenedTasks(moduleID uint, today string, teacherID uint, restrictTeacher bool) ([]OpenedTaskOption, error) {
	restrict := 0
	if restrictTeacher {
		restrict = 1
	}
	var rows []OpenedTaskOption
	err := r.DB.Raw(`
SELECT m.id AS module_id, m.name AS module_name,
       tw.id AS weekly_task_id, tw.week_start_date AS week_start, tw.week_end_date AS week_end,
       ti.id AS task_item_id, ti.title AS task_title
FROM c_programming_resources m
JOIN teacher_weekly_tasks tw ON tw.resource_module_id = m.id AND tw.deleted_at IS NULL
JOIN task_items ti ON ti.weekly_task_id = tw.id AND ti.deleted_at IS NULL
WHERE m.deleted_at IS NULL AND m.enabled = 1
  AND DATE(tw.week_start_date) <= ?
  AND (? = 0 OR m.id = ?)
  AND (? = 0 OR tw.teacher_id = ?)
ORDER BY m.`+"`order`"+`, m.id, tw.week_start_date DESC, ti.id
LIMIT 500`, today, moduleID, moduleID, restrict, teacherID).Scan(&rows).Error
	return rows, err
}

// Module summaries stay out of this query. A module row has no weekly task,
// and no class roster proves that this teacher teaches this student in that module.
const staffReflectionSelect = `
SELECT e.id, e.user_id, e.resource_module_id, e.weekly_task_id, e.task_item_id,
       e.reflection_type, e.summary, e.challenges, e.connections, e.next_steps,
       e.version, e.source, e.legacy_reflection_id, e.created_at, e.updated_at, e.deleted_at,
       u.name AS student_name, m.name AS module_name, ti.title AS task_title
FROM reflection_entries e
JOIN users u ON u.id = e.user_id
JOIN teacher_weekly_tasks tw ON tw.id = e.weekly_task_id AND tw.deleted_at IS NULL
JOIN task_items ti ON ti.id = e.task_item_id AND ti.weekly_task_id = tw.id AND ti.deleted_at IS NULL
JOIN c_programming_resources m ON m.id = e.resource_module_id AND m.deleted_at IS NULL
WHERE e.deleted_at IS NULL
  AND e.resource_module_id IS NOT NULL
  AND e.task_item_id IS NOT NULL
  AND e.weekly_task_id IS NOT NULL
  AND e.resource_module_id = tw.resource_module_id
  AND (? = 1 OR tw.teacher_id = ?)
  AND (? = 0 OR e.user_id = ?)
  AND (? = 0 OR e.resource_module_id = ?)
  AND (? = 0 OR e.task_item_id = ?)
  AND (? = '' OR u.name LIKE ?)
  AND (? = 0 OR e.created_at >= ?)
  AND (? = 0 OR e.created_at <= ?)
`

func staffScopeArgs(f StaffHistoryFilter) (int, uint, string) {
	admin := 0
	if f.AdminScope {
		admin = 1
	}
	name := strings.TrimSpace(f.Name)
	like := ""
	if name != "" {
		like = "%" + name + "%"
	}
	return admin, f.ActorID, like
}

func (r *ReflectionEntryRepository) ListStaffReflections(f StaffHistoryFilter) ([]StaffReflectionRow, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	hasCursor := 0
	var cursor time.Time
	if f.CursorTime != nil {
		hasCursor = 1
		cursor = *f.CursorTime
	}
	hasFrom, fromAt := boundTime(f.From)
	hasTo, toAt := boundTime(f.To)
	admin, actorID, like := staffScopeArgs(f)
	name := strings.TrimSpace(f.Name)
	query := staffReflectionSelect + `
  AND (? = 0 OR e.created_at < ? OR (e.created_at = ? AND e.id < ?))
ORDER BY e.created_at DESC, e.id DESC
LIMIT ?`
	var rows []StaffReflectionRow
	err := r.DB.Raw(query,
		admin, actorID,
		f.StudentID, f.StudentID,
		f.ModuleID, f.ModuleID,
		f.TaskItemID, f.TaskItemID,
		name, like,
		hasFrom, fromAt,
		hasTo, toAt,
		hasCursor, cursor, cursor, f.CursorID,
		f.Limit,
	).Scan(&rows).Error
	for i := range rows {
		rows[i].HistoryKind = historyKindFor(rows[i].Source)
	}
	return rows, err
}

func (r *ReflectionEntryRepository) GetStaffReflection(f StaffHistoryFilter, entryID string) (*StaffReflectionRow, error) {
	admin, actorID, like := staffScopeArgs(f)
	name := strings.TrimSpace(f.Name)
	query := staffReflectionSelect + ` AND e.id = ? LIMIT 1`
	var row StaffReflectionRow
	err := r.DB.Raw(query,
		admin, actorID,
		f.StudentID, f.StudentID,
		f.ModuleID, f.ModuleID,
		f.TaskItemID, f.TaskItemID,
		name, like,
		0, time.Time{},
		0, time.Time{},
		entryID,
	).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	row.HistoryKind = historyKindFor(row.Source)
	return &row, nil
}

func historyKindFor(source string) string {
	if source == model.ReflectionSourceLegacy {
		return model.HistoryKindLegacy
	}
	return model.HistoryKindAppend
}
