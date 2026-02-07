package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type ChatService struct {
	ChatRepo *repository.ChatRepository
}

func NewChatService(chatRepo *repository.ChatRepository) *ChatService {
	return &ChatService{ChatRepo: chatRepo}
}

func (s *ChatService) CreateSystemMessage(convID string, content string) (*model.Message, error) {
	msg := &model.Message{
		ConversationID: convID,
		SenderID:       nil, // 系统消息 SenderID 为 nil
		Type:           "system",
		Content:        content,
	}
	err := s.ChatRepo.CreateMessage(msg)
	return msg, err
}

func (s *ChatService) CreateGroup(creatorID uint, name string, memberIDs []uint) (*model.Conversation, *model.Message, error) {
	conv := &model.Conversation{
		Type:      "group",
		Name:      name,
		CreatorID: creatorID,
	}

	if err := s.ChatRepo.CreateConversation(conv); err != nil {
		return nil, nil, err
	}

	admin := &model.ConversationMember{
		ConversationID: conv.ID,
		UserID:         creatorID,
		Role:           "admin",
	}
	if err := s.ChatRepo.AddMember(admin); err != nil {
		return nil, nil, err
	}

	// 获取创建者信息用于系统消息
	var creator model.User
	s.ChatRepo.DB.First(&creator, creatorID)

	for _, id := range memberIDs {
		if id == creatorID {
			continue
		}
		member := &model.ConversationMember{
			ConversationID: conv.ID,
			UserID:         id,
			Role:           "member",
		}
		s.ChatRepo.AddMember(member)
	}

	sysMsg, _ := s.CreateSystemMessage(conv.ID, fmt.Sprintf("%s 创建了群聊", creator.Name))

	fullConv, err := s.ChatRepo.GetConversation(conv.ID)
	return fullConv, sysMsg, err
}

func (s *ChatService) GetOrCreatePrivateChat(userID1, userID2 uint) (*model.Conversation, error) {
	if userID1 == userID2 {
		return nil, errors.New("不能和自己创建私聊")
	}

	// 1. 尝试查找已存在的私聊
	conv, err := s.ChatRepo.FindPrivateConversation(userID1, userID2)
	if err == nil {
		return conv, nil
	}

	// 2. 如果不存在，则创建新私聊
	newConv := &model.Conversation{
		Type:      "private",
		Name:      "",
		CreatorID: userID1,
	}

	if err := s.ChatRepo.CreateConversation(newConv); err != nil {
		return nil, err
	}

	// 3. 添加两个成员
	members := []uint{userID1, userID2}
	for _, id := range members {
		member := &model.ConversationMember{
			ConversationID: newConv.ID,
			UserID:         id,
			Role:           "member",
		}
		if err := s.ChatRepo.AddMember(member); err != nil {
			return nil, err
		}
	}

	return s.ChatRepo.GetConversation(newConv.ID)
}

func (s *ChatService) InviteMember(adminID uint, convID string, targetUserID uint) (*model.Message, error) {
	// 1. 获取会话信息，确保是群聊
	conv, err := s.ChatRepo.GetConversation(convID)
	if err != nil {
		return nil, err
	}
	if conv.Type != "group" {
		return nil, errors.New("只有群聊可以邀请成员")
	}

	// 2. 检查权限：必须是管理员或群主
	member, err := s.ChatRepo.GetMember(convID, adminID)
	if err != nil {
		return nil, errors.New("你不是该群成员")
	}
	isOwner := conv.CreatorID == adminID
	isAdmin := member.Role == "admin" || isOwner
	if !isAdmin {
		return nil, errors.New("只有管理员可以邀请成员")
	}

	// 3. 检查目标是否已在群里
	_, err = s.ChatRepo.GetMember(convID, targetUserID)
	if err == nil {
		return nil, errors.New("该用户已是群成员")
	}

	newMember := &model.ConversationMember{
		ConversationID: convID,
		UserID:         targetUserID,
		Role:           "member",
	}
	if err := s.ChatRepo.AddMember(newMember); err != nil {
		return nil, err
	}

	// 发送系统消息
	var targetUser model.User
	s.ChatRepo.DB.First(&targetUser, targetUserID)
	return s.CreateSystemMessage(convID, fmt.Sprintf("%s 加入了群聊", targetUser.Name))
}

