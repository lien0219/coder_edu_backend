package repository

import (
	"coder_edu_backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type MotivationRepository struct {
	DB *gorm.DB
}

func NewMotivationRepository(db *gorm.DB) *MotivationRepository {
	return &MotivationRepository{DB: db}
}

// 获取所有激励短句
func (r *MotivationRepository) GetAll() ([]*model.Motivation, error) {
	var motivations []*model.Motivation
	err := r.DB.Find(&motivations).Error
	return motivations, err
}

// 获取启用的激励短句
func (r *MotivationRepository) GetEnabled() ([]*model.Motivation, error) {
	var motivations []*model.Motivation
	err := r.DB.Where("is_enabled = ?", true).Find(&motivations).Error
	return motivations, err
}

// 获取当前使用的激励短句
func (r *MotivationRepository) GetCurrent() (*model.Motivation, error) {
	var motivation model.Motivation
	err := r.DB.Where("is_currently_used = ?", true).First(&motivation).Error
	return &motivation, err
}

// 创建激励短句
func (r *MotivationRepository) Create(motivation *model.Motivation) error {
	return r.DB.Create(motivation).Error
}

// 更新激励短句
func (r *MotivationRepository) Update(motivation *model.Motivation) error {
	return r.DB.Save(motivation).Error
}

// 删除激励短句
func (r *MotivationRepository) Delete(id uint) error {
	return r.DB.Delete(&model.Motivation{}, id).Error
}

// 设置当前使用的激励短句
func (r *MotivationRepository) SetCurrent(id uint) error {
	tx := r.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := tx.Model(&model.Motivation{}).Where("is_currently_used = ?", true).Update("is_currently_used", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&model.Motivation{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_currently_used": true,
		"last_used_at":      time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// 检查是否至少有一个启用的短句
func (r *MotivationRepository) HasEnabled() (bool, error) {
	var count int64
	err := r.DB.Model(&model.Motivation{}).Where("is_enabled = ?", true).Count(&count).Error
	return count > 0, err
}
