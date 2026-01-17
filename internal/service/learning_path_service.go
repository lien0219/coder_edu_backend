package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"

	"github.com/google/uuid"
)

type LearningPathService struct {
	Repo *repository.LearningPathRepository
}

func NewLearningPathService(repo *repository.LearningPathRepository) *LearningPathService {
	return &LearningPathService{Repo: repo}
}

type CreateMaterialRequest struct {
	Level         int    `json:"level" binding:"required"`
	TotalChapters int    `json:"totalChapters"`
	ChapterNumber int    `json:"chapterNumber"`
	Title         string `json:"title" binding:"required"`
	Content       string `json:"content" binding:"required"`
	Points        int    `json:"points"`
}

func (s *LearningPathService) CreateMaterial(creatorID uint, req CreateMaterialRequest) (*model.LearningPathMaterial, error) {
	material := &model.LearningPathMaterial{
		ID:            uuid.New().String(),
		Level:         req.Level,
		TotalChapters: req.TotalChapters,
		ChapterNumber: req.ChapterNumber,
		Title:         req.Title,
		Content:       req.Content,
		Points:        req.Points,
		CreatorID:     creatorID,
	}
	if err := s.Repo.CreateMaterial(material); err != nil {
		return nil, err
	}
	return material, nil
}

func (s *LearningPathService) ListMaterials(level int, page, limit int) ([]model.LearningPathMaterial, int64, error) {
	return s.Repo.ListMaterials(level, page, limit)
}

func (s *LearningPathService) GetMaterial(id string) (*model.LearningPathMaterial, error) {
	return s.Repo.FindMaterialByID(id)
}

func (s *LearningPathService) UpdateMaterial(id string, req CreateMaterialRequest) (*model.LearningPathMaterial, error) {
	material, err := s.Repo.FindMaterialByID(id)
	if err != nil {
		return nil, err
	}

	material.Level = req.Level
	material.TotalChapters = req.TotalChapters
	material.ChapterNumber = req.ChapterNumber
	material.Title = req.Title
	material.Content = req.Content
	material.Points = req.Points

	if err := s.Repo.UpdateMaterial(material); err != nil {
		return nil, err
	}
	return material, nil
}

func (s *LearningPathService) DeleteMaterial(id string) error {
	return s.Repo.DeleteMaterial(id)
}
