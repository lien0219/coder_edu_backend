package controller

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ChatController 处理IM系统相关的HTTP请求
type ChatController struct {
	ChatService       *service.ChatService
	FriendshipService *service.FriendshipService
	Hub               *service.ChatHub
	Config            *config.Config
}

// CreateGroupRequest 创建群聊请求
type CreateGroupRequest struct {
	Name      string `json:"name" binding:"required" example:"学习小组"`
	MemberIDs []uint `json:"memberIds" swaggertype:"array,number" example:"1,2,3"`
}

// CreatePrivateChatRequest 创建私聊请求
type CreatePrivateChatRequest struct {
	TargetUserID uint `json:"targetUserId" binding:"required" example:"2"`
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	Type        string `json:"type" binding:"required" example:"text"`
	Content     string `json:"content" binding:"required" example:"你好"`
	ClientMsgID string `json:"clientMsgId" example:"uuid-123"`
}

// SendFriendRequestRequest 发送好友申请请求
type SendFriendRequestRequest struct {
	ReceiverID uint   `json:"receiverId" binding:"required" example:"1"`
	Message    string `json:"message" example:"我是王小明"`
}

func NewChatController(chatService *service.ChatService, friendshipService *service.FriendshipService, hub *service.ChatHub, cfg *config.Config) *ChatController {
	return &ChatController{
		ChatService:       chatService,
		FriendshipService: friendshipService,
		Hub:               hub,
		Config:            cfg,
	}
}

// HandleWS godoc
// @Summary WebSocket 连接
// @Description 建立 WebSocket 连接以接收实时消息
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   token query string true "JWT Token"
// @Success 101 {string} string "Switching Protocols"
// @Router /api/chat/ws [get]
func (ctrl *ChatController) HandleWS(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	service.ServeWs(ctrl.Hub, c.Writer, c.Request, userID)
}

// CreateGroup godoc
// @Summary 创建群聊
// @Description 创建一个新的群聊会话
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   request body CreateGroupRequest true "创建群聊请求"
// @Success 200 {object} util.Response{data=model.Conversation} "成功"
// @Failure 400 {object} util.Response "参数错误"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/chat/groups [post]
func (ctrl *ChatController) CreateGroup(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	conv, sysMsg, err := ctrl.ChatService.CreateGroup(userID, req.Name, req.MemberIDs)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 推送系统消息
	if sysMsg != nil {
		var memberIDs []uint
		for _, m := range conv.Members {
			memberIDs = append(memberIDs, m.UserID)
		}
		ctrl.Hub.PushToUsers(memberIDs, service.WSMessage{
			Type: "NEW_MESSAGE",
			Data: sysMsg,
		})
	}

	util.Success(c, conv)
}

// CreatePrivateChat godoc
// @Summary 创建或获取私聊
// @Description 创建一个新的私聊会话，如果已存在则返回现有会话
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   request body CreatePrivateChatRequest true "创建私聊请求"
// @Success 200 {object} util.Response{data=model.Conversation} "成功"
// @Failure 400 {object} util.Response "参数错误"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/chat/privates [post]
func (ctrl *ChatController) CreatePrivateChat(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID

	var req CreatePrivateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	conv, err := ctrl.ChatService.GetOrCreatePrivateChat(userID, req.TargetUserID)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 如果是私聊，填充对方的昵称和头像作为会话名
	if conv.Type == "private" {
		for _, m := range conv.Members {
			if m.UserID != userID {
				conv.Name = m.User.Name
				conv.Avatar = m.User.Avatar
				break
			}
		}
	}

	// 填充扁平化的 MemberIDs
	conv.MemberIDs = make([]uint, 0)
	for _, m := range conv.Members {
		conv.MemberIDs = append(conv.MemberIDs, m.UserID)
	}

	util.Success(c, conv)
}

