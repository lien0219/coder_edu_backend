package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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
	UserRepo    *repository.UserRepository
	CheckinRepo *repository.CheckinRepository
	DB          *gorm.DB
}

// UserStatsResponse 用户统计数据响应
type UserStatsResponse struct {
	ActiveDays            int     `json:"activeDays"`            // 活跃天数（连续签到天数）
	LevelAverageScore     float64 `json:"levelAverageScore"`     // 关卡挑战平均分
	TotalLearningDuration int     `json:"totalLearningDuration"` // 总学习时长（分钟）
	LevelCompletionCount  int     `json:"levelCompletionCount"`  // 关卡挑战完成个数
}

// NewUserService 创建一个新的用户服务实例
func NewUserService(userRepo *repository.UserRepository, checkinRepo *repository.CheckinRepository) *UserService {
	return &UserService{
		UserRepo:    userRepo,
		CheckinRepo: checkinRepo,
	}
}

// NewUserServiceWithDB 创建一个新的用户服务实例（包含数据库连接）
func NewUserServiceWithDB(userRepo *repository.UserRepository, checkinRepo *repository.CheckinRepository, db *gorm.DB) *UserService {
	return &UserService{
		UserRepo:    userRepo,
		CheckinRepo: checkinRepo,
		DB:          db,
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
		query = query.Where("last_seen > ?", time.Now().Add(-5*time.Minute))
	} else if filter.Status == "offline" {
		query = query.Where("last_seen <= ?", time.Now().Add(-5*time.Minute))
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

// 用户签到功能
func (s *UserService) Checkin(userID uint) (bool, error) {
	// 检查今天是否已经签到
	_, err := s.CheckinRepo.FindByUserAndDate(userID, time.Now())
	if err == nil {
		// 今天已经签到
		return false, nil
	}

	// 创建新的签到记录
	checkin := &model.Checkin{
		UserID:     userID,
		CheckinAt:  time.Now(),
		StreakDays: 1,
	}

	// 检查用户最近的签到记录，更新连续签到天数
	latestCheckin, err := s.CheckinRepo.FindLatestByUser(userID)
	if err == nil {
		// 检查是否是连续签到（昨天）
		yesterday := time.Now().Add(-24 * time.Hour)
		if latestCheckin.CheckinAt.Year() == yesterday.Year() &&
			latestCheckin.CheckinAt.Month() == yesterday.Month() &&
			latestCheckin.CheckinAt.Day() == yesterday.Day() {
			// 连续签到，增加连续签到天数
			checkin.StreakDays = latestCheckin.StreakDays + 1
		}
	}

	// 保存签到记录
	err = s.CheckinRepo.Create(checkin)
	if err != nil {
		return false, err
	}

	// 计算签到积分
	points := calculateCheckinPoints(checkin.StreakDays)

	// 更新用户积分
	err = s.UpdateUserPoints(userID, points)
	if err != nil {
		// 积分更新失败不应影响签到成功状态，但应记录错误
		// 在实际应用中应该添加日志记录
	}

	return true, nil
}

// calculateCheckinPoints 根据连续签到天数计算应得积分
// 规则：签到一天加5积分，连续签到一周加100积分，以此类推
func calculateCheckinPoints(streakDays int) int {
	// 基础积分：每天5积分
	basePoints := 5

	// 连续签到周数奖励（一周=7天）
	// 整周部分：每整周奖励100积分
	weeks := streakDays / 7
	weekBonus := weeks * 100

	// 剩余天数部分：每天5积分
	remainingDays := streakDays % 7
	remainingPoints := remainingDays * basePoints

	// 总积分
	totalPoints := weekBonus + remainingPoints

	return totalPoints
}

// 检查用户当天是否已签到
func (s *UserService) IsCheckedInToday(userID uint) (bool, error) {
	_, err := s.CheckinRepo.FindByUserAndDate(userID, time.Now())
	if err == nil {
		return true, nil
	}
	return false, nil
}

// 获取用户的签到统计信息
func (s *UserService) GetCheckinStats(userID uint) (map[string]interface{}, error) {
	// 检查今天是否已签到
	isCheckedInToday, err := s.IsCheckedInToday(userID)
	if err != nil {
		return nil, err
	}

	// 获取总签到次数
	checkinCount, err := s.CheckinRepo.GetCheckinCountByUser(userID)
	if err != nil {
		return nil, err
	}

	// 获取连续签到天数和对应的积分
	streakDays := 0
	streakPoints := 0
	latestCheckin, err := s.CheckinRepo.FindLatestByUser(userID)
	if err == nil {
		streakDays = latestCheckin.StreakDays
		streakPoints = calculateCheckinPoints(streakDays)
	}

	// 获取用户当前积分
	user, err := s.UserRepo.FindByID(userID)
	currentPoints := 0
	if err == nil {
		currentPoints = user.XP
	}

	return map[string]interface{}{
		"isCheckedInToday": isCheckedInToday,
		"totalCheckins":    checkinCount,
		"currentStreak":    streakDays,
		"streakPoints":     streakPoints,
		"currentPoints":    currentPoints,
	}, nil
}

// GetUserStats 获取用户的统计数据
func (s *UserService) GetUserStats(userID uint) (*UserStatsResponse, error) {
	if s.DB == nil {
		return nil, errors.New("database connection not available")
	}

	response := &UserStatsResponse{}

	// 1. 获取活跃天数（连续签到天数）
	streakDays := 0
	latestCheckin, err := s.CheckinRepo.FindLatestByUser(userID)
	if err == nil {
		streakDays = latestCheckin.StreakDays
	}
	response.ActiveDays = streakDays

	// 2. 获取关卡挑战平均分
	var averageScore float64
	scoreQuery := `
		SELECT COALESCE(AVG(score), 0) as avg_score
		FROM level_attempts
		WHERE user_id = ? AND ended_at IS NOT NULL AND score > 0 AND deleted_at IS NULL
	`
	if err := s.DB.Raw(scoreQuery, userID).Scan(&averageScore).Error; err != nil {
		return nil, err
	}
	response.LevelAverageScore = averageScore

	// 3. 获取总学习时长（从learning_logs表获取总duration，单位为分钟）
	var totalDuration int
	durationQuery := `
		SELECT COALESCE(SUM(duration), 0) as total_duration
		FROM learning_logs
		WHERE user_id = ? AND deleted_at IS NULL
	`
	if err := s.DB.Raw(durationQuery, userID).Scan(&totalDuration).Error; err != nil {
		return nil, err
	}
	response.TotalLearningDuration = totalDuration

	// 4. 获取关卡挑战完成个数（成功完成的关卡数量）
	var completionCount int
	completionQuery := `
		SELECT COUNT(*) as completion_count
		FROM level_attempts
		WHERE user_id = ? AND success = true AND deleted_at IS NULL
	`
	if err := s.DB.Raw(completionQuery, userID).Scan(&completionCount).Error; err != nil {
		return nil, err
	}
	response.LevelCompletionCount = completionCount

	return response, nil
}
