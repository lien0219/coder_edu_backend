package repository

import (
	"coder_edu_backend/internal/model"
	"strings"
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
	// return r.DB.Create(user).Error
	// 确保创建时间被设置
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	// 使用事务来处理插入操作
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// 尝试直接插入
		if err := tx.Create(user).Error; err != nil {
			// 如果因为id字段错误失败，尝试使用另一种方式
			if strings.Contains(err.Error(), "Field 'id' doesn't have a default value") {
				// 先获取当前最大ID
				var maxID uint
				tx.Model(&model.User{}).Select("MAX(id)").Scan(&maxID)

				// 设置新ID
				user.ID = maxID + 1

				// 再次尝试插入
				return tx.Create(user).Error
			}
			return err
		}
		return nil
	})
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
