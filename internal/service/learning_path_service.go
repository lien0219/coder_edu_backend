package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/util"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

// StudentPathListParams 学生学习路径分页与筛选
type StudentPathListParams struct {
	Page      int
	Limit     int
	Search    string
	Level     int // 1-4；0 表示不限
	Unlocked  *bool
	Completed *bool
}

type StudentPathResult struct {
	Items           []StudentMaterialResponse   `json:"items"`
	Total           int64                       `json:"total"`
	Recommendations MaterialRecommendationBlock `json:"recommendations"`
}

func (s *LearningPathService) GetStudentPath(userID uint, p StudentPathListParams) (*StudentPathResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 10
	}

	confirmed, err := s.loadConfirmedDiagnosis(userID)
	if err != nil {
		return nil, err
	}
	recommendedLevel := recommendedLevelOf(confirmed)

	completions, _ := s.Repo.GetUserCompletions(userID)
	completedMap := make(map[string]bool)
	for _, c := range completions {
		completedMap[c.MaterialID] = true
	}

	recs, recErr := s.BuildMaterialRecommendations(userID, confirmed, completedMap)
	if recErr != nil {
		return nil, recErr
	}

	levelFilter := p.Level
	if !(levelFilter >= 1 && levelFilter <= model.LearningLevelAdvanced) {
		levelFilter = 0
	}
	search := strings.TrimSpace(p.Search)
	materials, err := s.Repo.FindMaterialsForStudentListing(levelFilter, search)
	if err != nil {
		return nil, err
	}

	res := make([]StudentMaterialResponse, 0, len(materials))
	for _, m := range materials {
		res = append(res, StudentMaterialResponse{
			ID:            m.ID,
			Title:         m.Title,
			Level:         m.Level,
			Points:        m.Points,
			ChapterNumber: m.ChapterNumber,
			IsUnlocked:    m.Level <= recommendedLevel && recommendedLevel > 0,
			IsCompleted:   completedMap[m.ID],
		})
	}

	var filtered []StudentMaterialResponse
	for _, row := range res {
		if p.Unlocked != nil && row.IsUnlocked != *p.Unlocked {
			continue
		}
		if p.Completed != nil && row.IsCompleted != *p.Completed {
			continue
		}
		filtered = append(filtered, row)
	}

	total := int64(len(filtered))
	offset := (p.Page - 1) * p.Limit
	items := []StudentMaterialResponse{}
	if offset < len(filtered) {
		end := offset + p.Limit
		if end > len(filtered) {
			end = len(filtered)
		}
		items = filtered[offset:end]
	}
	return &StudentPathResult{
		Items:           items,
		Total:           total,
		Recommendations: recs,
	}, nil
}

type CreateMaterialRequest struct {
	Level             int       `json:"level" binding:"required"`
	TotalChapters     int       `json:"totalChapters"`
	ChapterNumber     int       `json:"chapterNumber"`
	Title             string    `json:"title" binding:"required"`
	Content           string    `json:"content" binding:"required"`
	Points            int       `json:"points"`
	KnowledgePointIDs *[]string `json:"knowledgePointIds"` // nil=不改关联；[]=清空
}

type MaterialView struct {
	model.LearningPathMaterial
	KnowledgePointIDs []string `json:"knowledgePointIds"`
}

func (s *LearningPathService) attachMaterialKnowledgePoints(ms []model.LearningPathMaterial) ([]MaterialView, error) {
	ids := make([]string, len(ms))
	for i, m := range ms {
		ids[i] = m.ID
	}
	linked, err := s.Repo.ListKnowledgePointIDsByMaterialIDs(ids)
	if err != nil {
		return nil, err
	}
	views := make([]MaterialView, len(ms))
	for i, m := range ms {
		kps := linked[m.ID]
		if kps == nil {
			kps = []string{}
		}
		views[i] = MaterialView{LearningPathMaterial: m, KnowledgePointIDs: kps}
	}
	return views, nil
}

