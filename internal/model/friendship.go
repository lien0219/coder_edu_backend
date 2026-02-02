package model

import "time"

// Friendship 好友关系表
type Friendship struct {
	UserID    uint      `gorm:"primaryKey" json:"userId"`
	FriendID  uint      `gorm:"primaryKey" json:"friendId"`
	Status    string    `gorm:"type:enum('accepted');default:'accepted'" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (Friendship) TableName() string {
	return "friendships"
}

// FriendRequest 好友申请表
type FriendRequest struct {
	UUIDBase
	SenderID   uint      `gorm:"index;not null" json:"senderId"`
	Sender     User      `gorm:"foreignKey:SenderID;references:ID;constraint:false" json:"sender,omitempty"`
	ReceiverID uint      `gorm:"index;not null" json:"receiverId"`
	Receiver   User      `gorm:"foreignKey:ReceiverID;references:ID;constraint:false" json:"receiver,omitempty"`
	Status     string    `gorm:"type:enum('pending','accepted','rejected');default:'pending'" json:"status"`
	Message    string    `gorm:"size:255" json:"message"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (FriendRequest) TableName() string {
	return "friend_requests"
}