func (s *ChatService) KickMember(adminID uint, convID string, targetUserID uint) (*model.Message, error) {
	if adminID == targetUserID {
		return nil, errors.New("不能踢出自己")
	}

	// 1. 获取会话信息，确保是群聊
	conv, err := s.ChatRepo.GetConversation(convID)
	if err != nil {
		return nil, err
	}
	if conv.Type != "group" {
		return nil, errors.New("只有群聊可以踢出成员")
	}

	// 2. 检查调用者权限
	caller, err := s.ChatRepo.GetMember(convID, adminID)
	if err != nil {
		return nil, errors.New("你不是该群成员")
	}
	isOwner := conv.CreatorID == adminID
	isAdmin := caller.Role == "admin" || isOwner
	if !isAdmin {
		return nil, errors.New("只有管理员可以踢出成员")
	}

	// 3. 不能踢出群主
	if conv.CreatorID == targetUserID {
		return nil, errors.New("不能踢出群主")
	}

	// 4. 普通管理员不能踢出其他管理员
	targetMember, err := s.ChatRepo.GetMember(convID, targetUserID)
	if err != nil {
		return nil, errors.New("目标用户不是群成员")
	}
	if targetMember.Role == "admin" && !isOwner {
		return nil, errors.New("只有群主可以踢出管理员")
	}

	if err := s.ChatRepo.RemoveMember(convID, targetUserID); err != nil {
		return nil, err
	}

	// 发送系统消息
	var targetUser model.User
	s.ChatRepo.DB.First(&targetUser, targetUserID)
	return s.CreateSystemMessage(convID, fmt.Sprintf("%s 被移出了群聊", targetUser.Name))
}

func (s *ChatService) UpdateGroupInfo(adminID uint, convID string, name string, avatar string) (*model.Message, error) {
	// 1. 获取会话信息，确保是群聊
	conv, err := s.ChatRepo.GetConversation(convID)
	if err != nil {
		return nil, err
	}
	if conv.Type != "group" {
		return nil, errors.New("只有群聊可以修改信息")
	}

	// 2. 检查权限：必须是管理员或群主
	member, err := s.ChatRepo.GetMember(convID, adminID)
	if err != nil {
		return nil, errors.New("你不是该群成员")
	}
	isOwner := conv.CreatorID == adminID
	isAdmin := member.Role == "admin" || isOwner
	if !isAdmin {
		return nil, errors.New("只有管理员可以修改群信息")
	}

	updates := make(map[string]interface{})
	oldName := conv.Name
	if name != "" && name != oldName {
		updates["name"] = name
	}
	if avatar != "" {
		updates["avatar"] = avatar
	}

	if len(updates) == 0 {
		return nil, nil
	}

	if err := s.ChatRepo.DB.Model(&model.Conversation{}).Where("id = ? AND type = ?", convID, "group").Updates(updates).Error; err != nil {
		return nil, err
	}

	// 如果改了名字，发送系统消息
	if name != "" && name != oldName {
		var adminUser model.User
		s.ChatRepo.DB.First(&adminUser, adminID)
		return s.CreateSystemMessage(convID, fmt.Sprintf("%s 修改群名为 \"%s\"", adminUser.Name, name))
	}
	return nil, nil
}

