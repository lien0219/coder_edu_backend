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
	Submissions      []ExerciseSubmissionItem `json:"submissions"`
	IsAutoSubmit     bool                     `json:"isAutoSubmit"` // 是否为自动提交
	Duration         int                      `json:"duration"`     // 答题时长（秒）
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

	// 2. 获取提交状态 (如果最新一次提交被驳回，则视为未提交，允许重交)
	var submissions []model.KnowledgePointSubmission
	if err := s.db.Where("user_id = ?", userID).Order("created_at ASC").Find(&submissions).Error; err != nil {
		return nil, err
	}
	submissionMap := make(map[string]bool)
	for _, sub := range submissions {
		// 采用覆盖逻辑，最后一次提交的状态决定了 IsSubmitted 的值
		// 只有在已提交待审核或已通过时，才认为已提交
		submissionMap[sub.KnowledgePointID] = sub.Status == "pending" || sub.Status == "approved"
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

	// 2. 检查记录，仅获取现有状态，不再自动创建
	var submission model.KnowledgePointSubmission
	var submissionDetails interface{}
	var startTime time.Time

	err := s.db.Where("user_id = ? AND knowledge_point_id = ?", userID, id).Order("created_at DESC").First(&submission).Error

	isSubmitted := false
	if err == nil {
		// 如果已提交待审核或已通过，则返回提交详情
		if submission.Status == "pending" || submission.Status == "approved" {
			isSubmitted = true
			var details []interface{}
			if err := json.Unmarshal([]byte(submission.Details), &details); err == nil {
				submissionDetails = details
			}
			startTime = submission.StartedAt
		} else if submission.Status == "draft" {
			// 如果是进行中的草稿，返回其开始时间供前端恢复倒计时
			startTime = submission.StartedAt
		}
	}

	return map[string]interface{}{
		"knowledgePoint":    kp,
		"isCompleted":       isCompleted,
		"isSubmitted":       isSubmitted,
		"submissionDetails": submissionDetails,
		"startTime":         startTime, // 如果没开始答题，则为零值
	}, nil
}

func (s *KnowledgePointService) StartExercises(userID uint, id string) (time.Time, error) {
	// 1. 检查是否已经有正在进行的计时或已提交的记录
	var existing model.KnowledgePointSubmission
	err := s.db.Where("user_id = ? AND knowledge_point_id = ?", userID, id).Order("created_at DESC").First(&existing).Error

	// 2. 如果已经有记录且不是被驳回的状态，则直接返回原有的开始时间（防止重复点按钮重置时间）
	if err == nil && existing.Status != "rejected" {
		return existing.StartedAt, nil
	}

	// 3. 真正的开启计时逻辑：创建草稿记录
	startTime := time.Now()
	newDraft := model.KnowledgePointSubmission{
		ID:               uuid.New().String(),
		UserID:           userID,
		KnowledgePointID: id,
		Status:           "draft",
		StartedAt:        startTime,
		CreatedAt:        startTime,
	}

	if err := s.db.Create(&newDraft).Error; err != nil {
		return time.Time{}, err
	}
	return startTime, nil
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

	// 查找该知识点的“草稿”记录进行更新
	var submission model.KnowledgePointSubmission
	err := s.db.Where("user_id = ? AND knowledge_point_id = ? AND status = ?", userID, req.KnowledgePointID, "draft").Order("created_at DESC").First(&submission).Error

	if err != nil {
		// 如果没找到草稿（极端情况），则创建新记录
		submission = model.KnowledgePointSubmission{
			ID:               uuid.New().String(),
			UserID:           userID,
			KnowledgePointID: req.KnowledgePointID,
			StartedAt:        time.Now(),
		}
	}

	submission.Details = string(detailsJSON)
	submission.Score = totalScore
	submission.Status = "pending"
	submission.IsAutoSubmit = req.IsAutoSubmit
	// 如果前端没传 Duration，后端根据开始时间算一个
	if req.Duration > 0 {
		submission.Duration = req.Duration
	} else {
		submission.Duration = int(time.Since(submission.StartedAt).Seconds())
	}
	submission.CreatedAt = time.Now()

	if err := s.db.Save(&submission).Error; err != nil {
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

		// 当老师保存编辑后，清理掉该知识点下所有“正在进行中”的草稿记录
		// 强制正在答题的学生下次进入时重新同步新版本的题目和计时规则
		if err := tx.Where("knowledge_point_id = ? AND status = ?", id, "draft").Delete(&model.KnowledgePointSubmission{}).Error; err != nil {
			return err
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
		// 1. 删除关联的视频
		if err := tx.Where("knowledge_point_id = ?", id).Delete(&model.KnowledgePointVideo{}).Error; err != nil {
			return err
		}

		// 2. 删除关联的练习题
		if err := tx.Where("knowledge_point_id = ?", id).Delete(&model.KnowledgePointExercise{}).Error; err != nil {
			return err
		}

		// 3. 删除学生的完成状态
		if err := tx.Where("knowledge_point_id = ?", id).Delete(&model.KnowledgePointCompletion{}).Error; err != nil {
			return err
		}

		// 4. 删除学生的提交记录及答题数据
		if err := tx.Where("knowledge_point_id = ?", id).Delete(&model.KnowledgePointSubmission{}).Error; err != nil {
			return err
		}

		// 5. 最后删除知识点本体
		if err := tx.Delete(&model.KnowledgePoint{}, "id = ?", id).Error; err != nil {
			return err
		}

		return nil
	})
}

