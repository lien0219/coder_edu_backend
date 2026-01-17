package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type LearningPathService struct {
	Repo            *repository.LearningPathRepository
	AssessmentRepo  *repository.AssessmentRepository
	LearningLogRepo *repository.LearningLogRepository
	UserRepo        *repository.UserRepository
}

func NewLearningPathService(
	repo *repository.LearningPathRepository,
	assessmentRepo *repository.AssessmentRepository,
	learningLogRepo *repository.LearningLogRepository,
	userRepo *repository.UserRepository,
) *LearningPathService {
	return &LearningPathService{
		Repo:            repo,
		AssessmentRepo:  assessmentRepo,
		LearningLogRepo: learningLogRepo,
		UserRepo:        userRepo,
	}
}

type StudentMaterialResponse struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Level         int    `json:"level"`
	Points        int    `json:"points"`
	IsUnlocked    bool   `json:"isUnlocked"`
	IsCompleted   bool   `json:"isCompleted"`
	ChapterNumber int    `json:"chapterNumber"`
}

func (s *LearningPathService) GetStudentPath(userID uint) ([]StudentMaterialResponse, error) {
	// 1. 获取学生的学前测试建议等级
	var recommendedLevel int
	// 先获取默认评估 ID
	var assessmentID uint = 1 // 默认
	as, _, err := s.AssessmentRepo.ListAssessments(1, 1)
	if err == nil && len(as) > 0 {
		assessmentID = as[0].ID
	}

	submission, err := s.AssessmentRepo.FindSubmissionByUserAndAssessment(userID, assessmentID)
	if err == nil && submission != nil {
		recommendedLevel = submission.RecommendedLevel
	}

	// 获取用户已完成的记录
	completions, _ := s.Repo.GetUserCompletions(userID)
	completedMap := make(map[string]bool)
	for _, c := range completions {
		completedMap[c.MaterialID] = true
	}

	// 2. 获取所有学习资料
	materials, _, err := s.Repo.ListMaterials(0, 1, 1000) // 获取全部
	if err != nil {
		return nil, err
	}

	// 3. 构建返回列表并设置解锁状态
	res := make([]StudentMaterialResponse, len(materials))
	for i, m := range materials {
		res[i] = StudentMaterialResponse{
			ID:            m.ID,
			Title:         m.Title,
			Level:         m.Level,
			Points:        m.Points,
			ChapterNumber: m.ChapterNumber,
			IsUnlocked:    m.Level <= recommendedLevel && recommendedLevel > 0,
			IsCompleted:   completedMap[m.ID],
		}
	}

	return res, nil
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

type MaterialDetailResponse struct {
	model.LearningPathMaterial
	IsCompleted bool `json:"isCompleted"`
}

func (s *LearningPathService) GetMaterialsByLevel(userID uint, level int) ([]MaterialDetailResponse, error) {
	// 1. 检查权限：只有当学生建议等级 >= 请求等级时，才允许获取详细内容
	var recommendedLevel int
	var assessmentID uint = 1
	as, _, err := s.AssessmentRepo.ListAssessments(1, 1)
	if err == nil && len(as) > 0 {
		assessmentID = as[0].ID
	}

	submission, err := s.AssessmentRepo.FindSubmissionByUserAndAssessment(userID, assessmentID)
	if err == nil && submission != nil {
		recommendedLevel = submission.RecommendedLevel
	}

	if level > recommendedLevel {
		return nil, nil // 或者返回一个特定的错误，表示未解锁
	}

	// 获取用户已完成的记录
	completions, _ := s.Repo.GetUserCompletions(userID)
	completedMap := make(map[string]bool)
	for _, c := range completions {
		completedMap[c.MaterialID] = true
	}

	// 2. 获取该等级的所有资料
	materials, _, err := s.Repo.ListMaterials(level, 1, 1000)
	if err != nil {
		return nil, err
	}

	res := make([]MaterialDetailResponse, len(materials))
	for i, m := range materials {
		res[i] = MaterialDetailResponse{
			LearningPathMaterial: m,
			IsCompleted:          completedMap[m.ID],
		}
	}

	return res, nil
}

func (s *LearningPathService) CompleteMaterial(userID uint, materialID string) error {
	// 1. 检查是否已经完成过
	existing, err := s.Repo.FindCompletion(userID, materialID)
	if err == nil && existing != nil {
		return nil // 已经完成过了
	}

	// 2. 获取资料信息以获取积分
	material, err := s.Repo.FindMaterialByID(materialID)
	if err != nil {
		return err
	}

	// 3. 创建完成记录
	completion := &model.LearningPathCompletion{
		UserID:      userID,
		MaterialID:  materialID,
		CompletedAt: time.Now(),
	}

	if err := s.Repo.CreateCompletion(completion); err != nil {
		return err
	}

	// 4. 奖励积分 (如果 Points > 0)
	if material.Points > 0 {
		log := &model.LearningLog{
			UserID:    userID,
			Activity:  "learning_path_complete",
			Content:   fmt.Sprintf("完成了资料学习: %s", material.Title),
			Score:     material.Points,
			Completed: true,
		}
		_ = s.LearningLogRepo.Create(log)

		// 显式更新用户表中的 XP 字段
		_ = s.UserRepo.UpdateXP(userID, material.Points)
	}

	return nil
}

type RecordLearningTimeRequest struct {
	Duration int `json:"duration" binding:"required,min=1"`
}

func (s *LearningPathService) RecordLearningTime(userID uint, materialID string, duration int) error {
	material, err := s.Repo.FindMaterialByID(materialID)
	if err != nil {
		return err
	}

	log := &model.LearningLog{
		UserID:   userID,
		Activity: "learning_path_material",
		Content:  fmt.Sprintf("学习了资料: %s", material.Title),
		Duration: duration,
	}

	return s.LearningLogRepo.Create(log)
}
