package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"

	"gorm.io/gorm"
)

// FlexibleTime 自定义时间类型，能够处理不完整的时间格式
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON 实现自定义的 JSON 解析逻辑
func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	// 去掉引号
	str := strings.Trim(string(data), `"`)
	if str == "" || str == "null" {
		ft.Time = time.Time{}
		return nil
	}

	// 尝试不同的时间格式
	formats := []string{
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,      // "2006-01-02T15:04:05.999999999Z07:00"
		"2006-01-02T15:04:05", // 没有时区
		"2006-01-02T15:04",    // 只有年月日时分
		"2006-01-02 15:04:05", // 空格分隔，没有时区
		"2006-01-02 15:04",    // 空格分隔，只有年月日时分
		"2006-01-02",          // 只有日期
	}

	var err error
	for _, format := range formats {
		ft.Time, err = time.Parse(format, str)
		if err == nil {
			// 如果解析成功，但格式不完整，需要补充默认值
			if format == "2006-01-02T15:04" {
				// 补充秒数和本地时区
				ft.Time = time.Date(ft.Time.Year(), ft.Time.Month(), ft.Time.Day(),
					ft.Time.Hour(), ft.Time.Minute(), 0, 0, time.Local)
			} else if format == "2006-01-02 15:04" {
				ft.Time = time.Date(ft.Time.Year(), ft.Time.Month(), ft.Time.Day(),
					ft.Time.Hour(), ft.Time.Minute(), 0, 0, time.Local)
			} else if format == "2006-01-02" {
				ft.Time = time.Date(ft.Time.Year(), ft.Time.Month(), ft.Time.Day(),
					0, 0, 0, 0, time.Local)
			}
			return nil
		}
	}

	return err
}

// MarshalJSON 实现自定义的 JSON 序列化
func (ft FlexibleTime) MarshalJSON() ([]byte, error) {
	if ft.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(ft.Time.Format(time.RFC3339))
}

// TimePtr 返回指向内部 time.Time 的指针，如果时间为零值则返回 nil
func (ft *FlexibleTime) TimePtr() *time.Time {
	if ft == nil || ft.Time.IsZero() {
		return nil
	}
	return &ft.Time
}

type LevelService struct {
	LevelRepo *repository.LevelRepository
	DB        *gorm.DB
}

func NewLevelService(levelRepo *repository.LevelRepository, db *gorm.DB) *LevelService {
	return &LevelService{
		LevelRepo: levelRepo,
		DB:        db,
	}
}

type LevelQuestionRequest struct {
	QuestionType  string      `json:"questionType"`
	Content       interface{} `json:"content"`
	Options       interface{} `json:"options,omitempty"`
	CorrectAnswer interface{} `json:"correctAnswer,omitempty"`
	Points        int         `json:"points"`
	ScoringRule   string      `json:"scoringRule,omitempty"`
	Weight        int         `json:"weight,omitempty"`
	ManualGrading bool        `json:"manualGrading,omitempty"`
	Explanation   string      `json:"explanation,omitempty"`
}

// LevelFullResponse 包含关卡完整信息的响应结构体
type LevelFullResponse struct {
	ID                 uint                    `json:"id"`
	CreatedAt          time.Time               `json:"createdAt"`
	UpdatedAt          time.Time               `json:"updatedAt"`
	CreatorID          uint                    `json:"creatorId"`
	Title              string                  `json:"title"`
	Description        string                  `json:"description"`
	CoverURL           string                  `json:"coverUrl"`
	Difficulty         string                  `json:"difficulty"`
	EstimatedMinutes   int                     `json:"estimatedMinutes"`
	AttemptLimit       int                     `json:"attemptLimit"`
	PassingScore       int                     `json:"passingScore"`
	BasePoints         int                     `json:"basePoints"`
	AllowPause         bool                    `json:"allowPause"`
	LevelType          string                  `json:"levelType"`
	IsPublished        bool                    `json:"isPublished"`
	PublishedAt        *time.Time              `json:"publishedAt,omitempty"`
	ScheduledPublishAt *time.Time              `json:"scheduledPublishAt,omitempty"`
	VisibleScope       string                  `json:"visibleScope"`
	VisibleTo          json.RawMessage         `json:"visibleTo"`
	AvailableFrom      *time.Time              `json:"availableFrom,omitempty"`
	AvailableTo        *time.Time              `json:"availableTo,omitempty"`
	CurrentVersion     uint                    `json:"currentVersion"`
	Abilities          []uint                  `json:"abilityIds"`
	KnowledgeTags      []uint                  `json:"knowledgeTagIds"`
	Questions          []LevelQuestionResponse `json:"questions"`
}

// LevelQuestionResponse 题目完整信息响应结构体
type LevelQuestionResponse struct {
	ID            uint            `json:"id"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
	LevelID       uint            `json:"levelId"`
	QuestionType  string          `json:"questionType"`
	Content       json.RawMessage `json:"content"`
	Options       json.RawMessage `json:"options"`
	CorrectAnswer json.RawMessage `json:"correctAnswer"`
	Points        int             `json:"points"`
	Weight        int             `json:"weight"`
	ManualGrading bool            `json:"manualGrading"`
	Order         int             `json:"order"`
	ScoringRule   string          `json:"scoringRule"`
	Explanation   string          `json:"explanation"`
	CodeTemplate  string          `json:"codeTemplate"`
}

// StudentLevelResponse 学生端关卡列表响应结构体
type StudentLevelResponse struct {
	ID               uint      `json:"id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	CoverURL         string    `json:"coverUrl"`
	Difficulty       string    `json:"difficulty"`
	EstimatedMinutes int       `json:"estimatedMinutes"`
	AttemptLimit     int       `json:"attemptLimit"`
	PassingScore     int       `json:"passingScore"`
	BasePoints       int       `json:"basePoints"`     // 积分奖励分数（所有题目积分总和）
	QuestionsCount   int       `json:"questionsCount"` // 题目数量
	Status           string    `json:"status"`         // "not_started", "in_progress", "completed"
	BestScore        int       `json:"bestScore,omitempty"`
	AttemptsUsed     int       `json:"attemptsUsed,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

// StudentLevelDetailResponse 学生端关卡详情响应结构体
type StudentLevelDetailResponse struct {
	// 基础信息
	ID               uint   `json:"id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	CoverURL         string `json:"coverUrl"`
	Difficulty       string `json:"difficulty"`
	EstimatedMinutes int    `json:"estimatedMinutes"`
	AttemptLimit     int    `json:"attemptLimit"`
	PassingScore     int    `json:"passingScore"`
	BasePoints       int    `json:"basePoints"`
	QuestionsCount   int    `json:"questionsCount"`

	// 关卡元数据
	UpdatedAt          *time.Time    `json:"updatedAt,omitempty"`
	Author             *UserInfo     `json:"author,omitempty"`
	Abilities          []AbilityInfo `json:"abilities,omitempty"` // 能力分类详细信息
	Tags               []TagInfo     `json:"tags,omitempty"`
	Prerequisites      []string      `json:"prerequisites"`
	LearningObjectives []string      `json:"learningObjectives"`

	// 统计数据
	TotalAttempts  int     `json:"totalAttempts"`  // 总挑战次数
	AverageScore   float64 `json:"averageScore"`   // 平均分数
	CompletionRate float64 `json:"completionRate"` // 完成率

	// 用户进度
	Status        string     `json:"status"` // "not_started", "in_progress", "completed"
	BestScore     int        `json:"bestScore,omitempty"`
	AttemptsUsed  int        `json:"attemptsUsed,omitempty"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`   // 完成时间
	LastAttemptAt *time.Time `json:"lastAttemptAt,omitempty"` // 最后尝试时间
	TimeSpent     int        `json:"timeSpent,omitempty"`     // 用时(秒)

	// 题目信息
	Questions []StudentQuestionResponse `json:"questions"`
	CreatedAt time.Time                 `json:"createdAt"`
}

