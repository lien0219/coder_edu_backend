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
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
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
	Hub    *ChatHub
	Conn   *websocket.Conn
	Send   chan []byte
	UserID uint
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

		// 解析前端发来的实时信号
		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			continue
		}

		// 处理正在输入状态 (TYPING)
		if wsMsg.Type == "TYPING" {
			data, ok := wsMsg.Data.(map[string]interface{})
			if !ok {
				continue
			}
			convID, _ := data["conversationId"].(string)
			if convID == "" {
				continue
			}

			// 获取该会话的其他成员并转发
			// 这里简单起见直接调用一个 Hub 内部方法进行转发
			c.Hub.HandleTransientEvent(c.UserID, convID, wsMsg)
		}
	}
}

// HandleTransientEvent 处理不需要存库的瞬时事件转发 (如 TYPING)
func (h *ChatHub) HandleTransientEvent(senderID uint, convID string, msg WSMessage) {
	if data, ok := msg.Data.(map[string]interface{}); ok {
		// 补充发送者 ID，让接收方知道是谁在输入
		data["userId"] = senderID
		msg.Data = data

		// 优先级 1: 如果前端传了目标用户 ID 列表，则直接推送
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

		// 优先级 2: 如果没传目标 ID，则根据 convID 查找所有成员进行推送 (适用于群聊)
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

type ChatHub struct {
	clients    map[uint]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	Redis      *redis.Client
	ChatRepo   *repository.ChatRepository
	ctx        context.Context
}

func NewChatHub(rdb *redis.Client, chatRepo *repository.ChatRepository) *ChatHub {
	return &ChatHub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uint]*Client),
		Redis:      rdb,
		ChatRepo:   chatRepo,
		ctx:        context.Background(),
	}
}

type PubSubMessage struct {
	TargetUsers []uint    `json:"targetUsers"`
	Payload     WSMessage `json:"payload"`
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
			h.pushToLocalUsers(psMsg.TargetUsers, psMsg.Payload)
		}
	}()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = client
			h.mu.Unlock()
			h.Redis.Set(h.ctx, fmt.Sprintf("user:online:%d", client.UserID), "true", 0)
			h.NotifyStatus(client.UserID, "online")
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()
			h.Redis.Del(h.ctx, fmt.Sprintf("user:online:%d", client.UserID))
			h.NotifyStatus(client.UserID, "offline")
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
	h.PushToUsers(nil, msg)
}

func (h *ChatHub) PushToUsers(userIDs []uint, msg WSMessage) {
	psMsg := PubSubMessage{
		TargetUsers: userIDs,
		Payload:     msg,
	}
	payload, _ := json.Marshal(psMsg)
	h.Redis.Publish(h.ctx, "chat_channel", payload)
}

func (h *ChatHub) pushToLocalUsers(userIDs []uint, msg WSMessage) {
	payload, _ := json.Marshal(msg)
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(userIDs) == 0 {
		for _, client := range h.clients {
			select {
			case client.Send <- payload:
			default:
			}
		}
		return
	}

	for _, id := range userIDs {
		if client, ok := h.clients[id]; ok {
			select {
			case client.Send <- payload:
			default:
			}
		}
	}
}

func (h *ChatHub) IsUserOnline(userID uint) bool {
	val, err := h.Redis.Get(h.ctx, fmt.Sprintf("user:online:%d", userID)).Result()
	return err == nil && val == "true"
}

func ServeWs(hub *ChatHub, w http.ResponseWriter, r *http.Request, userID uint) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{Hub: hub, Conn: conn, Send: make(chan []byte, 256), UserID: userID}
	client.Hub.register <- client

	go client.writePump()
	go client.readPump()
}
