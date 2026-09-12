package model

import "encoding/json"

const (
	SubmissionStatusPending   = "pending"
	SubmissionStatusCompleted = "completed"

	AutoResultCorrect            = "correct"
	AutoResultWrong              = "wrong"
	AutoResultUnanswered         = "unanswered"
	AutoResultUnscoredAmbiguous  = "unscored_ambiguous"
	AutoResultPendingManual      = "pending_manual"

	ScoringStatusAwaitingTeacher = "awaiting_teacher"
	ScoringStatusTeacherCompleted  = "teacher_completed"
)

// swagger:model AssessmentSubmission
type AssessmentSubmission struct {
	BaseModel
	UserID           uint            `gorm:"index;uniqueIndex:idx_assessment_submit_idempotent;uniqueIndex:idx_assessment_attempt;type:bigint unsigned" json:"userId"`
	User             *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	AssessmentID     uint            `gorm:"index;uniqueIndex:idx_assessment_attempt;type:bigint unsigned" json:"assessmentId"`
	AttemptNo        int             `gorm:"default:1;uniqueIndex:idx_assessment_attempt" json:"attemptNo"`
	ClientRequestID  string          `gorm:"size:64;uniqueIndex:idx_assessment_submit_idempotent" json:"clientRequestId,omitempty"`
	PaperVersion     string          `gorm:"size:64" json:"paperVersion,omitempty"`
	Answers          json.RawMessage `gorm:"type:json" json:"answers"`
	ItemResults      json.RawMessage `gorm:"type:json" json:"itemResults,omitempty"`
	TotalScore       int             `json:"totalScore"`
	AutoScore        int             `json:"autoScore"`
	ObjectiveMax     int             `json:"objectiveMax"`
	PendingManualCount int           `json:"pendingManualCount"`
	ScoringStatus    string          `gorm:"size:32;default:'awaiting_teacher'" json:"scoringStatus"`
	Status           string          `gorm:"size:20;default:'pending'" json:"status"` // pending, completed
	Feedback         string          `gorm:"type:text" json:"feedback"`
	RecommendedLevel int             `json:"recommendedLevel"` // 1:基础, 2:初级, 3:中级, 4:高级
}

func (AssessmentSubmission) TableName() string {
	return "assessment_submissions"
}

type QuestionAnswer struct {
	QuestionID uint   `json:"questionId"`
	Answer     string `json:"answer"`
}

type AssessmentItemResult struct {
	QuestionID      uint   `json:"questionId"`
	QuestionType    string `json:"questionType"`
	AutoResult      string `json:"autoResult"`
	PointsAwarded   int    `json:"pointsAwarded"`
	NeedsTeacherFix bool   `json:"needsTeacherFix,omitempty"`
}
