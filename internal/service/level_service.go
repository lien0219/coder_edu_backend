package service

import (
	"encoding/json"
	"errors"
	"time"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"

	"gorm.io/gorm"
)

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
	AvailableFrom    *time.Time             `json:"availableFrom"`
	AvailableTo      *time.Time             `json:"availableTo"`
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
			AvailableFrom:    req.AvailableFrom,
			AvailableTo:      req.AvailableTo,
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
		level.AvailableFrom = req.AvailableFrom
		level.AvailableTo = req.AvailableTo

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
			// ensure questions.LevelID set to level.ID
			for i := range snap.Questions {
				snap.Questions[i].LevelID = level.ID
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
		LevelID:      levelID,
		UserID:       userID,
		AttemptsUsed: int(count) + 1,
		StartedAt:    time.Now(),
		VersionID:    level.CurrentVersion,
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
	level.VisibleTo = string(vtBytes)
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
