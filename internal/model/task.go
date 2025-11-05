package model

import (
	"time"

	"gorm.io/gorm"
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskProgress  TaskStatus = "in_progress"
	TaskCompleted TaskStatus = "completed"
)

// Weekday 表示星期几
type Weekday string

const (
	Monday    Weekday = "monday"
	Tuesday   Weekday = "tuesday"
	Wednesday Weekday = "wednesday"
	Thursday  Weekday = "thursday"
	Friday    Weekday = "friday"
	Saturday  Weekday = "saturday"
	Sunday    Weekday = "sunday"
)

// TaskItemType 表示任务项类型
type TaskItemType string

const (
	TaskItemVideo    TaskItemType = "video"
	TaskItemArticle  TaskItemType = "article"
	TaskItemExercise TaskItemType = "exercise"
)

type Task struct {
	gorm.Model
	// ID          uint       `gorm:"primaryKey"`
	Title       string     `gorm:"size:255;not null"`
	Description string     `gorm:"type:text"`
	ModuleType  string     `gorm:"size:50;not null"` // pre-class, in-class, post-class
	Status      TaskStatus `gorm:"type:enum('pending','in_progress','completed');default:'pending'"`
	UserID      uint       `gorm:"index;type:bigint unsigned"`
	ModuleID    uint       `gorm:"index;type:bigint unsigned"`
	DueDate     time.Time
	Order       int    `gorm:"default:0"`
	Difficulty  string `gorm:"size:10"` // 难度字段
}

func (Task) TableName() string {
	return "tasks"
}

// TeacherWeeklyTask 老师布置的周任务
// swagger:model TeacherWeeklyTask
type TeacherWeeklyTask struct {
	BaseModel
	TeacherID          uint       `gorm:"index" json:"teacherId"`
	ResourceModuleID   uint       `gorm:"index" json:"resourceModuleId"`
	ResourceModuleName string     `json:"resourceModuleName"`
	WeekStartDate      time.Time  `gorm:"index" json:"weekStartDate"` // 周开始日期（周一）
	WeekEndDate        time.Time  `gorm:"index" json:"weekEndDate"`   // 周结束日期（周日）
	TaskItems          []TaskItem `gorm:"foreignKey:WeeklyTaskID" json:"taskItems,omitempty"`
}

func (TeacherWeeklyTask) TableName() string {
	return "teacher_weekly_tasks"
}

// TaskItem 任务项
type TaskItem struct {
	BaseModel
	WeeklyTaskID uint         `gorm:"index" json:"weeklyTaskId"`
	DayOfWeek    Weekday      `gorm:"index" json:"dayOfWeek"`
	ItemType     TaskItemType `json:"itemType"`
	ResourceID   uint         `json:"resourceId"`           // 视频或文章ID
	ExerciseID   uint         `json:"exerciseId,omitempty"` // 练习题ID
	Title        string       `json:"title"`
	Description  string       `json:"description,omitempty"`
	ContentType  string       `json:"contentType"` // "video", "article", "exercise"
}

func (TaskItem) TableName() string {
	return "task_items"
}

// DailyTaskCompletion 每日任务完成状态
type DailyTaskCompletion struct {
	BaseModel
	UserID            uint      `gorm:"index" json:"userId"`
	TaskItemID        uint      `gorm:"index" json:"taskItemId"`
	CompletionDate    time.Time `gorm:"index" json:"completionDate"`
	IsCompleted       bool      `gorm:"default:false" json:"isCompleted"`
	Progress          float64   `gorm:"default:0" json:"progress"`              // 0-100
	ResourceCompleted bool      `gorm:"default:false" json:"resourceCompleted"` // 对应资源是否完成
}

func (DailyTaskCompletion) TableName() string {
	return "daily_task_completions"
}
