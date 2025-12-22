package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
)

type KnowledgeTagService struct {
	Repo *repository.KnowledgeTagRepository
}

func NewKnowledgeTagService(repo *repository.KnowledgeTagRepository) *KnowledgeTagService {
	return &KnowledgeTagService{Repo: repo}
}

func (s *KnowledgeTagService) ListTags() ([]model.KnowledgeTag, error) {
	return s.Repo.FindAll()
}

func (s *KnowledgeTagService) GetTagsByIDs(ids []uint) ([]model.KnowledgeTag, error) {
	return s.Repo.FindByIDs(ids)
}
