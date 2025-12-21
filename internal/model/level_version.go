package model

import "time"

// swagger:model LevelVersion
type LevelVersion struct {
	BaseModel

	LevelID       uint       `gorm:"index;type:bigint unsigned" json:"levelId"`
	VersionNumber int        `gorm:"default:1" json:"versionNumber"`
	EditorID      uint       `gorm:"index;type:bigint unsigned" json:"editorId"`
	ChangeNote    string     `gorm:"type:text" json:"changeNote"`
	Content       string     `gorm:"type:json" json:"content"`
	IsPublished   bool       `gorm:"default:false" json:"isPublished"`
	PublishedAt   *time.Time `json:"publishedAt,omitempty"`
}

func (LevelVersion) TableName() string {
	return "level_versions"
}
