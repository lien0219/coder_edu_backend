package model

import (
	"time"
)

// AIQAHistory 存储 AI 问答的历史记录，支持多轮对话
type AIQAHistory struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"index" json:"userId"`
	SessionID string    `gorm:"size:50;index" json:"sessionId"` // 会话 ID，用于切断历史边界
	Question  string    `gorm:"type:text;not null" json:"question"`
	Answer    string    `gorm:"type:text;not null" json:"answer"`
	Source    string    `gorm:"size:20" json:"source"` // knowledge_base 或 llm
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}

func (AIQAHistory) TableName() string {
	return "ai_qa_histories"
}
