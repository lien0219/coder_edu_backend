package service

import (
	"coder_edu_backend/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	shardCount     = 32
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
				log.Printf("error: %v", err)
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

		// 处理正在输入状态 (TYPING)
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

			// 获取该会话的其他成员并转发
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
	clients map[uint]*Client
	mu      sync.RWMutex
}

type ChatHub struct {
	shards         [shardCount]*shard
	broadcast      chan []byte
	register       chan *Client
	unregister     chan *Client
	Redis          *redis.Client
	ChatRepo       *repository.ChatRepository
	FriendshipRepo *repository.FriendshipRepository
	ctx            context.Context
}

func NewChatHub(rdb *redis.Client, chatRepo *repository.ChatRepository, friendRepo *repository.FriendshipRepository) *ChatHub {
	h := &ChatHub{
		broadcast:      make(chan []byte),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		Redis:          rdb,
		ChatRepo:       chatRepo,
		FriendshipRepo: friendRepo,
		ctx:            context.Background(),
	}
	for i := 0; i < shardCount; i++ {
		h.shards[i] = &shard{
			clients: make(map[uint]*Client),
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
	pubsub := h.Redis.Subscribe(h.ctx, "chat_channel")
	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			var psMsg PubSubMessage
			if err := json.Unmarshal([]byte(msg.Payload), &psMsg); err != nil {
				log.Printf("PubSub unmarshal error: %v", err)
				continue
			}
			h.pushToLocalRawUsers(psMsg.TargetUsers, psMsg.Payload)
		}
	}()

	// 优化 3: 批量处理状态更新 (使用 Pipeline)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

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
			s.mu.Unlock()
			pendingUpdates = append(pendingUpdates, statusUpdate{client.UserID, "online"})

		case client := <-h.unregister:
			s := h.getShard(client.UserID)
			s.mu.Lock()
			if _, ok := s.clients[client.UserID]; ok {
				delete(s.clients, client.UserID)
				close(client.Send)
			}
			s.mu.Unlock()
			pendingUpdates = append(pendingUpdates, statusUpdate{client.UserID, "offline"})

		case <-ticker.C:
			if len(pendingUpdates) == 0 {
				continue
			}

			pipe := h.Redis.Pipeline()
			for _, update := range pendingUpdates {
				key := fmt.Sprintf("user:online:%d", update.userID)
				if update.status == "online" {
					pipe.Set(h.ctx, key, "true", 0)
				} else {
					pipe.Del(h.ctx, key)
				}
			}
			_, err := pipe.Exec(h.ctx)
			if err != nil {
				log.Printf("Redis pipeline error: %v", err)
			}

			// 发送状态通知
			for _, update := range pendingUpdates {
				h.NotifyStatus(update.userID, update.status)
			}
			pendingUpdates = pendingUpdates[:0]
		}
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

// getRelatedUserIDs 获取与该用户有关联的所有用户 ID (好友 + 所在群成员)
func (h *ChatHub) getRelatedUserIDs(userID uint) []uint {
	userMap := make(map[uint]bool)

	// 1. 获取好友 ID
	if h.FriendshipRepo != nil {
		friends, err := h.FriendshipRepo.GetFriends(userID, "")
		if err == nil {
			for _, f := range friends {
				userMap[f.ID] = true
			}
		}
	}

	// 2. 获取所在群聊的所有成员 ID
	if h.ChatRepo != nil {
		convs, _, err := h.ChatRepo.GetUserConversations(userID, "", 100, 0)
		if err == nil {
			for _, conv := range convs {
				for _, member := range conv.Members {
					if member.UserID != userID {
						userMap[member.UserID] = true
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

func (h *ChatHub) PushToUsers(userIDs []uint, msg WSMessage) {
	// 优化 1: 避免二次序列化
	msgBytes, _ := json.Marshal(msg)
	psMsg := PubSubMessage{
		TargetUsers: userIDs,
		Payload:     msgBytes,
	}
	payload, _ := json.Marshal(psMsg)
	h.Redis.Publish(h.ctx, "chat_channel", payload)
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
	return err == nil && val == "true"
}

func ServeWs(hub *ChatHub, w http.ResponseWriter, r *http.Request, userID uint) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
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
