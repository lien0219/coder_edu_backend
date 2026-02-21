package model

import (
	"time"
)

// Conversation 存储会话（私聊、群聊信息）
type Conversation struct {
	UUIDBase
	Type      string               `gorm:"type:enum('private','group');default:'group'" json:"type"`
	Name      string               `gorm:"size:100" json:"name"`
	Avatar    string               `gorm:"size:255" json:"avatar"`
	CreatorID uint                 `gorm:"index" json:"creatorId"` // 指向 User.ID (uint)
	Members   []ConversationMember `gorm:"foreignKey:ConversationID" json:"members"`
	MemberIDs []uint               `gorm:"-" json:"memberIds"` // 扁平化的成员ID列表
	Messages  []Message            `gorm:"foreignKey:ConversationID" json:"messages"`
}

func (Conversation) TableName() string {
	return "conversations"
}

// ConversationMember 维护成员关系、未读数、角色
type ConversationMember struct {
	ConversationID  string     `gorm:"primaryKey;type:varchar(36)" json:"conversationId"`
	UserID          uint       `gorm:"primaryKey;index" json:"userId"` // 优化按用户查询会话
	User            User       `gorm:"foreignKey:UserID" json:"user"`  // 关联用户信息
	Role            string     `gorm:"type:enum('admin','member');default:'member'" json:"role"`
	Nickname        string     `gorm:"size:50" json:"nickname"`
	LastReadMsgID   string     `gorm:"type:varchar(36);default:''" json:"lastReadMsgId"` // 记录最后读到的 UUID 消息 ID
	LastReadMsgTime *time.Time `json:"lastReadMsgTime"`                                  // 最后阅读消息的时间戳
	HiddenAt        *time.Time `gorm:"index" json:"hiddenAt,omitempty"`                  // 用户隐藏会话的时间，为 nil 表示未隐藏
	JoinedAt        time.Time  `gorm:"autoCreateTime" json:"joinedAt"`
}

func (ConversationMember) TableName() string {
	return "conversation_members"
}

// Message 消息记录
type Message struct {
	UUIDBase
	ConversationID string       `gorm:"index;index:idx_conv_created;type:varchar(36);not null" json:"conversationId"`
	CreatedAt      time.Time    `gorm:"index:idx_conv_created" json:"createdAt"` // 优化历史消息查询 (conversation_id, created_at)
	SenderID       *uint        `gorm:"index" json:"senderId"`
	Sender         User         `gorm:"foreignKey:SenderID" json:"sender"`             // 关联发送者用户信息
	Conversation   Conversation `gorm:"foreignKey:ConversationID" json:"conversation"` // 关联会话信息
	Type           string       `gorm:"type:enum('text','image','voice_call','file','system');default:'text'" json:"type"`
	Content        string       `gorm:"type:text" json:"content"`
	Duration       int          `gorm:"default:0" json:"duration"` // 语音通话时长或音视频时长（秒）
	IsRevoked      bool         `gorm:"default:false" json:"isRevoked"`
	CanRevoke      bool         `gorm:"-" json:"canRevoke"`               // 动态字段：是否可撤回
	ThumbnailURL   string       `gorm:"size:255" json:"thumbnailUrl"`     // 缩略图 URL
	ClientMsgID    string       `gorm:"size:50;index" json:"clientMsgId"` // 用于识别重复消息
	SeqID          uint64       `gorm:"index" json:"seqId"`               // 消息序列号，用于可靠性保证
}

func (Message) TableName() string {
	return "messages"
}
