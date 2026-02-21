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
	DB         *gorm.DB
	Redis      *redis.Client
	ctx        context.Context
	streamName string
	groupName  string
	bufferSize int
}

func NewChatRepository(db *gorm.DB, rdb *redis.Client) *ChatRepository {
	r := &ChatRepository{
		DB:         db,
		Redis:      rdb,
		ctx:        context.Background(),
		streamName: "chat:messages:stream",
		groupName:  "chat:messages:group",
		bufferSize: 100,
	}

	if rdb != nil {
		// 初始化Redis Stream消费组
		rdb.XGroupCreateMkStream(r.ctx, r.streamName, r.groupName, "0")
		// 启动后台Redis Stream消费者
		go r.messageStreamConsumer()
	}

	return r
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
		Where("conversation_members.user_id = ?", userID).
		Where("conversation_members.hidden_at IS NULL") // 过滤掉用户隐藏的会话

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

// HideConversation 用户隐藏某个会话（从会话列表中移除，收到新消息时自动恢复）
func (r *ChatRepository) HideConversation(convID string, userID uint) error {
	now := time.Now()
	return r.DB.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Update("hidden_at", now).Error
}

// UnhideConversation 取消隐藏会话（收到新消息时自动调用）
func (r *ChatRepository) UnhideConversation(convID string) error {
	return r.DB.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND hidden_at IS NOT NULL", convID).
		Update("hidden_at", nil).Error
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
	err := r.DB.Create(member).Error
	if err == nil && r.Redis != nil {
		r.Redis.Del(r.ctx, fmt.Sprintf("chat:relation:group_members:%s", member.ConversationID))
		r.Redis.Del(r.ctx, fmt.Sprintf("chat:relation:user_groups:%d", member.UserID))
	}
	return err
}

func (r *ChatRepository) RemoveMember(convID string, userID uint) error {
	err := r.DB.Delete(&model.ConversationMember{}, "conversation_id = ? AND user_id = ?", convID, userID).Error
	if err == nil && r.Redis != nil {
		r.Redis.Del(r.ctx, fmt.Sprintf("chat:relation:group_members:%s", convID))
		r.Redis.Del(r.ctx, fmt.Sprintf("chat:relation:user_groups:%d", userID))
	}
	return err
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
	// 获取所有成员 ID 以便清除缓存
	var memberIDs []uint
	r.DB.Table("conversation_members").Where("conversation_id = ?", convID).Pluck("user_id", &memberIDs)

	err := r.DB.Transaction(func(tx *gorm.DB) error {
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

	if err == nil && r.Redis != nil {
		r.Redis.Del(r.ctx, fmt.Sprintf("chat:relation:group_members:%s", convID))
		for _, uid := range memberIDs {
			r.Redis.Del(r.ctx, fmt.Sprintf("chat:relation:user_groups:%d", uid))
		}
	}
	return err
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
	// 1. 预先设置好 ID 和时间戳
	if msg.ID == "" {
		msg.ID = model.GenerateUUID()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	// 2. 生成会话内的连续 SeqID (使用 Redis 原子递增)
	if r.Redis != nil {
		seqKey := fmt.Sprintf("chat:seq:%s", msg.ConversationID)
		seq, err := r.Redis.Incr(r.ctx, seqKey).Result()
		if err == nil {
			msg.SeqID = uint64(seq)
		}
	}

	// 自动恢复被隐藏的会话：有新消息时，取消该会话所有成员的隐藏状态
	go r.UnhideConversation(msg.ConversationID)

	// 3. 写入 Redis Stream 实现持久化异步队列
	if r.Redis != nil {
		msgData, _ := json.Marshal(msg)
		_, err := r.Redis.XAdd(r.ctx, &redis.XAddArgs{
			Stream: r.streamName,
			Values: map[string]interface{}{"data": msgData},
		}).Result()

		if err != nil {
			// 如果 Redis 写入失败，降级为同步写入 MySQL
			return r.DB.Transaction(func(tx *gorm.DB) error {
				if err := tx.Create(msg).Error; err != nil {
					return err
				}
				return tx.Model(&model.Conversation{}).Where("id = ?", msg.ConversationID).Update("updated_at", msg.CreatedAt).Error
			})
		}
		// 3. 实时更新缓存
		go r.cacheMessage(msg)
	} else {
		// 无 Redis 环境，同步写入
		return r.DB.Create(msg).Error
	}

	return nil
}

func (r *ChatRepository) messageStreamConsumer() {
	consumerName := fmt.Sprintf("consumer-%d", time.Now().UnixNano())
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		// 批量读取消息
		streams, err := r.Redis.XReadGroup(r.ctx, &redis.XReadGroupArgs{
			Group:    r.groupName,
			Consumer: consumerName,
			Streams:  []string{r.streamName, ">"},
			Count:    int64(r.bufferSize),
			Block:    0,
		}).Result()

		if err != nil || len(streams) == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		var batch []*model.Message
		var msgIDs []string

		for _, xmsg := range streams[0].Messages {
			var msg model.Message
			if data, ok := xmsg.Values["data"].(string); ok {
				if err := json.Unmarshal([]byte(data), &msg); err == nil {
					batch = append(batch, &msg)
					msgIDs = append(msgIDs, xmsg.ID)
				}
			}
		}

		if len(batch) > 0 {
			r.flushMessages(batch)
			// 确认消息处理完毕
			r.Redis.XAck(r.ctx, r.streamName, r.groupName, msgIDs...)
		}
	}
}

func (r *ChatRepository) flushMessages(messages []*model.Message) {
	if len(messages) == 0 {
		return
	}

	// 1. 批量插入消息
	err := r.DB.Create(&messages).Error
	if err != nil {
		// 生产环境增加错误重试或者持久化日志
		return
	}

	// 2. 批量更新会话的活跃时间
	convUpdates := make(map[string]time.Time)
	for _, m := range messages {
		if t, ok := convUpdates[m.ConversationID]; !ok || m.CreatedAt.After(t) {
			convUpdates[m.ConversationID] = m.CreatedAt
		}
	}

	for convID, lastTime := range convUpdates {
		r.DB.Model(&model.Conversation{}).Where("id = ?", convID).Update("updated_at", lastTime)
	}
}

func (r *ChatRepository) cacheMessage(msg *model.Message) {
	// 确保Sender信息已加载
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

func (r *ChatRepository) GetMessages(convID string, query string, limit int, offset int, beforeID string, afterID string, afterSeq uint64) ([]model.Message, error) {
	var cacheMsgs []model.Message
	// 尝试从缓存读取 (仅针对第一页无搜索条件的请求)
	if query == "" && offset == 0 && beforeID == "" && afterID == "" && afterSeq == 0 && r.Redis != nil {
		key := fmt.Sprintf("chat:cache:%s", convID)
		cached, err := r.Redis.LRange(r.ctx, key, 0, int64(limit-1)).Result()
		if err == nil && len(cached) > 0 {
			for _, item := range cached {
				var m model.Message
				if err := json.Unmarshal([]byte(item), &m); err == nil {
					cacheMsgs = append(cacheMsgs, m)
				}
			}
			// 如果缓存的消息已经足够满足limit，直接返回
			if len(cacheMsgs) >= limit {
				return cacheMsgs, nil
			}
			// 如果缓存不足，以最后一条缓存消息为起点，调整参数去数据库补齐
			if len(cacheMsgs) > 0 {
				beforeID = cacheMsgs[len(cacheMsgs)-1].ID
				limit = limit - len(cacheMsgs)
			}
		}
	}

	var msgs []model.Message
	db := r.DB.Preload("Sender").Where("conversation_id = ?", convID)

	if query != "" {
		db = db.Where("content LIKE ?", "%"+query+"%")
	}

	// 支持基于 SeqID 的增量同步 (用于空洞修复)
	if afterSeq > 0 {
		db = db.Where("seq_id > ?", afterSeq)
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

	// 如果是 afterID 或 afterSeq，说明是在获取更新的消息，按时间正序
	order := "created_at DESC"
	if afterID != "" || afterSeq > 0 {
		order = "created_at ASC"
	}

	err := db.Order(order).
		Limit(limit).
		Offset(offset).
		Find(&msgs).Error

	// 如果是正序查出来的，反转回 DESC (保持前端展示一致性)
	if (afterID != "" || afterSeq > 0) && err == nil {
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}
	}

	// 合并缓存和数据库的结果
	if len(cacheMsgs) > 0 {
		return append(cacheMsgs, msgs...), err
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

// GetUserGroupIDs 获取用户参与的所有会话 ID
func (r *ChatRepository) GetUserGroupIDs(userID uint) ([]string, error) {
	var ids []string
	err := r.DB.Table("conversation_members").
		Where("user_id = ?", userID).
		Pluck("conversation_id", &ids).Error
	return ids, err
}

// GetUserGroupIDsCached 获取用户参与的所有会话 ID (带缓存)
func (r *ChatRepository) GetUserGroupIDsCached(userID uint) ([]string, error) {
	if r.Redis == nil {
		return r.GetUserGroupIDs(userID)
	}

	key := fmt.Sprintf("chat:relation:user_groups:%d", userID)
	cached, err := r.Redis.SMembers(r.ctx, key).Result()
	if err == nil && len(cached) > 0 {
		return cached, nil
	}

	ids, err := r.GetUserGroupIDs(userID)
	if err == nil && len(ids) > 0 {
		pipe := r.Redis.Pipeline()
		for _, id := range ids {
			pipe.SAdd(r.ctx, key, id)
		}
		pipe.Expire(r.ctx, key, 24*time.Hour)
		pipe.Exec(r.ctx)
	}
	return ids, err
}

// GetGroupMemberIDs 获取会话中的所有成员 ID
func (r *ChatRepository) GetGroupMemberIDs(convID string) ([]uint, error) {
	var ids []uint
	err := r.DB.Table("conversation_members").
		Where("conversation_id = ?", convID).
		Pluck("user_id", &ids).Error
	return ids, err
}

// GetGroupMemberIDsCached 获取会话成员 ID (带缓存)
func (r *ChatRepository) GetGroupMemberIDsCached(convID string) ([]uint, error) {
	if r.Redis == nil {
		return r.GetGroupMemberIDs(convID)
	}

	key := fmt.Sprintf("chat:relation:group_members:%s", convID)
	cached, err := r.Redis.SMembers(r.ctx, key).Result()
	if err == nil && len(cached) > 0 {
		var ids []uint
		for _, s := range cached {
			var id uint
			fmt.Sscanf(s, "%d", &id)
			if id > 0 {
				ids = append(ids, id)
			}
		}
		return ids, nil
	}

	ids, err := r.GetGroupMemberIDs(convID)
	if err == nil && len(ids) > 0 {
		pipe := r.Redis.Pipeline()
		for _, id := range ids {
			pipe.SAdd(r.ctx, key, id)
		}
		pipe.Expire(r.ctx, key, 24*time.Hour)
		pipe.Exec(r.ctx)
	}
	return ids, err
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

// CountActiveDiscussions 统计用户所在群聊中最近 since 以来有消息的会话数
func (r *ChatRepository) CountActiveDiscussions(userID uint, since time.Time) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Conversation{}).
		Joins("JOIN conversation_members ON conversation_members.conversation_id = conversations.id").
		Where("conversation_members.user_id = ?", userID).
		Where("conversations.type = ?", "group").
		Where("conversations.updated_at >= ?", since).
		Count(&count).Error
	return count, err
}

// GetRecentActiveUsers 获取用户所在会话中最近发过消息的用户（去重，按最近活跃时间倒序）
func (r *ChatRepository) GetRecentActiveUsers(userID uint, limit int) ([]model.User, error) {
	var users []model.User
	err := r.DB.Raw(`
		SELECT u.id, u.name, u.avatar, MAX(m.created_at) AS last_active
		FROM users u
		INNER JOIN messages m ON m.sender_id = u.id
		INNER JOIN conversation_members cm ON cm.conversation_id = m.conversation_id
		WHERE cm.user_id = ? AND m.sender_id != ? AND m.created_at >= ?
		GROUP BY u.id, u.name, u.avatar
		ORDER BY last_active DESC
		LIMIT ?
	`, userID, userID, time.Now().Add(-24*time.Hour), limit).Scan(&users).Error
	return users, err
}

// GetLatestMessageForUser 获取用户所有会话中最新的一条消息（含发送者信息和会话名称）
func (r *ChatRepository) GetLatestMessageForUser(userID uint) (*model.Message, error) {
	var msg model.Message
	err := r.DB.Preload("Sender").Preload("Conversation").
		Joins("JOIN conversation_members ON conversation_members.conversation_id = messages.conversation_id").
		Where("conversation_members.user_id = ?", userID).
		Order("messages.created_at DESC").
		First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// SetupPartitions 为消息表创建分区-----暂时不用，留作后续优化
func (r *ChatRepository) SetupPartitions() error {
	_ = `
	ALTER TABLE messages PARTITION BY RANGE (TO_DAYS(created_at)) (
		PARTITION p202501 VALUES LESS THAN (TO_DAYS('2025-02-01')),
		PARTITION p202502 VALUES LESS THAN (TO_DAYS('2025-03-01')),
		PARTITION p202503 VALUES LESS THAN (TO_DAYS('2025-04-01')),
		PARTITION p_future VALUES LESS THAN MAXVALUE
	);`
	return nil
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
