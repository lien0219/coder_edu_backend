package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type SuggestionRepository struct {
	DB *gorm.DB
}

func NewSuggestionRepository(db *gorm.DB) *SuggestionRepository {
	return &SuggestionRepository{DB: db}
}

func (r *SuggestionRepository) Create(suggestion *model.Suggestion) error {
	return r.DB.Create(suggestion).Error
}

func (r *SuggestionRepository) Update(suggestion *model.Suggestion) error {
	return r.DB.Save(suggestion).Error
}

func (r *SuggestionRepository) FindByID(id uint) (*model.Suggestion, error) {
	var suggestion model.Suggestion
	err := r.DB.First(&suggestion, id).Error
	return &suggestion, err
}

func (r *SuggestionRepository) ListForStudent(studentID uint) ([]model.Suggestion, error) {
	var suggestions []model.Suggestion
	// Find suggestions assigned to this student OR all students (student_id = 0)
	err := r.DB.Where("student_id = ? OR student_id = 0", studentID).Order("created_at desc").Find(&suggestions).Error
	return suggestions, err
}

func (r *SuggestionRepository) ListByTeacher(teacherID uint) ([]model.Suggestion, error) {
	var suggestions []model.Suggestion
	err := r.DB.Where("teacher_id = ?", teacherID).Order("created_at desc").Find(&suggestions).Error
	return suggestions, err
}

func (r *SuggestionRepository) Delete(id uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// Also delete completions when a suggestion is deleted
		if err := tx.Where("suggestion_id = ?", id).Delete(&model.SuggestionCompletion{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Suggestion{}, id).Error
	})
}

// Completion related methods

func (r *SuggestionRepository) GetCompletion(suggestionID, studentID uint) (*model.SuggestionCompletion, error) {
	var completion model.SuggestionCompletion
	err := r.DB.Where("suggestion_id = ? AND student_id = ?", suggestionID, studentID).First(&completion).Error
	if err != nil {
		return nil, err
	}
	return &completion, nil
}

func (r *SuggestionRepository) UpsertCompletion(completion *model.SuggestionCompletion) error {
	var existing model.SuggestionCompletion
	err := r.DB.Where("suggestion_id = ? AND student_id = ?", completion.SuggestionID, completion.StudentID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.DB.Create(completion).Error
	} else if err != nil {
		return err
	}
	existing.Status = completion.Status
	return r.DB.Save(&existing).Error
}
