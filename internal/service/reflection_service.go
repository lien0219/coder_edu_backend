package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"errors"

	"gorm.io/gorm"
)

var ErrLegacyReflectionWriteDisabled = errors.New("legacy reflection write is disabled")

type ReflectionService struct {
	repo               *repository.ReflectionRepository
	entries            *repository.ReflectionEntryRepository
	db                 *gorm.DB
	LegacyWriteEnabled bool
}

func NewReflectionService(repo *repository.ReflectionRepository, entries *repository.ReflectionEntryRepository, db *gorm.DB) *ReflectionService {
	// 环境变量缺失时保持拒绝。只有调用方显式设置为 true 才恢复旧的单条覆盖写入。
	return &ReflectionService{repo: repo, entries: entries, db: db, LegacyWriteEnabled: false}
}

func (s *ReflectionService) SaveReflection(userID uint, summary, challenges, connections, nextSteps string) (*model.Reflection, error) {
	if !s.LegacyWriteEnabled {
		return nil, ErrLegacyReflectionWriteDisabled
	}
	reflection, err := s.repo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			reflection = &model.Reflection{
				UserID: userID,
			}
		} else {
			return nil, err
		}
	}

	reflection.Summary = summary
	reflection.Challenges = challenges
	reflection.Connections = connections
	reflection.NextSteps = nextSteps

	err = s.repo.Save(reflection)
	return reflection, err
}

func (s *ReflectionService) GetReflectionByUserID(userID uint) (*model.Reflection, error) {
	reflection, err := s.repo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.Reflection{UserID: userID}, nil
		}
		return nil, err
	}
	return reflection, nil
}

func (s *ReflectionService) ListAllReflections(name string, page, pageSize int) ([]model.Reflection, int64, error) {
	return s.repo.ListAll(name, page, pageSize)
}

func (s *ReflectionService) UpdateReflectionByUserID(userID uint, summary, challenges, connections, nextSteps string) (*model.Reflection, error) {
	return s.SaveReflection(userID, summary, challenges, connections, nextSteps)
}

func (s *ReflectionService) GetReflectionByID(id string) (*model.Reflection, error) {
	return s.repo.FindByID(id)
}
