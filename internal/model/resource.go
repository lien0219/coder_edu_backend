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
	ModuleType  string       `gorm:"size:50;not null"`
	ModuleID    uint         `gorm:"index;type:bigint unsigned"`
	UploaderID  uint         `gorm:"index;type:bigint unsigned"`
	ViewCount   int          `gorm:"column:view_count;default:0"`
	Duration    float64      `gorm:"column:duration;default:0"` // 视频时长（秒）
	Size        int64        `gorm:"column:size;default:0"`     // 文件大小（字节）
	Format      string       `gorm:"size:50"`                   // 视频格式
	Thumbnail   string       `gorm:"size:255"`                  // 缩略图URL
	Points      int          `gorm:"default:0"`                 // 完成此资源可获得的积分
}

func (Resource) TableName() string {
	return "resources"
}
