package model

import (
	"time"
)

type UserRole string

const (
	Student UserRole = "student"
	Teacher UserRole = "teacher"
	Admin   UserRole = "admin"
)

// swagger:model User
type User struct {
	BaseModel
	Name              string    `gorm:"size:100;not null"`
	Email             string    `gorm:"size:100;unique;not null"`
	Password          string    `gorm:"size:100;not null"`
	Role              UserRole  `gorm:"type:enum('student','teacher','admin');default:'student'"`
	XP                int       `gorm:"default:0"` // 总经验/等级积分
	Points            int       `gorm:"default:0"` // 独立积分系统（课中知识点测试积分）
	Language          string    `gorm:"size:10;default:'en'"`
	Avatar            string    `gorm:"size:255" json:"avatar"`
	Disabled          bool      `gorm:"default:false"`
	CanTakeAssessment bool      `gorm:"default:true" json:"canTakeAssessment"`
	LastLogin         time.Time `gorm:"default:CURRENT_TIMESTAMP(3)"`
	LastSeen          time.Time `gorm:"default:CURRENT_TIMESTAMP(3)"`
}

func (User) TableName() string {
	return "users"
}
