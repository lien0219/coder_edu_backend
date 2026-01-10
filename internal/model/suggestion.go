package model

type SuggestionPriority string

const (
	PriorityHigh   SuggestionPriority = "High"
	PriorityMedium SuggestionPriority = "Medium"
	PriorityLow    SuggestionPriority = "Low"
)

type SuggestionStatus string

const (
	StatusPending   SuggestionStatus = "pending"
	StatusCompleted SuggestionStatus = "completed"
)

// Suggestion represents a teacher's suggestion to a student or all students
// swagger:model Suggestion
type Suggestion struct {
	BaseModel
	TeacherID      uint               `gorm:"index;not null" json:"teacherId"`
	StudentID      uint               `gorm:"index;default:0" json:"studentId"` // 0 means for all students
	Title          string             `gorm:"size:255;not null" json:"title"`
	Subtitle       string             `gorm:"type:text" json:"subtitle"`
	Priority       SuggestionPriority `gorm:"type:varchar(20);default:'Medium'" json:"priority"`
	CompletionTime string             `gorm:"size:50" json:"completionTime"`
	RelatedLevelID *uint              `gorm:"index" json:"relatedLevelId"`

	// Virtual field for student side status
	Status SuggestionStatus `gorm:"-" json:"status"`
}

// SuggestionCompletion records the completion status for each student
// swagger:model SuggestionCompletion
type SuggestionCompletion struct {
	BaseModel
	SuggestionID uint             `gorm:"uniqueIndex:idx_suggestion_student" json:"suggestionId"`
	StudentID    uint             `gorm:"uniqueIndex:idx_suggestion_student" json:"studentId"`
	Status       SuggestionStatus `gorm:"type:varchar(20);default:'completed'" json:"status"`
}

func (Suggestion) TableName() string {
	return "suggestions"
}

func (SuggestionCompletion) TableName() string {
	return "suggestion_completions"
}