// GetConversations godoc
// @Summary 获取会话列表
// @Description 获取当前用户的所有会话列表，支持分页和模糊搜索
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   page query int false "页码 (从1开始)" default(1)
// @Param   limit query int false "每页条数" default(20)
// @Param   query query string false "搜索关键字 (群名或好友名)"
// @Success 200 {object} util.Response{data=util.PageResponse{list=[]model.Conversation}} "成功"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/chat/conversations [get]
func (ctrl *ChatController) GetConversations(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	query := c.Query("query")

	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	convs, total, err := ctrl.ChatService.ChatRepo.GetUserConversations(userID, query, limit, offset)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 补充私聊对象的在线状态（如果是私聊）
	type convWithStatus struct {
		model.Conversation
		IsOnline bool `json:"isOnline,omitempty"`
	}
	var list []convWithStatus
	for _, conv := range convs {
		// 填充扁平化的 MemberIDs
		conv.MemberIDs = make([]uint, 0)
		for _, m := range conv.Members {
			conv.MemberIDs = append(conv.MemberIDs, m.UserID)
		}

		isOnline := false
		if conv.Type == "private" {
			for _, m := range conv.Members {
				if m.UserID != userID {
					isOnline = ctrl.Hub.IsUserOnline(m.UserID)
					// 私聊默认使用对方的昵称和头像
					conv.Name = m.User.Name
					conv.Avatar = m.User.Avatar
					break
				}
			}
		}
		list = append(list, convWithStatus{
			Conversation: conv,
			IsOnline:     isOnline,
		})
	}

	util.Success(c, util.PageResponse{
		List:  list,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// SendMessage godoc
// @Summary 发送消息
// @Description 向指定会话发送消息
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "会话ID"
// @Param   request body SendMessageRequest true "发送消息请求"
// @Success 200 {object} util.Response{data=model.Message} "成功"
// @Failure 400 {object} util.Response "参数错误"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/chat/conversations/{id}/messages [post]
func (ctrl *ChatController) SendMessage(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	convID := c.Param("id")
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	msg, err := ctrl.ChatService.SendMessage(userID, convID, req.Type, req.Content, req.ClientMsgID)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 刚发送的消息默认可以撤回
	msg.CanRevoke = true

	// 补充在线状态发送给 WS
	type msgWithStatus struct {
		*model.Message
		IsOnline  bool `json:"isOnline"`
		IsRead    bool `json:"isRead"`
		ReadCount int  `json:"readCount"`
	}
	wsData := msgWithStatus{
		Message:   msg,
		IsOnline:  true,
		IsRead:    false,
		ReadCount: 0,
	}

	conv, _ := ctrl.ChatService.ChatRepo.GetConversation(convID)
	var memberIDs []uint
	for _, m := range conv.Members {
		memberIDs = append(memberIDs, m.UserID)
	}
	ctrl.Hub.PushToUsers(memberIDs, service.WSMessage{
		Type: "NEW_MESSAGE",
		Data: wsData,
	})

	util.Success(c, wsData)
}

// GetHistory godoc
// @Summary 获取历史消息
// @Description 获取指定会话的历史消息记录，支持模糊搜索内容和 ID 分页
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "会话ID"
// @Param   limit query int false "限制条数" default(20)
// @Param   offset query int false "偏移量" default(0)
// @Param   query query string false "搜索关键字"
// @Param   before_id query string false "在此消息 ID 之前的消息"
// @Param   after_id query string false "在此消息 ID 之后的消息"
// @Success 200 {object} util.Response{data=[]object} "成功"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/chat/conversations/{id}/messages [get]
func (ctrl *ChatController) GetHistory(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	convID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	query := c.Query("query")
	beforeID := c.Query("before_id")
	afterID := c.Query("after_id")

	msgs, err := ctrl.ChatService.GetHistory(userID, convID, query, limit, offset, beforeID, afterID)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 获取会话成员以计算已读状态
	conv, _ := ctrl.ChatService.ChatRepo.GetConversation(convID)

	// 补充发送者的在线状态、消息的已读状态和已读人数
	type msgWithStatus struct {
		model.Message
		IsOnline  bool `json:"isOnline"`
		IsRead    bool `json:"isRead"`
		ReadCount int  `json:"readCount"`
	}
	var list []msgWithStatus

	// 提前准备好所有成员的已读时间，用于批量计算 ReadCount
	memberReadTimes := make(map[uint]time.Time)
	for _, m := range conv.Members {
		if m.LastReadMsgTime != nil {
			memberReadTimes[m.UserID] = *m.LastReadMsgTime
		}
	}

	for _, m := range msgs {
		isRead := false
		readCount := 0

		// 计算已读人数：遍历成员已读时间
		for uid, lastReadTime := range memberReadTimes {
			if m.SenderID != nil && uid == *m.SenderID {
				continue
			}
			if !lastReadTime.Before(m.CreatedAt) { // lastReadTime >= m.CreatedAt
				readCount++
			}
		}

		if conv.Type == "private" {
			// 私聊逻辑：ReadCount > 0 即为对方已读
			if m.SenderID != nil && *m.SenderID == userID {
				isRead = readCount > 0
			} else {
				isRead = true
			}
		} else {
			// 群聊逻辑：如果是自己发的，显示 ReadCount；如果是别人发的，自己查到历史说明自己已看过
			if m.SenderID == nil || *m.SenderID != userID {
				isRead = true
			}
		}

		// 计算是否可撤回：必须是自己发的，且在 2 分钟内，且未被撤回
		m.CanRevoke = m.SenderID != nil && *m.SenderID == userID && !m.IsRevoked && time.Since(m.CreatedAt) < 2*time.Minute

		senderID := uint(0)
		if m.SenderID != nil {
			senderID = *m.SenderID
		}

		list = append(list, msgWithStatus{
			Message:   m,
			IsOnline:  ctrl.Hub.IsUserOnline(senderID),
			IsRead:    isRead,
			ReadCount: readCount,
		})
	}

	util.Success(c, list)
}

// GetMessageContext godoc
// @Summary 获取消息上下文
// @Description 获取指定消息及其前后的上下文消息
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "消息ID"
// @Param   limit query int false "总条数" default(20)
// @Success 200 {object} util.Response{data=[]object} "成功"
// @Router /api/chat/messages/{id}/context [get]
func (ctrl *ChatController) GetMessageContext(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	msgID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	msgs, err := ctrl.ChatService.GetMessageContext(userID, msgID, limit)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	for i := range msgs {
		msgs[i].CanRevoke = msgs[i].SenderID != nil && *msgs[i].SenderID == userID && !msgs[i].IsRevoked && time.Since(msgs[i].CreatedAt) < 2*time.Minute
	}

	util.Success(c, msgs)
}

// RevokeMessage godoc
// @Summary 撤回消息
// @Description 撤回自己发送的消息（通常有时间限制）
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "消息ID"
// @Success 200 {object} util.Response "成功"
// @Router /api/chat/messages/{id}/revoke [put]
func (ctrl *ChatController) RevokeMessage(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	msgID := c.Param("id")

	msg, err := ctrl.ChatService.RevokeMessage(userID, msgID)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 推送撤回事件
	conv, _ := ctrl.ChatService.ChatRepo.GetConversation(msg.ConversationID)
	var memberIDs []uint
	for _, m := range conv.Members {
		memberIDs = append(memberIDs, m.UserID)
	}

	ctrl.Hub.PushToUsers(memberIDs, service.WSMessage{
		Type: "MESSAGE_REVOKE",
		Data: map[string]interface{}{
			"conversationId": msg.ConversationID,
			"messageId":      msgID,
			"senderId":       userID,
		},
	})

	util.Success(c, nil)
}

// DisbandGroup godoc
// @Summary 解散群聊
// @Description 仅群主可以解散群聊
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "会话ID"
// @Success 200 {object} util.Response "成功"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/chat/conversations/{id} [delete]
func (ctrl *ChatController) DisbandGroup(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	convID := c.Param("id")

	memberIDs, err := ctrl.ChatService.DisbandGroup(userID, convID)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 推送群解散事件
	ctrl.Hub.PushToUsers(memberIDs, service.WSMessage{
		Type: "GROUP_DISBANDED",
		Data: map[string]interface{}{
			"conversationId": convID,
		},
	})

	util.Success(c, nil)
}

// LeaveGroup godoc
// @Summary 退出群聊
// @Description 普通成员退出群聊，群主必须先转让或解散
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "会话ID"
// @Success 200 {object} util.Response "成功"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/chat/conversations/{id}/leave [post]
func (ctrl *ChatController) LeaveGroup(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	convID := c.Param("id")

	if err := ctrl.ChatService.LeaveGroup(userID, convID); err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 推送成员退出事件给群内其他成员
	conv, _ := ctrl.ChatService.ChatRepo.GetConversation(convID)
	var memberIDs []uint
	for _, m := range conv.Members {
		memberIDs = append(memberIDs, m.UserID)
	}
	ctrl.Hub.PushToUsers(memberIDs, service.WSMessage{
		Type: "MEMBER_LEFT",
		Data: map[string]interface{}{
			"conversationId": convID,
			"userId":         userID,
		},
	})

	util.Success(c, nil)
}

// MarkAsReadRequest 标记已读请求
type MarkAsReadRequest struct {
	MessageID string `json:"messageId" binding:"required" example:"uuid-msg-123"`
}

// UpdateGroupRequest 修改群信息请求
type UpdateGroupRequest struct {
	Name   string `json:"name" example:"新的群名称"`
	Avatar string `json:"avatar" example:"http://..."`
}

// InviteMemberRequest 邀请成员请求
type InviteMemberRequest struct {
	UserID uint `json:"userId" binding:"required" example:"10"`
}

// TransferAdminRequest 转让群主请求
type TransferAdminRequest struct {
	NewAdminID uint `json:"newAdminId" binding:"required" example:"10"`
}

// UpdateGroupInfo godoc
// @Summary 修改群信息
// @Description 仅管理员可修改群名称和头像
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "会话ID"
// @Param   request body UpdateGroupRequest true "更新内容"
// @Success 200 {object} util.Response "成功"
// @Router /api/chat/conversations/{id} [put]
func (ctrl *ChatController) UpdateGroupInfo(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	convID := c.Param("id")

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	sysMsg, err := ctrl.ChatService.UpdateGroupInfo(userID, convID, req.Name, req.Avatar)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 推送系统消息
	if sysMsg != nil {
		conv, _ := ctrl.ChatService.ChatRepo.GetConversation(convID)
		var memberIDs []uint
		for _, m := range conv.Members {
			memberIDs = append(memberIDs, m.UserID)
		}
		ctrl.Hub.PushToUsers(memberIDs, service.WSMessage{
			Type: "NEW_MESSAGE",
			Data: sysMsg,
		})
	}

	// 推送群信息更新事件
	ctrl.Hub.PushToUsers(nil, service.WSMessage{
		Type: "GROUP_INFO_UPDATED",
		Data: map[string]interface{}{
			"conversationId": convID,
			"name":           req.Name,
			"avatar":         req.Avatar,
		},
	})

	util.Success(c, nil)
}

// InviteMember godoc
// @Summary 邀请成员入群
// @Description 仅管理员可以邀请新成员
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "会话ID"
// @Param   request body InviteMemberRequest true "邀请用户ID"
// @Success 200 {object} util.Response "成功"
// @Router /api/chat/conversations/{id}/members [post]
func (ctrl *ChatController) InviteMember(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	convID := c.Param("id")

	var req InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	sysMsg, err := ctrl.ChatService.InviteMember(userID, convID, req.UserID)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 推送系统消息
	if sysMsg != nil {
		conv, _ := ctrl.ChatService.ChatRepo.GetConversation(convID)
		var memberIDs []uint
		for _, m := range conv.Members {
			memberIDs = append(memberIDs, m.UserID)
		}
		ctrl.Hub.PushToUsers(memberIDs, service.WSMessage{
			Type: "NEW_MESSAGE",
			Data: sysMsg,
		})
	}

	util.Success(c, nil)
}

// KickMember godoc
// @Summary 踢出群成员
// @Description 仅管理员可以踢出成员
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "会话ID"
// @Param   userId path uint true "被踢出的用户ID"
// @Success 200 {object} util.Response "成功"
// @Router /api/chat/conversations/{id}/members/{userId} [delete]
func (ctrl *ChatController) KickMember(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	convID := c.Param("id")
	targetIDStr := c.Param("userId")
	targetID, _ := strconv.ParseUint(targetIDStr, 10, 32)

	sysMsg, err := ctrl.ChatService.KickMember(userID, convID, uint(targetID))
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 推送系统消息
	if sysMsg != nil {
		conv, _ := ctrl.ChatService.ChatRepo.GetConversation(convID)
		var memberIDs []uint
		for _, m := range conv.Members {
			memberIDs = append(memberIDs, m.UserID)
		}
		memberIDs = append(memberIDs, uint(targetID))

		ctrl.Hub.PushToUsers(memberIDs, service.WSMessage{
			Type: "NEW_MESSAGE",
			Data: sysMsg,
		})

		ctrl.Hub.PushToUsers(memberIDs, service.WSMessage{
			Type: "MEMBER_LEFT",
			Data: map[string]interface{}{
				"conversationId": convID,
				"userId":         uint(targetID),
				"reason":         "kicked",
			},
		})
	}

	util.Success(c, nil)
}

// TransferAdmin godoc
// @Summary 转让群主
// @Description 仅群主可以转让管理权限
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "会话ID"
// @Param   request body TransferAdminRequest true "新群主ID"
// @Success 200 {object} util.Response "成功"
// @Router /api/chat/conversations/{id}/transfer [post]
func (ctrl *ChatController) TransferAdmin(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	convID := c.Param("id")

	var req TransferAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	if err := ctrl.ChatService.TransferAdmin(userID, convID, req.NewAdminID); err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 推送群主变更事件
	ctrl.Hub.PushToUsers(nil, service.WSMessage{
		Type: "ADMIN_TRANSFERRED",
		Data: map[string]interface{}{
			"conversationId": convID,
			"oldAdminId":     userID,
			"newAdminId":     req.NewAdminID,
		},
	})

	util.Success(c, nil)
}

// MarkAsRead godoc
// @Summary 标记消息为已读
// @Description 标记指定会话的消息为已读，并通知会话其他成员
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "会话ID"
// @Param   request body MarkAsReadRequest true "已读消息ID"
// @Success 200 {object} util.Response "成功"
// @Router /api/chat/conversations/{id}/read [put]
func (ctrl *ChatController) MarkAsRead(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	convID := c.Param("id")

	var req MarkAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	if err := ctrl.ChatService.MarkAsRead(userID, convID, req.MessageID); err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 推送已读事件给会话其他成员
	conv, _ := ctrl.ChatService.ChatRepo.GetConversation(convID)
	var targetIDs []uint
	for _, m := range conv.Members {
		if m.UserID != userID {
			targetIDs = append(targetIDs, m.UserID)
		}
	}

	ctrl.Hub.PushToUsers(targetIDs, service.WSMessage{
		Type: "MESSAGE_READ",
		Data: map[string]interface{}{
			"conversationId": convID,
			"userId":         userID,
			"messageId":      req.MessageID,
		},
	})

	util.Success(c, nil)
}

// GetMembers godoc
// @Summary 获取会话成员列表
// @Description 获取指定会话的成员列表，支持模糊筛选和分页，包含成员在线状态
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "会话ID"
// @Param   page query int false "页码 (从1开始)" default(1)
// @Param   limit query int false "每页条数" default(20)
// @Param   query query string false "搜索关键字 (姓名或邮箱)"
// @Success 200 {object} util.Response{data=util.PageResponse{list=[]object}} "成功"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/chat/conversations/{id}/members [get]
func (ctrl *ChatController) GetMembers(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	convID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	query := c.Query("query")

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	members, total, err := ctrl.ChatService.GetConversationMembers(userID, convID, query, limit, offset)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 补充成员在线状态
	type memberWithStatus struct {
		model.ConversationMember
		IsOnline bool `json:"isOnline"`
	}
	var list []memberWithStatus
	for _, m := range members {
		list = append(list, memberWithStatus{
			ConversationMember: m,
			IsOnline:           ctrl.Hub.IsUserOnline(m.UserID),
		})
	}

	util.Success(c, util.PageResponse{
		List:  list,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// SearchUser godoc
// @Summary 搜索用户
// @Description 通过邮箱搜索用户以添加好友
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   email query string true "用户邮箱"
// @Success 200 {object} util.Response{data=model.User} "成功"
// @Failure 404 {object} util.Response "用户不存在"
// @Router /api/chat/users/search [get]
func (ctrl *ChatController) SearchUser(c *gin.Context) {
	email := c.Query("email")
	user, err := ctrl.FriendshipService.SearchUserByEmail(email)
	if err != nil {
		util.Error(c, 404, err.Error())
		return
	}
	util.Success(c, user)
}

// SearchUsers godoc
// @Summary 模糊搜索用户
// @Description 通过昵称或邮箱模糊搜索用户
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   query query string true "搜索关键字"
// @Success 200 {object} util.Response{data=[]model.User} "成功"
// @Router /api/chat/users/search-fuzzy [get]
func (ctrl *ChatController) SearchUsers(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		util.BadRequest(c, "搜索关键字不能为空")
		return
	}

	users, err := ctrl.FriendshipService.FuzzySearchUsers(query)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}
	util.Success(c, users)
}

// SendFriendRequest godoc
// @Summary 发送好友申请
// @Description 向指定用户发送好友申请
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   request body SendFriendRequestRequest true "发送好友申请请求"
// @Success 200 {object} util.Response "成功"
// @Failure 400 {object} util.Response "参数错误"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/chat/friend-requests [post]
func (ctrl *ChatController) SendFriendRequest(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	var req SendFriendRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	err := ctrl.FriendshipService.SendFriendRequest(userID, req.ReceiverID, req.Message)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}
	util.Success(c, gin.H{"message": "申请已发送"})
}

// GetFriends godoc
// @Summary 获取好友列表
// @Description 获取当前用户的好友列表，支持根据昵称或邮箱模糊搜索
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   query query string false "搜索关键字 (昵称或邮箱)"
// @Success 200 {object} util.Response{data=[]model.User} "成功"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/chat/friends [get]
func (ctrl *ChatController) GetFriends(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	query := c.Query("query")
	friends, err := ctrl.FriendshipService.GetFriends(userID, query)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 补充在线状态
	type friendWithStatus struct {
		model.User
		IsOnline bool `json:"isOnline"`
	}
	var result []friendWithStatus
	for _, f := range friends {
		result = append(result, friendWithStatus{
			User:     f,
			IsOnline: ctrl.Hub.IsUserOnline(f.ID),
		})
	}

	util.Success(c, result)
}

// GlobalSearch godoc
// @Summary 全局搜索聊天记录
// @Description 在用户参与的所有会话中搜索包含关键字的消息内容，支持分页
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   query query string true "搜索关键字"
// @Param   page query int false "页码" default(1)
// @Param   limit query int false "每页条数" default(20)
// @Success 200 {object} util.Response{data=util.PageResponse{list=[]model.Message}} "成功"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/chat/search [get]
func (ctrl *ChatController) GlobalSearch(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	query := c.Query("query")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if query == "" {
		util.Success(c, util.PageResponse{List: []model.Message{}, Total: 0, Page: page, Limit: limit})
		return
	}

	offset := (page - 1) * limit
	msgs, total, err := ctrl.ChatService.GlobalSearch(userID, query, limit, offset)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	// 统一处理私聊会话的名称显示
	for i := range msgs {
		if msgs[i].Conversation.Type == "private" {
			for _, m := range msgs[i].Conversation.Members {
				if m.UserID != userID {
					msgs[i].Conversation.Name = m.User.Name
					msgs[i].Conversation.Avatar = m.User.Avatar
					break
				}
			}
		}
	}

	util.Success(c, util.PageResponse{
		List:  msgs,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// DeleteFriend godoc
// @Summary 删除好友
// @Description 解除与指定用户的好友关系
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path uint true "好友用户ID"
// @Success 200 {object} util.Response "成功"
// @Router /api/chat/friends/{id} [delete]
func (ctrl *ChatController) DeleteFriend(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	friendIDStr := c.Param("id")
	friendID, err := strconv.ParseUint(friendIDStr, 10, 32)
	if err != nil {
		util.BadRequest(c, "无效的好友ID")
		return
	}

	err = ctrl.FriendshipService.DeleteFriend(userID, uint(friendID))
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}
	util.Success(c, gin.H{"message": "好友已删除"})
}

// GetFriendRequests godoc
// @Summary 获取好友申请列表
// @Description 获取所有发送给我的好友申请记录（含历史状态），支持分页和模糊搜索
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   page query int false "页码 (从1开始)" default(1)
// @Param   limit query int false "每页条数" default(10)
// @Param   query query string false "搜索关键字 (发送者或接收者昵称/邮箱)"
// @Success 200 {object} util.Response{data=util.PageResponse{list=[]model.FriendRequest}} "成功"
// @Router /api/chat/friend-requests [get]
func (ctrl *ChatController) GetFriendRequests(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	query := c.Query("query")

	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	reqs, total, err := ctrl.FriendshipService.GetFriendRequests(userID, query, limit, offset)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	util.Success(c, util.PageResponse{
		List:  reqs,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// HandleFriendRequestRequest 处理好友申请请求
type HandleFriendRequestRequest struct {
	Action string `json:"action" binding:"required" example:"accept" enums:"accept,reject"`
}

// HandleFriendRequest godoc
// @Summary 处理好友申请
// @Description 同意或拒绝好友申请
// @Tags IM系统
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path string true "申请ID"
// @Param   request body HandleFriendRequestRequest true "处理动作"
// @Success 200 {object} util.Response "成功"
// @Router /api/chat/friend-requests/{id} [put]
func (ctrl *ChatController) HandleFriendRequest(c *gin.Context) {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		util.Unauthorized(c)
		return
	}
	userID := claims.UserID
	requestID := c.Param("id")
	var req HandleFriendRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	accept := req.Action == "accept"
	err := ctrl.FriendshipService.HandleFriendRequest(requestID, userID, accept)
	if err != nil {
		util.Error(c, 500, err.Error())
		return
	}

	msg := "已拒绝"
	if accept {
		msg = "已同意，你们现在是好友了"
	}
	util.Success(c, gin.H{"message": msg})
}

// UploadFile godoc
// @Summary 上传聊天文件
// @Description 上传图片或文件用于聊天，返回文件URL
// @Tags IM系统
// @Accept  multipart/form-data
// @Produce  json
// @Security ApiKeyAuth
// @Param   file formData file true "文件"
// @Success 200 {object} util.Response{data=map[string]string} "成功，返回文件URL"
// @Router /api/chat/upload [post]
func (ctrl *ChatController) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		util.BadRequest(c, "文件不能为空")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	// 支持的扩展名
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".pdf": true, ".docx": true, ".txt": true, ".zip": true,
		".mp4": true, ".mp3": true,
	}
	if !allowedExts[ext] {
		util.BadRequest(c, "不支持的文件类型")
		return
	}

	// 生成唯一文件名
	newFilename := fmt.Sprintf("%s-%s", time.Now().Format("20060102150405"), strings.ReplaceAll(file.Filename, " ", "-"))

	var fileURL string
	if ctrl.Config.Storage.Type == "local" {
		// 确保目录存在
		if _, err := os.Stat(ctrl.Config.Storage.LocalPath); os.IsNotExist(err) {
			os.MkdirAll(ctrl.Config.Storage.LocalPath, 0755)
		}

		dst := filepath.Join(ctrl.Config.Storage.LocalPath, newFilename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			util.Error(c, 500, "保存文件失败: "+err.Error())
			return
		}
		fileURL = "/uploads/" + newFilename
	} else {
		util.Error(c, 500, "当前存储配置暂不支持上传")
		return
	}

	util.Success(c, gin.H{"url": fileURL})
}
