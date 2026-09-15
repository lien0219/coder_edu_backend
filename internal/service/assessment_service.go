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
	AssessmentID      uint            `json:"assessmentId"` // 可选，用于后续扩展
	QuestionType      string          `json:"questionType" binding:"required"`
	Title             string          `json:"title"`
	Content           string          `json:"content" binding:"required"`
	Options           json.RawMessage `json:"options"`
	Answer            string          `json:"answer"`
	Points            int             `json:"points"`
	Order             int             `json:"order"`
	Explanation       string          `json:"explanation"`
	KnowledgePointIDs *[]string       `json:"knowledgePointIds"` // nil=不改关联；[]=清空
}

type AssessmentQuestionView struct {
	model.AssessmentQuestion
	KnowledgePointIDs []string `json:"knowledgePointIds"`
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

func emptyKnowledgePointIDs() []string {
	return []string{}
}

func (s *AssessmentService) attachQuestionKnowledgePoints(qs []model.AssessmentQuestion) ([]AssessmentQuestionView, error) {
	ids := make([]uint, len(qs))
	for i, q := range qs {
		ids[i] = q.ID
	}
	linked, err := s.Repo.ListKnowledgePointIDsByQuestionIDs(ids)
	if err != nil {
		return nil, err
	}
	views := make([]AssessmentQuestionView, len(qs))
	for i, q := range qs {
		kps := linked[q.ID]
		if kps == nil {
			kps = emptyKnowledgePointIDs()
		}
		views[i] = AssessmentQuestionView{AssessmentQuestion: q, KnowledgePointIDs: kps}
	}
	return views, nil
}

func (s *AssessmentService) CreateQuestion(req AssessmentQuestionRequest) (*AssessmentQuestionView, error) {
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
	err := s.Repo.WithTx(func(tx *repository.AssessmentRepository) error {
		if err := tx.CreateQuestion(q); err != nil {
			return err
		}
		if req.KnowledgePointIDs != nil {
			return tx.ReplaceQuestionKnowledgePoints(q.ID, *req.KnowledgePointIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	views, err := s.attachQuestionKnowledgePoints([]model.AssessmentQuestion{*q})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

func (s *AssessmentService) ListQuestions(assessmentID uint, page, limit int) ([]AssessmentQuestionView, int64, error) {
	if assessmentID == 0 {
		defaultA, err := s.getOrCreateDefaultAssessment()
		if err == nil {
			assessmentID = defaultA.ID
		}
	}
	qs, total, err := s.Repo.ListQuestions(assessmentID, page, limit)
	if err != nil {
		return nil, 0, err
	}
	views, err := s.attachQuestionKnowledgePoints(qs)
	if err != nil {
		return nil, 0, err
	}
	return views, total, nil
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

func sanitizeStudentOptions(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("[]")
	}
	var objects []map[string]interface{}
	if err := json.Unmarshal(raw, &objects); err == nil {
		clean := make([]map[string]string, 0, len(objects))
		for _, opt := range objects {
			item := map[string]string{}
			if v := stringifyOpt(opt["label"]); v != "" {
				item["label"] = v
			}
			if v := stringifyOpt(opt["text"]); v != "" {
				item["text"] = v
			}
			clean = append(clean, item)
		}
		out, err := json.Marshal(clean)
		if err != nil {
			return json.RawMessage("[]")
		}
		return out
	}
	var generic []interface{}
	if err := json.Unmarshal(raw, &generic); err == nil {
		out, err := json.Marshal(generic)
		if err != nil {
			return json.RawMessage("[]")
		}
		return out
	}
	return json.RawMessage("[]")
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
			Options:      sanitizeStudentOptions(q.Options),
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

func (s *AssessmentService) GetQuestion(id uint) (*AssessmentQuestionView, error) {
	q, err := s.Repo.FindQuestionByID(id)
	if err != nil {
		return nil, err
	}
	views, err := s.attachQuestionKnowledgePoints([]model.AssessmentQuestion{*q})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

func (s *AssessmentService) UpdateQuestion(id uint, req AssessmentQuestionRequest) (*AssessmentQuestionView, error) {
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
	err = s.Repo.WithTx(func(tx *repository.AssessmentRepository) error {
		if err := tx.UpdateQuestion(q); err != nil {
			return err
		}
		if req.KnowledgePointIDs != nil {
			return tx.ReplaceQuestionKnowledgePoints(q.ID, *req.KnowledgePointIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	views, err := s.attachQuestionKnowledgePoints([]model.AssessmentQuestion{*q})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
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

		if snapErr := attachKnowledgePointSnapshot(tx, questions, results); snapErr != nil {
			return snapErr
		}
		itemJSON, marshalErr := json.Marshal(results)
		if marshalErr != nil {
			return marshalErr
		}

		maxAttempt, maxErr := tx.MaxAttemptNo(userID, req.AssessmentID)
		if maxErr != nil {
			return maxErr
		}

		status, scoringStatus, level := applyObjectiveAutoComplete(questions, results, autoScore, objectiveMax)
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
			ScoringStatus:      scoringStatus,
			Status:             status,
			RecommendedLevel:   level,
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

func attachKnowledgePointSnapshot(repo *repository.AssessmentRepository, questions []model.AssessmentQuestion, results []model.AssessmentItemResult) error {
	ids := make([]uint, len(questions))
	for i, q := range questions {
		ids[i] = q.ID
	}
	linked, err := repo.ListKnowledgePointIDsByQuestionIDs(ids)
	if err != nil {
		return err
	}
	for i := range results {
		kps := linked[results[i].QuestionID]
		if kps == nil {
			kps = emptyKnowledgePointIDs()
		}
		results[i].KnowledgePointIDs = kps
	}
	return nil
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
	Score            *int   `json:"score"`
	Feedback         string `json:"feedback"`
	RecommendedLevel int    `json:"recommendedLevel"`
}

func (s *AssessmentService) GradeSubmission(id uint, req GradeSubmissionRequest) error {
	submission, err := s.Repo.FindSubmissionByID(id)
	if err != nil {
		return err
	}

	// 推荐等级与分数独立：纯客观卷覆盖等级时保留 autoScore/totalScore，
	// 不按基础/初级/中级/高级反推 60/75/85/100。
	if shouldPreserveObjectiveAutoScore(submission) {
		submission.TotalScore = submission.AutoScore
	} else if req.Score != nil {
		submission.TotalScore = *req.Score
	}
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
	ScoringStatus    string `json:"scoringStatus,omitempty"`
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
	KnowledgeMastery       []KnowledgeMastery          `json:"knowledgeMastery,omitempty"`
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
		ScoringStatus:    sub.ScoringStatus,
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

func (s *AssessmentService) attachKnowledgeMastery(status *StudentAssessmentStatus, latest *model.AssessmentSubmission, questions []model.AssessmentQuestion) error {
	if status == nil || latest == nil {
		return nil
	}
	var items []model.AssessmentItemResult
	if len(latest.ItemResults) > 0 {
		if err := json.Unmarshal(latest.ItemResults, &items); err != nil {
			status.KnowledgeMastery = []KnowledgeMastery{}
			return nil
		}
	}
	kpIDs := make([]string, 0)
	seen := map[string]struct{}{}
	for _, item := range items {
		for _, id := range item.KnowledgePointIDs {
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			kpIDs = append(kpIDs, id)
		}
	}
	metas := map[string]KnowledgePointMasteryMeta{}
	if len(kpIDs) > 0 {
		rows, err := s.Repo.FindKnowledgePointMasteryMeta(kpIDs)
		if err != nil {
			return err
		}
		for id, kp := range rows {
			metas[id] = KnowledgePointMasteryMeta{
				ID:              kp.ID,
				Title:           kp.Title,
				Order:           kp.Order,
				CompletionScore: kp.CompletionScore,
			}
		}
	}
	fallbackMax := map[uint]int{}
	for _, q := range questions {
		fallbackMax[q.ID] = q.Points
	}
	status.KnowledgeMastery = AggregateKnowledgeMastery(items, metas, fallbackMax)
	return nil
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
	if err := s.attachKnowledgeMastery(status, latest, qs); err != nil {
		return nil, err
	}

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
