package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) Create(user *model.User) error {
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
		Update("xp", gorm.Expr("xp + ?", xp)).
		Error
}
func (r *UserRepository) FindTopByXP(limit int) ([]model.User, error) {
	var users []model.User
	err := r.DB.Order("xp DESC").Limit(limit).Find(&users).Error
	return users, err
}

// 获取指定用户的所有成就
func (r *UserRepository) GetAchievements(userID uint) ([]model.Achievement, error) {
	var achievements []model.Achievement
	err := r.DB.Where("user_id = ?", userID).Find(&achievements).Error
	return achievements, err
}
