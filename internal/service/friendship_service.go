package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"errors"
)

type FriendshipService struct {
	FriendRepo *repository.FriendshipRepository
	UserRepo   *repository.UserRepository
}

func NewFriendshipService(friendRepo *repository.FriendshipRepository, userRepo *repository.UserRepository) *FriendshipService {
	return &FriendshipService{
		FriendRepo: friendRepo,
		UserRepo:   userRepo,
	}
}

func (s *FriendshipService) SearchUserByEmail(email string) (*model.User, error) {
	user, err := s.UserRepo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	user.Password = ""
	return user, nil
}

func (s *FriendshipService) FuzzySearchUsers(query string) ([]model.User, error) {
	var users []model.User
	searchTerm := "%" + query + "%"
	err := s.UserRepo.DB.Select("id, name, email, avatar").
		Where("disabled = ?", false).
		Where("name LIKE ? OR email LIKE ?", searchTerm, searchTerm).
		Limit(20).
		Find(&users).Error
	return users, err
}

func (s *FriendshipService) SendFriendRequest(senderID uint, receiverID uint, message string) error {
	if senderID == receiverID {
		return errors.New("不能添加自己为好友")
	}

	isFriend, _ := s.FriendRepo.IsFriend(senderID, receiverID)
	if isFriend {
		return errors.New("已经是好友了")
	}

	// 优化：检查对方是否已经给自己发过申请了
	var reciprocalReq model.FriendRequest
	err := s.FriendRepo.DB.Where("sender_id = ? AND receiver_id = ? AND status = ?", receiverID, senderID, "pending").
		First(&reciprocalReq).Error
	if err == nil {
		// 如果对方已经发过申请，直接调用同意接口逻辑
		return s.HandleFriendRequest(reciprocalReq.ID, senderID, true)
	}

	req := &model.FriendRequest{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Message:    message,
		Status:     "pending",
	}
	return s.FriendRepo.CreateRequest(req)
}

func (s *FriendshipService) HandleFriendRequest(requestID string, receiverID uint, accept bool) error {
	req, err := s.FriendRepo.GetRequest(requestID)
	if err != nil {
		return errors.New("申请不存在")
	}

	if req.ReceiverID != receiverID {
		return errors.New("无权处理此申请")
	}

	if req.Status != "pending" {
		return errors.New("申请已处理")
	}

	if accept {
		// 1. 更新当前申请状态
		err = s.FriendRepo.UpdateRequestStatus(requestID, "accepted")
		if err != nil {
			return err
		}

		// 2. 检查是否已经是好友（处理互相加好友的并发/冲突情况）
		isFriend, _ := s.FriendRepo.IsFriend(req.SenderID, req.ReceiverID)
		if isFriend {
			return nil // 已经是好友了，直接返回成功
		}

		// 3. 同步处理反向的申请（如果对方也发了申请，自动设为已接受）
		_ = s.FriendRepo.DB.Model(&model.FriendRequest{}).
			Where("sender_id = ? AND receiver_id = ? AND status = ?", req.ReceiverID, req.SenderID, "pending").
			Update("status", "accepted").Error

		// 4. 创建双向好友关系
		friendship := &model.Friendship{
			UserID:   req.SenderID,
			FriendID: req.ReceiverID,
			Status:   "accepted",
		}
		return s.FriendRepo.CreateFriendship(friendship)
	} else {
		return s.FriendRepo.UpdateRequestStatus(requestID, "rejected")
	}
}

func (s *FriendshipService) GetFriends(userID uint, query string) ([]model.User, error) {
	return s.FriendRepo.GetFriends(userID, query)
}

func (s *FriendshipService) GetFriendRequests(userID uint, query string, limit, offset int) ([]model.FriendRequest, int64, error) {
	return s.FriendRepo.GetRequests(userID, query, limit, offset)
}

func (s *FriendshipService) GetPendingRequests(userID uint) ([]model.FriendRequest, error) {
	return s.FriendRepo.GetPendingRequests(userID)
}

func (s *FriendshipService) DeleteFriend(userID, friendID uint) error {
	return s.FriendRepo.DeleteFriendship(userID, friendID)
}
