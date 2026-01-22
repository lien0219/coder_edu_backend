package repository

import (
	"coder_edu_backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) Create(user *model.User) error {
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	return r.DB.Create(user).Error
}

func (r *UserRepository) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.DB.First(&user, id).Error
	return &user, err
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.DB.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *UserRepository) Update(user *model.User) error {
	return r.DB.Save(user).Error
}

func (r *UserRepository) UpdateXP(userID uint, xp int) error {
	return r.DB.Model(&model.User{}).
		Where("id = ?", userID).
		UpdateColumn("xp", gorm.Expr("xp + ?", xp)).
		Error
}

func (r *UserRepository) UpdateLastLogin(userID uint) error {
	return r.DB.Model(&model.User{}).
		Where("id = ?", userID).
		Update("last_login", time.Now()).
		Error
}

func (r *UserRepository) UpdateLastSeen(userID uint) error {
	return r.DB.Model(&model.User{}).
		Where("id = ?", userID).
		Update("last_seen", time.Now()).
		Error
}
func (r *UserRepository) FindTopByXP(limit int) ([]model.User, error) {
	var users []model.User
	err := r.DB.Where("disabled = ?", false).Order("xp DESC").Limit(limit).Find(&users).Error
	return users, err
}

func (r *UserRepository) FindTopByPoints(limit int) ([]model.User, error) {
	var users []model.User
	err := r.DB.Where("disabled = ? AND role = ?", false, model.Student).Order("points DESC").Limit(limit).Find(&users).Error
	return users, err
}

func (r *UserRepository) GetAchievements(userID uint) ([]model.Achievement, error) {
	var achievements []model.Achievement
	err := r.DB.Where("user_id = ?", userID).Find(&achievements).Error
	return achievements, err
}
