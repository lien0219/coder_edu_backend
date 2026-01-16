package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"encoding/json"
)

type AssessmentService struct {
	Repo *repository.AssessmentRepository
}

func NewAssessmentService(repo *repository.AssessmentRepository) *AssessmentService {
	return &AssessmentService{Repo: repo}
}

type AssessmentQuestionRequest struct {
	AssessmentID uint            `json:"assessmentId"` // 可选，用于后续扩展
	QuestionType string          `json:"questionType" binding:"required"`
	Title        string          `json:"title"`
	Content      string          `json:"content" binding:"required"`
	Options      json.RawMessage `json:"options"`
	Answer       string          `json:"answer"`
	Points       int             `json:"points"`
	Order        int             `json:"order"`
	Explanation  string          `json:"explanation"`
}

func (s *AssessmentService) getOrCreateDefaultAssessment() (*model.Assessment, error) {
	as, _, err := s.Repo.ListAssessments(1, 1)
	if err == nil && len(as) > 0 {
		return &as[0], nil
	}

	newAssessment := &model.Assessment{
		Title:       "学前测试评估",
		Description: "默认学前测试评估题库",
		TimeLimit:   0,
	}
	if err := s.Repo.CreateAssessment(newAssessment); err != nil {
		return nil, err
	}
	return newAssessment, nil
}

