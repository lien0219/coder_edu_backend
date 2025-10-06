package repository

import (
	"coder_edu_backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type CheckinRepository struct {
	DB *gorm.DB
}

// NewCheckinRepository 创建新的签到仓库实例
func NewCheckinRepository(db *gorm.DB) *CheckinRepository {
	return &CheckinRepository{DB: db}
}

// Create 创建新的签到记录
func (r *CheckinRepository) Create(checkin *model.Checkin) error {
	return r.DB.Create(checkin).Error
}

// FindByUserAndDate 检查用户在指定日期是否已签到
func (r *CheckinRepository) FindByUserAndDate(userID uint, date time.Time) (*model.Checkin, error) {
	var checkin model.Checkin
	// 获取日期的开始和结束时间
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-1 * time.Nanosecond)

	err := r.DB.Where("user_id = ? AND checkin_at BETWEEN ? AND ?", userID, startOfDay, endOfDay).First(&checkin).Error
	if err != nil {
		return nil, err
	}
	return &checkin, nil
}

// FindLatestByUser 获取用户最近的签到记录
func (r *CheckinRepository) FindLatestByUser(userID uint) (*model.Checkin, error) {
	var checkin model.Checkin
	err := r.DB.Where("user_id = ?", userID).Order("checkin_at DESC").First(&checkin).Error
	if err != nil {
		return nil, err
	}
	return &checkin, nil
}

// GetCheckinCountByUser 获取用户的总签到次数
func (r *CheckinRepository) GetCheckinCountByUser(userID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Checkin{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