// UserInfo 用户信息结构体
type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// TagInfo 标签信息结构体
type TagInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// AbilityInfo 能力信息结构体
type AbilityInfo struct {
	ID          uint   `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// StudentQuestionResponse 学生端题目信息响应结构体（不包含答案）
type StudentQuestionResponse struct {
	ID           uint            `json:"id"`
	Title        string          `json:"title"` // 题目标题
	QuestionType string          `json:"questionType"`
	Content      json.RawMessage `json:"content"`
	Options      json.RawMessage `json:"options,omitempty"`
	Points       int             `json:"points"`
	Weight       int             `json:"weight"`
	Order        int             `json:"order"`
	Difficulty   string          `json:"difficulty,omitempty"` // 题目难度
}

type LevelCreateRequest struct {
	Title            string                 `json:"title" binding:"required"`
	Description      string                 `json:"description"`
	CoverURL         string                 `json:"coverUrl"`
	Difficulty       string                 `json:"difficulty"`
	EstimatedMinutes int                    `json:"estimatedMinutes"`
	AttemptLimit     int                    `json:"attemptLimit"`
	PassingScore     int                    `json:"passingScore"`
	BasePoints       int                    `json:"basePoints"`
	AllowPause       bool                   `json:"allowPause"`
	LevelType        string                 `json:"levelType"`
	AbilityIDs       []uint                 `json:"abilityIds"`
	KnowledgeTagIDs  []uint                 `json:"knowledgeTagIds"`
	Questions        []LevelQuestionRequest `json:"questions"`
	IsPublished      bool                   `json:"isPublished"`
	VisibleScope     string                 `json:"visibleScope"`
	VisibleTo        []uint                 `json:"visibleTo"`
	AvailableFrom    *FlexibleTime          `json:"availableFrom"`
	AvailableTo      *FlexibleTime          `json:"availableTo"`
}

func (s *LevelService) CreateLevel(creatorID uint, req LevelCreateRequest) (*model.Level, error) {
	if req.Title == "" {
		return nil, errors.New("title required")
	}
	// validate abilities: must choose at least 1
	if len(req.AbilityIDs) == 0 {
		return nil, errors.New("at least one ability must be selected")
	}
	// validate visible scope
	if req.VisibleScope == "specific" && len(req.VisibleTo) == 0 {
		return nil, errors.New("visibleTo must be provided when visibleScope is 'specific'")
	}
	var createdLevel *model.Level
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		level := &model.Level{
			CreatorID:        creatorID,
			Title:            req.Title,
			Description:      req.Description,
			CoverURL:         req.CoverURL,
			Difficulty:       req.Difficulty,
			EstimatedMinutes: req.EstimatedMinutes,
			AttemptLimit:     req.AttemptLimit,
			PassingScore:     req.PassingScore,
			BasePoints:       req.BasePoints,
			AllowPause:       req.AllowPause,
			LevelType:        req.LevelType,
			IsPublished:      req.IsPublished,
			VisibleScope:     req.VisibleScope,
			AvailableFrom:    req.AvailableFrom.TimePtr(),
			AvailableTo:      req.AvailableTo.TimePtr(),
		}
		// Ensure VisibleTo is valid JSON (store empty array if nil)
		{
			var vtBytes []byte
			if len(req.VisibleTo) > 0 {
				vtBytes, _ = json.Marshal(req.VisibleTo)
			} else {
				vtBytes = []byte("[]")
			}
			level.VisibleTo = json.RawMessage(vtBytes)
		}

		if err := tx.Create(level).Error; err != nil {
			return err
		}

		// abilities
		if len(req.AbilityIDs) > 0 {
			var links []model.LevelAbility
			for _, aid := range req.AbilityIDs {
				links = append(links, model.LevelAbility{
					LevelID:   level.ID,
					AbilityID: aid,
				})
			}
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}

		// knowledge tags
		if len(req.KnowledgeTagIDs) > 0 {
			var links []model.LevelKnowledge
			for _, kid := range req.KnowledgeTagIDs {
				links = append(links, model.LevelKnowledge{
					LevelID:        level.ID,
					KnowledgeTagID: kid,
				})
			}
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}

		// questions
		if len(req.Questions) > 0 {
			for idx, q := range req.Questions {
				contentBytes, _ := json.Marshal(q.Content)
				optionsBytes, _ := json.Marshal(q.Options)
				correctBytes, _ := json.Marshal(q.CorrectAnswer)
				question := &model.LevelQuestion{
					LevelID:       level.ID,
					QuestionType:  q.QuestionType,
					Content:       string(contentBytes),
					Options:       string(optionsBytes),
					CorrectAnswer: string(correctBytes),
					Points:        q.Points,
					Weight:        q.Weight,
					ManualGrading: q.ManualGrading,
					Order:         idx + 1,
					ScoringRule:   q.ScoringRule,
					Explanation:   q.Explanation,
				}
				if err := tx.Create(question).Error; err != nil {
					return err
				}
			}
		}

		// create initial version snapshot
		// snapshot content include level + questions
		var questions []model.LevelQuestion
		if err := tx.Where("level_id = ?", level.ID).Find(&questions).Error; err != nil {
			return err
		}
		snapshot := map[string]interface{}{
			"level":     level,
			"questions": questions,
		}
		snapshotBytes, _ := json.Marshal(snapshot)
		version := &model.LevelVersion{
			LevelID:       level.ID,
			VersionNumber: 1,
			EditorID:      creatorID,
			ChangeNote:    "Initial version",
			Content:       string(snapshotBytes),
			IsPublished:   level.IsPublished,
		}
		if err := tx.Create(version).Error; err != nil {
			return err
		}

		// set current version
		level.CurrentVersion = version.ID
		if err := tx.Save(level).Error; err != nil {
			return err
		}

		createdLevel = level
		return nil
	})

	if err != nil {
		return nil, err
	}
	return createdLevel, nil
}

// UpdateLevel edits an existing level and creates a new version snapshot
func (s *LevelService) UpdateLevel(editorID uint, levelID uint, req LevelCreateRequest) (*model.Level, error) {
	var updatedLevel *model.Level
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		level, err := s.LevelRepo.FindByID(levelID)
		if err != nil {
			return err
		}
		// update scalar fields
		level.Title = req.Title
		level.Description = req.Description
		level.CoverURL = req.CoverURL
		level.Difficulty = req.Difficulty
		level.EstimatedMinutes = req.EstimatedMinutes
		level.AttemptLimit = req.AttemptLimit
		level.PassingScore = req.PassingScore
		level.BasePoints = req.BasePoints
		level.AllowPause = req.AllowPause
		level.LevelType = req.LevelType
		level.IsPublished = req.IsPublished
		level.VisibleScope = req.VisibleScope
		level.AvailableFrom = req.AvailableFrom.TimePtr()
		level.AvailableTo = req.AvailableTo.TimePtr()

		if err := tx.Save(level).Error; err != nil {
			return err
		}

		// validate abilities: must choose at least 1
		if len(req.AbilityIDs) == 0 {
			return errors.New("at least one ability must be selected")
		}
		// validate visible scope
		if req.VisibleScope == "specific" && len(req.VisibleTo) == 0 {
			return errors.New("visibleTo must be provided when visibleScope is 'specific'")
		}

		// replace abilities
		if err := tx.Where("level_id = ?", level.ID).Delete(&model.LevelAbility{}).Error; err != nil {
			return err
		}
		if len(req.AbilityIDs) > 0 {
			var links []model.LevelAbility
			for _, aid := range req.AbilityIDs {
				links = append(links, model.LevelAbility{LevelID: level.ID, AbilityID: aid})
			}
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}

		// replace knowledge tags
		if err := tx.Where("level_id = ?", level.ID).Delete(&model.LevelKnowledge{}).Error; err != nil {
			return err
		}
		if len(req.KnowledgeTagIDs) > 0 {
			var links []model.LevelKnowledge
			for _, kid := range req.KnowledgeTagIDs {
				links = append(links, model.LevelKnowledge{LevelID: level.ID, KnowledgeTagID: kid})
			}
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}

		// replace questions
		if err := s.LevelRepo.DeleteQuestionsByLevel(level.ID); err != nil {
			return err
		}
		if len(req.Questions) > 0 {
			var qEntities []model.LevelQuestion
			for idx, q := range req.Questions {
				cb, _ := json.Marshal(q.Content)
				ob, _ := json.Marshal(q.Options)
				cb2, _ := json.Marshal(q.CorrectAnswer)
				qEntities = append(qEntities, model.LevelQuestion{
					LevelID:       level.ID,
					QuestionType:  q.QuestionType,
					Content:       string(cb),
					Options:       string(ob),
					CorrectAnswer: string(cb2),
					Points:        q.Points,
					Weight:        q.Weight,
					ManualGrading: q.ManualGrading,
					Order:         idx + 1,
					ScoringRule:   q.ScoringRule,
					Explanation:   q.Explanation,
				})
			}
			if err := s.LevelRepo.CreateQuestions(qEntities); err != nil {
				return err
			}
		}

		// create new version snapshot
		var questions []model.LevelQuestion
		if err := tx.Where("level_id = ?", level.ID).Find(&questions).Error; err != nil {
			return err
		}
		snapshot := map[string]interface{}{
			"level":     level,
			"questions": questions,
		}
		snapshotBytes, _ := json.Marshal(snapshot)

		// compute next version number
		versions, err := s.LevelRepo.GetVersions(level.ID)
		if err != nil {
			return err
		}
		nextVersion := 1
		if len(versions) > 0 {
			nextVersion = versions[0].VersionNumber + 1
		}

		version := &model.LevelVersion{
			LevelID:       level.ID,
			VersionNumber: nextVersion,
			EditorID:      editorID,
			ChangeNote:    "Edit",
			Content:       string(snapshotBytes),
			IsPublished:   level.IsPublished,
		}
		if err := tx.Create(version).Error; err != nil {
			return err
		}

		level.CurrentVersion = version.ID
		if err := tx.Save(level).Error; err != nil {
			return err
		}

		updatedLevel = level
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updatedLevel, nil
}

// Publish toggles publication and sets PublishedAt
func (s *LevelService) PublishLevel(editorID, levelID uint, publish bool) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		level, err := s.LevelRepo.FindByID(levelID)
		if err != nil {
			return err
		}
		level.IsPublished = publish
		if publish {
			now := time.Now()
			level.PublishedAt = &now
		} else {
			level.PublishedAt = nil
		}
		if err := tx.Save(level).Error; err != nil {
			return err
		}
		// create a version record to mark publish state
		var questions []model.LevelQuestion
		if err := tx.Where("level_id = ?", level.ID).Find(&questions).Error; err != nil {
			return err
		}
		snapshot := map[string]interface{}{"level": level, "questions": questions}
		content, _ := json.Marshal(snapshot)
		versions, err := s.LevelRepo.GetVersions(level.ID)
		if err != nil {
			return err
		}
		nextVersion := 1
		if len(versions) > 0 {
			nextVersion = versions[0].VersionNumber + 1
		}
		v := &model.LevelVersion{
			LevelID:       level.ID,
			VersionNumber: nextVersion,
			EditorID:      editorID,
			ChangeNote:    "Publish",
			Content:       string(content),
			IsPublished:   publish,
			PublishedAt:   level.PublishedAt,
		}
		if err := tx.Create(v).Error; err != nil {
			return err
		}
		level.CurrentVersion = v.ID
		if err := tx.Save(level).Error; err != nil {
			return err
		}
		return nil
	})
}

// BulkUpdateLevels performs field updates for multiple levels
func (s *LevelService) BulkUpdateLevels(editorID uint, ids []uint, updates map[string]interface{}) error {
	if len(ids) == 0 {
		return nil
	}
	return s.LevelRepo.BulkUpdate(ids, updates)
}

// GetVersions returns versions for a level
func (s *LevelService) GetVersions(levelID uint) ([]model.LevelVersion, error) {
	return s.LevelRepo.GetVersions(levelID)
}

// RollbackToVersion applies a version snapshot as a new version and updates current level
func (s *LevelService) RollbackToVersion(editorID uint, levelID uint, versionID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		v, err := s.LevelRepo.GetVersionByID(versionID)
		if err != nil {
			return err
		}

		// parse content
		var snap struct {
			Level     model.Level           `json:"level"`
			Questions []model.LevelQuestion `json:"questions"`
		}
		if err := json.Unmarshal([]byte(v.Content), &snap); err != nil {
			return err
		}
		// find target level
		level, err := s.LevelRepo.FindByID(levelID)
		if err != nil {
			return err
		}
		// update scalar fields from snapshot.Level (only selected fields)
		level.Title = snap.Level.Title
		level.Description = snap.Level.Description
		level.CoverURL = snap.Level.CoverURL
		level.Difficulty = snap.Level.Difficulty
		level.EstimatedMinutes = snap.Level.EstimatedMinutes
		level.AttemptLimit = snap.Level.AttemptLimit
		level.PassingScore = snap.Level.PassingScore
		level.BasePoints = snap.Level.BasePoints
		level.AllowPause = snap.Level.AllowPause
		level.LevelType = snap.Level.LevelType
		level.IsPublished = snap.Level.IsPublished
		level.VisibleScope = snap.Level.VisibleScope
		level.AvailableFrom = snap.Level.AvailableFrom
		level.AvailableTo = snap.Level.AvailableTo

		if err := tx.Save(level).Error; err != nil {
			return err
		}

		// replace questions
		if err := s.LevelRepo.DeleteQuestionsByLevel(level.ID); err != nil {
			return err
		}
		if len(snap.Questions) > 0 {
			for i := range snap.Questions {
				snap.Questions[i].LevelID = level.ID
				snap.Questions[i].ID = 0
			}
			if err := s.LevelRepo.CreateQuestions(snap.Questions); err != nil {
				return err
			}
		}

		// create a new version marking the rollback
		contents := v.Content
		versions, err := s.LevelRepo.GetVersions(level.ID)
		if err != nil {
			return err
		}
		nextVersion := 1
		if len(versions) > 0 {
			nextVersion = versions[0].VersionNumber + 1
		}
		newV := &model.LevelVersion{
			LevelID:       level.ID,
			VersionNumber: nextVersion,
			EditorID:      editorID,
			ChangeNote:    "Rollback to version",
			Content:       contents,
			IsPublished:   level.IsPublished,
			PublishedAt:   level.PublishedAt,
		}
		if err := tx.Create(newV).Error; err != nil {
			return err
		}
		level.CurrentVersion = newV.ID
		if err := tx.Save(level).Error; err != nil {
			return err
		}
		return nil
	})
}

// StartAttempt 创建并开始一次关卡挑战
func (s *LevelService) StartAttempt(userID, levelID uint) (*model.LevelAttempt, error) {
	var level *model.Level
	lev, err := s.LevelRepo.FindByID(levelID)
	if err != nil {
		return nil, err
	}
	level = lev

	// count attempts used
	var count int64
	if err := s.DB.Model(&model.LevelAttempt{}).Where("user_id = ? AND level_id = ?", userID, levelID).Count(&count).Error; err != nil {
		return nil, err
	}
	if level.AttemptLimit > 0 && int(count) >= level.AttemptLimit {
		return nil, errors.New("attempt limit reached")
	}

	attempt := &model.LevelAttempt{
		LevelID:          levelID,
		UserID:           userID,
		AttemptsUsed:     int(count) + 1,
		StartedAt:        time.Now(),
		VersionID:        level.CurrentVersion,
		PerQuestionTimes: "{}",
	}
	if err := s.DB.Create(attempt).Error; err != nil {
		return nil, err
	}
	return attempt, nil
}

// SubmitAttempt 提交挑战，计算分数并记录每题耗时
type SubmitAnswer struct {
	QuestionID uint        `json:"questionId"`
	Answer     interface{} `json:"answer"`
}

type PerQuestionTime struct {
	QuestionID  uint `json:"questionId"`
	TimeSeconds int  `json:"timeSeconds"`
}

func (s *LevelService) SubmitAttempt(userID, levelID, attemptID uint, answers []SubmitAnswer, times []PerQuestionTime) (*model.LevelAttempt, error) {
	// load attempt
	var attempt model.LevelAttempt
	if err := s.DB.First(&attempt, attemptID).Error; err != nil {
		return nil, err
	}
	if attempt.UserID != userID || attempt.LevelID != levelID {
		return nil, errors.New("unauthorized attempt")
	}
	if attempt.EndedAt != nil {
		return nil, errors.New("attempt already submitted")
	}

	// load questions: prefer snapshot from version if available
	qMap := make(map[uint]model.LevelQuestion)
	if attempt.VersionID > 0 {
		v, err := s.LevelRepo.GetVersionByID(attempt.VersionID)
		if err == nil {
			var snap struct {
				Level     model.Level           `json:"level"`
				Questions []model.LevelQuestion `json:"questions"`
			}
			if err := json.Unmarshal([]byte(v.Content), &snap); err == nil {
				for _, q := range snap.Questions {
					qMap[q.ID] = q
				}
			}
		}
	}
	// fallback to current questions if none loaded
	if len(qMap) == 0 {
		var questions []model.LevelQuestion
		if err := s.DB.Where("level_id = ?", levelID).Find(&questions).Error; err != nil {
			return nil, err
		}
		for _, q := range questions {
			qMap[q.ID] = q
		}
	}

	// grade with support for manual grading and weight
	totalScore := 0
	needsManual := false
	for _, a := range answers {
		if q, ok := qMap[a.QuestionID]; ok {
			if q.ManualGrading {
				needsManual = true
				continue
			}
			// compare JSON string of provided answer with stored correctAnswer
			provided, _ := json.Marshal(a.Answer)
			correct := q.CorrectAnswer
			if string(provided) == correct {
				weight := q.Weight
				if weight <= 0 {
					weight = 1
				}
				totalScore += q.Points * weight
			}
		}
	}

	// update attempt
	now := time.Now()
	duration := int(now.Sub(attempt.StartedAt).Seconds())
	attempt.Score = totalScore
	attempt.TotalTimeSeconds = duration
	attempt.EndedAt = &now
	attempt.NeedsManual = needsManual

	// success check: load level
	level, err := s.LevelRepo.FindByID(levelID)
	if err != nil {
		return nil, err
	}
	// if manual grading required, mark success = false and wait for manual review
	if needsManual {
		attempt.Success = false
	} else {
		attempt.Success = totalScore >= level.PassingScore && attempt.AttemptsUsed <= level.AttemptLimit
	}

	// save attempt, answers and times in transaction
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&attempt).Error; err != nil {
			return err
		}
		// save answers
		if len(answers) > 0 {
			var ansEntities []model.LevelAttemptAnswer
			for _, a := range answers {
				bytes, _ := json.Marshal(a.Answer)
				ansEntities = append(ansEntities, model.LevelAttemptAnswer{
					AttemptID:  attempt.ID,
					QuestionID: a.QuestionID,
					Answer:     string(bytes),
				})
			}
			if err := tx.Create(&ansEntities).Error; err != nil {
				return err
			}
		}
		// per-question times
		if len(times) > 0 {
			var timesEntities []model.LevelAttemptQuestionTime
			for _, t := range times {
				timesEntities = append(timesEntities, model.LevelAttemptQuestionTime{
					AttemptID:   attempt.ID,
					QuestionID:  t.QuestionID,
					TimeSeconds: t.TimeSeconds,
				})
			}
			if err := tx.Create(&timesEntities).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &attempt, nil
}

// AddQuestion 向关卡添加单个题目
func (s *LevelService) AddQuestion(editorID, levelID uint, req LevelQuestionRequest) (*model.LevelQuestion, error) {
	// basic validation
	if req.QuestionType == "" {
		return nil, errors.New("questionType required")
	}
	if req.Content == nil {
		return nil, errors.New("content required")
	}
	cb, _ := json.Marshal(req.Content)
	ob, _ := json.Marshal(req.Options)
	correct, _ := json.Marshal(req.CorrectAnswer)
	q := &model.LevelQuestion{
		LevelID:       levelID,
		QuestionType:  req.QuestionType,
		Content:       string(cb),
		Options:       string(ob),
		CorrectAnswer: string(correct),
		Points:        req.Points,
		Weight:        req.Weight,
		ManualGrading: req.ManualGrading,
		ScoringRule:   req.ScoringRule,
		Explanation:   req.Explanation,
	}
	if err := s.LevelRepo.CreateQuestion(q); err != nil {
		return nil, err
	}
	return q, nil
}

// UpdateQuestion 更新题目
func (s *LevelService) UpdateQuestion(editorID, levelID, questionID uint, req LevelQuestionRequest) (*model.LevelQuestion, error) {
	q, err := s.LevelRepo.FindQuestionByID(questionID)
	if err != nil {
		return nil, err
	}
	if q.LevelID != levelID {
		return nil, errors.New("question not belong to level")
	}
	if req.Content != nil {
		cb, _ := json.Marshal(req.Content)
		q.Content = string(cb)
	}
	if req.Options != nil {
		ob, _ := json.Marshal(req.Options)
		q.Options = string(ob)
	}
	if req.CorrectAnswer != nil {
		correct, _ := json.Marshal(req.CorrectAnswer)
		q.CorrectAnswer = string(correct)
	}
	q.QuestionType = req.QuestionType
	q.Points = req.Points
	q.Weight = req.Weight
	q.ManualGrading = req.ManualGrading
	q.ScoringRule = req.ScoringRule
	q.Explanation = req.Explanation
	if err := s.LevelRepo.UpdateQuestion(q); err != nil {
		return nil, err
	}
	return q, nil
}

// DeleteQuestion 删除题目
func (s *LevelService) DeleteQuestion(levelID, questionID uint) error {
	q, err := s.LevelRepo.FindQuestionByID(questionID)
	if err != nil {
		return err
	}
	if q.LevelID != levelID {
		return errors.New("question not belong to level")
	}
	return s.LevelRepo.DeleteQuestionByID(questionID)
}

// GetAttemptStats 返回关卡尝试统计（count, avgScore, avgTime, successRate）
func (s *LevelService) GetAttemptStats(levelID uint, start *time.Time, end *time.Time, studentID uint) (map[string]interface{}, error) {
	query := s.DB.Model(&model.LevelAttempt{}).Where("level_id = ?", levelID)
	if start != nil {
		query = query.Where("started_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("started_at <= ?", *end)
	}
	if studentID > 0 {
		query = query.Where("user_id = ?", studentID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var avgScore float64
	var avgTime float64
	var successCount int64
	if total > 0 {
		if err := query.Select("AVG(score)").Scan(&avgScore).Error; err != nil {
			return nil, err
		}
		if err := query.Select("AVG(total_time_seconds)").Scan(&avgTime).Error; err != nil {
			return nil, err
		}
		if err := query.Select("SUM(success)").Scan(&successCount).Error; err != nil {
			return nil, err
		}
	}
	stats := map[string]interface{}{
		"totalAttempts": total,
		"avgScore":      avgScore,
		"avgTime":       avgTime,
		"successRate": func() float64 {
			if total == 0 {
				return 0
			}
			return float64(successCount) / float64(total)
		}(),
	}
	return stats, nil
}

// QuestionScore represents a manual score input
type QuestionScore struct {
	QuestionID uint
	Score      int
	Comment    string
	GraderID   uint
	GradedAt   *time.Time
}

// ListAttemptsNeedingManual returns attempts for a level that need manual grading
func (s *LevelService) ListAttemptsNeedingManual(levelID uint) ([]model.LevelAttempt, error) {
	var attempts []model.LevelAttempt
	err := s.DB.Where("level_id = ? AND needs_manual = ?", levelID, true).Find(&attempts).Error
	return attempts, err
}

// ManualGradeAttempt 保存人工评分并完成尝试（若全部题目评分完成）
func (s *LevelService) ManualGradeAttempt(graderID uint, attemptID uint, scores []QuestionScore) error {
	lar := repository.NewLevelAttemptRepository(s.DB)

	// save scores
	var scoreEntities []model.LevelAttemptQuestionScore
	now := time.Now()
	for _, sc := range scores {
		scoreEntities = append(scoreEntities, model.LevelAttemptQuestionScore{
			AttemptID:  attemptID,
			QuestionID: sc.QuestionID,
			Score:      sc.Score,
			GraderID:   graderID,
			Comment:    sc.Comment,
			GradedAt:   &now,
		})
	}

	err := lar.CreateOrUpdateQuestionScores(scoreEntities)
	if err != nil {
		return err
	}

	// recalc total score: auto-graded + manual scores
	attempt, err := lar.FindByID(attemptID)
	if err != nil {
		return err
	}

	// get auto-scored questions (those not manual) -- prefer snapshot
	var questions []model.LevelQuestion
	if attempt.VersionID > 0 {
		if v, err := s.LevelRepo.GetVersionByID(attempt.VersionID); err == nil {
			var snap struct {
				Level     model.Level           `json:"level"`
				Questions []model.LevelQuestion `json:"questions"`
			}
			if err := json.Unmarshal([]byte(v.Content), &snap); err == nil {
				questions = snap.Questions
			}
		}
	}
	if len(questions) == 0 {
		questions, err = s.LevelRepo.GetQuestionsByLevel(attempt.LevelID)
		if err != nil {
			return err
		}
	}
	autoScore := 0
	for _, q := range questions {
		if q.ManualGrading {
			continue
		}
		// find stored answer for this question
		var ans model.LevelAttemptAnswer
		if err := s.DB.Where("attempt_id = ? AND question_id = ?", attemptID, q.ID).First(&ans).Error; err == nil {
			// compare
			var provided interface{}
			if json.Unmarshal([]byte(ans.Answer), &provided) == nil {
				providedBytes, _ := json.Marshal(provided)
				if string(providedBytes) == q.CorrectAnswer {
					w := q.Weight
					if w <= 0 {
						w = 1
					}
					autoScore += q.Points * w
				}
			}
		}
	}

	// sum manual scores
	var manualTotal int64
	if err := s.DB.Model(&model.LevelAttemptQuestionScore{}).Where("attempt_id = ?", attemptID).Select("SUM(score)").Scan(&manualTotal).Error; err != nil {
		return err
	}

	newTotal := autoScore + int(manualTotal)
	// update attempt
	now2 := time.Now()
	attempt.Score = newTotal
	attempt.NeedsManual = false
	// determine success based on level.passing score
	level, err := s.LevelRepo.FindByID(attempt.LevelID)
	if err != nil {
		return err
	}
	attempt.Success = newTotal >= level.PassingScore
	attempt.EndedAt = &now2

	if err := lar.Update(attempt); err != nil {
		return err
	}
	return nil
}

// BulkPublish 批量发布/下架（会为每个关卡创建版本记录）
func (s *LevelService) BulkPublish(editorID uint, ids []uint, publish bool) error {
	for _, id := range ids {
		level, err := s.LevelRepo.FindByID(id)
		if err != nil {
			return fmt.Errorf("level with id %d not found", id)
		}

		if level.IsPublished == publish {
			continue
		}

		if err := s.PublishLevel(editorID, id, publish); err != nil {
			return err
		}
	}
	return nil
}

// SchedulePublish 设置/取消定时发布
func (s *LevelService) SchedulePublish(editorID, levelID uint, scheduledAt *time.Time) error {
	level, err := s.LevelRepo.FindByID(levelID)
	if err != nil {
		return err
	}
	level.ScheduledPublishAt = scheduledAt
	return s.LevelRepo.UpdateLevel(level)
}

// UpdateVisibility 更新关卡可见范围与特定可见学生列表
func (s *LevelService) UpdateVisibility(editorID, levelID uint, visibleScope string, visibleTo []uint) error {
	level, err := s.LevelRepo.FindByID(levelID)
	if err != nil {
		return err
	}
	if visibleScope == "specific" && len(visibleTo) == 0 {
		return errors.New("visibleTo required when visibleScope is 'specific'")
	}
	// marshal visibleTo
	vtBytes, _ := json.Marshal(visibleTo)
	level.VisibleScope = visibleScope
	level.VisibleTo = vtBytes
	return s.LevelRepo.UpdateLevel(level)
}

// ProcessScheduledPublishes 查找并发布到期的关卡（被后台定时触发）
func (s *LevelService) ProcessScheduledPublishes() error {
	var levels []model.Level
	now := time.Now()
	if err := s.DB.Where("scheduled_publish_at IS NOT NULL AND scheduled_publish_at <= ? AND is_published = ?", now, false).Find(&levels).Error; err != nil {
		return err
	}
	for _, lvl := range levels {
		// publish using existing logic
		if err := s.PublishLevel(0, lvl.ID, true); err != nil {
			// log and continue
			continue
		}
		// clear scheduled time
		lvl.ScheduledPublishAt = nil
		s.LevelRepo.UpdateLevel(&lvl)
	}
	return nil
}

// ListLevelsFull 获取包含完整关联信息的关卡列表
func (s *LevelService) ListLevelsFull(creatorID uint, page, limit int) ([]LevelFullResponse, int, error) {
	// 获取基本关卡信息
	levels, total, err := s.LevelRepo.ListByCreator(creatorID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	var fullLevels []LevelFullResponse
	for _, level := range levels {
		// 获取能力关联
		var abilityIDs []uint
		var levelAbilities []model.LevelAbility
		if err := s.DB.Where("level_id = ?", level.ID).Find(&levelAbilities).Error; err == nil {
			for _, la := range levelAbilities {
				abilityIDs = append(abilityIDs, la.AbilityID)
			}
		}

		// 获取知识标签关联
		var knowledgeTagIDs []uint
		var levelKnowledge []model.LevelKnowledge
		if err := s.DB.Where("level_id = ?", level.ID).Find(&levelKnowledge).Error; err == nil {
			for _, lk := range levelKnowledge {
				knowledgeTagIDs = append(knowledgeTagIDs, lk.KnowledgeTagID)
			}
		}

		// 获取题目信息
		var questions []LevelQuestionResponse
		var levelQuestions []model.LevelQuestion
		if err := s.DB.Where("level_id = ?", level.ID).Order("`order` asc").Find(&levelQuestions).Error; err == nil {
			for _, q := range levelQuestions {
				questions = append(questions, LevelQuestionResponse{
					ID:            q.ID,
					CreatedAt:     q.CreatedAt,
					UpdatedAt:     q.UpdatedAt,
					LevelID:       q.LevelID,
					QuestionType:  q.QuestionType,
					Content:       json.RawMessage(q.Content),
					Options:       json.RawMessage(q.Options),
					CorrectAnswer: json.RawMessage(q.CorrectAnswer),
					Points:        q.Points,
					Weight:        q.Weight,
					ManualGrading: q.ManualGrading,
					Order:         q.Order,
					ScoringRule:   q.ScoringRule,
					Explanation:   q.Explanation,
					CodeTemplate:  "", // 如果有的话需要从 Content 中解析
				})
			}
		}

		fullLevel := LevelFullResponse{
			ID:                 level.ID,
			CreatedAt:          level.CreatedAt,
			UpdatedAt:          level.UpdatedAt,
			CreatorID:          level.CreatorID,
			Title:              level.Title,
			Description:        level.Description,
			CoverURL:           level.CoverURL,
			Difficulty:         level.Difficulty,
			EstimatedMinutes:   level.EstimatedMinutes,
			AttemptLimit:       level.AttemptLimit,
			PassingScore:       level.PassingScore,
			BasePoints:         level.BasePoints,
			AllowPause:         level.AllowPause,
			LevelType:          level.LevelType,
			IsPublished:        level.IsPublished,
			PublishedAt:        level.PublishedAt,
			ScheduledPublishAt: level.ScheduledPublishAt,
			VisibleScope:       level.VisibleScope,
			VisibleTo:          json.RawMessage(level.VisibleTo),
			AvailableFrom:      level.AvailableFrom,
			AvailableTo:        level.AvailableTo,
			CurrentVersion:     level.CurrentVersion,
			Abilities:          abilityIDs,
			KnowledgeTags:      knowledgeTagIDs,
			Questions:          questions,
		}
		fullLevels = append(fullLevels, fullLevel)
	}

	return fullLevels, total, nil
}

// DeleteLevel 删除关卡
func (s *LevelService) DeleteLevel(deleterID, levelID uint) error {
	// 检查关卡是否存在以及权限
	level, err := s.LevelRepo.FindByID(levelID)
	if err != nil {
		return err
	}
	if level.CreatorID != deleterID {
		return errors.New("only creator can delete level")
	}

	// 删除关卡及其所有关联数据
	return s.LevelRepo.DeleteLevel(levelID)
}

// ListLevelsForStudent 获取学生端关卡列表
func (s *LevelService) ListLevelsForStudent(userID uint, search, difficulty, status string, page, limit int) ([]StudentLevelResponse, int, error) {
	// 获取关卡列表
	levels, total, err := s.LevelRepo.ListLevelsForStudent(userID, search, difficulty, page, limit)
	if err != nil {
		return nil, 0, err
	}

	var responses []StudentLevelResponse
	for _, level := range levels {
		// 获取关卡的题目信息
		var levelQuestions []model.LevelQuestion
		basePoints := 0
		questionsCount := 0
		if err := s.DB.Where("level_id = ?", level.ID).Find(&levelQuestions).Error; err == nil {
			for _, question := range levelQuestions {
				basePoints += question.Points
			}
			questionsCount = len(levelQuestions)
		}

		// 获取学生对该关卡的尝试记录
		attempts, err := s.getAttemptsByUserLevel(userID, level.ID)
		if err != nil {
			continue // 跳过有错误的数据
		}

		response := StudentLevelResponse{
			ID:               level.ID,
			Title:            level.Title,
			Description:      level.Description,
			CoverURL:         level.CoverURL,
			Difficulty:       level.Difficulty,
			EstimatedMinutes: level.EstimatedMinutes,
			AttemptLimit:     level.AttemptLimit,
			PassingScore:     level.PassingScore,
			BasePoints:       basePoints,
			QuestionsCount:   questionsCount,
			CreatedAt:        level.CreatedAt,
		}

		// 确定关卡状态
		if len(attempts) == 0 {
			response.Status = "not_started"
		} else {
			// 检查是否有成功的尝试
			hasCompleted := false
			bestScore := 0

			for _, attempt := range attempts {
				if attempt.Score > bestScore {
					bestScore = attempt.Score
				}
				if attempt.Success {
					hasCompleted = true
				}
			}

			if hasCompleted {
				response.Status = "completed"
				response.BestScore = bestScore
			} else {
				response.Status = "in_progress"
			}

			// 获取用户对该关卡的总尝试次数
			var totalAttempts int64
			s.DB.Model(&model.LevelAttempt{}).Where("user_id = ? AND level_id = ?", userID, level.ID).Count(&totalAttempts)
			response.AttemptsUsed = int(totalAttempts)
		}

		// 如果指定了状态筛选，只返回符合条件的关卡
		if status != "" && status != "all" && response.Status != status {
			continue
		}

		responses = append(responses, response)
	}

	// 如果有状态筛选，需要重新计算总数
	if status != "" && status != "all" {
		// 获取所有符合条件的关卡总数
		allLevels, _, err := s.LevelRepo.ListLevelsForStudent(userID, search, difficulty, 1, 10000) // 获取所有数据
		if err == nil {
			filteredCount := 0
			for _, level := range allLevels {
				attempts, err := s.getAttemptsByUserLevel(userID, level.ID)
				if err != nil {
					continue
				}

				levelStatus := "not_started"
				if len(attempts) > 0 {
					hasCompleted := false
					for _, attempt := range attempts {
						if attempt.Success {
							hasCompleted = true
							break
						}
					}
					if hasCompleted {
						levelStatus = "completed"
					} else {
						levelStatus = "in_progress"
					}
				}

				if levelStatus == status {
					filteredCount++
				}
			}
			total = filteredCount
		}
	}

	return responses, total, nil
}

// getAttemptsByUserLevel 获取用户对特定关卡的所有尝试记录
func (s *LevelService) getAttemptsByUserLevel(userID, levelID uint) ([]model.LevelAttempt, error) {
	var attempts []model.LevelAttempt
	err := s.DB.Where("user_id = ? AND level_id = ?", userID, levelID).Find(&attempts).Error
	return attempts, err
}

// GetStudentLevelDetail 获取学生端关卡详情
func (s *LevelService) GetStudentLevelDetail(userID, levelID uint) (*StudentLevelDetailResponse, error) {
	// 验证关卡是否存在且对学生可见
	level, err := s.LevelRepo.FindByID(levelID)
	if err != nil {
		return nil, err
	}

	// 验证可见性
	if level.IsPublished != true {
		return nil, errors.New("level not found")
	}

	// 可见性检查
	if level.VisibleScope != "all" {
		if level.VisibleScope != "specific" {
			return nil, errors.New("level not accessible")
		}
		// 检查用户是否在可见列表中
		canAccess := false
		if level.VisibleTo != nil {
			var visibleTo []uint
			if err := json.Unmarshal(level.VisibleTo, &visibleTo); err == nil {
				for _, uid := range visibleTo {
					if uid == userID {
						canAccess = true
						break
					}
				}
			}
		}
		if !canAccess {
			return nil, errors.New("level not accessible")
		}
	}

	// 时间范围检查（如果是指定学生可见的关卡）
	if level.VisibleScope == "specific" {
		now := time.Now()
		if level.AvailableFrom != nil && level.AvailableFrom.After(now) {
			return nil, errors.New("level not yet available")
		}
		if level.AvailableTo != nil && level.AvailableTo.Before(now) {
			return nil, errors.New("level no longer available")
		}
	}

	// 获取关卡的题目信息
	var levelQuestions []model.LevelQuestion
	if err := s.DB.Where("level_id = ?", levelID).Order("`order` asc").Find(&levelQuestions).Error; err != nil {
		return nil, err
	}

	// 计算积分和题目数量
	basePoints := 0
	questionsCount := len(levelQuestions)
	questions := make([]StudentQuestionResponse, 0, questionsCount)

	for _, q := range levelQuestions {
		basePoints += q.Points

		// 从content中提取标题
		title := extractQuestionTitle(q.Content)

		questions = append(questions, StudentQuestionResponse{
			ID:           q.ID,
			Title:        title,
			QuestionType: q.QuestionType,
			Content:      json.RawMessage(q.Content),
			Options:      json.RawMessage(q.Options),
			Points:       q.Points,
			Weight:       q.Weight,
			Order:        q.Order,
			Difficulty:   level.Difficulty, // 使用关卡难度数据
		})
	}

	// 获取学生对该关卡的尝试记录
	attempts, err := s.getAttemptsByUserLevel(userID, levelID)
	if err != nil {
		return nil, err
	}

	// 获取关卡元数据
	var updatedAt *time.Time
	updatedAt = &level.UpdatedAt

	// 获取创建者信息
	var author *UserInfo
	var creator model.User
	if err := s.DB.Where("id = ?", level.CreatorID).First(&creator).Error; err == nil {
		author = &UserInfo{
			ID:       creator.ID,
			Username: creator.Name, // User模型使用Name字段
			Email:    creator.Email,
		}
	}

	// 获取能力信息
	var abilities []AbilityInfo
	var levelAbilities []model.LevelAbility
	if err := s.DB.Where("level_id = ?", levelID).Find(&levelAbilities).Error; err == nil {
		for _, la := range levelAbilities {
			var ability model.Ability
			if err := s.DB.Where("id = ?", la.AbilityID).First(&ability).Error; err == nil {
				abilities = append(abilities, AbilityInfo{
					ID:          ability.ID,
					Code:        ability.Code,
					Name:        ability.Name,
					Description: ability.Description,
				})
			}
		}
	}

	// 获取标签信息
	var tags []TagInfo
	var levelKnowledge []model.LevelKnowledge
	if err := s.DB.Where("level_id = ?", levelID).Find(&levelKnowledge).Error; err == nil {
		for _, lk := range levelKnowledge {
			var tag model.KnowledgeTag
			if err := s.DB.Where("id = ?", lk.KnowledgeTagID).First(&tag).Error; err == nil {
				tags = append(tags, TagInfo{
					ID:   tag.ID,
					Name: tag.Name,
				})
			}
		}
	}

	// 生成学习目标（基于能力描述）
	var learningObjectives []string
	for _, ability := range abilities {
		if ability.Description != "" {
			learningObjectives = append(learningObjectives, ability.Description)
		}
	}

	// 获取统计数据
	var totalAttempts int64
	var averageScore float64
	var completionRate float64

	// 计算总尝试次数
	s.DB.Model(&model.LevelAttempt{}).Where("level_id = ?", levelID).Count(&totalAttempts)

	// 计算平均分数和完成率
	var successfulAttempts int64
	var totalSuccessfulScore int
	if totalAttempts > 0 {
		// 获取所有成功的尝试
		var successfulAttemptsList []model.LevelAttempt
		s.DB.Where("level_id = ? AND success = ?", levelID, true).Find(&successfulAttemptsList)
		successfulAttempts = int64(len(successfulAttemptsList))

		// 计算成功尝试的平均分
		if successfulAttempts > 0 {
			for _, attempt := range successfulAttemptsList {
				totalSuccessfulScore += attempt.Score
			}
			averageScore = float64(totalSuccessfulScore) / float64(successfulAttempts)
		}

		completionRate = float64(successfulAttempts) / float64(totalAttempts) * 100
	}

	response := &StudentLevelDetailResponse{
		// 基础信息
		ID:               level.ID,
		Title:            level.Title,
		Description:      level.Description,
		CoverURL:         level.CoverURL,
		Difficulty:       level.Difficulty,
		EstimatedMinutes: level.EstimatedMinutes,
		AttemptLimit:     level.AttemptLimit,
		PassingScore:     level.PassingScore,
		BasePoints:       basePoints,
		QuestionsCount:   questionsCount,

		// 关卡元数据
		UpdatedAt:          updatedAt,
		Author:             author,
		Abilities:          abilities, // 能力分类详细信息
		Tags:               tags,
		Prerequisites:      []string{},         // 暂时为空，后续可扩展
		LearningObjectives: learningObjectives, // 基于能力描述生成

		// 统计数据
		TotalAttempts:  int(totalAttempts),
		AverageScore:   averageScore,
		CompletionRate: completionRate,

		// 题目信息
		Questions: questions,
		CreatedAt: level.CreatedAt,
	}

	// 确定关卡状态和用户进度信息
	var completedAt *time.Time
	var lastAttemptAt *time.Time
	var timeSpent int

	if len(attempts) == 0 {
		response.Status = "not_started"
	} else {
		// 检查是否有成功的尝试
		hasCompleted := false
		bestScore := 0

		// 按开始时间排序，找到最早的成功完成时间
		for _, attempt := range attempts {

			// 计算该次尝试的用时（提交时间 - 开始时间）
			if attempt.EndedAt != nil {
				attemptTimeSpent := int(attempt.EndedAt.Sub(attempt.StartedAt).Seconds())
				timeSpent += attemptTimeSpent
			}

			if attempt.Success {
				hasCompleted = true
				// 记录第一次成功的完成时间（提交时间）
				if completedAt == nil {
					completedAt = attempt.EndedAt
				} else if attempt.EndedAt != nil && attempt.EndedAt.Before(*completedAt) {
					// 找到最早的成功完成时间
					completedAt = attempt.EndedAt
				}
			}

			// 记录最后一次尝试的时间（优先使用提交时间）
			if attempt.EndedAt != nil {
				if lastAttemptAt == nil || attempt.EndedAt.After(*lastAttemptAt) {
					lastAttemptAt = attempt.EndedAt
				}
			} else {
				// 如果没有提交时间，使用开始时间
				if lastAttemptAt == nil || attempt.StartedAt.After(*lastAttemptAt) {
					lastAttemptAt = &attempt.StartedAt
				}
			}
		}

		if hasCompleted {
			response.Status = "completed"
			response.BestScore = bestScore
			response.CompletedAt = completedAt
		} else {
			response.Status = "in_progress"
		}

		// 获取用户对该关卡的总尝试次数
		var totalAttempts int64
		s.DB.Model(&model.LevelAttempt{}).Where("user_id = ? AND level_id = ?", userID, levelID).Count(&totalAttempts)
		response.AttemptsUsed = int(totalAttempts)
		response.LastAttemptAt = lastAttemptAt
		response.TimeSpent = timeSpent
	}

	return response, nil
}

// GetStudentLevelQuestions 获取学生端关卡题目列表
func (s *LevelService) GetStudentLevelQuestions(userID, levelID uint) ([]StudentQuestionResponse, error) {
	// 验证关卡是否存在且对学生可见
	level, err := s.LevelRepo.FindByID(levelID)
	if err != nil {
		return nil, err
	}

	// 验证可见性
	if level.IsPublished != true {
		return nil, errors.New("level not found")
	}

	// 可见性检查
	if level.VisibleScope != "all" {
		if level.VisibleScope != "specific" {
			return nil, errors.New("level not accessible")
		}
		// 检查用户是否在可见列表中
		canAccess := false
		if level.VisibleTo != nil {
			var visibleTo []uint
			if err := json.Unmarshal(level.VisibleTo, &visibleTo); err == nil {
				for _, uid := range visibleTo {
					if uid == userID {
						canAccess = true
						break
					}
				}
			}
		}
		if !canAccess {
			return nil, errors.New("level not accessible")
		}
	}

	// 时间范围检查（如果是指定学生可见的关卡）
	if level.VisibleScope == "specific" {
		now := time.Now()
		if level.AvailableFrom != nil && level.AvailableFrom.After(now) {
			return nil, errors.New("level not yet available")
		}
		if level.AvailableTo != nil && level.AvailableTo.Before(now) {
			return nil, errors.New("level no longer available")
		}
	}

	// 获取关卡的所有题目信息
	var levelQuestions []model.LevelQuestion
	if err := s.DB.Where("level_id = ?", levelID).Order("`order` asc").Find(&levelQuestions).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	questions := make([]StudentQuestionResponse, 0, len(levelQuestions))
	for _, q := range levelQuestions {
		// 从content中提取标题
		title := extractQuestionTitle(q.Content)

		questions = append(questions, StudentQuestionResponse{
			ID:           q.ID,
			Title:        title,
			QuestionType: q.QuestionType,
			Content:      json.RawMessage(q.Content),
			Options:      json.RawMessage(q.Options),
			Points:       q.Points,
			Weight:       q.Weight,
			Order:        q.Order,
			Difficulty:   level.Difficulty, // 使用关卡难度
		})
	}

	return questions, nil
}

// extractQuestionTitle 从题目content中提取标题
func extractQuestionTitle(content string) string {
	// 解析content JSON
	var contentData map[string]interface{}
	if err := json.Unmarshal([]byte(content), &contentData); err != nil {
		return "题目" // 解析失败时返回默认标题
	}

	// 提取stem字段作为标题
	if stem, ok := contentData["stem"].(string); ok {
		// 移除HTML标签，提取纯文本作为标题
		return stripHTMLTags(stem)
	}

	return "题目" // 默认标题
}

// stripHTMLTags 移除HTML标签，提取纯文本
func stripHTMLTags(html string) string {
	// 简单移除HTML标签的逻辑
	// 这里可以根据需要使用更复杂的HTML解析库
	result := strings.ReplaceAll(html, "<p>", "")
	result = strings.ReplaceAll(result, "</p>", "")
	result = strings.ReplaceAll(result, "<br>", "")
	result = strings.ReplaceAll(result, "<br/>", "")
	result = strings.ReplaceAll(result, "<div>", "")
	result = strings.ReplaceAll(result, "</div>", "")

	// 限制长度，避免标题过长
	if len(result) > 50 {
		return result[:47] + "..."
	}

	return result
}

// BatchSubmitAnswers 批量提交关卡答案
func (s *LevelService) BatchSubmitAnswers(userID, levelID, attemptID uint, req interface{}) (*BatchSubmitAnswersResponse, error) {
	// 验证关卡可见性
	level, err := s.LevelRepo.FindByID(levelID)
	if err != nil {
		return nil, errors.New("level not found")
	}

	// 验证可见性
	if level.IsPublished != true {
		return nil, errors.New("level not found")
	}

	if level.VisibleScope != "all" {
		if level.VisibleScope != "specific" {
			return nil, errors.New("level not accessible")
		}
		// 检查用户是否在可见列表中
		canAccess := false
		if level.VisibleTo != nil {
			var visibleTo []uint
			if err := json.Unmarshal(level.VisibleTo, &visibleTo); err == nil {
				for _, uid := range visibleTo {
					if uid == userID {
						canAccess = true
						break
					}
				}
			}
		}
		if !canAccess {
			return nil, errors.New("level not accessible")
		}
	}

	// 验证尝试记录
	var attempt model.LevelAttempt
	if err := s.DB.Where("id = ? AND user_id = ? AND level_id = ?", attemptID, userID, levelID).First(&attempt).Error; err != nil {
		return nil, errors.New("attempt not found")
	}

	// 获取关卡的所有问题
	var questions []model.LevelQuestion
	if err := s.DB.Where("level_id = ?", levelID).Order("`order` asc").Find(&questions).Error; err != nil {
		return nil, err
	}

	// 解析请求数据
	reqMap, ok := req.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid request format")
	}

	answersInterface, ok := reqMap["answers"]
	if !ok {
		return nil, errors.New("answers field missing")
	}

	answersSlice, ok := answersInterface.([]interface{})
	if !ok {
		return nil, errors.New("answers field must be array")
	}

	// 创建答案映射，便于查找
	answerMap := make(map[uint]interface{})
	for _, answerItem := range answersSlice {
		answerMapItem, ok := answerItem.(map[string]interface{})
		if !ok {
			continue
		}

		questionIDFloat, ok1 := answerMapItem["questionId"].(float64)
		answer, ok2 := answerMapItem["answer"]

		if ok1 && ok2 {
			answerMap[uint(questionIDFloat)] = answer
		}
	}

	// 对所有问题进行评分
	results := make([]QuestionResult, 0, len(questions))
	totalScore := 0
	maxScore := 0

	for _, question := range questions {
		maxScore += question.Points
		result := QuestionResult{
			QuestionID: question.ID,
		}

		// 检查是否提交了答案
		if answer, submitted := answerMap[question.ID]; submitted {
			// 提交了答案，进行评分
			correct, score, explanation := s.checkAnswer(question, answer)
			result.Correct = correct
			result.Score = score
			result.Explanation = explanation
			result.Status = "correct"
			if !correct {
				result.Status = "incorrect"
				// 显示正确答案
				var correctAnswer interface{}
				if err := json.Unmarshal([]byte(question.CorrectAnswer), &correctAnswer); err == nil {
					result.CorrectAnswer = correctAnswer
				}
			}
			totalScore += score
		} else {
			// 未提交答案
			result.Status = "unanswered"
			result.Score = 0
		}

		results = append(results, result)
	}

	// 更新尝试记录
	now := time.Now()
	attempt.EndedAt = &now
	attempt.Score = totalScore
	attempt.Success = totalScore >= level.PassingScore

	// 计算总时间（从开始到现在的时长）
	if attempt.StartedAt.Before(now) {
		attempt.TotalTimeSeconds = int(now.Sub(attempt.StartedAt).Seconds())
	}

	if err := s.DB.Save(&attempt).Error; err != nil {
		return nil, err
	}

	// 保存答案记录（可选，用于历史记录和分析）
	for _, answerItem := range answersSlice {
		answerMapItem, ok := answerItem.(map[string]interface{})
		if !ok {
			continue
		}

		questionIDFloat, ok1 := answerMapItem["questionId"].(float64)
		answer, ok2 := answerMapItem["answer"]

		if ok1 && ok2 {
			answerRecord := &model.LevelAttemptAnswer{
				AttemptID:  attemptID,
				QuestionID: uint(questionIDFloat),
			}

			// 将答案转换为JSON字符串存储
			if answerBytes, err := json.Marshal(answer); err == nil {
				answerRecord.Answer = string(answerBytes)
			}

			s.DB.Create(answerRecord) // 忽略错误，继续处理
		}
	}

	response := &BatchSubmitAnswersResponse{
		Results:        results,
		TotalScore:     totalScore,
		MaxScore:       maxScore,
		AttemptID:      attemptID,
		IsCompleted:    attempt.Success,
		SubmittedCount: len(answersSlice),
	}

	return response, nil
}

// BatchSubmitAnswersRequest 批量提交答案请求结构体
type BatchSubmitAnswersRequest struct {
	Answers []QuestionAnswer `json:"answers"` // 问题答案数组
}

// QuestionAnswer 单个问题答案结构体
type QuestionAnswer struct {
	QuestionID uint        `json:"questionId"` // 问题ID
	Answer     interface{} `json:"answer"`     // 答案内容
}

// BatchSubmitAnswersResponse 批量提交答案响应结构体
type BatchSubmitAnswersResponse struct {
	Results        []QuestionResult `json:"results"`        // 每个问题的评分结果
	TotalScore     int              `json:"totalScore"`     // 总得分
	MaxScore       int              `json:"maxScore"`       // 最高得分
	AttemptID      uint             `json:"attemptId"`      // 尝试ID
	IsCompleted    bool             `json:"isCompleted"`    // 是否完成挑战
	SubmittedCount int              `json:"submittedCount"` // 提交答案的数量
}

// QuestionResult 单个问题评分结果结构体
type QuestionResult struct {
	QuestionID    uint        `json:"questionId"`              // 问题ID
	Correct       bool        `json:"correct"`                 // 答案是否正确
	Score         int         `json:"score"`                   // 获得分数
	CorrectAnswer interface{} `json:"correctAnswer,omitempty"` // 正确答案（仅在错误时显示）
	Explanation   string      `json:"explanation,omitempty"`   // 答案解析
	Status        string      `json:"status"`                  // "correct", "incorrect", "unanswered"
}

// checkAnswer 检查答案是否正确
func (s *LevelService) checkAnswer(question model.LevelQuestion, userAnswer interface{}) (bool, int, string) {
	// 根据问题类型检查答案
	switch question.QuestionType {
	case "multiple_choice":
		// 选择题：比较用户答案和正确答案
		var correctAnswer int
		if err := json.Unmarshal([]byte(question.CorrectAnswer), &correctAnswer); err != nil {
			return false, 0, "答案解析错误"
		}

		// 支持数字格式 (float64) 和字符串格式
		var userAnswerInt int
		if answerFloat, ok := userAnswer.(float64); ok {
			userAnswerInt = int(answerFloat)
		} else if answerStr, ok := userAnswer.(string); ok {
			// 将字符串转换为整数
			if parsed, err := strconv.Atoi(answerStr); err == nil {
				userAnswerInt = parsed
			} else {
				return false, 0, "答案格式错误：无效的选项索引"
			}
		} else {
			return false, 0, "答案格式错误：选择题答案必须是数字或字符串"
		}

		return userAnswerInt == correctAnswer, question.Points, question.Explanation

	case "fill_blank":
		// 填空题：字符串比较（可以扩展为更复杂的匹配逻辑）
		var correctAnswer string
		if err := json.Unmarshal([]byte(question.CorrectAnswer), &correctAnswer); err != nil {
			return false, 0, "答案解析错误"
		}

		if userAnswerStr, ok := userAnswer.(string); ok {
			return strings.TrimSpace(strings.ToLower(userAnswerStr)) == strings.TrimSpace(strings.ToLower(correctAnswer)), question.Points, question.Explanation
		}
		return false, 0, "答案格式错误"

	case "essay":
		// 论述题：通常需要人工评分，这里暂时返回待评分状态
		return false, 0, "论述题需要教师评分"

	case "code":
		// 编程题：通常需要运行测试，这里暂时返回待评分状态
		return false, 0, "编程题需要运行测试"

	default:
		return false, 0, "不支持的问题类型"
	}
}

// LevelRankingEntry 排行榜条目结构
type LevelRankingEntry struct {
	Ranking        int    `json:"ranking"`
	Username       string `json:"username"`
	BestLevelTitle string `json:"bestLevelTitle"`
	TotalScore     int    `json:"totalScore"`
}

// UserLevelStatsResponse 用户关卡挑战统计响应结构
type UserLevelStatsResponse struct {
	UserID             uint    `json:"userId"`
	WeeklyTimeHours    float64 `json:"weeklyTimeHours"`    // 本周挑战总时长（小时）
	AverageSuccessRate float64 `json:"averageSuccessRate"` // 平均成功率（百分比）
	SolvedChallenges   int     `json:"solvedChallenges"`   // 已解决的挑战总数
	TotalScore         int     `json:"totalScore"`         // 关卡挑战总积分
}

// GetLevelRanking 获取关卡挑战排行榜
func (s *LevelService) GetLevelRanking(limit int) ([]LevelRankingEntry, error) {
	// 修复积分计算逻辑：每个关卡只获得一次最高分，防止重复挑战刷分
	query := `
		WITH user_level_best_scores AS (
			SELECT
				la.user_id,
				la.level_id,
				MAX(la.score) as best_score
			FROM level_attempts la
			WHERE la.success = true
			GROUP BY la.user_id, la.level_id
		),
		user_stats AS (
			SELECT
				u.id as user_id,
				u.name as username,
				SUM(ulbs.best_score) as total_score,
				MAX(ulbs.best_score) as max_score
			FROM users u
			INNER JOIN user_level_best_scores ulbs ON u.id = ulbs.user_id
			WHERE u.role = 'student'
			GROUP BY u.id, u.name
			HAVING SUM(ulbs.best_score) > 0
		),
		user_best_levels AS (
			SELECT
				us.user_id,
				us.username,
				us.total_score,
				l.title as best_level_title,
				ROW_NUMBER() OVER (PARTITION BY us.user_id ORDER BY ulbs.best_score DESC) as rn
			FROM user_stats us
			INNER JOIN user_level_best_scores ulbs ON us.user_id = ulbs.user_id AND ulbs.best_score = us.max_score
			INNER JOIN levels l ON ulbs.level_id = l.id
		)
		SELECT
			ROW_NUMBER() OVER (ORDER BY total_score DESC, user_id ASC) as ranking,
			username,
			best_level_title,
			total_score
		FROM user_best_levels
		WHERE rn = 1
		ORDER BY total_score DESC, user_id ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	var rankings []LevelRankingEntry
	err := s.DB.Raw(query).Scan(&rankings).Error
	if err != nil {
		return nil, err
	}

	return rankings, nil
}

