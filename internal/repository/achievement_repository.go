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

type GoalRepository struct {
	DB *gorm.DB
}

func NewGoalRepository(db *gorm.DB) *GoalRepository {
	return &GoalRepository{DB: db}
}

func (r *GoalRepository) FindByUserID(userID uint) ([]model.Goal, error) {
	var goals []model.Goal
	err := r.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&goals).Error
	if err != nil {
		return nil, err
	}
	return goals, nil
}

func (r *GoalRepository) FindByIDAndUserID(goalID, userID uint) (*model.Goal, error) {
	var goal model.Goal
	err := r.DB.Where("id = ? AND user_id = ?", goalID, userID).First(&goal).Error
	if err != nil {
		return nil, err
	}
	return &goal, nil
}

func (r *GoalRepository) Create(goal *model.Goal) error {
	return r.DB.Create(goal).Error
}

func (r *GoalRepository) Update(goal *model.Goal) error {
	return r.DB.Save(goal).Error
}
