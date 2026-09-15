package service

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/util"

	"gorm.io/gorm"
)

func openRecommendTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openAssessmentTestDB(t)
	stmts := []string{
		`CREATE TABLE knowledge_points (
			id TEXT PRIMARY KEY,
			title TEXT,
			description TEXT,
			type TEXT,
			article_content TEXT,
			time_limit INTEGER DEFAULT 0,
			` + "`order`" + ` INTEGER DEFAULT 0,
			completion_score INTEGER DEFAULT 0,
			tags TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE learning_path_material_knowledge_points (
			material_id TEXT NOT NULL,
			knowledge_point_id TEXT NOT NULL,
			PRIMARY KEY (material_id, knowledge_point_id)
		)`,
		`CREATE TABLE learning_path_materials (
			id TEXT PRIMARY KEY,
			level INTEGER NOT NULL,
			total_chapters INTEGER DEFAULT 0,
			chapter_number INTEGER DEFAULT 0,
			title TEXT NOT NULL,
			content TEXT,
			points INTEGER DEFAULT 0,
			creator_id INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE learning_path_completions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			material_id TEXT,
			completed_at DATETIME
		)`,
		`CREATE UNIQUE INDEX idx_learning_path_completion_user_material
			ON learning_path_completions (user_id, material_id)`,
		`CREATE TABLE learning_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			user_id INTEGER,
			module_id INTEGER,
			activity TEXT,
			content TEXT,
			tags TEXT,
			insights TEXT,
			challenges TEXT,
			next_steps TEXT,
			duration INTEGER DEFAULT 0,
			completed INTEGER DEFAULT 0,
			score INTEGER DEFAULT 0
		)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func newPathBundle(t *testing.T) (*AssessmentService, *LearningPathService, *gorm.DB) {
	t.Helper()
	db := openRecommendTestDB(t)
	aRepo := repository.NewAssessmentRepository(db)
	pRepo := repository.NewLearningPathRepository(db)
	return NewAssessmentService(aRepo), NewLearningPathService(
		pRepo,
		aRepo,
		repository.NewLearningLogRepository(db),
		repository.NewUserRepository(db),
	), db
}

func insertKnowledgePoint(t *testing.T, db *gorm.DB, id, title string) {
	t.Helper()
	if err := db.Exec(
		`INSERT INTO knowledge_points (id, title, type, created_at, updated_at) VALUES (?,?,?,?,?)`,
		id, title, "concept", time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatal(err)
	}
}

func insertMaterial(t *testing.T, path *LearningPathService, title string, level, chapter int, kpIDs []string) *MaterialView {
	t.Helper()
	ids := append([]string(nil), kpIDs...)
	m, err := path.CreateMaterial(1, CreateMaterialRequest{
		Level:             level,
		TotalChapters:     3,
		ChapterNumber:     chapter,
		Title:             title,
		Content:           title + " body",
		Points:            0,
		KnowledgePointIDs: &ids,
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func publishLinkedPaper(t *testing.T, svc *AssessmentService, db *gorm.DB) (*model.Assessment, string, []model.AssessmentQuestion) {
	t.Helper()
	insertKnowledgePoint(t, db, "kp-pointer", "指针")
	insertKnowledgePoint(t, db, "kp-loop", "循环")
	insertKnowledgePoint(t, db, "kp-array", "数组")
	a, ver := publishPaper(t, svc, "diag-paper", []model.AssessmentQuestion{
		{QuestionType: "single_choice", Content: "pointer q", Options: choiceOptions(), Answer: "0", Points: 10},
		{QuestionType: "single_choice", Content: "loop q", Options: choiceOptions(), Answer: "0", Points: 10},
		{QuestionType: "essay", Content: "essay q", Points: 10},
	})
	qs, err := svc.Repo.ListAllQuestions(a.ID)
	if err != nil || len(qs) < 3 {
		t.Fatal(err)
	}
	if err := svc.Repo.ReplaceQuestionKnowledgePoints(qs[0].ID, []string{"kp-pointer"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Repo.ReplaceQuestionKnowledgePoints(qs[1].ID, []string{"kp-loop"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Repo.ReplaceQuestionKnowledgePoints(qs[2].ID, []string{"kp-array"}); err != nil {
		t.Fatal(err)
	}
	return a, ver, qs
}

func publishObjectiveLinkedPaper(t *testing.T, svc *AssessmentService, db *gorm.DB) (*model.Assessment, string, []model.AssessmentQuestion) {
	t.Helper()
	insertKnowledgePoint(t, db, "kp-pointer", "指针")
	insertKnowledgePoint(t, db, "kp-loop", "循环")
	a, ver := publishPaper(t, svc, "obj-diag-paper", []model.AssessmentQuestion{
		{QuestionType: "single_choice", Content: "pointer q", Options: choiceOptions(), Answer: "0", Points: 20},
		{QuestionType: "single_choice", Content: "loop q", Options: choiceOptions(), Answer: "0", Points: 20},
		{QuestionType: "single_choice", Content: "extra q", Options: choiceOptions(), Answer: "0", Points: 20},
	})
	qs, err := svc.Repo.ListAllQuestions(a.ID)
	if err != nil || len(qs) < 3 {
		t.Fatal(err)
	}
	if err := svc.Repo.ReplaceQuestionKnowledgePoints(qs[0].ID, []string{"kp-pointer"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Repo.ReplaceQuestionKnowledgePoints(qs[1].ID, []string{"kp-loop"}); err != nil {
		t.Fatal(err)
	}
	return a, ver, qs
}

func confirmWrongOn(t *testing.T, svc *AssessmentService, userID uint, paper *model.Assessment, ver string, qs []model.AssessmentQuestion, wrongIndex int, clientID string) *model.AssessmentSubmission {
	t.Helper()
	answers := []model.QuestionAnswer{
		{QuestionID: qs[0].ID, Answer: "0"},
		{QuestionID: qs[1].ID, Answer: "0"},
		{QuestionID: qs[2].ID, Answer: "text"},
	}
	if wrongIndex >= 0 && wrongIndex < 2 {
		answers[wrongIndex].Answer = "1"
	}
	sub, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: clientID,
		Answers:         answers,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.GradeSubmission(sub.ID, GradeSubmissionRequest{Score: intPtr(10), RecommendedLevel: 2, Feedback: "ok"}); err != nil {
		t.Fatal(err)
	}
	loaded, err := svc.Repo.FindSubmissionByID(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func TestRecommendDifferentWrongKnowledgePoints(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	pointerMat := insertMaterial(t, path, "pointer-mat", 2, 1, []string{"kp-pointer"})
	loopMat := insertMaterial(t, path, "loop-mat", 2, 2, []string{"kp-loop"})
	_ = insertMaterial(t, path, "array-mat", 2, 3, []string{"kp-array"})

	u1 := createTestUser(t, db, true)
	u2 := createTestUser(t, db, true)
	confirmWrongOn(t, svc, u1, paper, ver, qs, 0, "u1-wrong-pointer")
	confirmWrongOn(t, svc, u2, paper, ver, qs, 1, "u2-wrong-loop")

	r1, err := path.GetStudentPath(u1, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := path.GetStudentPath(u2, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if r1.Recommendations.Status != RecStatusOK || r2.Recommendations.Status != RecStatusOK {
		t.Fatalf("status %s / %s", r1.Recommendations.Status, r2.Recommendations.Status)
	}
	if len(r1.Recommendations.Items) != 1 || r1.Recommendations.Items[0].ID != pointerMat.ID {
		t.Fatalf("user1 recs %+v", r1.Recommendations.Items)
	}
	if r1.Recommendations.Items[0].Reasons[0].Cause != RecCauseWrong || r1.Recommendations.Items[0].Reasons[0].KnowledgePointID != "kp-pointer" {
		t.Fatalf("user1 reasons %+v", r1.Recommendations.Items[0].Reasons)
	}
	if len(r2.Recommendations.Items) != 1 || r2.Recommendations.Items[0].ID != loopMat.ID {
		t.Fatalf("user2 recs %+v", r2.Recommendations.Items)
	}
	if len(r1.Items) < 3 || len(r2.Items) < 3 {
		t.Fatalf("ordinary list must remain, got %d and %d", len(r1.Items), len(r2.Items))
	}
}

func TestRecommendUsesPriorConfirmedOnPendingRetest(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	pointerMat := insertMaterial(t, path, "pointer-mat", 2, 1, []string{"kp-pointer"})
	userID := createTestUser(t, db, true)
	confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "first-confirmed")

	if err := svc.Repo.UpdateUserAssessmentStatus(userID, true); err != nil {
		t.Fatal(err)
	}
	_, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "pending-retest",
		Answers:         []model.QuestionAnswer{{QuestionID: qs[1].ID, Answer: "1"}, {QuestionID: qs[0].ID, Answer: "0"}, {QuestionID: qs[2].ID, Answer: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendations.Status != RecStatusOK || len(result.Recommendations.Items) != 1 || result.Recommendations.Items[0].ID != pointerMat.ID {
		t.Fatalf("pending retest must keep prior confirmed recs: %+v", result.Recommendations)
	}
}

func TestStudentAssessmentStatusMasteryKeepsDiagnosisRule(t *testing.T) {
	svc, _, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	userID := createTestUser(t, db, true)
	confirmed := confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "mastery-confirmed")

	status, err := svc.GetStudentAssessmentStatus(userID)
	if err != nil {
		t.Fatal(err)
	}
	if status.LastConfirmedDiagnosis == nil || status.LastConfirmedDiagnosis.SubmissionID != confirmed.ID {
		t.Fatalf("confirmed diagnosis selection changed %+v", status.LastConfirmedDiagnosis)
	}
	if status.LastConfirmedDiagnosis.RecommendedLevel != 2 {
		t.Fatalf("recommended level %+v", status.LastConfirmedDiagnosis)
	}

	foundPointer, foundLoop := false, false
	for _, row := range status.KnowledgeMastery {
		raw, _ := json.Marshal(row)
		if strings.Contains(strings.ToLower(string(raw)), "\"answer\"") {
			t.Fatalf("mastery must not include answers: %s", raw)
		}
		switch row.KnowledgePointID {
		case "kp-pointer":
			foundPointer = true
			if row.Percent != 0 || row.Status != KnowledgeMasteryStatusNeedsReview {
				t.Fatalf("pointer %+v", row)
			}
		case "kp-loop":
			foundLoop = true
			if row.Percent != 100 || row.Status != KnowledgeMasteryStatusMastered {
				t.Fatalf("loop %+v", row)
			}
		case "kp-array":
			t.Fatalf("essay item must not enter mastery %+v", row)
		}
	}
	if !foundPointer || !foundLoop {
		t.Fatalf("%+v", status.KnowledgeMastery)
	}

	if err := svc.Repo.UpdateUserAssessmentStatus(userID, true); err != nil {
		t.Fatal(err)
	}
	pending, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "mastery-pending",
		Answers: []model.QuestionAnswer{
			{QuestionID: qs[0].ID, Answer: "0"},
			{QuestionID: qs[1].ID, Answer: "1"},
			{QuestionID: qs[2].ID, Answer: "x"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	status2, err := svc.GetStudentAssessmentStatus(userID)
	if err != nil {
		t.Fatal(err)
	}
	if status2.LastConfirmedDiagnosis == nil || status2.LastConfirmedDiagnosis.SubmissionID != confirmed.ID {
		t.Fatalf("pending retest must keep confirmed diagnosis %+v", status2.LastConfirmedDiagnosis)
	}
	if status2.Submission == nil || status2.Submission.ID != pending.ID {
		t.Fatalf("latest attempt should be pending %+v", status2.Submission)
	}
}

func TestStaleDiagnosisNotUsedForRecommend(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	_ = insertMaterial(t, path, "pointer-mat", 2, 1, []string{"kp-pointer"})
	userID := createTestUser(t, db, true)
	confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "old-version")

	updated, err := svc.UpdateQuestion(qs[0].ID, AssessmentQuestionRequest{
		QuestionType: qs[0].QuestionType,
		Content:      "pointer q rewritten",
		Options:      qs[0].Options,
		Answer:       "1",
		Points:       qs[0].Points + 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = updated
	result, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendations.Status != RecStatusNoConfirmedDiagnosis {
		t.Fatalf("stale diagnosis must not recommend, got %s", result.Recommendations.Status)
	}
	if len(result.Items) == 0 {
		t.Fatal("ordinary material list must not disappear")
	}
}

func TestPendingManualAndAmbiguousAreNotWrongEvidence(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	_ = insertMaterial(t, path, "array-mat", 2, 1, []string{"kp-array"})
	_ = insertMaterial(t, path, "pointer-mat", 2, 2, []string{"kp-pointer"})
	userID := createTestUser(t, db, true)
	sub, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "manual-only",
		Answers: []model.QuestionAnswer{
			{QuestionID: qs[0].ID, Answer: "0"},
			{QuestionID: qs[1].ID, Answer: "0"},
			{QuestionID: qs[2].ID, Answer: "essay"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	itemJSON, _ := json.Marshal([]model.AssessmentItemResult{
		{QuestionID: qs[0].ID, QuestionType: "single_choice", AutoResult: model.AutoResultUnscoredAmbiguous, KnowledgePointIDs: []string{"kp-pointer"}},
		{QuestionID: qs[2].ID, QuestionType: "essay", AutoResult: model.AutoResultPendingManual, KnowledgePointIDs: []string{"kp-array"}},
	})
	sub.ItemResults = itemJSON
	if err := svc.GradeSubmission(sub.ID, GradeSubmissionRequest{Score: intPtr(10), RecommendedLevel: 2}); err != nil {
		t.Fatal(err)
	}
	loaded, err := svc.Repo.FindSubmissionByID(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	loaded.ItemResults = itemJSON
	if err := svc.Repo.UpdateSubmission(loaded); err != nil {
		t.Fatal(err)
	}

	result, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendations.Status != RecStatusNoWeakEvidence {
		t.Fatalf("got %s %+v", result.Recommendations.Status, result.Recommendations)
	}
}

func TestUnansweredReasonIsExplicit(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	mat := insertMaterial(t, path, "pointer-mat", 2, 1, []string{"kp-pointer"})
	userID := createTestUser(t, db, true)
	sub, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "blank-q",
		Answers:         []model.QuestionAnswer{{QuestionID: qs[1].ID, Answer: "0"}, {QuestionID: qs[2].ID, Answer: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.GradeSubmission(sub.ID, GradeSubmissionRequest{Score: intPtr(10), RecommendedLevel: 2}); err != nil {
		t.Fatal(err)
	}
	result, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendations.Status != RecStatusOK || len(result.Recommendations.Items) != 1 || result.Recommendations.Items[0].ID != mat.ID {
		t.Fatalf("%+v", result.Recommendations)
	}
	if result.Recommendations.Items[0].Reasons[0].Cause != RecCauseUnanswered {
		t.Fatalf("want unanswered, got %+v", result.Recommendations.Items[0].Reasons)
	}
}

func TestMultiKnowledgeMaterialDedupAndLockedExcluded(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	shared := insertMaterial(t, path, "shared-mat", 2, 1, []string{"kp-pointer", "kp-loop"})
	locked := insertMaterial(t, path, "locked-mat", 3, 1, []string{"kp-pointer"})
	userID := createTestUser(t, db, true)
	sub, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "two-wrong",
		Answers: []model.QuestionAnswer{
			{QuestionID: qs[0].ID, Answer: "1"},
			{QuestionID: qs[1].ID, Answer: "1"},
			{QuestionID: qs[2].ID, Answer: "x"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.GradeSubmission(sub.ID, GradeSubmissionRequest{Score: intPtr(0), RecommendedLevel: 2}); err != nil {
		t.Fatal(err)
	}
	result, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Recommendations.Items) != 1 || result.Recommendations.Items[0].ID != shared.ID {
		t.Fatalf("expected one unlocked shared material, got %+v", result.Recommendations.Items)
	}
	if len(result.Recommendations.Items[0].Reasons) != 2 {
		t.Fatalf("expected two reasons after dedupe, got %+v", result.Recommendations.Items[0].Reasons)
	}
	for _, item := range result.Recommendations.Items {
		if item.ID == locked.ID {
			t.Fatal("locked material must not be recommended")
		}
	}
	if err := path.CompleteMaterial(userID, locked.ID); !errors.Is(err, util.ErrMaterialNotAccessible) {
		t.Fatalf("complete locked want not accessible, got %v", err)
	}
}

func TestRecommendDegradesWithoutMappingResultsOrMaterials(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	userID := createTestUser(t, db, true)

	noDiag, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if noDiag.Recommendations.Status != RecStatusNoConfirmedDiagnosis {
		t.Fatalf("got %s", noDiag.Recommendations.Status)
	}

	sub := confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "no-material")
	result, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendations.Status != RecStatusNoMatchingMaterials {
		t.Fatalf("got %s", result.Recommendations.Status)
	}
	if len(result.Items) != 0 {
		t.Fatalf("no materials seeded, list should be empty, got %d", len(result.Items))
	}

	loaded, err := svc.Repo.FindSubmissionByID(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	loaded.ItemResults = nil
	if err := svc.Repo.UpdateSubmission(loaded); err != nil {
		t.Fatal(err)
	}
	missing, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if missing.Recommendations.Status != RecStatusNoItemResults {
		t.Fatalf("got %s", missing.Recommendations.Status)
	}

	legacyJSON, _ := json.Marshal([]map[string]interface{}{
		{"questionId": qs[0].ID, "questionType": "single_choice", "autoResult": "wrong", "pointsAwarded": 0},
	})
	loaded.ItemResults = legacyJSON
	if err := svc.Repo.UpdateSubmission(loaded); err != nil {
		t.Fatal(err)
	}
	legacy, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Recommendations.Status != RecStatusLegacyNoSnapshot {
		t.Fatalf("legacy missing snapshot must not use live links, got %s", legacy.Recommendations.Status)
	}

	emptyJSON, _ := json.Marshal([]model.AssessmentItemResult{
		{QuestionID: qs[0].ID, QuestionType: "single_choice", AutoResult: model.AutoResultWrong, KnowledgePointIDs: []string{}},
	})
	loaded.ItemResults = emptyJSON
	if err := svc.Repo.UpdateSubmission(loaded); err != nil {
		t.Fatal(err)
	}
	emptySnap, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if emptySnap.Recommendations.Status != RecStatusNoMappedKnowledgePoints {
		t.Fatalf("explicit empty snapshot must differ from missing field, got %s", emptySnap.Recommendations.Status)
	}
}

func TestChangingQuestionLinksDoesNotReinterpretOldDiagnosis(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	pointerMat := insertMaterial(t, path, "pointer-mat", 2, 1, []string{"kp-pointer"})
	loopMat := insertMaterial(t, path, "loop-mat", 2, 2, []string{"kp-loop"})
	userID := createTestUser(t, db, true)
	confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "snapshot-a")

	before, err := ComputePaperVersion(paper.ID, qs)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Repo.ReplaceQuestionKnowledgePoints(qs[0].ID, []string{"kp-loop"}); err != nil {
		t.Fatal(err)
	}
	qsAfter, _ := svc.Repo.ListAllQuestions(paper.ID)
	after, err := ComputePaperVersion(paper.ID, qsAfter)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("knowledge-point links must not change paper version")
	}

	result, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendations.Status != RecStatusOK || len(result.Recommendations.Items) != 1 || result.Recommendations.Items[0].ID != pointerMat.ID {
		t.Fatalf("old snapshot must still recommend pointer material, got %+v", result.Recommendations)
	}
	for _, item := range result.Recommendations.Items {
		if item.ID == loopMat.ID {
			t.Fatal("must not silently apply new question links to old diagnosis")
		}
	}
}

func TestTeacherKnowledgeLinksSaveEchoAndClear(t *testing.T) {
	svc, path, db := newPathBundle(t)
	insertKnowledgePoint(t, db, "kp-pointer", "指针")
	insertKnowledgePoint(t, db, "kp-loop", "循环")
	a, err := svc.CreateAssessment(AssessmentRequest{Title: "link-paper"})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{"kp-pointer", "kp-loop"}
	created, err := svc.CreateQuestion(AssessmentQuestionRequest{
		AssessmentID:      a.ID,
		QuestionType:      "essay",
		Content:           "q",
		KnowledgePointIDs: &ids,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.KnowledgePointIDs) != 2 {
		t.Fatalf("create echo %+v", created.KnowledgePointIDs)
	}
	got, err := svc.GetQuestion(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.KnowledgePointIDs) != 2 {
		t.Fatalf("get echo %+v", got.KnowledgePointIDs)
	}

	keep, err := svc.UpdateQuestion(created.ID, AssessmentQuestionRequest{
		QuestionType: "essay",
		Content:      "q-updated",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(keep.KnowledgePointIDs) != 2 {
		t.Fatalf("omitted field must keep links, got %+v", keep.KnowledgePointIDs)
	}

	empty := []string{}
	cleared, err := svc.UpdateQuestion(created.ID, AssessmentQuestionRequest{
		QuestionType:      "essay",
		Content:           "q-cleared",
		KnowledgePointIDs: &empty,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.KnowledgePointIDs == nil || len(cleared.KnowledgePointIDs) != 0 {
		t.Fatalf("empty list must clear, got %+v", cleared.KnowledgePointIDs)
	}

	bad := []string{"missing-kp"}
	if _, err := svc.UpdateQuestion(created.ID, AssessmentQuestionRequest{
		QuestionType:      "essay",
		Content:           "q-bad",
		KnowledgePointIDs: &bad,
	}); !errors.Is(err, util.ErrInvalidKnowledgePoint) {
		t.Fatalf("want invalid kp, got %v", err)
	}

	matIDs := []string{"kp-pointer"}
	mat, err := path.CreateMaterial(1, CreateMaterialRequest{
		Level: 1, Title: "m", Content: "c", KnowledgePointIDs: &matIDs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(mat.KnowledgePointIDs) != 1 || mat.KnowledgePointIDs[0] != "kp-pointer" {
		t.Fatalf("material create echo %+v", mat.KnowledgePointIDs)
	}
	kept, err := path.UpdateMaterial(mat.ID, CreateMaterialRequest{Level: 1, Title: "m2", Content: "c2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(kept.KnowledgePointIDs) != 1 {
		t.Fatalf("material omit must keep, got %+v", kept.KnowledgePointIDs)
	}
	clearedMat, err := path.UpdateMaterial(mat.ID, CreateMaterialRequest{
		Level: 1, Title: "m3", Content: "c3", KnowledgePointIDs: &empty,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(clearedMat.KnowledgePointIDs) != 0 {
		t.Fatalf("material empty must clear, got %+v", clearedMat.KnowledgePointIDs)
	}

	listed, _, err := svc.ListQuestions(a.ID, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if listed[0].KnowledgePointIDs == nil {
		t.Fatal("list must return empty array not omit")
	}
}

func TestSubmitSnapshotsKnowledgePoints(t *testing.T) {
	svc, _, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	userID := createTestUser(t, db, true)
	sub := confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "snap")
	var items []model.AssessmentItemResult
	if err := json.Unmarshal(sub.ItemResults, &items); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.QuestionID == qs[0].ID {
			found = true
			if len(item.KnowledgePointIDs) != 1 || item.KnowledgePointIDs[0] != "kp-pointer" {
				t.Fatalf("snapshot %+v", item.KnowledgePointIDs)
			}
		}
		if item.KnowledgePointIDs == nil {
			t.Fatalf("snapshot must always write knowledgePointIds, question %d", item.QuestionID)
		}
	}
	if !found {
		t.Fatal("missing item")
	}
}

func TestSoftDeletedKnowledgePointRejected(t *testing.T) {
	svc, _, db := newPathBundle(t)
	insertKnowledgePoint(t, db, "kp-dead", "已删")
	if err := db.Exec(`UPDATE knowledge_points SET deleted_at = ? WHERE id = ?`, time.Now(), "kp-dead").Error; err != nil {
		t.Fatal(err)
	}
	a, err := svc.CreateAssessment(AssessmentRequest{Title: "dead-kp"})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{"kp-dead"}
	_, err = svc.CreateQuestion(AssessmentQuestionRequest{
		AssessmentID: a.ID, QuestionType: "essay", Content: "q", KnowledgePointIDs: &ids,
	})
	if !errors.Is(err, util.ErrInvalidKnowledgePoint) {
		t.Fatalf("got %v", err)
	}
}

func TestIncompleteMaterialsRankFirst(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	done := insertMaterial(t, path, "done-mat", 2, 1, []string{"kp-pointer"})
	todo := insertMaterial(t, path, "todo-mat", 2, 9, []string{"kp-pointer"})
	userID := createTestUser(t, db, true)
	confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "rank")
	if err := db.Exec(
		`INSERT INTO learning_path_completions (user_id, material_id, completed_at) VALUES (?,?,?)`,
		userID, done.ID, time.Now(),
	).Error; err != nil {
		t.Fatal(err)
	}
	result, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Recommendations.Items) != 2 {
		t.Fatalf("%+v", result.Recommendations.Items)
	}
	if result.Recommendations.Items[0].ID != todo.ID || result.Recommendations.Items[1].ID != done.ID {
		t.Fatalf("incomplete first, got %s then %s", result.Recommendations.Items[0].ID, result.Recommendations.Items[1].ID)
	}
}

func TestItemResultJSONEmptyVsMissing(t *testing.T) {
	var missing model.AssessmentItemResult
	if err := json.Unmarshal([]byte(`{"questionId":1,"autoResult":"wrong"}`), &missing); err != nil {
		t.Fatal(err)
	}
	if missing.KnowledgePointIDs != nil {
		t.Fatalf("missing field must stay nil, got %#v", missing.KnowledgePointIDs)
	}

	var empty model.AssessmentItemResult
	if err := json.Unmarshal([]byte(`{"questionId":1,"autoResult":"wrong","knowledgePointIds":[]}`), &empty); err != nil {
		t.Fatal(err)
	}
	if empty.KnowledgePointIDs == nil || len(empty.KnowledgePointIDs) != 0 {
		t.Fatalf("empty array must be non-nil empty, got %#v", empty.KnowledgePointIDs)
	}

	raw, err := json.Marshal(model.AssessmentItemResult{QuestionID: 1, KnowledgePointIDs: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"questionId":1,"questionType":"","autoResult":"","pointsAwarded":0,"knowledgePointIds":[]}` {
		t.Fatalf("new snapshot must write empty array, got %s", raw)
	}
}

