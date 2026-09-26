package model

import (
	"encoding/json"
	"time"
)

const (
	LearningProfileProgramSelfAssessment = "platform_self_assessment"

	LearningProfileWavePretest  = "pretest"
	LearningProfileWavePosttest = "posttest"

	LearningProfileStatusDraft     = "draft"
	LearningProfileStatusSubmitted = "submitted"

	LearningProfileScoringForward = "forward"

	LearningProfileInstrumentDL  = "DL-C56-v1"
	LearningProfileInstrumentSDL = "SDL-C20-v1"
)

// LearningProfileInstrument is a versioned Likert instrument.
// Not registered with AutoMigrate; created by handwritten SQL only.
type LearningProfileInstrument struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Code      string    `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Version   string    `gorm:"size:32;not null" json:"version"`
	Program   string    `gorm:"size:64;not null;index" json:"program"`
	ScaleMin  int       `gorm:"not null" json:"scaleMin"`
	ScaleMax  int       `gorm:"not null" json:"scaleMax"`
	ItemCount int       `gorm:"not null" json:"itemCount"`
	Status    string    `gorm:"size:32;not null;default:active" json:"status"`
}

func (LearningProfileInstrument) TableName() string {
	return "learning_profile_instruments"
}

// LearningProfileItem is one frozen scale item for a specific instrument version.
type LearningProfileItem struct {
	ID                 uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
	InstrumentID       uint            `gorm:"index;uniqueIndex:idx_lp_items_instrument_code;type:bigint unsigned;not null" json:"instrumentId"`
	ItemCode           string          `gorm:"size:32;uniqueIndex:idx_lp_items_instrument_code;not null" json:"itemCode"`
	SourceItemNo       string          `gorm:"size:32;not null" json:"sourceItemNo"`
	Stem               string          `gorm:"type:text;not null" json:"stem"`
	PrimaryDimension   string          `gorm:"size:64;not null;index" json:"primaryDimension"`
	SecondaryDimension string          `gorm:"size:64;not null" json:"secondaryDimension"`
	ScaleMin           int             `gorm:"not null" json:"scaleMin"`
	ScaleMax           int             `gorm:"not null" json:"scaleMax"`
	ScoringDirection   string          `gorm:"size:16;not null" json:"scoringDirection"`
	SortOrder          int             `gorm:"not null" json:"sortOrder"`
	OptionsJSON        json.RawMessage `gorm:"type:json;not null" json:"optionsJson"`
}

func (LearningProfileItem) TableName() string {
	return "learning_profile_items"
}

// LearningProfileAdministration is one student attempt of one instrument wave.
// Draft and official submit share this row; status submitted is immutable.
type LearningProfileAdministration struct {
	ID           uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	UserID       uint            `gorm:"uniqueIndex:idx_lp_admin_user_inst_wave_program;type:bigint unsigned;not null" json:"userId"`
	InstrumentID uint            `gorm:"uniqueIndex:idx_lp_admin_user_inst_wave_program;index;type:bigint unsigned;not null" json:"instrumentId"`
	WaveType     string          `gorm:"size:16;uniqueIndex:idx_lp_admin_user_inst_wave_program;not null" json:"waveType"`
	Program      string          `gorm:"size:64;uniqueIndex:idx_lp_admin_user_inst_wave_program;not null" json:"program"`
	Status       string          `gorm:"size:16;not null;default:draft" json:"status"`
	PaperVersion string          `gorm:"size:64;not null" json:"paperVersion"`
	// ItemSnapshot stores FrozenPaper JSON on official submit (instrument code/version,
	// stems, dimensions, scale, full option labels/values, and raw answers). Draft is NULL.
	ItemSnapshot json.RawMessage `gorm:"type:json" json:"itemSnapshot,omitempty"`
	StartedAt    *time.Time      `json:"startedAt,omitempty"`
	SubmittedAt  *time.Time      `json:"submittedAt,omitempty"`
}

func (LearningProfileAdministration) TableName() string {
	return "learning_profile_administrations"
}

// LearningProfileAnswer stores one raw Likert value.
type LearningProfileAnswer struct {
	ID               uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AdministrationID uint      `gorm:"uniqueIndex:idx_lp_answers_admin_item;index;type:bigint unsigned;not null" json:"administrationId"`
	ItemID           uint      `gorm:"uniqueIndex:idx_lp_answers_admin_item;type:bigint unsigned;not null" json:"itemId"`
	ItemCode         string    `gorm:"size:32;not null" json:"itemCode"`
	RawValue         int       `gorm:"not null" json:"rawValue"`
	AnsweredAt       time.Time `json:"answeredAt"`
}

func (LearningProfileAnswer) TableName() string {
	return "learning_profile_answers"
}

// LearningProfileDimensionScore is written only on official submit.
type LearningProfileDimensionScore struct {
	ID               uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AdministrationID uint      `gorm:"uniqueIndex:idx_lp_dim_admin_code;index;type:bigint unsigned;not null" json:"administrationId"`
	DimensionCode    string    `gorm:"size:64;uniqueIndex:idx_lp_dim_admin_code;not null" json:"dimensionCode"`
	RawMean          float64   `gorm:"type:decimal(8,4);not null" json:"rawMean"`
	ScaleMin         int       `gorm:"not null" json:"scaleMin"`
	ScaleMax         int       `gorm:"not null" json:"scaleMax"`
	DisplayScore     float64   `gorm:"type:decimal(8,4);not null" json:"displayScore"`
	ItemCount        int       `gorm:"not null" json:"itemCount"`
	ComputedAt       time.Time `json:"computedAt"`
}

func (LearningProfileDimensionScore) TableName() string {
	return "learning_profile_dimension_scores"
}
