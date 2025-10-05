package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

// ExerciseSubmissionRepository 处理练习提交记录的数据库操作
type ExerciseSubmissionRepository struct {
	DB *gorm.DB
}

// NewExerciseSubmissionRepository 创建新的练习提交记录仓库实例
func NewExerciseSubmissionRepository(db *gorm.DB) *ExerciseSubmissionRepository {
	return &ExerciseSubmissionRepository{DB: db}
}

// Create 创建新的练习提交记录
func (r *ExerciseSubmissionRepository) Create(submission *model.ExerciseSubmission) error {
	return r.DB.Create(submission).Error
}

// FindByUserAndQuestion 检查用户是否提交过特定题目
func (r *ExerciseSubmissionRepository) FindByUserAndQuestion(userID, questionID uint) (*model.ExerciseSubmission, error) {
	var submission model.ExerciseSubmission
	err := r.DB.Where("user_id = ? AND question_id = ?", userID, questionID).First(&submission).Error
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

// Update 更新练习提交记录
func (r *ExerciseSubmissionRepository) Update(submission *model.ExerciseSubmission) error {
	return r.DB.Save(submission).Error
}
