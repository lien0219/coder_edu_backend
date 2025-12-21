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
