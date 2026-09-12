package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/util"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
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

func (s *AssessmentService) ListQuestions(assessmentID uint, page, limit int) ([]model.AssessmentQuestion, int64, error) {
	if assessmentID == 0 {
		defaultA, err := s.getOrCreateDefaultAssessment()
		if err == nil {
			assessmentID = defaultA.ID
		}
	}
	return s.Repo.ListQuestions(assessmentID, page, limit)
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

type StudentPaperResponse struct {
	AssessmentID uint                        `json:"assessmentId"`
	Title        string                      `json:"title"`
	PaperVersion string                      `json:"paperVersion"`
	Questions    []StudentAssessmentQuestion `json:"questions"`
}

func (s *AssessmentService) ListStudentQuestions() (*StudentPaperResponse, error) {
	paper, qs, err := s.Repo.FindPublishedAssessmentWithQuestions()
	if err != nil {
		return nil, err
	}
	version, err := ComputePaperVersion(paper.ID, qs)
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
	return &StudentPaperResponse{
		AssessmentID: paper.ID,
		Title:        paper.Title,
		PaperVersion: version,
		Questions:    res,
	}, nil
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

type PublishAssessmentRequest struct {
	IsPublished bool `json:"isPublished"`
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

func (s *AssessmentService) SetAssessmentPublished(id uint, published bool) (*model.Assessment, error) {
	a, err := s.Repo.FindAssessmentByID(id)
	if err != nil {
		return nil, err
	}
	a.IsPublished = published
	if published {
		now := time.Now()
		a.PublishedAt = &now
	}
	if err := s.Repo.UpdateAssessment(a); err != nil {
		return nil, err
	}
	return a, nil
}

type AssessmentSubmissionRequest struct {
	AssessmentID    uint                   `json:"assessmentId"`
	PaperVersion    string                 `json:"paperVersion"`
	ClientRequestID string                 `json:"clientRequestId"`
	Answers         []model.QuestionAnswer `json:"answers"`
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}

func (s *AssessmentService) SubmitAssessment(userID uint, req AssessmentSubmissionRequest) (*model.AssessmentSubmission, error) {
	req.PaperVersion = strings.TrimSpace(req.PaperVersion)
	req.ClientRequestID = strings.TrimSpace(req.ClientRequestID)
	if req.AssessmentID == 0 || req.PaperVersion == "" || req.ClientRequestID == "" {
		return nil, util.ErrAssessmentSubmitMissingMeta
	}

	existing, err := s.Repo.FindSubmissionByClientRequestID(userID, req.ClientRequestID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return replayIfCompatible(existing, req)
	}

	paper, err := s.Repo.FindAssessmentByID(req.AssessmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.ErrAssessmentSubmitInvalid
		}
		return nil, err
	}
	if !paper.IsPublished {
		return nil, util.ErrAssessmentPaperUnavailable
	}

	questions, err := s.Repo.ListAllQuestions(req.AssessmentID)
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, util.ErrAssessmentPaperUnavailable
	}

	version, err := ComputePaperVersion(req.AssessmentID, questions)
	if err != nil {
		return nil, err
	}
	if version != req.PaperVersion {
		return nil, util.ErrAssessmentPaperStale
	}

	results, autoScore, objectiveMax, pendingManual, err := validateAndScoreAnswers(questions, req.Answers)
	if err != nil {
		return nil, err
	}

	answersJSON, err := json.Marshal(req.Answers)
	if err != nil {
		return nil, err
	}
	itemJSON, err := json.Marshal(results)
	if err != nil {
		return nil, err
	}

	var created *model.AssessmentSubmission
	txErr := s.Repo.WithTx(func(tx *repository.AssessmentRepository) error {
		replay, findErr := tx.FindSubmissionByClientRequestID(userID, req.ClientRequestID)
		if findErr != nil {
			return findErr
		}
		if replay != nil {
			matched, replayErr := replayIfCompatible(replay, req)
			if replayErr != nil {
				return replayErr
			}
			created = matched
			return nil
		}

		user, lockErr := tx.LockUserForUpdate(userID)
		if lockErr != nil {
			return lockErr
		}
		if !user.CanTakeAssessment {
			return util.ErrAssessmentRetestDenied
		}

		maxAttempt, maxErr := tx.MaxAttemptNo(userID, req.AssessmentID)
		if maxErr != nil {
			return maxErr
		}

		created = &model.AssessmentSubmission{
			UserID:             userID,
			AssessmentID:       req.AssessmentID,
			AttemptNo:          maxAttempt + 1,
			ClientRequestID:    req.ClientRequestID,
			PaperVersion:       req.PaperVersion,
			Answers:            answersJSON,
			ItemResults:        itemJSON,
			TotalScore:         autoScore,
			AutoScore:          autoScore,
			ObjectiveMax:       objectiveMax,
			PendingManualCount: pendingManual,
			ScoringStatus:      model.ScoringStatusAwaitingTeacher,
			Status:             model.SubmissionStatusPending,
			RecommendedLevel:   0,
		}
		if createErr := tx.CreateSubmission(created); createErr != nil {
			if isUniqueConstraintError(createErr) {
				replay, replayErr := tx.FindSubmissionByClientRequestID(userID, req.ClientRequestID)
				if replayErr != nil {
					return replayErr
				}
				if replay != nil {
					matched, replayErr := replayIfCompatible(replay, req)
					if replayErr != nil {
						return replayErr
					}
					created = matched
					return nil
				}
				return util.ErrAssessmentRetestDenied
			}
			return createErr
		}
		return tx.UpdateUserAssessmentStatus(userID, false)
	})
	if txErr != nil {
		return nil, txErr
	}
	return created, nil
}

func replayIfCompatible(existing *model.AssessmentSubmission, req AssessmentSubmissionRequest) (*model.AssessmentSubmission, error) {
	if existing.AssessmentID != req.AssessmentID || existing.PaperVersion != req.PaperVersion {
		return nil, util.ErrAssessmentIdempotencyConflict
	}
	if !sameSubmittedAnswers(existing.Answers, req.Answers) {
		return nil, util.ErrAssessmentIdempotencyConflict
	}
	return existing, nil
}

func sameSubmittedAnswers(stored json.RawMessage, incoming []model.QuestionAnswer) bool {
	var storedAns []model.QuestionAnswer
	if len(stored) > 0 {
		if err := json.Unmarshal(stored, &storedAns); err != nil {
			return false
		}
	}
	return answersFingerprint(storedAns) == answersFingerprint(incoming)
}

func answersFingerprint(answers []model.QuestionAnswer) string {
	cloned := append([]model.QuestionAnswer(nil), answers...)
	sort.Slice(cloned, func(i, j int) bool { return cloned[i].QuestionID < cloned[j].QuestionID })
	for i := range cloned {
		cloned[i].Answer = strings.TrimSpace(cloned[i].Answer)
	}
	raw, err := json.Marshal(cloned)
	if err != nil {
		return ""
	}
	return string(raw)
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
	submission.Status = model.SubmissionStatusCompleted
	submission.ScoringStatus = model.ScoringStatusTeacherCompleted

	return s.Repo.UpdateSubmission(submission)
}

func (s *AssessmentService) DeleteSubmission(id uint) error {
	return s.Repo.DeleteSubmission(id)
}

type ConfirmedDiagnosis struct {
	AssessmentID     uint   `json:"assessmentId"`
	SubmissionID     uint   `json:"submissionId"`
	AttemptNo        int    `json:"attemptNo"`
	RecommendedLevel int    `json:"recommendedLevel"`
	Status           string `json:"status"`
	PaperVersion     string `json:"paperVersion,omitempty"`
}

type StudentAssessmentStatus struct {
	Submission             *model.AssessmentSubmission `json:"submission"`
	LastConfirmedDiagnosis *ConfirmedDiagnosis         `json:"lastConfirmedDiagnosis"`
	StaleDiagnosis         *ConfirmedDiagnosis         `json:"staleDiagnosis,omitempty"`
	HistoricalDiagnosis    *ConfirmedDiagnosis         `json:"historicalDiagnosis,omitempty"`
	CanTakeAssessment      bool                        `json:"canTakeAssessment"`
	HasPublishedPaper      bool                        `json:"hasPublishedPaper"`
	AssessmentID           uint                        `json:"assessmentId,omitempty"`
	PaperVersion           string                      `json:"paperVersion,omitempty"`
}

func missingPaperVersion(v string) bool {
	return strings.TrimSpace(v) == ""
}

func diagnosisFromSubmission(sub *model.AssessmentSubmission) *ConfirmedDiagnosis {
	if sub == nil {
		return nil
	}
	return &ConfirmedDiagnosis{
		AssessmentID:     sub.AssessmentID,
		SubmissionID:     sub.ID,
		AttemptNo:        sub.AttemptNo,
		RecommendedLevel: sub.RecommendedLevel,
		Status:           sub.Status,
		PaperVersion:     sub.PaperVersion,
	}
}

func (s *AssessmentService) GetConfirmedDiagnosisForUser(userID uint) (*ConfirmedDiagnosis, error) {
	paper, qs, err := s.Repo.FindPublishedAssessmentWithQuestions()
	if err != nil {
		if errors.Is(err, util.ErrNoPublishedAssessment) {
			return nil, nil
		}
		return nil, err
	}
	version, err := ComputePaperVersion(paper.ID, qs)
	if err != nil {
		return nil, err
	}
	sub, err := s.Repo.FindConfirmedDiagnosis(userID, paper.ID, version)
	if err != nil {
		return nil, err
	}
	return diagnosisFromSubmission(sub), nil
}

func (s *AssessmentService) GetStudentAssessmentStatus(userID uint) (*StudentAssessmentStatus, error) {
	canTake, err := s.Repo.GetUserAssessmentStatus(userID)
	if err != nil {
		return nil, err
	}

	status := &StudentAssessmentStatus{
		CanTakeAssessment: canTake,
	}

	paper, qs, err := s.Repo.FindPublishedAssessmentWithQuestions()
	if errors.Is(err, util.ErrNoPublishedAssessment) {
		return status, nil
	}
	if err != nil {
		return nil, err
	}

	version, err := ComputePaperVersion(paper.ID, qs)
	if err != nil {
		return nil, err
	}

	status.HasPublishedPaper = true
	status.AssessmentID = paper.ID
	status.PaperVersion = version

	latest, err := s.Repo.FindLatestAttempt(userID, paper.ID)
	if err != nil {
		return nil, err
	}
	status.Submission = latest

	confirmed, err := s.Repo.FindConfirmedDiagnosis(userID, paper.ID, version)
	if err != nil {
		return nil, err
	}
	if confirmed != nil {
		status.LastConfirmedDiagnosis = diagnosisFromSubmission(confirmed)
		return status, nil
	}

	anyConfirmed, err := s.Repo.FindConfirmedDiagnosis(userID, paper.ID, "")
	if err != nil {
		return nil, err
	}
	if anyConfirmed == nil {
		return status, nil
	}
	if missingPaperVersion(anyConfirmed.PaperVersion) {
		status.HistoricalDiagnosis = diagnosisFromSubmission(anyConfirmed)
		return status, nil
	}
	if anyConfirmed.PaperVersion != version {
		status.StaleDiagnosis = diagnosisFromSubmission(anyConfirmed)
	}
	return status, nil
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
	canTake, err := s.Repo.GetUserAssessmentStatus(userID)
	if err != nil {
		return nil, err
	}

	result := &StudentAssessmentResult{
		CanTakeAssessment: canTake,
		HasSubmitted:      false,
		Status:            "untested",
	}

	paper, _, err := s.Repo.FindPublishedAssessmentWithQuestions()
	if errors.Is(err, util.ErrNoPublishedAssessment) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}

	submission, err := s.Repo.FindLatestAttempt(userID, paper.ID)
	if err != nil {
		return nil, err
	}
	if submission == nil {
		return result, nil
	}

	result.HasSubmitted = true
	result.Status = submission.Status
	result.TotalScore = submission.TotalScore
	result.Feedback = submission.Feedback
	result.RecommendedLevel = submission.RecommendedLevel
	return result, nil
}
