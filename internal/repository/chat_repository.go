package repository

import (
	"coder_edu_backend/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type ChatRepository struct {
	DB    *gorm.DB
	Redis *redis.Client
	ctx   context.Context
}

func NewChatRepository(db *gorm.DB, rdb *redis.Client) *ChatRepository {
	return &ChatRepository{
		DB:    db,
		Redis: rdb,
		ctx:   context.Background(),
	}
}

func (r *ChatRepository) CreateConversation(conv *model.Conversation) error {
	return r.DB.Create(conv).Error
}

func (r *ChatRepository) GetConversation(id string) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.DB.Preload("Members.User").First(&conv, "id = ?", id).Error
	return &conv, err
}

func (r *ChatRepository) GetUserConversations(userID uint, query string, limit, offset int) ([]model.Conversation, int64, error) {
	var convs []model.Conversation
	var total int64

	db := r.DB.Model(&model.Conversation{}).
		Joins("JOIN conversation_members ON conversation_members.conversation_id = conversations.id").
		Where("conversation_members.user_id = ?", userID)

	if query != "" {
		searchTerm := "%" + query + "%"
		db = db.Where("conversations.name LIKE ?", searchTerm)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询数据
	err := db.Preload("Members.User").
		Order("conversations.updated_at DESC").
		Limit(limit).Offset(offset).
		Find(&convs).Error

	return convs, total, err
}

func (r *ChatRepository) FindPrivateConversation(userID1, userID2 uint) (*model.Conversation, error) {
	var conv model.Conversation
	// 查找两个用户共同参与且类型为private的会话
	err := r.DB.Table("conversations").
		Joins("JOIN conversation_members cm1 ON cm1.conversation_id = conversations.id").
		Joins("JOIN conversation_members cm2 ON cm2.conversation_id = conversations.id").
		Where("conversations.type = ?", "private").
		Where("cm1.user_id = ?", userID1).
		Where("cm2.user_id = ?", userID2).
		Preload("Members.User").
		First(&conv).Error

	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ChatRepository) AddMember(member *model.ConversationMember) error {
	return r.DB.Create(member).Error
}

func (r *ChatRepository) RemoveMember(convID string, userID uint) error {
	return r.DB.Delete(&model.ConversationMember{}, "conversation_id = ? AND user_id = ?", convID, userID).Error
}

func (r *ChatRepository) UpdateMemberRole(convID string, userID uint, role string) error {
	return r.DB.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Update("role", role).Error
}

func (r *ChatRepository) GetMember(convID string, userID uint) (*model.ConversationMember, error) {
	var member model.ConversationMember
	err := r.DB.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error
	return &member, err
}

func (r *ChatRepository) DeleteConversation(convID string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 删除消息
		if err := tx.Where("conversation_id = ?", convID).Delete(&model.Message{}).Error; err != nil {
			return err
		}
		// 2. 删除成员
		if err := tx.Where("conversation_id = ?", convID).Delete(&model.ConversationMember{}).Error; err != nil {
			return err
		}
		// 3. 删除会话
		if err := tx.Where("id = ?", convID).Delete(&model.Conversation{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *ChatRepository) UpdateLastReadMessage(convID string, userID uint, msgID string) error {
	var msg model.Message
	if err := r.DB.First(&msg, "id = ?", msgID).Error; err != nil {
		return err
	}
	readTime := msg.CreatedAt.UTC()
	return r.DB.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Updates(map[string]interface{}{
			"last_read_msg_id":   msgID,
			"last_read_msg_time": readTime,
		}).Error
}

func (r *ChatRepository) GetConversationMembers(convID string, query string, limit, offset int) ([]model.ConversationMember, int64, error) {
	var members []model.ConversationMember
	var total int64

	db := r.DB.Model(&model.ConversationMember{}).
		Where("conversation_id = ?", convID).
		Joins("User")

	if query != "" {
		searchTerm := "%" + query + "%"
		db = db.Where("User.name LIKE ? OR User.email LIKE ?", searchTerm, searchTerm)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Preload("User").
		Limit(limit).Offset(offset).
		Order("role ASC, joined_at ASC").
		Find(&members).Error

	return members, total, err
}

const maxCacheMessages = 50 // 每个会话缓存最近50条消息

func (r *ChatRepository) CreateMessage(msg *model.Message) error {
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		return tx.Model(&model.Conversation{}).Where("id = ?", msg.ConversationID).Update("updated_at", msg.CreatedAt).Error
	})

	if err == nil && r.Redis != nil {
		// 异步缓存消息，避免阻塞主流程
		go r.cacheMessage(msg)
	}
	return err
}

func (r *ChatRepository) cacheMessage(msg *model.Message) {
	// 确保 Sender 信息已加载
	if msg.SenderID != nil && msg.Sender.ID == 0 {
		r.DB.Preload("Sender").First(msg, "id = ?", msg.ID)
	}

	key := fmt.Sprintf("chat:cache:%s", msg.ConversationID)
	data, _ := json.Marshal(msg)

	pipe := r.Redis.Pipeline()
	pipe.LPush(r.ctx, key, data)
	pipe.LTrim(r.ctx, key, 0, maxCacheMessages-1)
	pipe.Expire(r.ctx, key, 24*time.Hour) // 缓存 24 小时
	pipe.Exec(r.ctx)
}

func (r *ChatRepository) GetMessages(convID string, query string, limit int, offset int, beforeID string, afterID string) ([]model.Message, error) {
	// 尝试从缓存读取 (仅针对第一页无搜索条件的请求)
	if query == "" && offset == 0 && beforeID == "" && afterID == "" && r.Redis != nil {
		key := fmt.Sprintf("chat:cache:%s", convID)
		cached, err := r.Redis.LRange(r.ctx, key, 0, int64(limit-1)).Result()
		if err == nil && len(cached) > 0 {
			var msgs []model.Message
			for _, item := range cached {
				var m model.Message
				if err := json.Unmarshal([]byte(item), &m); err == nil {
					msgs = append(msgs, m)
				}
			}
			if len(msgs) > 0 {
				return msgs, nil
			}
		}
	}

	var msgs []model.Message
	db := r.DB.Preload("Sender").Where("conversation_id = ?", convID)

	if query != "" {
		db = db.Where("content LIKE ?", "%"+query+"%")
	}

	if beforeID != "" {
		var beforeMsg model.Message
		if err := r.DB.First(&beforeMsg, "id = ?", beforeID).Error; err == nil {
			db = db.Where("created_at < ?", beforeMsg.CreatedAt)
		}
	}

	if afterID != "" {
		var afterMsg model.Message
		if err := r.DB.First(&afterMsg, "id = ?", afterID).Error; err == nil {
			db = db.Where("created_at > ?", afterMsg.CreatedAt)
		}
	}

	// 如果是 afterID，说明是在向上滚动加载更新的消息，应该按时间正序取最旧的 N 条
	order := "created_at DESC"
	if afterID != "" {
		order = "created_at ASC"
	}

	err := db.Order(order).
		Limit(limit).
		Offset(offset).
		Find(&msgs).Error

	// 如果是 ASC 查出来的，反转回 DESC
	if afterID != "" && err == nil {
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}
	}

	return msgs, err
}

func (r *ChatRepository) GetMessageContext(msgID string, limit int) ([]model.Message, error) {
	var targetMsg model.Message
	if err := r.DB.First(&targetMsg, "id = ?", msgID).Error; err != nil {
		return nil, err
	}

	half := limit / 2
	var prevMsgs []model.Message
	var nextMsgs []model.Message

	// 获取之前的消息（含自己）
	r.DB.Preload("Sender").
		Where("conversation_id = ? AND created_at <= ?", targetMsg.ConversationID, targetMsg.CreatedAt).
		Order("created_at DESC").
		Limit(half + 1).
		Find(&prevMsgs)

	// 获取之后的消息
	r.DB.Preload("Sender").
		Where("conversation_id = ? AND created_at > ?", targetMsg.ConversationID, targetMsg.CreatedAt).
		Order("created_at ASC").
		Limit(half).
		Find(&nextMsgs)

	// 合并并排序
	for i, j := 0, len(prevMsgs)-1; i < j; i, j = i+1, j-1 {
		prevMsgs[i], prevMsgs[j] = prevMsgs[j], prevMsgs[i]
	}

	return append(prevMsgs, nextMsgs...), nil
}

// GetUserRelatedIDs 获取用户参与的所有会话中的所有成员 ID
func (r *ChatRepository) GetUserRelatedIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := r.DB.Table("conversation_members").
		Where("conversation_id IN (SELECT conversation_id FROM conversation_members WHERE user_id = ?)", userID).
		Where("user_id != ?", userID).
		Distinct("user_id").
		Pluck("user_id", &ids).Error
	return ids, err
}

