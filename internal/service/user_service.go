package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// UserFilter 定义用户筛选条件
// swagger:model UserFilter
type UserFilter struct {
	Role      string
	Status    string
	Search    string
	StartDate time.Time
	EndDate   time.Time
}

// UserService 处理用户相关的业务逻辑
type UserService struct {
	UserRepo *repository.UserRepository
}

// NewUserService 创建一个新的用户服务实例
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{
		UserRepo: userRepo,
	}
}

// GetUsers 获取用户列表，支持分页和筛选
func (s *UserService) GetUsers(page, pageSize int, filter UserFilter) ([]model.User, int, error) {
	var users []model.User
	var total int64

	query := s.UserRepo.DB.Model(&model.User{})

	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}

	if filter.Status == "online" {
		query = query.Where("last_login > ?", time.Now().Add(-24*time.Hour))
	} else if filter.Status == "offline" {
		query = query.Where("last_login <= ?", time.Now().Add(-24*time.Hour))
	} else if filter.Status == "disabled" {
		query = query.Where("disabled = ?", true)
	}

	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("name LIKE ? OR email LIKE ?", searchTerm, searchTerm)
	}

	if !filter.StartDate.IsZero() {
		query = query.Where("created_at >= ?", filter.StartDate)
	}

	if !filter.EndDate.IsZero() {
		query = query.Where("created_at <= ?", filter.EndDate)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users)

	return users, int(total), nil
}

// GetUserByID 根据ID获取用户信息
func (s *UserService) GetUserByID(id uint) (*model.User, error) {
	return s.UserRepo.FindByID(id)
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(user *model.User) error {
	existingUser, err := s.UserRepo.FindByID(user.ID)
	if err != nil {
		return errors.New("用户不存在")
	}

	existingUser.Name = user.Name
	existingUser.Email = user.Email
	existingUser.Role = user.Role
	existingUser.Language = user.Language
	existingUser.Disabled = user.Disabled
	existingUser.UpdatedAt = time.Now()

	return s.UserRepo.Update(existingUser)
}

// ResetPassword 重置用户密码
func (s *UserService) ResetPassword(userID uint) (string, error) {
	user, err := s.UserRepo.FindByID(userID)
	if err != nil {
		return "", errors.New("用户不存在")
	}

	// 生成临时密码
	tempPassword := generateTempPassword()

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()

	if err := s.UserRepo.Update(user); err != nil {
		return "", err
	}

	return tempPassword, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(id uint) error {
	user, err := s.UserRepo.FindByID(id)
	if err != nil {
		return errors.New("用户不存在")
	}

	return s.UserRepo.DB.Delete(user).Error
}

// DisableUser 禁用/启用用户
func (s *UserService) DisableUser(id uint, disable bool) error {
	user, err := s.UserRepo.FindByID(id)
	if err != nil {
		return errors.New("用户不存在")
	}

	user.Disabled = disable
	user.UpdatedAt = time.Now()

	return s.UserRepo.Update(user)
}

// generateTempPassword 生成临时密码
func generateTempPassword() string {
	// 生成8位随机密码
	return fmt.Sprintf("temp%d", time.Now().UnixNano()%100000000)
}

// UpdateUserWithPassword 更新用户信息并修改密码
func (s *UserService) UpdateUserWithPassword(user *model.User, newPassword string) error {
	existingUser, err := s.UserRepo.FindByID(user.ID)
	if err != nil {
		return errors.New("用户不存在")
	}

	// 更新基本信息
	existingUser.Name = user.Name
	existingUser.Email = user.Email
	existingUser.Role = user.Role
	existingUser.Language = user.Language
	existingUser.Disabled = user.Disabled
	existingUser.UpdatedAt = time.Now()

	// 如果提供了新密码，则进行加密并更新
	if newPassword != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		existingUser.Password = string(hashedPassword)
	}

	return s.UserRepo.Update(existingUser)
}

// UpdateUserPoints 更新用户的积分
func (s *UserService) UpdateUserPoints(userID uint, points int) error {
	_, err := s.UserRepo.FindByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	return s.UserRepo.UpdateXP(userID, points)
}
