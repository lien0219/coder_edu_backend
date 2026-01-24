package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"encoding/json"
	"errors"
)

type PostClassTestService struct {
	Repo *repository.PostClassTestRepository
}

func NewPostClassTestService(repo *repository.PostClassTestRepository) *PostClassTestService {
	return &PostClassTestService{Repo: repo}
}

type PostClassTestQuestionReq struct {
	ID           string          `json:"id"`
	QuestionType string          `json:"questionType" binding:"required"`
	Content      string          `json:"content" binding:"required"`
	Options      json.RawMessage `json:"options"`
	Answer       string          `json:"answer" binding:"required"`
	Points       int             `json:"points"`
	RewardXP     int             `json:"rewardXp"`
	Explanation  string          `json:"explanation"`
	Order        int             `json:"order"`
}

type PostClassTestReq struct {
	Title       *string                     `json:"title"`
	Description *string                     `json:"description"`
	TimeLimit   *int                        `json:"timeLimit"`
	IsPublished *bool                       `json:"isPublished"`
	Questions   *[]PostClassTestQuestionReq `json:"questions"`
}

func (s *PostClassTestService) CreateTest(creatorID uint, req PostClassTestReq) (*model.PostClassTest, error) {
	if req.Title == nil || *req.Title == "" {
		return nil, errors.New("title is required")
	}

	test := &model.PostClassTest{
		Title:     *req.Title,
		CreatorID: creatorID,
	}

	if req.Description != nil {
		test.Description = *req.Description
	}
	if req.TimeLimit != nil {
		test.TimeLimit = *req.TimeLimit
	}
	if req.IsPublished != nil {
		test.IsPublished = *req.IsPublished
	}

	if err := s.Repo.CreateTest(test); err != nil {
		return nil, err
	}

	if test.IsPublished {
		_ = s.Repo.UnpublishAllExcept(test.ID)
	}

	if req.Questions != nil {
		for _, qReq := range *req.Questions {
			q := &model.PostClassTestQuestion{
				TestID:       test.ID,
				QuestionType: qReq.QuestionType,
				Content:      qReq.Content,
				Options:      qReq.Options,
				Answer:       qReq.Answer,
				Points:       qReq.Points,
				RewardXP:     qReq.RewardXP,
				Explanation:  qReq.Explanation,
				Order:        qReq.Order,
			}
			if err := s.Repo.CreateQuestion(q); err != nil {
				return nil, err
			}
		}
	}

	return test, nil
}

func (s *PostClassTestService) UpdateTest(testID string, req PostClassTestReq) (*model.PostClassTest, error) {
	test, err := s.Repo.FindTestByID(testID)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		test.Title = *req.Title
	}
	if req.Description != nil {
		test.Description = *req.Description
	}
	if req.TimeLimit != nil {
		test.TimeLimit = *req.TimeLimit
	}
	if req.IsPublished != nil {
		test.IsPublished = *req.IsPublished
	}

	if err := s.Repo.UpdateTest(test); err != nil {
		return nil, err
	}

	if test.IsPublished {
		_ = s.Repo.UnpublishAllExcept(test.ID)
	}

	if req.Questions != nil {
		existingQs, _ := s.Repo.ListQuestions(testID)
		existingMap := make(map[string]*model.PostClassTestQuestion)
		for i := range existingQs {
			existingMap[existingQs[i].ID] = &existingQs[i]
		}

		newQIDs := make(map[string]bool)
		for _, qReq := range *req.Questions {
			if qReq.ID != "" {
				if q, ok := existingMap[qReq.ID]; ok {
					q.QuestionType = qReq.QuestionType
					q.Content = qReq.Content
					q.Options = qReq.Options
					q.Answer = qReq.Answer
					q.Points = qReq.Points
					q.RewardXP = qReq.RewardXP
					q.Explanation = qReq.Explanation
					q.Order = qReq.Order
					s.Repo.UpdateQuestion(q)
					newQIDs[q.ID] = true
				}
			} else {
				q := &model.PostClassTestQuestion{
					TestID:       testID,
					QuestionType: qReq.QuestionType,
					Content:      qReq.Content,
					Options:      qReq.Options,
					Answer:       qReq.Answer,
					Points:       qReq.Points,
					RewardXP:     qReq.RewardXP,
					Explanation:  qReq.Explanation,
					Order:        qReq.Order,
				}
				s.Repo.CreateQuestion(q)
			}
		}

		for id := range existingMap {
			if !newQIDs[id] {
				s.Repo.DeleteQuestion(id)
			}
		}
	}

	return test, nil
}

func (s *PostClassTestService) DeleteTest(testID string) error {
	return s.Repo.DeleteTest(testID)
}

func (s *PostClassTestService) GetTest(testID string) (*model.PostClassTest, []model.PostClassTestQuestion, error) {
	test, err := s.Repo.FindTestByID(testID)
	if err != nil {
		return nil, nil, err
	}
	qs, err := s.Repo.ListQuestions(testID)
	return test, qs, err
}

func (s *PostClassTestService) ListTests(page, limit int) ([]repository.PostClassTestListRow, int64, error) {
	return s.Repo.ListTests(page, limit)
}

func (s *PostClassTestService) ListSubmissions(testID string, page, limit int, studentName string) ([]map[string]interface{}, int64, error) {
	return s.Repo.ListSubmissions(testID, page, limit, studentName)
}

func (s *PostClassTestService) GetSubmissionDetail(submissionID string) (map[string]interface{}, error) {
	submission, answers, err := s.Repo.GetSubmissionDetail(submissionID)
	if err != nil {
		return nil, err
	}

	test, qs, err := s.GetTest(submission.TestID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"submission": submission,
		"answers":    answers,
		"test":       test,
		"questions":  qs,
	}, nil
}

func (s *PostClassTestService) ResetStudentTest(submissionID string) error {
	return s.Repo.DeleteSubmission(submissionID)
}

func (s *PostClassTestService) BatchResetStudentTests(submissionIDs []string) error {
	return s.Repo.BatchDeleteSubmissions(submissionIDs)
}