type SubmissionListResponse struct {
	ID                  string    `json:"id"`
	UserID              uint      `json:"userId"`
	UserName            string    `json:"userName"`
	KnowledgePointID    string    `json:"knowledgePointId"`
	KnowledgePointTitle string    `json:"knowledgePointTitle"`
	Score               int       `json:"score"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"createdAt"`
}

func (s *KnowledgePointService) ListSubmissions(kpID string, status string, studentName string, page int, limit int) ([]SubmissionListResponse, int64, error) {
	// 1. 获取所有学生总数
	var total int64
	studentQuery := s.db.Model(&model.User{}).Where("role = ?", model.Student)
	if studentName != "" {
		studentQuery = studentQuery.Where("name LIKE ?", "%"+studentName+"%")
	}
	studentQuery.Count(&total)

	// 2. 分页获取学生
	var students []model.User
	offset := (page - 1) * limit
	if err := studentQuery.
		Order("id ASC").
		Offset(offset).
		Limit(limit).
		Find(&students).Error; err != nil {
		return nil, 0, err
	}

	var res []SubmissionListResponse

	// 3. 遍历分页后的学生，查询其提交记录
	for _, student := range students {
		var subs []model.KnowledgePointSubmission
		db := s.db.Where("user_id = ?", student.ID)

		if kpID != "" {
			db = db.Where("knowledge_point_id = ?", kpID)
		}
		if status != "" {
			db = db.Where("status = ?", status)
		}

		// 获取该学生匹配条件的提交记录
		if err := db.Order("created_at DESC").Find(&subs).Error; err != nil {
			continue
		}

		if len(subs) > 0 {
			for _, sub := range subs {
				var kp model.KnowledgePoint
				s.db.Select("title").First(&kp, "id = ?", sub.KnowledgePointID)

				res = append(res, SubmissionListResponse{
					ID:                  sub.ID,
					UserID:              student.ID,
					UserName:            student.Name,
					KnowledgePointID:    sub.KnowledgePointID,
					KnowledgePointTitle: kp.Title,
					Score:               sub.Score,
					Status:              sub.Status,
					CreatedAt:           sub.CreatedAt,
				})
			}
		} else if status == "" || status == "unsubmitted" {
			title := "待分配"
			if kpID != "" {
				var kp model.KnowledgePoint
				s.db.Select("title").First(&kp, "id = ?", kpID)
				title = kp.Title
			}

			res = append(res, SubmissionListResponse{
				ID:                  "",
				UserID:              student.ID,
				UserName:            student.Name,
				KnowledgePointID:    kpID,
				KnowledgePointTitle: title,
				Score:               0,
				Status:              "unsubmitted",
				CreatedAt:           time.Time{},
			})
		}
	}

	return res, total, nil
}

func (s *KnowledgePointService) GetSubmissionDetail(id string) (*model.KnowledgePointSubmission, error) {
	var sub model.KnowledgePointSubmission
	if err := s.db.First(&sub, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *KnowledgePointService) AuditSubmission(id string, status string, manualScore *int) error {
	if status != "approved" && status != "rejected" {
		return fmt.Errorf("invalid status")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var sub model.KnowledgePointSubmission
		if err := tx.First(&sub, "id = ?", id).Error; err != nil {
			return err
		}

		// 记录原始状态，用于判断是否需要发放积分
		oldStatus := sub.Status

		// 如果老师提供了手动分数，更新提交记录中的分数
		finalScore := sub.Score
		if manualScore != nil {
			finalScore = *manualScore
			if err := tx.Model(&sub).Update("score", finalScore).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&sub).Update("status", status).Error; err != nil {
			return err
		}

		// 如果审核通过，且之前不是已通过状态，则更新完成状态并按最终分数发放积分
		if status == "approved" && oldStatus != "approved" {
			completion := model.KnowledgePointCompletion{
				UserID:           sub.UserID,
				KnowledgePointID: sub.KnowledgePointID,
				IsCompleted:      true,
				CompletedAt:      time.Now(),
			}
			if err := tx.Save(&completion).Error; err != nil {
				return err
			}

			var user model.User
			if err := tx.First(&user, sub.UserID).Error; err != nil {
				return err
			}
			user.Points += finalScore // 发放到独立积分系统
			if err := tx.Save(&user).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
