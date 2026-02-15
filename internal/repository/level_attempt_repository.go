package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type LevelAttemptRepository struct {
	DB *gorm.DB
}

func NewLevelAttemptRepository(db *gorm.DB) *LevelAttemptRepository {
	return &LevelAttemptRepository{DB: db}
}

func (r *LevelAttemptRepository) Create(attempt *model.LevelAttempt) error {
	return r.DB.Create(attempt).Error
}

func (r *LevelAttemptRepository) Update(attempt *model.LevelAttempt) error {
	return r.DB.Save(attempt).Error
}

func (r *LevelAttemptRepository) FindByID(id uint) (*model.LevelAttempt, error) {
	var a model.LevelAttempt
	if err := r.DB.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *LevelAttemptRepository) CountByUserAndLevel(userID, levelID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.LevelAttempt{}).Where("user_id = ? AND level_id = ?", userID, levelID).Count(&count).Error
	return count, err
}

func (r *LevelAttemptRepository) CreateQuestionTimes(times []model.LevelAttemptQuestionTime) error {
	if len(times) == 0 {
		return nil
	}
	return r.DB.Create(&times).Error
}

func (r *LevelAttemptRepository) GetQuestionTimes(attemptID uint) ([]model.LevelAttemptQuestionTime, error) {
	var times []model.LevelAttemptQuestionTime
	err := r.DB.Where("attempt_id = ?", attemptID).Find(&times).Error
	return times, err
}

func (r *LevelAttemptRepository) CreateAnswers(answers []model.LevelAttemptAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	return r.DB.Create(&answers).Error
}

func (r *LevelAttemptRepository) GetAnswers(attemptID uint) ([]model.LevelAttemptAnswer, error) {
	var answers []model.LevelAttemptAnswer
	err := r.DB.Where("attempt_id = ?", attemptID).Find(&answers).Error
	return answers, err
}

func (r *LevelAttemptRepository) CreateOrUpdateQuestionScores(scores []model.LevelAttemptQuestionScore) error {
	if len(scores) == 0 {
		return nil
	}
	for _, s := range scores {
		var existing model.LevelAttemptQuestionScore
		err := r.DB.Where("attempt_id = ? AND question_id = ?", s.AttemptID, s.QuestionID).First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if existing.ID == 0 {
			if err := r.DB.Create(&s).Error; err != nil {
				return err
			}
		} else {
			existing.Score = s.Score
			existing.Comment = s.Comment
			existing.GraderID = s.GraderID
			now := s.GradedAt
			existing.GradedAt = now
			if err := r.DB.Save(&existing).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *LevelAttemptRepository) GetQuestionScores(attemptID uint) ([]model.LevelAttemptQuestionScore, error) {
	var scores []model.LevelAttemptQuestionScore
	err := r.DB.Where("attempt_id = ?", attemptID).Find(&scores).Error
	return scores, err
}

func (r *LevelAttemptRepository) GetWeeklyStats(userID uint, weeks int, specificWeek string) ([]model.ChallengeWeeklyData, error) {
	var stats []model.ChallengeWeeklyData

	db := r.DB.Table("level_attempts").
		Select("DATE_FORMAT(started_at, '%x-%v') as week, AVG(score) as average_score, COUNT(CASE WHEN success = true THEN 1 END) as completed_count").
		Where("user_id = ? AND ended_at IS NOT NULL AND deleted_at IS NULL", userID).
		Group("week").
		Order("week DESC")

	if specificWeek != "" {
		db = db.Where("DATE_FORMAT(started_at, '%x-%v') = ?", specificWeek)
	} else {
		db = db.Limit(weeks)
	}

	err := db.Scan(&stats).Error
	return stats, err
}

func (r *LevelAttemptRepository) GetAbilityScores(userID uint) (map[uint]float64, error) {
	type Result struct {
		AbilityID uint
		AvgScore  float64
	}
	var results []Result
	err := r.DB.Table("level_attempts att").
		Select("la.ability_id, AVG(att.score) as avg_score").
		Joins("JOIN level_abilities la ON att.level_id = la.level_id").
		Where("att.user_id = ? AND att.success = ? AND att.deleted_at IS NULL", userID, true).
		Group("la.ability_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	scoreMap := make(map[uint]float64)
	for _, r := range results {
		scoreMap[r.AbilityID] = r.AvgScore
	}
	return scoreMap, nil
}

func (r *LevelAttemptRepository) GetLatestAttemptLevelID(userID uint) (uint, error) {
	var levelID uint
	err := r.DB.Model(&model.LevelAttempt{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("started_at DESC").
		Limit(1).
		Pluck("level_id", &levelID).Error
	return levelID, err
}

func (r *LevelAttemptRepository) GetLevelAttemptsHistory(userID, levelID uint, limit int) ([]model.LevelAttempt, error) {
	var attempts []model.LevelAttempt
	err := r.DB.Where("user_id = ? AND level_id = ? AND ended_at IS NOT NULL AND deleted_at IS NULL", userID, levelID).
		Order("started_at ASC").
		Limit(limit).
		Find(&attempts).Error
	return attempts, err
}

func (r *LevelAttemptRepository) GetLevelStats(levelID uint) (total int64, successful int64, totalSuccessfulScore int, err error) {
	err = r.DB.Model(&model.LevelAttempt{}).Where("level_id = ? AND deleted_at IS NULL", levelID).Count(&total).Error
	if err != nil {
		return
	}
	if total == 0 {
		return
	}
	err = r.DB.Model(&model.LevelAttempt{}).Where("level_id = ? AND success = ? AND deleted_at IS NULL", levelID, true).Count(&successful).Error
	if err != nil {
		return
	}
	if successful > 0 {
		err = r.DB.Model(&model.LevelAttempt{}).Where("level_id = ? AND success = ? AND deleted_at IS NULL", levelID, true).Select("SUM(score)").Scan(&totalSuccessfulScore).Error
	}
	return
}

func (r *LevelAttemptRepository) GetTotalManualScore(attemptID uint) (int64, error) {
	var total int64
	err := r.DB.Model(&model.LevelAttemptQuestionScore{}).Where("attempt_id = ?", attemptID).Select("SUM(score)").Scan(&total).Error
	return total, err
}

func (r *LevelAttemptRepository) GetAnswerByQuestion(attemptID, questionID uint) (*model.LevelAttemptAnswer, error) {
	var ans model.LevelAttemptAnswer
	err := r.DB.Where("attempt_id = ? AND question_id = ?", attemptID, questionID).First(&ans).Error
	return &ans, err
}

func (r *LevelAttemptRepository) ListNeedingManual(levelID uint) ([]model.LevelAttempt, error) {
	var attempts []model.LevelAttempt
	err := r.DB.Where("level_id = ? AND needs_manual = ?", levelID, true).Find(&attempts).Error
	return attempts, err
}
