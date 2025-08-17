package model

type ResourceType string

const (
	PDF       ResourceType = "pdf"
	Video     ResourceType = "video"
	Article   ResourceType = "article"
	Worksheet ResourceType = "worksheet"
)

// Resource represents a learning resource
// swagger:model Resource
type Resource struct {
	BaseModel
	Title       string       `gorm:"size:255;not null"`
	Description string       `gorm:"type:text"`
	Type        ResourceType `gorm:"type:enum('pdf','video','article','worksheet');not null"`
	URL         string       `gorm:"size:255;not null"`
	ModuleType  string       `gorm:"size:50"`
	UploaderID  uint         `gorm:"index"`
	ViewCount   int          `gorm:"default:0"`
}

func (Resource) TableName() string {
	return "resources"
}
