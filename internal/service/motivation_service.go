package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"errors"
	"math/rand"
	"time"
)

type MotivationService struct {
	MotivationRepo *repository.MotivationRepository
}

func NewMotivationService(motivationRepo *repository.MotivationRepository) *MotivationService {
	return &MotivationService{MotivationRepo: motivationRepo}
}

// 获取所有激励短句
func (s *MotivationService) GetAllMotivations() ([]*model.Motivation, error) {
	return s.MotivationRepo.GetAll()
}

// 获取当前显示的激励短句
func (s *MotivationService) GetCurrentMotivation() (string, error) {
	current, err := s.MotivationRepo.GetCurrent()
	if err != nil {
		// 如果没有当前使用的，或者出错，获取启用的激励短句列表
		enabledMotivations, err := s.MotivationRepo.GetEnabled()
		if err != nil || len(enabledMotivations) == 0 {
			return "", err
		}
		// 设置第一个启用的为当前使用
		s.MotivationRepo.SetCurrent(enabledMotivations[0].ID)
		return enabledMotivations[0].Content, nil
	}

	// 检查是否需要切换（每12小时切换一次）
	now := time.Now()
	elapsed := now.Sub(current.LastUsedAt)
	enabledMotivations, err := s.MotivationRepo.GetEnabled()

	// 如果只有一条启用的短句，则不切换
	if err == nil && len(enabledMotivations) > 1 && elapsed.Hours() >= 12 {
		// 从启用的列表中随机选择一个，排除当前使用的
		var candidates []*model.Motivation
		for _, m := range enabledMotivations {
			if m.ID != current.ID {
				candidates = append(candidates, m)
			}
		}
		if len(candidates) > 0 {
			// 随机选择一个
			newCurrent := candidates[rand.Intn(len(candidates))]
			s.MotivationRepo.SetCurrent(newCurrent.ID)
			return newCurrent.Content, nil
		}
	}

	return current.Content, nil
}

// 创建新的激励短句
func (s *MotivationService) CreateMotivation(content string) error {
	motivation := &model.Motivation{
		Content:         content,
		IsEnabled:       true,
		IsCurrentlyUsed: false,
	}
	return s.MotivationRepo.Create(motivation)
}

// 更新激励短句
func (s *MotivationService) UpdateMotivation(id uint, content string, isEnabled bool) error {
	var motivation model.Motivation
	err := s.MotivationRepo.DB.First(&motivation, id).Error
	if err != nil {
		return err
	}

	current, err := s.MotivationRepo.GetCurrent()
	if err == nil && current.ID == id && !isEnabled {
		enabled, err := s.MotivationRepo.GetEnabled()
		if err != nil {
			return err
		}
		if len(enabled) <= 1 {
			return errors.New("至少需要保留一个启用的激励短句")
		}
	}

	motivation.Content = content
	motivation.IsEnabled = isEnabled
	return s.MotivationRepo.Update(&motivation)
}

// 删除激励短句
func (s *MotivationService) DeleteMotivation(id uint) error {
	// 检查是否是当前使用的短句
	current, err := s.MotivationRepo.GetCurrent()
	if err == nil && current.ID == id {
		// 检查是否有其他启用的短句
		enabled, err := s.MotivationRepo.GetEnabled()
		if err != nil {
			return err
		}
		// 如果只有这一个启用的，则不允许删除
		if len(enabled) <= 1 {
			return errors.New("至少需要保留一个启用的激励短句")
		}
	}

	return s.MotivationRepo.Delete(id)
}

// 立即切换到指定的激励短句
func (s *MotivationService) SwitchToMotivation(id uint) error {
	// 检查是否启用
	motivations, err := s.MotivationRepo.GetAll()
	if err != nil {
		return err
	}

	found := false
	for _, m := range motivations {
		if m.ID == id {
			found = true
			if !m.IsEnabled {
				return errors.New("该激励短句未启用")
			}
			break
		}
	}

	if !found {
		return errors.New("未找到指定的激励短句")
	}

	return s.MotivationRepo.SetCurrent(id)
}
