package model

import "time"

const (
	ReflectionSourceStudent = "student"
	ReflectionSourceLegacy  = "legacy"
	ReflectionTypePostClass = "post_class"
	ReflectionTypeTask      = "task"
	ReflectionTypeModule    = "module"
	HistoryKindAppend       = "append"
	HistoryKindLegacy       = "legacy_snapshot"
)

// ReflectionEntry is an append-only reflection. Legacy rows are snapshots of
// reflections and are not a complete reconstruction of overwritten history.
type ReflectionEntry struct {
	ID                 string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID             uint       `gorm:"index;type:bigint unsigned;not null" json:"userId"`
	ResourceModuleID   *uint      `gorm:"type:bigint unsigned" json:"resourceModuleId,omitempty"`
	WeeklyTaskID       *uint      `gorm:"type:bigint unsigned" json:"weeklyTaskId,omitempty"`
	TaskItemID         *uint      `gorm:"type:bigint unsigned" json:"taskItemId,omitempty"`
	ReflectionType     string     `gorm:"size:32;not null;default:post_class" json:"reflectionType"`
	Summary            string     `gorm:"type:text" json:"summary"`
	Challenges         string     `gorm:"type:text" json:"challenges"`
	Connections        string     `gorm:"type:text" json:"connections"`
	NextSteps          string     `gorm:"type:text" json:"nextSteps"`
	Version            int        `gorm:"not null;default:1" json:"version"`
	Source             string     `gorm:"type:enum('student','legacy');not null" json:"source"`
	LegacyReflectionID *string    `gorm:"type:varchar(36)" json:"legacyReflectionId,omitempty"`
	IdempotencyKey     *string    `gorm:"type:varchar(64)" json:"-"`
	CreatedAt          time.Time  `gorm:"type:datetime(3);autoCreateTime:false" json:"createdAt"`
	UpdatedAt          time.Time  `gorm:"type:datetime(3);autoUpdateTime:false" json:"updatedAt"`
	DeletedAt          *time.Time `gorm:"type:datetime(3)" json:"deletedAt,omitempty"`
	HistoryKind        string     `gorm:"-" json:"historyKind"`
}

func (ReflectionEntry) TableName() string {
	return "reflection_entries"
}

type ReflectionEntryRevision struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	EntryID           string    `gorm:"type:varchar(36);not null;index" json:"entryId"`
	VersionSuperseded int       `gorm:"not null" json:"versionSuperseded"`
	Summary           string    `gorm:"type:text" json:"summary"`
	Challenges        string    `gorm:"type:text" json:"challenges"`
	Connections       string    `gorm:"type:text" json:"connections"`
	NextSteps         string    `gorm:"type:text" json:"nextSteps"`
	EditorUserID      uint      `gorm:"type:bigint unsigned;not null" json:"editorUserId"`
	CreatedAt         time.Time `gorm:"type:datetime(3);autoCreateTime:false" json:"createdAt"`
}

func (ReflectionEntryRevision) TableName() string {
	return "reflection_entry_revisions"
}

type GoalReflectionLink struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	EntryID           string    `gorm:"type:varchar(36);not null;uniqueIndex" json:"entryId"`
	GoalID            uint      `gorm:"type:bigint unsigned;not null" json:"goalId"`
	UserID            uint      `gorm:"type:bigint unsigned;not null" json:"userId"`
	NextStepsSnapshot string    `gorm:"type:text" json:"nextStepsSnapshot"`
	CreatedAt         time.Time `gorm:"type:datetime(3);autoCreateTime:false" json:"createdAt"`
}

func (GoalReflectionLink) TableName() string {
	return "goal_reflection_links"
}
