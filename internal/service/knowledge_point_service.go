package service

import (
	"coder_edu_backend/internal/model"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KnowledgePointService struct {
	db *gorm.DB
}

func NewKnowledgePointService(db *gorm.DB) *KnowledgePointService {
	return &KnowledgePointService{db: db}
}

type CreateVideoResourceRequest struct {
	ID          string `json:"id"` // Temporary ID from frontend
	Title       string `json:"title" binding:"required"`
	URL         string `json:"url" binding:"required"`
	Description string `json:"description"`
}

type CreateExerciseRequest struct {
	ID          string             `json:"id"` // Temporary ID from frontend
	Type        model.ExerciseType `json:"type" binding:"required"`
	Question    string             `json:"question" binding:"required"`
	Options     []string           `json:"options"`
	Answer      string             `json:"answer" binding:"required"`
	Explanation string             `json:"explanation"`
	Points      int                `json:"points"`
}

type CreateKnowledgePointRequest struct {
	Title           string                       `json:"title" binding:"required"`
	Description     string                       `json:"description"`
	Type            model.KnowledgePointType     `json:"type" binding:"required"`
	ArticleContent  string                       `json:"articleContent"`
	TimeLimit       int                          `json:"timeLimit"`
	Order           int                          `json:"order"`
	CompletionScore int                          `json:"completionScore"`
	Videos          []CreateVideoResourceRequest `json:"videos"`
	Exercises       []CreateExerciseRequest      `json:"exercises"`
}

func (s *KnowledgePointService) CreateKnowledgePoint(req CreateKnowledgePointRequest) (*model.KnowledgePoint, error) {
	kp := &model.KnowledgePoint{
		ID:              uuid.New().String(),
		Title:           req.Title,
		Description:     req.Description,
		Type:            req.Type,
		ArticleContent:  req.ArticleContent,
		TimeLimit:       req.TimeLimit,
		Order:           req.Order,
		CompletionScore: req.CompletionScore,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create Knowledge Point
		if err := tx.Create(kp).Error; err != nil {
			return err
		}

		// 2. Create Videos
		for _, v := range req.Videos {
			video := model.KnowledgePointVideo{
				ID:               uuid.New().String(),
				KnowledgePointID: kp.ID,
				Title:            v.Title,
				URL:              v.URL,
				Description:      v.Description,
			}
			if err := tx.Create(&video).Error; err != nil {
				return err
			}
			kp.Videos = append(kp.Videos, video)
		}

		// 3. Create Exercises
		for _, ex := range req.Exercises {
			optionsJSON, _ := json.Marshal(ex.Options)
			exercise := model.KnowledgePointExercise{
				ID:               uuid.New().String(),
				KnowledgePointID: kp.ID,
				Type:             ex.Type,
				Question:         ex.Question,
				Options:          string(optionsJSON),
				Answer:           ex.Answer,
				Explanation:      ex.Explanation,
				Points:           ex.Points,
			}
			if err := tx.Create(&exercise).Error; err != nil {
				return err
			}
			kp.Exercises = append(kp.Exercises, exercise)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return kp, nil
}

func (s *KnowledgePointService) ListKnowledgePoints(title string) ([]model.KnowledgePoint, error) {
	var kps []model.KnowledgePoint
	db := s.db.Preload("Videos").Preload("Exercises")

	if title != "" {
		db = db.Where("title LIKE ?", "%"+title+"%")
	}

	if err := db.Order("`order` ASC, created_at DESC").Find(&kps).Error; err != nil {
		return nil, err
	}

	return kps, nil
}

func (s *KnowledgePointService) UpdateKnowledgePoint(id string, req CreateKnowledgePointRequest) (*model.KnowledgePoint, error) {
	var kp model.KnowledgePoint
	if err := s.db.First(&kp, "id = ?", id).Error; err != nil {
		return nil, err
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"title":            req.Title,
			"description":      req.Description,
			"type":             req.Type,
			"article_content":  req.ArticleContent,
			"time_limit":       req.TimeLimit,
			"order":            req.Order,
			"completion_score": req.CompletionScore,
		}
		if err := tx.Model(&kp).Updates(updates).Error; err != nil {
			return err
		}

		if err := tx.Where("knowledge_point_id = ?", id).Delete(&model.KnowledgePointVideo{}).Error; err != nil {
			return err
		}
		kp.Videos = nil
		for _, v := range req.Videos {
			video := model.KnowledgePointVideo{
				ID:               uuid.New().String(),
				KnowledgePointID: id,
				Title:            v.Title,
				URL:              v.URL,
				Description:      v.Description,
			}
			if err := tx.Create(&video).Error; err != nil {
				return err
			}
			kp.Videos = append(kp.Videos, video)
		}

		if err := tx.Where("knowledge_point_id = ?", id).Delete(&model.KnowledgePointExercise{}).Error; err != nil {
			return err
		}
		kp.Exercises = nil
		for _, ex := range req.Exercises {
			optionsJSON, _ := json.Marshal(ex.Options)
			exercise := model.KnowledgePointExercise{
				ID:               uuid.New().String(),
				KnowledgePointID: id,
				Type:             ex.Type,
				Question:         ex.Question,
				Options:          string(optionsJSON),
				Answer:           ex.Answer,
				Explanation:      ex.Explanation,
				Points:           ex.Points,
			}
			if err := tx.Create(&exercise).Error; err != nil {
				return err
			}
			kp.Exercises = append(kp.Exercises, exercise)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &kp, nil
}

func (s *KnowledgePointService) DeleteKnowledgePoint(id string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("knowledge_point_id = ?", id).Delete(&model.KnowledgePointVideo{}).Error; err != nil {
			return err
		}

		if err := tx.Where("knowledge_point_id = ?", id).Delete(&model.KnowledgePointExercise{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(&model.KnowledgePoint{}, "id = ?", id).Error; err != nil {
			return err
		}

		return nil
	})
}
