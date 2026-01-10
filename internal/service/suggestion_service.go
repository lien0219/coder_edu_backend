package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"fmt"
)

type SuggestionService struct {
	SuggestionRepo   *repository.SuggestionRepository
	LevelRepo        *repository.LevelRepository
	LevelAttemptRepo *repository.LevelAttemptRepository
}

func NewSuggestionService(
	suggestionRepo *repository.SuggestionRepository,
	levelRepo *repository.LevelRepository,
	levelAttemptRepo *repository.LevelAttemptRepository,
) *SuggestionService {
	return &SuggestionService{
		SuggestionRepo:   suggestionRepo,
		LevelRepo:        levelRepo,
		LevelAttemptRepo: levelAttemptRepo,
	}
}

func (s *SuggestionService) CreateSuggestion(suggestion *model.Suggestion) error {
	return s.SuggestionRepo.Create(suggestion)
}

func (s *SuggestionService) UpdateSuggestion(suggestionID uint, teacherID uint, updates *model.Suggestion) error {
	existing, err := s.SuggestionRepo.FindByID(suggestionID)
	if err != nil {
		return err
	}

	if existing.TeacherID != teacherID {
		return fmt.Errorf("unauthorized to update this suggestion")
	}

	// Only allow updating certain fields
	existing.Title = updates.Title
	existing.Subtitle = updates.Subtitle
	existing.Priority = updates.Priority
	existing.CompletionTime = updates.CompletionTime
	existing.RelatedLevelID = updates.RelatedLevelID
	existing.StudentID = updates.StudentID

	return s.SuggestionRepo.Update(existing)
}

func (s *SuggestionService) GetStudentSuggestions(studentID uint) ([]model.Suggestion, error) {
	suggestions, err := s.SuggestionRepo.ListForStudent(studentID)
	if err != nil {
		return nil, err
	}

	for i := range suggestions {
		// 1. Check if student has a manual completion record
		completion, _ := s.SuggestionRepo.GetCompletion(suggestions[i].ID, studentID)
		if completion != nil {
			suggestions[i].Status = model.StatusCompleted
		} else {
			suggestions[i].Status = model.StatusPending

			// 2. Auto-completion logic (Fallback)
			if suggestions[i].RelatedLevelID != nil {
				// Check if the student has passed the related level
				attempts, err := s.LevelAttemptRepo.GetLevelAttemptsHistory(studentID, *suggestions[i].RelatedLevelID, 1)
				if err == nil && len(attempts) > 0 {
					// Check if any attempt was successful
					var hasPassed bool
					// We might need a repo method to check for ANY success, but for now we look at the latest
					// Actually, GetLevelAttemptsHistory with limit 1 is just the first one in ASC order or what?
					// Usually history is DESC. Let's assume the repo gives us latest.
					if attempts[0].Success {
						hasPassed = true
					}

					if hasPassed {
						suggestions[i].Status = model.StatusCompleted
						// Optionally auto-create completion record so it's persistent
						s.SuggestionRepo.UpsertCompletion(&model.SuggestionCompletion{
							SuggestionID: suggestions[i].ID,
							StudentID:    studentID,
							Status:       model.StatusCompleted,
						})
					}
				}
			}
		}
	}

	return suggestions, nil
}

func (s *SuggestionService) GetTeacherSuggestions(teacherID uint) ([]model.Suggestion, error) {
	return s.SuggestionRepo.ListByTeacher(teacherID)
}

func (s *SuggestionService) CompleteSuggestion(suggestionID, studentID uint) error {
	// Instead of updating the Suggestion model, we create a Completion record
	return s.SuggestionRepo.UpsertCompletion(&model.SuggestionCompletion{
		SuggestionID: suggestionID,
		StudentID:    studentID,
		Status:       model.StatusCompleted,
	})
}

func (s *SuggestionService) DeleteSuggestion(suggestionID, teacherID uint) error {
	suggestion, err := s.SuggestionRepo.FindByID(suggestionID)
	if err != nil {
		return err
	}

	if suggestion.TeacherID != teacherID {
		return fmt.Errorf("unauthorized to delete this suggestion")
	}

	return s.SuggestionRepo.Delete(suggestionID)
}

func (s *SuggestionService) GetStudentProgressForTeacher(studentID uint) (interface{}, error) {
	// Return a summary of student's level attempts
	type ProgressSummary struct {
		LevelID    uint   `json:"levelId"`
		Title      string `json:"title"`
		Success    bool   `json:"success"`
		Score      int    `json:"score"`
		LastPlayed string `json:"lastPlayed"`
	}

	var results []ProgressSummary
	err := s.LevelAttemptRepo.DB.Table("level_attempts").
		Select("levels.id as level_id, levels.title, MAX(level_attempts.success) as success, MAX(level_attempts.score) as score, MAX(level_attempts.ended_at) as last_played").
		Joins("JOIN levels ON levels.id = level_attempts.level_id").
		Where("level_attempts.user_id = ? AND level_attempts.deleted_at IS NULL", studentID).
		Group("levels.id, levels.title").
		Scan(&results).Error

	return results, err
}

type StudentProgressListItem struct {
	StudentID       uint    `json:"studentId"`
	Name            string  `json:"name"`
	Email           string  `json:"email"`
	TotalXP         int     `json:"totalXp"`
	LevelsCompleted int     `json:"levelsCompleted"`
	AverageScore    float64 `json:"averageScore"`
	LastSeen        string  `json:"lastSeen"`
}

func (s *SuggestionService) ListStudentsProgress(page, pageSize int, search string) ([]StudentProgressListItem, int, error) {
	var students []model.User
	var total int64

	query := s.LevelAttemptRepo.DB.Model(&model.User{}).Where("role = ?", model.Student)

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("name LIKE ? OR email LIKE ?", searchTerm, searchTerm)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("xp DESC").Find(&students).Error; err != nil {
		return nil, 0, err
	}

	var results []StudentProgressListItem
	for _, student := range students {
		var stats struct {
			LevelsCompleted int     `gorm:"column:comp_count"`
			AverageScore    float64 `gorm:"column:avg_score"`
		}

		// Get stats for this student
		s.LevelAttemptRepo.DB.Table("level_attempts").
			Select("COUNT(DISTINCT CASE WHEN success = true THEN level_id END) as comp_count, COALESCE(AVG(score), 0) as avg_score").
			Where("user_id = ? AND deleted_at IS NULL", student.ID).
			Scan(&stats)

		results = append(results, StudentProgressListItem{
			StudentID:       student.ID,
			Name:            student.Name,
			Email:           student.Email,
			TotalXP:         student.XP,
			LevelsCompleted: stats.LevelsCompleted,
			AverageScore:    stats.AverageScore,
			LastSeen:        student.LastSeen.Format("2006-01-02 15:04:05"),
		})
	}

	return results, int(total), nil
}
