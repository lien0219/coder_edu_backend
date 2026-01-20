package service

import (
	"coder_edu_backend/internal/model"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

type ExerciseSubmissionItem struct {
	ExerciseID      string `json:"exerciseId" binding:"required"`
	Answer          string `json:"answer"`          // 普通题答案
	Code            string `json:"code"`            // 编程题代码
	ExecutionResult string `json:"executionResult"` // 编程题执行结果
}

type SubmitKnowledgePointExercisesRequest struct {
	KnowledgePointID string                   `json:"knowledgePointId" binding:"required"`
	Submissions      []ExerciseSubmissionItem `json:"submissions" binding:"required"`
}

type KnowledgePointStudentResponse struct {
	ID              string                   `json:"id"`
	Title           string                   `json:"title"`
	Description     string                   `json:"description"`
	Type            model.KnowledgePointType `json:"type"`
	Order           int                      `json:"order"`
	IsCompleted     bool                     `json:"isCompleted"`
	IsSubmitted     bool                     `json:"isSubmitted"`
	CompletionScore int                      `json:"completionScore"`
}

func (s *KnowledgePointService) ListKnowledgePointsForStudent(userID uint) ([]KnowledgePointStudentResponse, error) {
	var kps []model.KnowledgePoint
	if err := s.db.Order("`order` ASC, created_at DESC").Find(&kps).Error; err != nil {
		return nil, err
	}

	// 1. 获取完成状态 (老师审核通过)
	var completions []model.KnowledgePointCompletion
	if err := s.db.Where("user_id = ?", userID).Find(&completions).Error; err != nil {
		return nil, err
	}
	completionMap := make(map[string]bool)
	for _, c := range completions {
		completionMap[c.KnowledgePointID] = c.IsCompleted
	}

	// 2. 获取提交状态 (学生是否点击过提交)
	var submissions []model.KnowledgePointSubmission
	if err := s.db.Where("user_id = ?", userID).Find(&submissions).Error; err != nil {
		return nil, err
	}
	submissionMap := make(map[string]bool)
	for _, sub := range submissions {
		submissionMap[sub.KnowledgePointID] = true
	}

	var resp []KnowledgePointStudentResponse
	for _, kp := range kps {
		resp = append(resp, KnowledgePointStudentResponse{
			ID:              kp.ID,
			Title:           kp.Title,
			Description:     kp.Description,
			Type:            kp.Type,
			Order:           kp.Order,
			IsCompleted:     completionMap[kp.ID],
			IsSubmitted:     submissionMap[kp.ID],
			CompletionScore: kp.CompletionScore,
		})
	}

	return resp, nil
}

func (s *KnowledgePointService) GetKnowledgePointForStudent(id string, userID uint) (interface{}, error) {
	var kp model.KnowledgePoint
	if err := s.db.Preload("Videos").Preload("Exercises").First(&kp, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// 1. 检查是否最终完成 (老师审核通过后的状态)
	var completion model.KnowledgePointCompletion
	isCompleted := s.db.Where("user_id = ? AND knowledge_point_id = ? AND is_completed = ?", userID, id, true).First(&completion).Error == nil

	// 2. 检查并获取提交记录
	var submission model.KnowledgePointSubmission
	var submissionDetails interface{}
	isSubmitted := s.db.Where("user_id = ? AND knowledge_point_id = ?", userID, id).Order("created_at DESC").First(&submission).Error == nil

	if isSubmitted {
		// 解析 Details 字段
		var details []interface{}
		if err := json.Unmarshal([]byte(submission.Details), &details); err == nil {
			submissionDetails = details
		}
	}

	return map[string]interface{}{
		"knowledgePoint":    kp,
		"isCompleted":       isCompleted,
		"isSubmitted":       isSubmitted,
		"submissionDetails": submissionDetails, // 如果没提交则为 null
	}, nil
}

func (s *KnowledgePointService) SubmitExercises(userID uint, req SubmitKnowledgePointExercisesRequest) (interface{}, error) {
	var kp model.KnowledgePoint
	if err := s.db.Preload("Exercises").First(&kp, "id = ?", req.KnowledgePointID).Error; err != nil {
		return nil, err
	}

	totalScore := 0

	type DetailedResult struct {
		ExerciseID      string `json:"exerciseId"`
		Question        string `json:"question"`
		Type            string `json:"type"`
		UserAnswer      string `json:"userAnswer"`
		CorrectAnswer   string `json:"correctAnswer"`
		Code            string `json:"code,omitempty"`
		ExecutionResult string `json:"executionResult,omitempty"`
		IsCorrect       bool   `json:"isCorrect"`
		Points          int    `json:"points"`
	}

	detailedResults := make([]DetailedResult, 0)
	submissionMap := make(map[string]ExerciseSubmissionItem)
	for _, sub := range req.Submissions {
		submissionMap[sub.ExerciseID] = sub
	}

	for _, ex := range kp.Exercises {
		sub := submissionMap[ex.ID]
		isCorrect := false

		// 判定逻辑：对所有答案都进行去除首尾空格处理
		if ex.Type == model.ExProgramming {
			// 编程题对比执行结果
			isCorrect = strings.TrimSpace(sub.ExecutionResult) == strings.TrimSpace(ex.Answer)
		} else {
			// 普通题对比答案字段 (如 T/F, A, B 等)
			isCorrect = strings.TrimSpace(sub.Answer) == strings.TrimSpace(ex.Answer)
		}

		points := 0
		if isCorrect {
			points = ex.Points
			totalScore += points
		}

		detailedResults = append(detailedResults, DetailedResult{
			ExerciseID:      ex.ID,
			Question:        ex.Question,
			Type:            string(ex.Type),
			UserAnswer:      sub.Answer,
			CorrectAnswer:   ex.Answer,
			Code:            sub.Code,
			ExecutionResult: sub.ExecutionResult,
			IsCorrect:       isCorrect,
			Points:          points,
		})
	}

	detailsJSON, _ := json.Marshal(detailedResults)
	submission := model.KnowledgePointSubmission{
		ID:               uuid.New().String(),
		UserID:           userID,
		KnowledgePointID: kp.ID,
		Details:          string(detailsJSON),
		Score:            totalScore,
		Status:           "pending",
		CreatedAt:        time.Now(),
	}

	if err := s.db.Create(&submission).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"score":   totalScore,
		"results": detailedResults,
		"message": "提交成功，系统已初步校验，请等待老师最终审核",
	}, nil
}

func (s *KnowledgePointService) RecordLearningTime(userID uint, id string, duration int) error {
	var kp model.KnowledgePoint
	if err := s.db.First(&kp, "id = ?", id).Error; err != nil {
		return err
	}

	log := &model.LearningLog{
		UserID:   userID,
		Activity: "knowledge_point",
		Content:  fmt.Sprintf("学习了知识点: %s", kp.Title),
		Duration: duration,
	}

	return s.db.Create(log).Error
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