func (r *ChatRepository) RevokeMessage(msgID string, senderID uint) (*model.Message, error) {
	var msg model.Message
	if err := r.DB.First(&msg, "id = ? AND sender_id = ?", msgID, senderID).Error; err != nil {
		return nil, err
	}

	if msg.IsRevoked {
		return &msg, nil
	}

	// 限制撤回时间
	if time.Since(msg.CreatedAt) > 2*time.Minute {
		return nil, fmt.Errorf("消息发送已超过 2 分钟，无法撤回")
	}

	msg.IsRevoked = true
	msg.Content = "消息已撤回"
	err := r.DB.Save(&msg).Error

	if err == nil && r.Redis != nil {
		// 撤回消息后清除缓存，强制下次拉取时回源数据库并更新缓存
		r.Redis.Del(r.ctx, fmt.Sprintf("chat:cache:%s", msg.ConversationID))
	}

	return &msg, err
}

func (r *ChatRepository) SearchMessages(userID uint, query string, limit, offset int) ([]model.Message, int64, error) {
	var msgs []model.Message
	var total int64

	db := r.DB.Model(&model.Message{}).
		Joins("JOIN conversation_members ON conversation_members.conversation_id = messages.conversation_id").
		Where("conversation_members.user_id = ? AND messages.content LIKE ?", userID, "%"+query+"%")

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Preload("Sender").Preload("Conversation.Members.User").
		Order("messages.created_at DESC").
		Limit(limit).Offset(offset).
		Find(&msgs).Error
	return msgs, total, err
}