func TestAcceptanceArrayVsLoopSameLevel(t *testing.T) {
	svc, path, db := newPathBundle(t)
	insertKnowledgePoint(t, db, "kp-array", "数组")
	insertKnowledgePoint(t, db, "kp-loop", "循环")
	paper, ver := publishPaper(t, svc, "array-loop-paper", []model.AssessmentQuestion{
		{QuestionType: "single_choice", Content: "array q", Options: choiceOptions(), Answer: "0", Points: 10},
		{QuestionType: "single_choice", Content: "loop q", Options: choiceOptions(), Answer: "0", Points: 10},
	})
	qs, err := svc.Repo.ListAllQuestions(paper.ID)
	if err != nil || len(qs) != 2 {
		t.Fatal(err)
	}
	if err := svc.Repo.ReplaceQuestionKnowledgePoints(qs[0].ID, []string{"kp-array"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Repo.ReplaceQuestionKnowledgePoints(qs[1].ID, []string{"kp-loop"}); err != nil {
		t.Fatal(err)
	}
	arrayMat := insertMaterial(t, path, "array-mat", 2, 1, []string{"kp-array"})
	loopMat := insertMaterial(t, path, "loop-mat", 2, 2, []string{"kp-loop"})

	studentA := createTestUser(t, db, true)
	studentB := createTestUser(t, db, true)
	subA, err := svc.SubmitAssessment(studentA, AssessmentSubmissionRequest{
		AssessmentID: paper.ID, PaperVersion: ver, ClientRequestID: "student-a",
		Answers: []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "1"}, {QuestionID: qs[1].ID, Answer: "0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	subB, err := svc.SubmitAssessment(studentB, AssessmentSubmissionRequest{
		AssessmentID: paper.ID, PaperVersion: ver, ClientRequestID: "student-b",
		Answers: []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "0"}, {QuestionID: qs[1].ID, Answer: "1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.GradeSubmission(subA.ID, GradeSubmissionRequest{Score: intPtr(10), RecommendedLevel: 2}); err != nil {
		t.Fatal(err)
	}
	if err := svc.GradeSubmission(subB.ID, GradeSubmissionRequest{Score: intPtr(10), RecommendedLevel: 2}); err != nil {
		t.Fatal(err)
	}

	recA, err := path.GetStudentPath(studentA, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	recB, err := path.GetStudentPath(studentB, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if recA.Recommendations.Status != RecStatusOK || recB.Recommendations.Status != RecStatusOK {
		t.Fatalf("status A=%s B=%s", recA.Recommendations.Status, recB.Recommendations.Status)
	}
	if len(recA.Recommendations.Items) != 1 || recA.Recommendations.Items[0].ID != arrayMat.ID {
		t.Fatalf("A expected array-mat, got %+v", recA.Recommendations.Items)
	}
	if recA.Recommendations.Items[0].Reasons[0].KnowledgePointID != "kp-array" || recA.Recommendations.Items[0].Reasons[0].Cause != RecCauseWrong {
		t.Fatalf("A reasons %+v", recA.Recommendations.Items[0].Reasons)
	}
	if len(recB.Recommendations.Items) != 1 || recB.Recommendations.Items[0].ID != loopMat.ID {
		t.Fatalf("B expected loop-mat, got %+v", recB.Recommendations.Items)
	}
	if recB.Recommendations.Items[0].Reasons[0].KnowledgePointID != "kp-loop" {
		t.Fatalf("B reasons %+v", recB.Recommendations.Items[0].Reasons)
	}
	if recA.Recommendations.Items[0].Reasons[0].QuestionIndex != 1 {
		t.Fatalf("A ordinal must be paper position 1, got %+v", recA.Recommendations.Items[0].Reasons[0])
	}
	if recB.Recommendations.Items[0].Reasons[0].QuestionIndex != 2 {
		t.Fatalf("B ordinal must be paper position 2, got %+v", recB.Recommendations.Items[0].Reasons[0])
	}
	if len(recA.Items) < 2 || len(recB.Items) < 2 {
		t.Fatalf("ordinary list must remain A=%d B=%d", len(recA.Items), len(recB.Items))
	}
}

func TestHistoricalRecommendDoesNotFabricateOrdinalFromQuestionID(t *testing.T) {
	svc, path, db := newPathBundle(t)
	insertKnowledgePoint(t, db, "kp-loop", "循环")
	paper, ver := publishPaper(t, svc, "hist-ordinal", []model.AssessmentQuestion{
		{QuestionType: "single_choice", Content: "循环题干摘要", Options: choiceOptions(), Answer: "0", Points: 10},
	})
	qs, err := svc.Repo.ListAllQuestions(paper.ID)
	if err != nil || len(qs) != 1 {
		t.Fatal(err)
	}
	if err := svc.Repo.ReplaceQuestionKnowledgePoints(qs[0].ID, []string{"kp-loop"}); err != nil {
		t.Fatal(err)
	}
	mat := insertMaterial(t, path, "loop-hist", 2, 1, []string{"kp-loop"})
	userID := createTestUser(t, db, true)
	sub, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID: paper.ID, PaperVersion: ver, ClientRequestID: "hist-ord",
		Answers: []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.GradeSubmission(sub.ID, GradeSubmissionRequest{Score: intPtr(0), RecommendedLevel: 2}); err != nil {
		t.Fatal(err)
	}
	loaded, err := svc.Repo.FindSubmissionByID(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	legacyJSON, err := json.Marshal([]model.AssessmentItemResult{{
		QuestionID:        qs[0].ID,
		QuestionType:      "single_choice",
		AutoResult:        model.AutoResultWrong,
		KnowledgePointIDs: []string{"kp-loop"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	loaded.ItemResults = legacyJSON
	if err := svc.Repo.UpdateSubmission(loaded); err != nil {
		t.Fatal(err)
	}

	result, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Recommendations.Items) != 1 || result.Recommendations.Items[0].ID != mat.ID {
		t.Fatalf("got %+v", result.Recommendations.Items)
	}
	reason := result.Recommendations.Items[0].Reasons[0]
	if reason.QuestionIndex != 0 {
		t.Fatalf("must not invent paper ordinal for legacy snapshot, got %+v", reason)
	}
	if reason.QuestionID != qs[0].ID {
		t.Fatalf("question id still identity %+v", reason)
	}
	if reason.QuestionSummary != "循环题干摘要" || reason.KnowledgePointTitle != "循环" {
		t.Fatalf("want kp+summary, got %+v", reason)
	}
}

func TestSoftDeletedKnowledgePointAndMaterialExcluded(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	liveMat := insertMaterial(t, path, "live-pointer", 2, 1, []string{"kp-pointer"})
	deadMat := insertMaterial(t, path, "dead-pointer", 2, 2, []string{"kp-pointer"})
	userID := createTestUser(t, db, true)
	confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "del-filter")

	if err := db.Exec(`UPDATE learning_path_materials SET deleted_at = ? WHERE id = ?`, time.Now(), deadMat.ID).Error; err != nil {
		t.Fatal(err)
	}
	result, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Recommendations.Items) != 1 || result.Recommendations.Items[0].ID != liveMat.ID {
		t.Fatalf("deleted material leaked: %+v", result.Recommendations.Items)
	}

	if err := db.Exec(`UPDATE knowledge_points SET deleted_at = ? WHERE id = ?`, time.Now(), "kp-pointer").Error; err != nil {
		t.Fatal(err)
	}
	afterKP, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if afterKP.Recommendations.Status != RecStatusNoMatchingMaterials {
		t.Fatalf("deleted knowledge point still recommended: %+v", afterKP.Recommendations)
	}
	if err := path.CompleteMaterial(userID, deadMat.ID); !errors.Is(err, util.ErrResourceNotFound) && !errors.Is(err, util.ErrMaterialNotAccessible) {
		t.Fatalf("deleted material complete got %v", err)
	}
}

func TestNewSubmitUsesPersistTimeLinks(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	pointerMat := insertMaterial(t, path, "pointer-mat", 2, 1, []string{"kp-pointer"})
	loopMat := insertMaterial(t, path, "loop-mat", 2, 2, []string{"kp-loop"})
	oldUser := createTestUser(t, db, true)
	confirmWrongOn(t, svc, oldUser, paper, ver, qs, 0, "old-snap")

	if err := svc.Repo.ReplaceQuestionKnowledgePoints(qs[0].ID, []string{"kp-loop"}); err != nil {
		t.Fatal(err)
	}
	newUser := createTestUser(t, db, true)
	confirmWrongOn(t, svc, newUser, paper, ver, qs, 0, "new-snap")

	oldRec, err := path.GetStudentPath(oldUser, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	newRec, err := path.GetStudentPath(newUser, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(oldRec.Recommendations.Items) != 1 || oldRec.Recommendations.Items[0].ID != pointerMat.ID {
		t.Fatalf("old diagnosis %+v", oldRec.Recommendations)
	}
	if len(newRec.Recommendations.Items) != 1 || newRec.Recommendations.Items[0].ID != loopMat.ID {
		t.Fatalf("new submit must snapshot persist-time links %+v", newRec.Recommendations)
	}
}

func TestStudentMaterialAccessRejectsLockedAndInvalidLevel(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	locked := insertMaterial(t, path, "locked-mat", 3, 1, []string{"kp-pointer"})
	userID := createTestUser(t, db, true)
	confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "access")

	if err := path.RecordLearningTime(userID, locked.ID, 5); !errors.Is(err, util.ErrMaterialNotAccessible) {
		t.Fatalf("learning-time locked got %v", err)
	}
	details, err := path.GetMaterialsByLevel(userID, 3)
	if err != nil {
		t.Fatal(err)
	}
	if details != nil {
		t.Fatalf("level 3 detail must be locked, got %d", len(details))
	}
	invalid, err := path.GetMaterialsByLevel(userID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if invalid != nil {
		t.Fatal("level 0 must not dump all materials")
	}
}

func TestRecommendDoesNotReadOtherStudentDiagnosis(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	_ = insertMaterial(t, path, "pointer-mat", 2, 1, []string{"kp-pointer"})
	a := createTestUser(t, db, true)
	b := createTestUser(t, db, true)
	confirmWrongOn(t, svc, a, paper, ver, qs, 0, "only-a")

	recB, err := path.GetStudentPath(b, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if recB.Recommendations.Status != RecStatusNoConfirmedDiagnosis {
		t.Fatalf("student B must not inherit A diagnosis, got %s %+v", recB.Recommendations.Status, recB.Recommendations)
	}
}

func TestAutoCompletedObjectiveDiagnosisDrivesPath(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishObjectiveLinkedPaper(t, svc, db)
	pointerMat := insertMaterial(t, path, "pointer-catchup", 2, 1, []string{"kp-pointer"})
	_ = insertMaterial(t, path, "loop-ok", 2, 2, []string{"kp-loop"})
	userID := createTestUser(t, db, true)

	sub, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "auto-path",
		Answers: []model.QuestionAnswer{
			{QuestionID: qs[0].ID, Answer: "1"},
			{QuestionID: qs[1].ID, Answer: "0"},
			{QuestionID: qs[2].ID, Answer: "0"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if sub.Status != model.SubmissionStatusCompleted || sub.RecommendedLevel != 2 {
		t.Fatalf("40/60 auto path diagnosis %+v", sub)
	}
	if sub.ScoringStatus != model.ScoringStatusSystemCompleted {
		t.Fatalf("must not wait for teacher %+v", sub)
	}

	confirmed, err := svc.GetConfirmedDiagnosisForUser(userID)
	if err != nil {
		t.Fatal(err)
	}
	if confirmed == nil || confirmed.SubmissionID != sub.ID || confirmed.RecommendedLevel != 2 {
		t.Fatalf("learning path must use auto diagnosis %+v", confirmed)
	}

	result, err := path.GetStudentPath(userID, StudentPathListParams{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Recommendations.Items) == 0 {
		t.Fatalf("auto diagnosis must generate catch-up path %+v", result.Recommendations)
	}
	if result.Recommendations.Items[0].ID != pointerMat.ID {
		t.Fatalf("want pointer catch-up, got %+v", result.Recommendations.Items)
	}
}