// GetUserLevelTotalScore 获取单个用户的关卡挑战总积分
func (s *LevelService) GetUserLevelTotalScore(userID uint) (int, error) {
	query := `
		WITH user_level_best_scores AS (
			SELECT
				la.user_id,
				la.level_id,
				MAX(la.score) as best_score
			FROM level_attempts la
			WHERE la.success = true AND la.user_id = ?
			GROUP BY la.user_id, la.level_id
		)
		SELECT COALESCE(SUM(best_score), 0) as total_score
		FROM user_level_best_scores
	`

	var totalScore int
	err := s.DB.Raw(query, userID).Scan(&totalScore).Error
	if err != nil {
		return 0, err
	}

	return totalScore, nil
}

// GetUserLevelStats 获取用户的关卡挑战统计数据
func (s *LevelService) GetUserLevelStats(userID uint) (*UserLevelStatsResponse, error) {
	// 计算本周的总时长（小时）
	weeklyTimeQuery := `
		SELECT COALESCE(SUM(total_time_seconds) / 3600.0, 0) as weekly_time_hours
		FROM level_attempts
		WHERE user_id = ?
			AND YEARWEEK(started_at, 1) = YEARWEEK(NOW(), 1)
			AND ended_at IS NOT NULL
	`

	// 计算平均成功率
	successRateQuery := `
		SELECT
			CASE
				WHEN COUNT(*) = 0 THEN 0
				ELSE ROUND((SUM(CASE WHEN success = true THEN 1 ELSE 0 END) * 100.0) / COUNT(*), 2)
			END as success_rate
		FROM level_attempts
		WHERE user_id = ? AND ended_at IS NOT NULL
	`

	// 计算已解决的挑战总数（成功完成的关卡数量）
	solvedChallengesQuery := `
		SELECT COUNT(DISTINCT level_id) as solved_count
		FROM level_attempts
		WHERE user_id = ? AND success = true
	`

	var weeklyTimeHours float64
	var averageSuccessRate float64
	var solvedChallenges int

	// 执行查询
	if err := s.DB.Raw(weeklyTimeQuery, userID).Scan(&weeklyTimeHours).Error; err != nil {
		return nil, err
	}

	if err := s.DB.Raw(successRateQuery, userID).Scan(&averageSuccessRate).Error; err != nil {
		return nil, err
	}

	if err := s.DB.Raw(solvedChallengesQuery, userID).Scan(&solvedChallenges).Error; err != nil {
		return nil, err
	}

	// 获取总积分
	totalScore, err := s.GetUserLevelTotalScore(userID)
	if err != nil {
		return nil, err
	}

	return &UserLevelStatsResponse{
		UserID:             userID,
		WeeklyTimeHours:    weeklyTimeHours,
		AverageSuccessRate: averageSuccessRate,
		SolvedChallenges:   solvedChallenges,
		TotalScore:         totalScore,
	}, nil
}
