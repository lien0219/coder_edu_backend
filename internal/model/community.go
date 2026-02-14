package model

import (
	"time"
)

type Post struct {
	UUIDBase
	Title    string    `gorm:"size:255;not null"`
	Content  string    `gorm:"type:text;not null"`
	AuthorID uint      `gorm:"index;type:bigint unsigned"`
	Author   User      `gorm:"foreignKey:AuthorID"`
	Tags     string    `gorm:"size:255"`
	Upvotes  int       `gorm:"default:0"`
	Views    int       `gorm:"default:0"`
	IsPinned bool      `gorm:"default:false"`
	Comments []Comment `gorm:"foreignKey:PostID"`
}

func (Post) TableName() string {
	return "posts"
}

type Comment struct {
	UUIDBase
	PostID      string  `gorm:"index;type:varchar(36)" json:"postId"`
	AuthorID    uint    `gorm:"index;type:bigint unsigned" json:"authorId"`
	Author      User    `gorm:"foreignKey:AuthorID" json:"author"`
	Content     string  `gorm:"type:text;not null" json:"content"`
	Upvotes     int     `gorm:"default:0" json:"likes"`
	ParentID    *string `gorm:"index;type:varchar(36)" json:"parentId"`       // 父评论ID
	ReplyToUID  *uint   `gorm:"index;type:bigint unsigned" json:"replyToUid"` // 被回复者ID
	ReplyToUser *User   `gorm:"foreignKey:ReplyToUID" json:"replyToUser"`
}

func (Comment) TableName() string {
	return "comments"
}

type Question struct {
	UUIDBase
	Title    string     `gorm:"size:255;not null" json:"title"`
	Content  string     `gorm:"type:text;not null" json:"content"`
	AuthorID uint       `gorm:"index;type:bigint unsigned" json:"authorId"`
	Author   User       `gorm:"foreignKey:AuthorID" json:"author"`
	Tags     string     `gorm:"size:255" json:"tags"`
	Upvotes  int        `gorm:"default:0" json:"likes"`
	Answers  []Answer   `gorm:"foreignKey:QuestionID" json:"answers"`
	IsSolved bool       `gorm:"default:false" json:"isSolved"`
	SolvedAt *time.Time `json:"solvedAt"`
}

func (Question) TableName() string {
	return "questions"
}

type Answer struct {
	UUIDBase
	QuestionID string     `gorm:"index;type:varchar(36)" json:"questionId"`
	AuthorID   uint       `gorm:"index;type:bigint unsigned" json:"authorId"`
	Author     User       `gorm:"foreignKey:AuthorID" json:"author"`
	Content    string     `gorm:"type:text;not null" json:"content"`
	Upvotes    int        `gorm:"default:0" json:"likes"`
	IsAccepted bool       `gorm:"default:false" json:"isAccepted"`
	AcceptedAt *time.Time `json:"acceptedAt"`
}

func (Answer) TableName() string {
	return "answers"
}

type CommunityLike struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	UserID      uint      `gorm:"uniqueIndex:idx_user_content;type:bigint unsigned" json:"userId"`
	ContentType string    `gorm:"uniqueIndex:idx_user_content;size:20" json:"contentType"` // post, comment, answer
	ContentID   string    `gorm:"uniqueIndex:idx_user_content;size:36" json:"contentId"`
}

func (CommunityLike) TableName() string {
	return "community_likes"
}

type CommunityResourceType string

const (
	ResourcePDF     CommunityResourceType = "pdf"
	ResourceVideo   CommunityResourceType = "video"
	ResourceWord    CommunityResourceType = "word"
	ResourceArticle CommunityResourceType = "article"
)

type CommunityResource struct {
	UUIDBase
	Title         string                `gorm:"size:255;not null" json:"title"`
	Description   string                `gorm:"type:text" json:"description"`
	AuthorID      uint                  `gorm:"index;type:bigint unsigned" json:"authorId"`
	Author        User                  `gorm:"foreignKey:AuthorID" json:"author"`
	Type          CommunityResourceType `gorm:"size:20;not null" json:"type"`
	FileURL       string                `gorm:"size:255" json:"fileUrl"`
	Content       string                `gorm:"type:text" json:"content"` // 用于手写文章
	DownloadCount int                   `gorm:"default:0" json:"downloadCount"`
	ViewCount     int                   `gorm:"default:0" json:"viewCount"`
	Upvotes       int                   `gorm:"default:0" json:"likes"`
}

func (CommunityResource) TableName() string {
	return "community_resources"
}
