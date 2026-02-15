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
	Name              string    `gorm:"size:100;not null" json:"Name"`
	Email             string    `gorm:"size:100;unique;not null" json:"Email"`
	Password          string    `gorm:"size:100;not null" json:"-"`
	Role              UserRole  `gorm:"type:enum('student','teacher','admin');default:'student'" json:"Role"`
	XP                int       `gorm:"default:0" json:"XP"`     // 总经验/等级积分
	Points            int       `gorm:"default:0" json:"Points"` // 独立积分系统（课中知识点测试积分）
	Language          string    `gorm:"size:10;default:'en'" json:"Language"`
	Avatar            string    `gorm:"size:255" json:"avatar"`
	Disabled          bool      `gorm:"default:false" json:"Disabled"`
	CanTakeAssessment bool      `gorm:"default:true" json:"canTakeAssessment"`
	LastLogin         time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"LastLogin"`
	LastSeen          time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"LastSeen"`
}

func (User) TableName() string {
	return "users"
}