func (s *ChatService) TransferAdmin(currentAdminID uint, convID string, newAdminID uint) error {
	if currentAdminID == newAdminID {
		return errors.New("不能转让给自己")
	}

	// 1. 获取会话信息，确保是群聊
	conv, err := s.ChatRepo.GetConversation(convID)
	if err != nil {
		return err
	}
	if conv.Type != "group" {
		return errors.New("只有群聊可以转让群主")
	}

	// 2. 只有当前群主（CreatorID）有权转让
	if conv.CreatorID != currentAdminID {
		return errors.New("只有群主可以转让权限")
	}

	// 3. 确保目标用户在群里
	_, err = s.ChatRepo.GetMember(convID, newAdminID)
	if err != nil {
		return errors.New("目标用户不是群成员")
	}

	return s.ChatRepo.DB.Transaction(func(tx *gorm.DB) error {
		// 4. 原群主降级为普通成员 (如果原来是 admin 也会变普通成员，群主身份已在下面 creator_id 体现)
		if err := tx.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id = ?", convID, currentAdminID).
			Update("role", "member").Error; err != nil {
			return err
		}
		// 5. 新群主升级为管理员
		if err := tx.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id = ?", convID, newAdminID).
			Update("role", "admin").Error; err != nil {
			return err
		}
		// 6. 重要：更新会话表的所有者 ID
		if err := tx.Model(&model.Conversation{}).
			Where("id = ?", convID).
			Update("creator_id", newAdminID).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *ChatService) SendMessage(senderID uint, convID string, msgType string, content string, clientMsgID string) (*model.Message, error) {
	_, err := s.ChatRepo.GetMember(convID, senderID)
	if err != nil {
		return nil, errors.New("非会话成员无法发送消息")
	}

	msg := &model.Message{
		ConversationID: convID,
		SenderID:       &senderID,
		Type:           msgType,
		Content:        content,
		ClientMsgID:    clientMsgID,
	}

	// 提前填充发送者信息，适配异步写入架构
	var user model.User
	s.ChatRepo.DB.First(&user, senderID)
	msg.Sender = user

	if err := s.ChatRepo.CreateMessage(msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *ChatService) GetHistory(userID uint, convID string, query string, limit int, offset int, beforeID string, afterID string, afterSeq uint64) ([]model.Message, error) {
	_, err := s.ChatRepo.GetMember(convID, userID)
	if err != nil {
		return nil, errors.New("无权查看此会话历史")
	}
	return s.ChatRepo.GetMessages(convID, query, limit, offset, beforeID, afterID, afterSeq)
}

func (s *ChatService) GetMessageContext(userID uint, msgID string, limit int) ([]model.Message, error) {
	// 先找到该消息，确定 conversation_id
	var msg model.Message
	if err := s.ChatRepo.DB.First(&msg, "id = ?", msgID).Error; err != nil {
		return nil, err
	}

	// 验证权限
	_, err := s.ChatRepo.GetMember(msg.ConversationID, userID)
	if err != nil {
		return nil, errors.New("无权查看此会话历史")
	}

	return s.ChatRepo.GetMessageContext(msgID, limit)
}

func (s *ChatService) RevokeMessage(userID uint, msgID string) (*model.Message, error) {
	return s.ChatRepo.RevokeMessage(msgID, userID)
}

func (s *ChatService) DisbandGroup(userID uint, convID string) ([]uint, error) {
	// 1. 获取会话信息
	conv, err := s.ChatRepo.GetConversation(convID)
	if err != nil {
		return nil, err
	}

	// 2. 检查类型
	if conv.Type != "group" {
		return nil, errors.New("只有群聊可以解散")
	}

	// 3. 检查权限：只有群主可以解散
	if conv.CreatorID != userID {
		return nil, errors.New("只有群主可以解散群聊")
	}

	// 4. 获取所有成员 ID，用于后续 WS 通知
	var memberIDs []uint
	for _, m := range conv.Members {
		memberIDs = append(memberIDs, m.UserID)
	}

	// 5. 调用 repository 解散
	if err := s.ChatRepo.DeleteConversation(convID); err != nil {
		return nil, err
	}

	return memberIDs, nil
}

func (s *ChatService) LeaveGroup(userID uint, convID string) error {
	// 1. 获取会话信息
	conv, err := s.ChatRepo.GetConversation(convID)
	if err != nil {
		return err
	}

	// 2. 检查类型
	if conv.Type != "group" {
		return errors.New("只有群聊可以退出")
	}

	// 3. 如果是群主，不能直接退出（必须先转让或解散）
	if conv.CreatorID == userID {
		return errors.New("群主不能直接退出群聊，请先转让群主或解散群聊")
	}

	// 4. 检查是否是成员
	_, err = s.ChatRepo.GetMember(convID, userID)
	if err != nil {
		return errors.New("你不是该群成员")
	}

	// 5. 调用 repository 移除成员
	return s.ChatRepo.RemoveMember(convID, userID)
}

func (s *ChatService) GetConversationMembers(userID uint, convID string, query string, limit, offset int) ([]model.ConversationMember, int64, error) {
	// 验证请求者是否在会话中
	_, err := s.ChatRepo.GetMember(convID, userID)
	if err != nil {
		return nil, 0, errors.New("无权查看此会话成员")
	}

	return s.ChatRepo.GetConversationMembers(convID, query, limit, offset)
}

func (s *ChatService) GlobalSearch(userID uint, query string, limit, offset int) ([]model.Message, int64, error) {
	return s.ChatRepo.SearchMessages(userID, query, limit, offset)
}

func (s *ChatService) MarkAsRead(userID uint, convID string, msgID string) error {
	return s.ChatRepo.UpdateLastReadMessage(convID, userID, msgID)
}
