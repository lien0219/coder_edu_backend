package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel is a replacement for gorm.Model with Swagger documentation
// swagger:model
type BaseModel struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
