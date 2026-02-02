package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type FriendshipRepository struct {
	DB *gorm.DB
}

func NewFriendshipRepository(db *gorm.DB) *FriendshipRepository {
	return &FriendshipRepository{DB: db}
}

func (r *FriendshipRepository) CreateFriendship(f *model.Friendship) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(f).Error; err != nil {
			return err
		}
		reverse := &model.Friendship{
			UserID:   f.FriendID,
			FriendID: f.UserID,
			Status:   f.Status,
		}
		return tx.Create(reverse).Error
	})
}

func (r *FriendshipRepository) GetFriends(userID uint, query string) ([]model.User, error) {
	var friends []model.User
	db := r.DB.Joins("JOIN friendships ON friendships.friend_id = users.id").
		Where("friendships.user_id = ?", userID)

	if query != "" {
		searchTerm := "%" + query + "%"
		db = db.Where("(users.name LIKE ? OR users.email LIKE ?)", searchTerm, searchTerm)
	}

	err := db.Find(&friends).Error
	return friends, err
}

func (r *FriendshipRepository) IsFriend(userID, friendID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Count(&count).Error
	return count > 0, err
}

func (r *FriendshipRepository) CreateRequest(req *model.FriendRequest) error {
	return r.DB.Create(req).Error
}

func (r *FriendshipRepository) GetRequest(id string) (*model.FriendRequest, error) {
	var req model.FriendRequest
	err := r.DB.First(&req, "id = ?", id).Error
	return &req, err
}

func (r *FriendshipRepository) UpdateRequestStatus(id string, status string) error {
	return r.DB.Model(&model.FriendRequest{}).Where("id = ?", id).Update("status", status).Error
}

func (r *FriendshipRepository) GetRequests(userID uint, query string, limit, offset int) ([]model.FriendRequest, int64, error) {
	var reqs []model.FriendRequest
	var total int64

	db := r.DB.Model(&model.FriendRequest{}).
		Preload("Sender").Preload("Receiver").
		Where("sender_id = ? OR receiver_id = ?", userID, userID)

	if query != "" {
		searchTerm := "%" + query + "%"
		// 搜索发送者或接收者的昵称/邮箱
		db = db.Joins("LEFT JOIN users AS sender ON sender.id = friend_requests.sender_id").
			Joins("LEFT JOIN users AS receiver ON receiver.id = friend_requests.receiver_id").
			Where("(sender.name LIKE ? OR sender.email LIKE ? OR receiver.name LIKE ? OR receiver.email LIKE ?)",
				searchTerm, searchTerm, searchTerm, searchTerm)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询数据
	err := db.Order("friend_requests.created_at DESC").
		Limit(limit).Offset(offset).
		Find(&reqs).Error

	return reqs, total, err
}

func (r *FriendshipRepository) GetPendingRequests(userID uint) ([]model.FriendRequest, error) {
	var reqs []model.FriendRequest
	err := r.DB.Where("receiver_id = ? AND status = ?", userID, "pending").Find(&reqs).Error
	return reqs, err
}

func (r *FriendshipRepository) DeleteFriendship(userID, friendID uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND friend_id = ?", userID, friendID).Delete(&model.Friendship{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ? AND friend_id = ?", friendID, userID).Delete(&model.Friendship{}).Error
	})
}
