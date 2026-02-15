package service

import (
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/pkg/logger"
	"coder_edu_backend/pkg/monitoring"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	shardCount     = 32
	onlineTTL      = 2 * time.Minute // 在线状态过期时间
)

var (
	// 内存复用 (sync.Pool)
	messagePool = sync.Pool{
		New: func() interface{} {
			return &WSMessage{}
		},
	}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Client struct {
	Hub     *ChatHub
	Conn    *websocket.Conn
	Send    chan []byte
	UserID  uint
	Limiter *rate.Limiter // 限流器
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log.Error("WebSocket unexpected close", zap.Error(err), zap.Uint("userId", c.UserID))
			}
			break
		}

		// 限流校验 (每秒最多 30 条消息，允许突发 50 条)
		if !c.Limiter.Allow() {
			continue
		}

		// 对象池解析消息
		wsMsg := messagePool.Get().(*WSMessage)
		if err := json.Unmarshal(message, wsMsg); err != nil {
			messagePool.Put(wsMsg)
			continue
		}

		monitoring.IMMessageCounter.WithLabelValues(wsMsg.Type, "in").Inc() // 记录上行消息

		// 收到任何有效消息都更新最后活动时间
		if c.Hub.UserRepo != nil {
			go c.Hub.UserRepo.UpdateLastSeen(c.UserID)
		}

		if wsMsg.Type == "TYPING" {
			data, ok := wsMsg.Data.(map[string]interface{})
			if !ok {
				messagePool.Put(wsMsg)
				continue
			}
			convID, _ := data["conversationId"].(string)
			if convID == "" {
				messagePool.Put(wsMsg)
				continue
			}

			c.Hub.HandleTransientEvent(c.UserID, convID, *wsMsg)
		}
		messagePool.Put(wsMsg)
	}
}