func (s *LearningPathService) CreateMaterial(creatorID uint, req CreateMaterialRequest) (*MaterialView, error) {
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
	err := s.Repo.WithTx(func(tx *repository.LearningPathRepository) error {
		if err := tx.CreateMaterial(material); err != nil {
			return err
		}
		if req.KnowledgePointIDs != nil {
			return tx.ReplaceMaterialKnowledgePoints(material.ID, *req.KnowledgePointIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	views, err := s.attachMaterialKnowledgePoints([]model.LearningPathMaterial{*material})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

func (s *LearningPathService) ListMaterials(level int, page, limit int) ([]MaterialView, int64, error) {
	ms, total, err := s.Repo.ListMaterials(level, page, limit)
	if err != nil {
		return nil, 0, err
	}
	views, err := s.attachMaterialKnowledgePoints(ms)
	if err != nil {
		return nil, 0, err
	}
	return views, total, nil
}

func (s *LearningPathService) GetMaterial(id string) (*MaterialView, error) {
	material, err := s.Repo.FindMaterialByID(id)
	if err != nil {
		return nil, err
	}
	views, err := s.attachMaterialKnowledgePoints([]model.LearningPathMaterial{*material})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

func (s *LearningPathService) UpdateMaterial(id string, req CreateMaterialRequest) (*MaterialView, error) {
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

	err = s.Repo.WithTx(func(tx *repository.LearningPathRepository) error {
		if err := tx.UpdateMaterial(material); err != nil {
			return err
		}
		if req.KnowledgePointIDs != nil {
			return tx.ReplaceMaterialKnowledgePoints(material.ID, *req.KnowledgePointIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	views, err := s.attachMaterialKnowledgePoints([]model.LearningPathMaterial{*material})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

func (s *LearningPathService) DeleteMaterial(id string) error {
	return s.Repo.DeleteMaterial(id)
}

type MaterialDetailResponse struct {
	model.LearningPathMaterial
	IsCompleted bool `json:"isCompleted"`
}

func (s *LearningPathService) GetMaterialsByLevel(userID uint, level int) ([]MaterialDetailResponse, error) {
	confirmed, err := s.loadConfirmedDiagnosis(userID)
	if err != nil {
		return nil, err
	}
	recommendedLevel := recommendedLevelOf(confirmed)

	if level < model.LearningLevelBasic || level > model.LearningLevelAdvanced || level > recommendedLevel {
		return nil, nil
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

func (s *LearningPathService) assertMaterialAccessible(userID uint, material *model.LearningPathMaterial) error {
	if material == nil {
		return util.ErrResourceNotFound
	}
	confirmed, err := s.loadConfirmedDiagnosis(userID)
	if err != nil {
		return err
	}
	level := recommendedLevelOf(confirmed)
	if level <= 0 || material.Level > level {
		return util.ErrMaterialNotAccessible
	}
	return nil
}

func (s *LearningPathService) CompleteMaterial(userID uint, materialID string) error {
	material, err := s.Repo.FindMaterialByID(materialID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return util.ErrResourceNotFound
		}
		return err
	}

	existing, err := s.Repo.FindCompletion(userID, materialID)
	if err == nil && existing != nil && existing.ID != 0 {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if err := s.assertMaterialAccessible(userID, material); err != nil {
		return err
	}

	return s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		pathTx := &repository.LearningPathRepository{DB: tx}
		replay, findErr := pathTx.FindCompletion(userID, materialID)
		if findErr == nil && replay != nil && replay.ID != 0 {
			return nil
		}
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		completion := &model.LearningPathCompletion{
			UserID:      userID,
			MaterialID:  materialID,
			CompletedAt: time.Now(),
		}
		if createErr := pathTx.CreateCompletion(completion); createErr != nil {
			if isLearningPathCompletionConflict(createErr) {
				return nil
			}
			return createErr
		}

		if material.Points <= 0 {
			return nil
		}

		logTx := &repository.LearningLogRepository{DB: tx}
		log := &model.LearningLog{
			UserID:    userID,
			Activity:  "learning_path_complete",
			Content:   fmt.Sprintf("完成了资料学习: %s", material.Title),
			Score:     material.Points,
			Completed: true,
		}
		if logErr := logTx.Create(log); logErr != nil {
			return logErr
		}

		userTx := &repository.UserRepository{DB: tx}
		return userTx.AddXP(userID, material.Points)
	})
}

type RecordLearningTimeRequest struct {
	Duration int `json:"duration" binding:"required,min=1"`
}

func (s *LearningPathService) RecordLearningTime(userID uint, materialID string, duration int) error {
	material, err := s.Repo.FindMaterialByID(materialID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return util.ErrResourceNotFound
		}
		return err
	}
	if err := s.assertMaterialAccessible(userID, material); err != nil {
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

// isLearningPathCompletionConflict 仅识别 (user_id, material_id) 目标唯一索引冲突。
// 其他唯一冲突、死锁、锁等待超时、连接错误不得当作“已完成”。
func isLearningPathCompletionConflict(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062 &&
			strings.Contains(mysqlErr.Message, "idx_learning_path_completion_user_material")
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "deadlock") || strings.Contains(msg, "lock wait timeout") {
		return false
	}
	hasIndex := strings.Contains(msg, "idx_learning_path_completion_user_material")
	hasCols := strings.Contains(msg, "learning_path_completions") &&
		strings.Contains(msg, "user_id") &&
		strings.Contains(msg, "material_id")
	if !hasIndex && !hasCols {
		return false
	}
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique") ||
		strings.Contains(msg, "constraint failed")
}
