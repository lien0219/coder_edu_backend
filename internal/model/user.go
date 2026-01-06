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

// User represents a platform user
// swagger:model User
type User struct {
	BaseModel
	Name      string    `gorm:"size:100;not null"`
	Email     string    `gorm:"size:100;unique;not null"`
	Password  string    `gorm:"size:100;not null"`
	Role      UserRole  `gorm:"type:enum('student','teacher','admin');default:'student'"`
	XP        int       `gorm:"default:0"`
	Language  string    `gorm:"size:10;default:'en'"`
	Disabled  bool      `gorm:"default:false"`
	LastLogin time.Time `gorm:"default:CURRENT_TIMESTAMP(3)"`
	LastSeen  time.Time `gorm:"default:CURRENT_TIMESTAMP(3)"`
}

func (User) TableName() string {
	return "users"
}
