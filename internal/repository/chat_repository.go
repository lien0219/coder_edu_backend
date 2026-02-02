package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type ChatRepository struct {
	DB *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{DB: db}
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
	// 查找两个用户共同参与且类型为 private 的会话
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
	// 统一使用 UTC 时间并去除纳秒干扰，确保比对一致性
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
		Joins("User") // 自动关联 User 表

	if query != "" {
		searchTerm := "%" + query + "%"
		db = db.Where("User.name LIKE ? OR User.email LIKE ?", searchTerm, searchTerm)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Preload("User").
		Limit(limit).Offset(offset).
		Order("role ASC, joined_at ASC"). // 管理员排前面，按加入时间排序
		Find(&members).Error

	return members, total, err
}

func (r *ChatRepository) CreateMessage(msg *model.Message) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		return tx.Model(&model.Conversation{}).Where("id = ?", msg.ConversationID).Update("updated_at", msg.CreatedAt).Error
	})
}

func (r *ChatRepository) GetMessages(convID string, query string, limit int, offset int) ([]model.Message, error) {
	var msgs []model.Message
	db := r.DB.Preload("Sender").Where("conversation_id = ?", convID)

	if query != "" {
		db = db.Where("content LIKE ?", "%"+query+"%")
	}

	err := db.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&msgs).Error
	return msgs, err
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
