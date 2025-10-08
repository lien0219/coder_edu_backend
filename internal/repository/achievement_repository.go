package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type AchievementRepository struct {
	DB *gorm.DB
}

func NewAchievementRepository(db *gorm.DB) *AchievementRepository {
	return &AchievementRepository{DB: db}
}

func (r *AchievementRepository) FindByUserID(userID uint) ([]model.Achievement, error) {
	var achievements []model.Achievement
	err := r.DB.Joins("JOIN user_achievements ON user_achievements.achievement_id = achievements.id").
		Where("user_achievements.user_id = ?", userID).
		Find(&achievements).Error
	if err != nil {
		return nil, err
	}
	return achievements, nil
}
