package model

import (
	"time"
)

type Post struct {
	BaseModel
	// gorm.Model
	// ID        uint      `gorm:"primaryKey"`
	Title     string    `gorm:"size:255;not null"`
	Content   string    `gorm:"type:text;not null"`
	AuthorID  uint      `gorm:"index;type:bigint unsigned"`
	Tags      string    `gorm:"size:255"`
	Upvotes   int       `gorm:"default:0"`
	Views     int       `gorm:"default:0"`
	IsPinned  bool      `gorm:"default:false"`
	Comments  []Comment `gorm:"foreignKey:PostID"`
	CreatedAt time.Time
}

func (Post) TableName() string {
	return "posts"
}

type Comment struct {
	BaseModel
	// gorm.Model
	// ID        uint   `gorm:"primaryKey"`
	PostID    uint   `gorm:"index;type:int unsigned"`
	AuthorID  uint   `gorm:"index;type:bigint unsigned"`
	Content   string `gorm:"type:text;not null"`
	Upvotes   int    `gorm:"default:0"`
	CreatedAt time.Time
}

func (Comment) TableName() string {
	return "comments"
}

type Question struct {
	BaseModel
	// gorm.Model
	// ID        uint     `gorm:"primaryKey"`
	Title     string   `gorm:"size:255;not null"`
	Content   string   `gorm:"type:text;not null"`
	AuthorID  uint     `gorm:"index;type:bigint unsigned"`
	Tags      string   `gorm:"size:255"`
	Upvotes   int      `gorm:"default:0"`
	Answers   []Answer `gorm:"foreignKey:QuestionID"`
	IsSolved  bool     `gorm:"default:false"`
	SolvedAt  *time.Time
	CreatedAt time.Time
}

func (Question) TableName() string {
	return "questions"
}

type Answer struct {
	BaseModel
	// gorm.Model
	// ID         uint   `gorm:"primaryKey"`
	QuestionID uint   `gorm:"index;type:int unsigned"`
	AuthorID   uint   `gorm:"index;type:bigint unsigned"`
	Content    string `gorm:"type:text;not null"`
	Upvotes    int    `gorm:"default:0"`
	IsAccepted bool   `gorm:"default:false"`
	AcceptedAt *time.Time
	CreatedAt  time.Time
}

func (Answer) TableName() string {
	return "answers"
}