func (s *AssessmentService) CreateQuestion(req AssessmentQuestionRequest) (*model.AssessmentQuestion, error) {
	if req.AssessmentID == 0 {
		defaultA, err := s.getOrCreateDefaultAssessment()
		if err != nil {
			return nil, err
		}
		req.AssessmentID = defaultA.ID
	}

	q := &model.AssessmentQuestion{
		AssessmentID: req.AssessmentID,
		QuestionType: req.QuestionType,
		Title:        req.Title,
		Content:      req.Content,
		Options:      req.Options,
		Answer:       req.Answer,
		Points:       req.Points,
		Order:        req.Order,
		Explanation:  req.Explanation,
	}
	if err := s.Repo.CreateQuestion(q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *AssessmentService) ListQuestions(assessmentID uint) ([]model.AssessmentQuestion, error) {
	if assessmentID == 0 {
		defaultA, err := s.getOrCreateDefaultAssessment()
		if err == nil {
			assessmentID = defaultA.ID
		}
	}
	return s.Repo.ListAllQuestions(assessmentID)
}

type StudentAssessmentQuestion struct {
	ID           uint            `json:"id"`
	QuestionType string          `json:"questionType"`
	Title        string          `json:"title"`
	Content      string          `json:"content"`
	Options      json.RawMessage `json:"options"`
	Points       int             `json:"points"`
	Order        int             `json:"order"`
}

func (s *AssessmentService) ListStudentQuestions() ([]StudentAssessmentQuestion, error) {
	defaultA, err := s.getOrCreateDefaultAssessment()
	if err != nil {
		return nil, err
	}

	qs, err := s.Repo.ListAllQuestions(defaultA.ID)
	if err != nil {
		return nil, err
	}

	res := make([]StudentAssessmentQuestion, len(qs))
	for i, q := range qs {
		res[i] = StudentAssessmentQuestion{
			ID:           q.ID,
			QuestionType: q.QuestionType,
			Title:        q.Title,
			Content:      q.Content,
			Options:      q.Options,
			Points:       q.Points,
			Order:        q.Order,
		}
	}
	return res, nil
}

func (s *AssessmentService) GetQuestion(id uint) (*model.AssessmentQuestion, error) {
	return s.Repo.FindQuestionByID(id)
}

func (s *AssessmentService) UpdateQuestion(id uint, req AssessmentQuestionRequest) (*model.AssessmentQuestion, error) {
	q, err := s.Repo.FindQuestionByID(id)
	if err != nil {
		return nil, err
	}

	if req.AssessmentID == 0 {
		req.AssessmentID = q.AssessmentID
	}

	q.AssessmentID = req.AssessmentID
	q.QuestionType = req.QuestionType
	q.Title = req.Title
	q.Content = req.Content
	q.Options = req.Options
	q.Answer = req.Answer
	q.Points = req.Points
	q.Order = req.Order
	q.Explanation = req.Explanation
	if err := s.Repo.UpdateQuestion(q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *AssessmentService) DeleteQuestion(id uint) error {
	return s.Repo.DeleteQuestion(id)
}

type AssessmentRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	TimeLimit   int    `json:"timeLimit"`
}

func (s *AssessmentService) CreateAssessment(req AssessmentRequest) (*model.Assessment, error) {
	a := &model.Assessment{
		Title:       req.Title,
		Description: req.Description,
		TimeLimit:   req.TimeLimit,
	}
	if err := s.Repo.CreateAssessment(a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AssessmentService) ListAssessments(page, limit int) ([]model.Assessment, int64, error) {
	return s.Repo.ListAssessments(page, limit)
}

func (s *AssessmentService) GetAssessment(id uint) (*model.Assessment, error) {
	return s.Repo.FindAssessmentByID(id)
}

type AssessmentSubmissionRequest struct {
	AssessmentID uint                   `json:"assessmentId"`
	Answers      []model.QuestionAnswer `json:"answers"`
}

func (s *AssessmentService) SubmitAssessment(userID uint, req AssessmentSubmissionRequest) (*model.AssessmentSubmission, error) {
	if req.AssessmentID == 0 {
		defaultA, err := s.getOrCreateDefaultAssessment()
		if err != nil {
			return nil, err
		}
		req.AssessmentID = defaultA.ID
	}

	questions, err := s.Repo.ListAllQuestions(req.AssessmentID)
	if err != nil {
		return nil, err
	}

	questionMap := make(map[uint]model.AssessmentQuestion)
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	totalScore := 0
	for _, ans := range req.Answers {
		if q, ok := questionMap[ans.QuestionID]; ok {
			if ans.Answer == q.Answer {
				totalScore += q.Points
			}
		}
	}

	answersJSON, _ := json.Marshal(req.Answers)

	// 查找是否已存在提交记录
	existing, err := s.Repo.FindSubmissionByUserAndAssessment(userID, req.AssessmentID)
	if err == nil && existing != nil {
		// 如果存在，则覆盖旧数据
		existing.Answers = answersJSON
		existing.TotalScore = totalScore
		existing.Status = "pending"   // 重新设为待审核
		existing.Feedback = ""        // 清空旧评语
		existing.RecommendedLevel = 0 // 重置建议等级
		if err := s.Repo.UpdateSubmission(existing); err != nil {
			return nil, err
		}
		_ = s.Repo.UpdateUserAssessmentStatus(userID, false)
		return existing, nil
	}

	// 如果不存在，创建新记录
	submission := &model.AssessmentSubmission{
		UserID:       userID,
		AssessmentID: req.AssessmentID,
		Answers:      answersJSON,
		TotalScore:   totalScore,
		Status:       "pending",
	}

	if err := s.Repo.CreateSubmission(submission); err != nil {
		return nil, err
	}

	// 自动更新用户状态为不可重测
	_ = s.Repo.UpdateUserAssessmentStatus(userID, false)

	return submission, nil
}

func (s *AssessmentService) ListSubmissions(page, limit int, status string, studentName string) ([]model.AssessmentSubmission, int64, error) {
	if status == "all" {
		status = ""
	}
	return s.Repo.ListSubmissions(page, limit, status, studentName)
}

func (s *AssessmentService) SetUserCanRetest(userIDs []uint, canTake bool) error {
	return s.Repo.BatchUpdateUserAssessmentStatus(userIDs, canTake)
}

func (s *AssessmentService) GetUserAssessmentStatus(userID uint) (bool, error) {
	return s.Repo.GetUserAssessmentStatus(userID)
}

type SubmissionDetailResponse struct {
	Submission *model.AssessmentSubmission `json:"submission"`
	Questions  []model.AssessmentQuestion  `json:"questions"`
}

func (s *AssessmentService) GetSubmissionDetail(id uint) (*SubmissionDetailResponse, error) {
	submission, err := s.Repo.FindSubmissionByID(id)
	if err != nil {
		return nil, err
	}

	questions, err := s.Repo.ListAllQuestions(submission.AssessmentID)
	if err != nil {
		return nil, err
	}

	return &SubmissionDetailResponse{
		Submission: submission,
		Questions:  questions,
	}, nil
}

type GradeSubmissionRequest struct {
	Score            int    `json:"score"`
	Feedback         string `json:"feedback"`
	RecommendedLevel int    `json:"recommendedLevel"`
}

func (s *AssessmentService) GradeSubmission(id uint, req GradeSubmissionRequest) error {
	submission, err := s.Repo.FindSubmissionByID(id)
	if err != nil {
		return err
	}

	submission.TotalScore = req.Score
	submission.Feedback = req.Feedback
	submission.RecommendedLevel = req.RecommendedLevel
	submission.Status = "completed"

	return s.Repo.UpdateSubmission(submission)
}

func (s *AssessmentService) DeleteSubmission(id uint) error {
	return s.Repo.DeleteSubmission(id)
}

type StudentAssessmentStatus struct {
	Submission        *model.AssessmentSubmission `json:"submission"`
	CanTakeAssessment bool                        `json:"canTakeAssessment"`
}

func (s *AssessmentService) GetStudentAssessmentStatus(userID uint) (*StudentAssessmentStatus, error) {
	// 获取用户重测权限状态
	canTake, err := s.Repo.GetUserAssessmentStatus(userID)
	if err != nil {
		return nil, err
	}

	// 获取默认评估 ID
	defaultA, err := s.getOrCreateDefaultAssessment()
	if err != nil {
		return nil, err
	}

	// 获取提交记录
	submission, _ := s.Repo.FindSubmissionByUserAndAssessment(userID, defaultA.ID)

	return &StudentAssessmentStatus{
		Submission:        submission,
		CanTakeAssessment: canTake,
	}, nil
}

type StudentAssessmentResult struct {
	HasSubmitted      bool   `json:"hasSubmitted"`
	Status            string `json:"status"` // pending, completed, untested
	TotalScore        int    `json:"totalScore"`
	Feedback          string `json:"feedback"`
	RecommendedLevel  int    `json:"recommendedLevel"`
	CanTakeAssessment bool   `json:"canTakeAssessment"`
}

func (s *AssessmentService) GetStudentAssessmentResult(userID uint) (*StudentAssessmentResult, error) {
	// 获取用户测试状态
	canTake, err := s.Repo.GetUserAssessmentStatus(userID)
	if err != nil {
		return nil, err
	}

	// 获取默认测试 ID
	defaultA, err := s.getOrCreateDefaultAssessment()
	if err != nil {
		return nil, err
	}

	// 获取提交记录
	submission, err := s.Repo.FindSubmissionByUserAndAssessment(userID, defaultA.ID)

	result := &StudentAssessmentResult{
		CanTakeAssessment: canTake,
	}

	if err != nil {
		// 没有提交记录
		result.HasSubmitted = false
		result.Status = "untested"
		return result, nil
	}

	// 有提交记录
	result.HasSubmitted = true
	result.Status = submission.Status
	result.TotalScore = submission.TotalScore
	result.Feedback = submission.Feedback
	result.RecommendedLevel = submission.RecommendedLevel

	return result, nil
}