// HandleTransientEvent 处理不需要存库的瞬时事件转发
func (h *ChatHub) HandleTransientEvent(senderID uint, convID string, msg WSMessage) {
	if data, ok := msg.Data.(map[string]interface{}); ok {
		if msg.Type == "TYPING" && h.ChatRepo != nil {
			conv, err := h.ChatRepo.GetConversation(convID)
			if err == nil && conv.Type == "group" {
				return
			}
		}

		data["userId"] = senderID
		msg.Data = data

		// 如果传了目标用户 ID 列表，则直接推送
		if targets, ok := data["targetUserIds"].([]interface{}); ok && len(targets) > 0 {
			var ids []uint
			for _, t := range targets {
				if id, ok := t.(float64); ok {
					if uint(id) != senderID {
						ids = append(ids, uint(id))
					}
				}
			}
			h.PushToUsers(ids, msg)
			return
		}

		// 如果没传目标 ID，则根据 convID 查找所有成员进行推送 (适用于群聊)
		if h.ChatRepo != nil {
			conv, err := h.ChatRepo.GetConversation(convID)
			if err == nil {
				var ids []uint
				for _, m := range conv.Members {
					if m.UserID != senderID {
						ids = append(ids, m.UserID)
					}
				}
				h.PushToUsers(ids, msg)
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if n := len(c.Send); n > 0 {
				for i := 0; i < n; i++ {
					w.Write(<-c.Send)
				}
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type shard struct {
	clients           map[uint]*Client
	localGroupMembers map[string]map[uint]bool // convID -> UserIDSet
	mu                sync.RWMutex
}

type ChatHub struct {
	shards         [shardCount]*shard
	broadcast      chan []byte
	register       chan *Client
	unregister     chan *Client
	Redis          *redis.Client
	ChatRepo       *repository.ChatRepository
	UserRepo       *repository.UserRepository
	FriendshipRepo *repository.FriendshipRepository
	ctx            context.Context
	instanceID     string
}

func NewChatHub(rdb *redis.Client, chatRepo *repository.ChatRepository, userRepo *repository.UserRepository, friendRepo *repository.FriendshipRepository) *ChatHub {
	// 暂时生成简单的实例ID(生产环境需要从配置或环境变量中读取)
	id := fmt.Sprintf("node_%d", time.Now().UnixNano())

	h := &ChatHub{
		broadcast:      make(chan []byte),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		Redis:          rdb,
		ChatRepo:       chatRepo,
		UserRepo:       userRepo,
		FriendshipRepo: friendRepo,
		ctx:            context.Background(),
		instanceID:     id,
	}
	for i := 0; i < shardCount; i++ {
		h.shards[i] = &shard{
			clients:           make(map[uint]*Client),
			localGroupMembers: make(map[string]map[uint]bool),
		}
	}
	return h
}

func (h *ChatHub) getShard(userID uint) *shard {
	return h.shards[userID%shardCount]
}

type PubSubMessage struct {
	TargetUsers []uint          `json:"targetUsers"`
	Payload     json.RawMessage `json:"payload"`
}

func (h *ChatHub) Run() {
	// 订阅本节点的专属频道、全局广播频道、节点级群组广播频道
	pubsub := h.Redis.Subscribe(h.ctx,
		fmt.Sprintf("chat:node:%s", h.instanceID),
		"chat:global",
		"chat:node_broadcast",
	)
	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			var psMsg PubSubMessage
			if err := json.Unmarshal([]byte(msg.Payload), &psMsg); err != nil {
				logger.Log.Error("PubSub unmarshal error", zap.Error(err))
				continue
			}

			// 如果是节点级广播psMsg.TargetUsers为空且来自chat:node_broadcast)
			if msg.Channel == "chat:node_broadcast" {
				h.pushToLocalGroupUsers(psMsg.Payload)
			} else {
				h.pushToLocalRawUsers(psMsg.TargetUsers, psMsg.Payload)
			}
		}
	}()

	// 批量处理状态更新
	ticker := time.NewTicker(500 * time.Millisecond)
	// 状态续期定时器 (Heartbeat)
	heartbeatTicker := time.NewTicker(1 * time.Minute)
	defer func() {
		ticker.Stop()
		heartbeatTicker.Stop()
	}()

	type statusUpdate struct {
		userID uint
		status string
	}
	var pendingUpdates []statusUpdate

	for {
		select {
		case client := <-h.register:
			s := h.getShard(client.UserID)
			s.mu.Lock()
			s.clients[client.UserID] = client
			// 预加载该用户的本地群组映射
			h.updateLocalGroupMapping(client.UserID, true)
			s.mu.Unlock()
			pendingUpdates = append(pendingUpdates, statusUpdate{client.UserID, "online"})
			monitoring.IMOnlineUsers.Inc()

			// 更新数据库最后活动时间
			if h.UserRepo != nil {
				go h.UserRepo.UpdateLastSeen(client.UserID)
			}

		case client := <-h.unregister:
			s := h.getShard(client.UserID)
			s.mu.Lock()
			if _, ok := s.clients[client.UserID]; ok {
				h.updateLocalGroupMapping(client.UserID, false)
				delete(s.clients, client.UserID)
				close(client.Send)
				monitoring.IMOnlineUsers.Dec()
			}
			s.mu.Unlock()
			pendingUpdates = append(pendingUpdates, statusUpdate{client.UserID, "offline"})

		case <-heartbeatTicker.C:
			h.refreshOnlineStatus()

		case <-ticker.C:
			if len(pendingUpdates) == 0 {
				continue
			}

			pipe := h.Redis.Pipeline()
			for _, update := range pendingUpdates {
				key := fmt.Sprintf("user:online:%d", update.userID)
				if update.status == "online" {
					// 存储实例ID，实现精准路由
					pipe.Set(h.ctx, key, h.instanceID, onlineTTL)
				} else {
					pipe.Del(h.ctx, key)
				}
			}
			_, err := pipe.Exec(h.ctx)
			if err != nil {
				logger.Log.Error("Redis pipeline error", zap.Error(err))
			}

			for _, update := range pendingUpdates {
				h.NotifyStatus(update.userID, update.status)
			}
			pendingUpdates = pendingUpdates[:0]
		}
	}
}

// refreshOnlineStatus 刷新当前服务器所有在线用户的过期时间
func (h *ChatHub) refreshOnlineStatus() {
	pipe := h.Redis.Pipeline()
	count := 0
	for i := 0; i < shardCount; i++ {
		s := h.shards[i]
		s.mu.RLock()
		for userID := range s.clients {
			// 使用Set放弃Expire，确保即使key意外丢失也能恢复，并锁定在当前实例
			pipe.Set(h.ctx, fmt.Sprintf("user:online:%d", userID), h.instanceID, onlineTTL)
			count++

			// 同时也更新数据库中的最后活动时间，确保管理员后台显示在线
			if h.UserRepo != nil {
				go h.UserRepo.UpdateLastSeen(userID)
			}
		}
		s.mu.RUnlock()
	}
	if count > 0 {
		pipe.Exec(h.ctx)
		logger.Log.Debug("Refreshed online status", zap.Int("count", count))
	}
}

func (h *ChatHub) NotifyStatus(userID uint, status string) {
	msg := WSMessage{
		Type: "USER_STATUS",
		Data: map[string]interface{}{
			"userId": userID,
			"status": status,
		},
	}

	relatedIDs := h.getRelatedUserIDs(userID)
	if len(relatedIDs) > 0 {
		h.PushToUsers(relatedIDs, msg)
	}
}

// getRelatedUserIDs 获取与该用户有关联的所有用户ID(好友 + 所在群成员)
func (h *ChatHub) getRelatedUserIDs(userID uint) []uint {
	userMap := make(map[uint]bool)

	// 1. 获取好友id (带缓存)
	if h.FriendshipRepo != nil {
		ids, err := h.FriendshipRepo.GetFriendIDsCached(userID)
		if err == nil {
			for _, id := range ids {
				if id > 0 { // 排除防止穿透的 0
					userMap[id] = true
				}
			}
		}
	}

	// 2. 获取所在群聊的所有成员id (带缓存)
	if h.ChatRepo != nil {
		// 先获取用户参加的所有群id(带缓存)
		convIDs, err := h.ChatRepo.GetUserGroupIDsCached(userID)
		if err == nil {
			for _, convID := range convIDs {
				memberIDs, err := h.ChatRepo.GetGroupMemberIDsCached(convID)
				if err == nil {
					for _, mid := range memberIDs {
						if mid != userID {
							userMap[mid] = true
						}
					}
				}
			}
		}
	}

	var ids []uint
	for id := range userMap {
		ids = append(ids, id)
	}
	return ids
}

// 关闭所有连接并清理在线状态
func (h *ChatHub) Stop() {
	logger.Log.Info("ChatHub stopping: clearing online status and closing connections...")

	var allUserIDs []uint
	for i := 0; i < shardCount; i++ {
		s := h.shards[i]
		s.mu.Lock()
		for userID, client := range s.clients {
			allUserIDs = append(allUserIDs, userID)
			close(client.Send)
			delete(s.clients, userID)
		}
		s.mu.Unlock()
	}

	if len(allUserIDs) > 0 {
		pipe := h.Redis.Pipeline()
		for _, userID := range allUserIDs {
			pipe.Del(h.ctx, fmt.Sprintf("user:online:%d", userID))
		}
		pipe.Exec(h.ctx)
	}

	monitoring.IMOnlineUsers.Set(0) // 停机时清空指标
	logger.Log.Info("ChatHub stopped", zap.Int("closedConnections", len(allUserIDs)))
}

func (h *ChatHub) updateLocalGroupMapping(userID uint, isRegister bool) {
	if h.ChatRepo == nil {
		return
	}
	convIDs, err := h.ChatRepo.GetUserGroupIDsCached(userID)
	if err != nil {
		return
	}

	for _, convID := range convIDs {
		// 仅在用户所属的shard中记录其所在的群
		s := h.getShard(userID)
		if isRegister {
			if s.localGroupMembers[convID] == nil {
				s.localGroupMembers[convID] = make(map[uint]bool)
			}
			s.localGroupMembers[convID][userID] = true
		} else {
			if s.localGroupMembers[convID] != nil {
				delete(s.localGroupMembers[convID], userID)
				if len(s.localGroupMembers[convID]) == 0 {
					delete(s.localGroupMembers, convID)
				}
			}
		}
	}
}

func (h *ChatHub) PushToUsers(userIDs []uint, msg WSMessage) {
	// 避免二次序列化
	msgBytes, _ := json.Marshal(msg)

	// 如果没有指定用户，则进行全服广播
	if len(userIDs) == 0 {
		psMsg := PubSubMessage{
			TargetUsers: nil,
			Payload:     msgBytes,
		}
		payload, _ := json.Marshal(psMsg)
		h.Redis.Publish(h.ctx, "chat:global", payload)
		monitoring.IMMessageCounter.WithLabelValues(msg.Type, "out").Inc()
		return
	}

	// 如果是群消息，采用节点级发布
	// 判断是否是群消息 (通过 WSMessage 的 Data 或补充 convID 参数)
	// 这里通过 data 中的 conversationId 尝试识别
	var convID string
	if data, ok := msg.Data.(map[string]interface{}); ok {
		convID, _ = data["conversationId"].(string)
	}

	if convID != "" {
		// 节点级发布：不再查每个人的位置，直接往全局频道发
		// 每台机器收到后在本地寻找该群成员
		psMsg := PubSubMessage{
			TargetUsers: nil, // 特殊标记：由接收端根据本地映射过滤
			Payload:     msgBytes,
		}
		// 复用chat:global或新建chat:node_broadcast频道
		// 让所有节点订阅 chat:node_broadcast
		payload, _ := json.Marshal(psMsg)
		h.Redis.Publish(h.ctx, "chat:node_broadcast", payload)
		monitoring.IMMessageCounter.WithLabelValues(msg.Type, "out").Inc()
		return
	}

	// 1：精准路由 (针对私聊)
	keys := make([]string, len(userIDs))
	for i, id := range userIDs {
		keys[i] = fmt.Sprintf("user:online:%d", id)
	}

	instanceMap := make(map[string][]uint)
	locations, err := h.Redis.MGet(h.ctx, keys...).Result()
	if err == nil {
		for i, loc := range locations {
			if loc == nil {
				continue
			}
			instanceID := loc.(string)
			instanceMap[instanceID] = append(instanceMap[instanceID], userIDs[i])
		}
	}

	for instanceID, ids := range instanceMap {
		psMsg := PubSubMessage{
			TargetUsers: ids,
			Payload:     msgBytes,
		}
		payload, _ := json.Marshal(psMsg)
		h.Redis.Publish(h.ctx, fmt.Sprintf("chat:node:%s", instanceID), payload)
	}

	monitoring.IMMessageCounter.WithLabelValues(msg.Type, "out").Inc()
}

func (h *ChatHub) pushToLocalRawUsers(userIDs []uint, payload []byte) {
	if len(userIDs) == 0 {
		for i := 0; i < shardCount; i++ {
			s := h.shards[i]
			s.mu.RLock()
			for _, client := range s.clients {
				select {
				case client.Send <- payload:
				default:
				}
			}
			s.mu.RUnlock()
		}
		return
	}

	for _, id := range userIDs {
		s := h.getShard(id)
		s.mu.RLock()
		if client, ok := s.clients[id]; ok {
			select {
			case client.Send <- payload:
			default:
			}
		}
		s.mu.RUnlock()
	}
}

// pushToLocalGroupUsers 在本地寻找该群成员并推送
func (h *ChatHub) pushToLocalGroupUsers(payload []byte) {
	// 解析出 convID
	var wsMsg WSMessage
	if err := json.Unmarshal(payload, &wsMsg); err != nil {
		return
	}
	data, ok := wsMsg.Data.(map[string]interface{})
	if !ok {
		return
	}
	convID, _ := data["conversationId"].(string)
	if convID == "" {
		return
	}

	// 遍历分片，只推送本地在该群的用户
	for i := 0; i < shardCount; i++ {
		s := h.shards[i]
		s.mu.RLock()
		if memberMap, ok := s.localGroupMembers[convID]; ok {
			for userID := range memberMap {
				if client, ok := s.clients[userID]; ok {
					select {
					case client.Send <- payload:
					default:
					}
				}
			}
		}
		s.mu.RUnlock()
	}
}

func (h *ChatHub) IsUserOnline(userID uint) bool {
	// 查本地分片
	s := h.getShard(userID)
	s.mu.RLock()
	_, ok := s.clients[userID]
	s.mu.RUnlock()
	if ok {
		return true
	}

	// 查 Redis (多实例部署)
	val, err := h.Redis.Get(h.ctx, fmt.Sprintf("user:online:%d", userID)).Result()
	return err == nil && val != ""
}

func ServeWs(hub *ChatHub, w http.ResponseWriter, r *http.Request, userID uint) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log.Error("WebSocket upgrade failed", zap.Error(err), zap.Uint("userId", userID))
		return
	}
	client := &Client{
		Hub:     hub,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		UserID:  userID,
		Limiter: rate.NewLimiter(rate.Limit(30), 50), // 每秒30条，允许突发50条
	}
	client.Hub.register <- client

	go client.writePump()
	go client.readPump()
}
