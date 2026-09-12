package service

// 本文件只连独立 SQLite 内存库，不读 configs、不调用 database.InitDB，
// 不跑业务 AutoMigrate/seed，不启动 HTTP 服务。
// SQLite 可覆盖服务规则与“插入被拒绝则不落库”；不能等价证明 MySQL 的
// SELECT FOR UPDATE、事务隔离、唯一约束冲突语义。

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/util"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openAssessmentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=busy_timeout(5000)", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	stmts := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			name TEXT,
			email TEXT,
			password TEXT,
			role TEXT,
			xp INTEGER DEFAULT 0,
			points INTEGER DEFAULT 0,
			language TEXT DEFAULT 'en',
			avatar TEXT,
			disabled INTEGER DEFAULT 0,
			can_take_assessment INTEGER DEFAULT 1,
			last_login DATETIME,
			last_seen DATETIME
		)`,
		`CREATE TABLE assessments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			title TEXT,
			description TEXT,
			time_limit INTEGER DEFAULT 0,
			is_published INTEGER DEFAULT 0,
			published_at DATETIME
		)`,
		`CREATE TABLE assessment_questions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			assessment_id INTEGER,
			question_type TEXT,
			title TEXT,
			content TEXT,
			options TEXT,
			answer TEXT,
			points INTEGER DEFAULT 0,
			` + "`order`" + ` INTEGER DEFAULT 0,
			explanation TEXT
		)`,
		`CREATE TABLE assessment_submissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			user_id INTEGER,
			assessment_id INTEGER,
			attempt_no INTEGER DEFAULT 1,
			client_request_id TEXT,
			paper_version TEXT,
			answers TEXT,
			item_results TEXT,
			total_score INTEGER DEFAULT 0,
			auto_score INTEGER DEFAULT 0,
			objective_max INTEGER DEFAULT 0,
			pending_manual_count INTEGER DEFAULT 0,
			scoring_status TEXT DEFAULT 'awaiting_teacher',
			status TEXT DEFAULT 'pending',
			feedback TEXT,
			recommended_level INTEGER DEFAULT 0
		)`,
		`CREATE UNIQUE INDEX idx_assessment_attempt ON assessment_submissions (user_id, assessment_id, attempt_no)`,
		`CREATE UNIQUE INDEX idx_assessment_submit_idempotent ON assessment_submissions (user_id, client_request_id)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func createTestUser(t *testing.T, db *gorm.DB, canTake bool) uint {
	t.Helper()
	email := uuid.NewString() + "@test.local"
	if err := db.Exec(
		`INSERT INTO users (name, email, password, role, can_take_assessment, created_at, updated_at) VALUES (?,?,?,?,?,?,?)`,
		"stu", email, "x", "student", canTake, time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatal(err)
	}
	var id uint
	if err := db.Raw(`SELECT id FROM users WHERE email = ?`, email).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	return id
}

func publishPaper(t *testing.T, svc *AssessmentService, title string, questions []model.AssessmentQuestion) (*model.Assessment, string) {
	t.Helper()
	a, err := svc.CreateAssessment(AssessmentRequest{Title: title})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for i, q := range questions {
		q.AssessmentID = a.ID
		q.Order = i + 1
		if q.Points == 0 {
			q.Points = 10
		}
		if err := svc.Repo.CreateQuestion(&q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.SetAssessmentPublished(a.ID, true); err != nil {
		t.Fatal(err)
	}
	loaded, err := svc.Repo.FindAssessmentByID(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	loaded.PublishedAt = &now
	_ = svc.Repo.UpdateAssessment(loaded)
	qs, err := svc.Repo.ListAllQuestions(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	ver, err := ComputePaperVersion(a.ID, qs)
	if err != nil {
		t.Fatal(err)
	}
	return loaded, ver
}

func newAssessmentService(t *testing.T) (*AssessmentService, *gorm.DB) {
	db := openAssessmentTestDB(t)
	return NewAssessmentService(repository.NewAssessmentRepository(db)), db
}

func TestStudentGetDoesNotCreateOrFallback(t *testing.T) {
	svc, db := newAssessmentService(t)
	_ = db
	if _, err := svc.CreateAssessment(AssessmentRequest{Title: "draft-only"}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.ListStudentQuestions()
	if !errors.Is(err, util.ErrNoPublishedAssessment) {
		t.Fatalf("want 暂无测验, got %v", err)
	}

	older, _ := publishPaper(t, svc, "older", []model.AssessmentQuestion{{
		QuestionType: "single_choice", Content: "q1", Options: choiceOptions(), Answer: "0",
	}})
	time.Sleep(20 * time.Millisecond)
	newer, ver := publishPaper(t, svc, "newer", []model.AssessmentQuestion{{
		QuestionType: "single_choice", Content: "q2", Options: choiceOptions(), Answer: "1",
	}})
	paper, err := svc.ListStudentQuestions()
	if err != nil {
		t.Fatal(err)
	}
	if paper.AssessmentID == 1 {
		t.Fatal("must not fall back to min/default id 1 when a later published paper exists")
	}
	if paper.AssessmentID != newer.ID {
		t.Fatalf("expected latest published %d (older=%d), got %d", newer.ID, older.ID, paper.AssessmentID)
	}
	if paper.PaperVersion != ver {
		t.Fatalf("version mismatch")
	}
}

func TestSubmitBindsClaimedPaperAndRejectsStaleVersion(t *testing.T) {
	svc, _ := newAssessmentService(t)
	userID := createTestUser(t, svc.Repo.DB, true)
	first, ver := publishPaper(t, svc, "paper-a", []model.AssessmentQuestion{{
		QuestionType: "single_choice", Content: "x", Options: choiceOptions(), Answer: "0",
	}})
	qs, _ := svc.Repo.ListAllQuestions(first.ID)
	second, _ := publishPaper(t, svc, "paper-b", []model.AssessmentQuestion{{
		QuestionType: "single_choice", Content: "y", Options: choiceOptions(), Answer: "1",
	}})

	sub, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    first.ID,
		PaperVersion:    ver,
		ClientRequestID: "req-keep-first",
		Answers:         []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if sub.AssessmentID != first.ID || sub.AssessmentID == second.ID {
		t.Fatalf("must bind claimed paper %d, not current %d", first.ID, second.ID)
	}

	qs[0].Answer = "1"
	if err := svc.Repo.UpdateQuestion(&qs[0]); err != nil {
		t.Fatal(err)
	}
	_, err = svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    first.ID,
		PaperVersion:    ver,
		ClientRequestID: "req-stale",
		Answers:         []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "0"}},
	})
	if !errors.Is(err, util.ErrAssessmentPaperStale) {
		t.Fatalf("edited paper must 409, got %v", err)
	}
	var count int64
	svc.Repo.DB.Model(&model.AssessmentSubmission{}).Where("client_request_id = ?", "req-stale").Count(&count)
	if count != 0 {
		t.Fatal("stale submit must not persist")
	}
}

func TestSubmitRequiresClaimedPaperMeta(t *testing.T) {
	svc, _ := newAssessmentService(t)
	userID := createTestUser(t, svc.Repo.DB, true)
	_, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		Answers: []model.QuestionAnswer{{QuestionID: 1, Answer: "0"}},
	})
	if !errors.Is(err, util.ErrAssessmentSubmitMissingMeta) {
		t.Fatalf("missing meta: %v", err)
	}
}

func TestInvalidSubmitDoesNotPersist(t *testing.T) {
	svc, _ := newAssessmentService(t)
	userID := createTestUser(t, svc.Repo.DB, true)
	paper, ver := publishPaper(t, svc, "p", []model.AssessmentQuestion{{
		QuestionType: "single_choice", Content: "x", Options: choiceOptions(), Answer: "0",
	}})
	qs, _ := svc.Repo.ListAllQuestions(paper.ID)

	_, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "bad-out",
		Answers:         []model.QuestionAnswer{{QuestionID: 9999, Answer: "0"}},
	})
	if !errors.Is(err, util.ErrAssessmentSubmitInvalid) {
		t.Fatalf("%v", err)
	}
	_, err = svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "bad-opt",
		Answers:         []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "9"}},
	})
	if !errors.Is(err, util.ErrAssessmentSubmitInvalid) {
		t.Fatalf("illegal option: %v", err)
	}
	var count int64
	svc.Repo.DB.Model(&model.AssessmentSubmission{}).Where("user_id = ?", userID).Count(&count)
	if count != 0 {
		t.Fatalf("invalid submits persisted: %d", count)
	}
}

func TestSubmitRejectsUnpublishedOrEmptyPaper(t *testing.T) {
	svc, _ := newAssessmentService(t)
	userID := createTestUser(t, svc.Repo.DB, true)
	paper, ver := publishPaper(t, svc, "to-unpublish", []model.AssessmentQuestion{{
		QuestionType: "single_choice", Content: "x", Options: choiceOptions(), Answer: "0",
	}})
	qs, _ := svc.Repo.ListAllQuestions(paper.ID)
	if _, err := svc.SetAssessmentPublished(paper.ID, false); err != nil {
		t.Fatal(err)
	}
	_, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "after-unpublish",
		Answers:         []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "0"}},
	})
	if !errors.Is(err, util.ErrAssessmentPaperUnavailable) {
		t.Fatalf("unpublished: %v", err)
	}
	var count int64
	svc.Repo.DB.Model(&model.AssessmentSubmission{}).Where("client_request_id = ?", "after-unpublish").Count(&count)
	if count != 0 {
		t.Fatal("unpublished submit must not persist")
	}

	emptyPaper, err := svc.CreateAssessment(AssessmentRequest{Title: "published-empty"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetAssessmentPublished(emptyPaper.ID, true); err != nil {
		t.Fatal(err)
	}
	_, err = svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    emptyPaper.ID,
		PaperVersion:    "any-version",
		ClientRequestID: "empty-paper",
		Answers:         nil,
	})
	if !errors.Is(err, util.ErrAssessmentPaperUnavailable) {
		t.Fatalf("empty published paper: %v", err)
	}
	svc.Repo.DB.Model(&model.AssessmentSubmission{}).Where("client_request_id = ?", "empty-paper").Count(&count)
	if count != 0 {
		t.Fatal("empty-paper submit must not persist")
	}
}

func TestSubmitIdempotentAndConcurrent(t *testing.T) {
	svc, _ := newAssessmentService(t)
	userID := createTestUser(t, svc.Repo.DB, true)
	paper, ver := publishPaper(t, svc, "p", []model.AssessmentQuestion{
		{QuestionType: "single_choice", Content: "x", Options: choiceOptions(), Answer: "0"},
		{QuestionType: "essay", Content: "write"},
	})
	qs, _ := svc.Repo.ListAllQuestions(paper.ID)

	req := AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "same-click",
		Answers: []model.QuestionAnswer{
			{QuestionID: qs[0].ID, Answer: "0"},
			{QuestionID: qs[1].ID, Answer: "hello"},
		},
	}
	first, err := svc.SubmitAssessment(userID, req)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := svc.SubmitAssessment(userID, req)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != replay.ID || first.AttemptNo != 1 {
		t.Fatalf("idempotent replay %+v vs %+v", first, replay)
	}
	if first.PendingManualCount != 1 || first.ObjectiveMax != 10 || first.AutoScore != 10 {
		t.Fatalf("scoring fields %+v", first)
	}

	_, err = svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "same-click",
		Answers: []model.QuestionAnswer{
			{QuestionID: qs[0].ID, Answer: "1"},
			{QuestionID: qs[1].ID, Answer: "changed"},
		},
	})
	if !errors.Is(err, util.ErrAssessmentIdempotencyConflict) {
		t.Fatalf("same request id different body: %v", err)
	}

	replayAfterLock, err := svc.SubmitAssessment(userID, req)
	if err != nil {
		t.Fatal(err)
	}
	if replayAfterLock.ID != first.ID {
		t.Fatal("replay after canTake=false must not be blocked")
	}

	_, err = svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID:    paper.ID,
		PaperVersion:    ver,
		ClientRequestID: "second-click",
		Answers:         req.Answers,
	})
	if !errors.Is(err, util.ErrAssessmentRetestDenied) {
		t.Fatalf("repeat without retest: %v", err)
	}

	other := createTestUser(t, svc.Repo.DB, true)
	paper2, ver2 := publishPaper(t, svc, "concurrent", []model.AssessmentQuestion{{
		QuestionType: "single_choice", Content: "z", Options: choiceOptions(), Answer: "0",
	}})
	qs2, _ := svc.Repo.ListAllQuestions(paper2.ID)
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	ids := make(chan uint, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sub, subErr := svc.SubmitAssessment(other, AssessmentSubmissionRequest{
				AssessmentID:    paper2.ID,
				PaperVersion:    ver2,
				ClientRequestID: fmt.Sprintf("conc-%d", i),
				Answers:         []model.QuestionAnswer{{QuestionID: qs2[0].ID, Answer: "0"}},
			})
			if subErr != nil {
				errCh <- subErr
				return
			}
			ids <- sub.ID
		}(i)
	}
	wg.Wait()
	close(errCh)
	close(ids)
	denied := 0
	ok := 0
	for e := range errCh {
		if errors.Is(e, util.ErrAssessmentRetestDenied) {
			denied++
		} else if e != nil {
			t.Fatal(e)
		}
	}
	for range ids {
		ok++
	}
	if ok != 1 || denied != 1 {
		t.Fatalf("concurrent submit want 1 success 1 denied, ok=%d denied=%d", ok, denied)
	}
	t.Log("SQLite serializes writers; this is not proof of MySQL FOR UPDATE / isolation / unique-error mapping")
}

func TestStudentIsolationAndDiagnosisScope(t *testing.T) {
	svc, _ := newAssessmentService(t)
	a := createTestUser(t, svc.Repo.DB, true)
	b := createTestUser(t, svc.Repo.DB, true)
	paper, ver := publishPaper(t, svc, "diag", []model.AssessmentQuestion{{
		QuestionType: "single_choice", Content: "x", Options: choiceOptions(), Answer: "0",
	}})
	qs, _ := svc.Repo.ListAllQuestions(paper.ID)
	subA, err := svc.SubmitAssessment(a, AssessmentSubmissionRequest{
		AssessmentID: paper.ID, PaperVersion: ver, ClientRequestID: "a1",
		Answers: []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.GradeSubmission(subA.ID, GradeSubmissionRequest{Score: 10, RecommendedLevel: 3, Feedback: "ok"}); err != nil {
		t.Fatal(err)
	}

	statusB, err := svc.GetStudentAssessmentStatus(b)
	if err != nil {
		t.Fatal(err)
	}
	if statusB.Submission != nil || statusB.LastConfirmedDiagnosis != nil {
		t.Fatalf("student B must not see A: %+v", statusB)
	}

	if err := svc.SetUserCanRetest([]uint{a}, true); err != nil {
		t.Fatal(err)
	}
	pending, err := svc.SubmitAssessment(a, AssessmentSubmissionRequest{
		AssessmentID: paper.ID, PaperVersion: ver, ClientRequestID: "a2",
		Answers: []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if pending.AttemptNo != 2 || pending.RecommendedLevel != 0 {
		t.Fatalf("new pending %+v", pending)
	}
	statusA, err := svc.GetStudentAssessmentStatus(a)
	if err != nil {
		t.Fatal(err)
	}
	if statusA.LastConfirmedDiagnosis == nil || statusA.LastConfirmedDiagnosis.RecommendedLevel != 3 {
		t.Fatalf("pending retest must keep confirmed diagnosis %+v", statusA.LastConfirmedDiagnosis)
	}
	if statusA.LastConfirmedDiagnosis.PaperVersion != ver {
		t.Fatalf("confirmed diagnosis must match claimed version %s", ver)
	}
	if statusA.Submission == nil || statusA.Submission.ID != pending.ID {
		t.Fatalf("latest attempt should be pending %+v", statusA.Submission)
	}

	qs[0].Answer = "1"
	if err := svc.Repo.UpdateQuestion(&qs[0]); err != nil {
		t.Fatal(err)
	}
	statusEdited, err := svc.GetStudentAssessmentStatus(a)
	if err != nil {
		t.Fatal(err)
	}
	if statusEdited.LastConfirmedDiagnosis != nil {
		t.Fatalf("edited paper must not silently reuse old diagnosis %+v", statusEdited.LastConfirmedDiagnosis)
	}
	if statusEdited.StaleDiagnosis == nil || statusEdited.StaleDiagnosis.RecommendedLevel != 3 {
		t.Fatalf("incompatible diagnosis should be flagged stale %+v", statusEdited.StaleDiagnosis)
	}

	otherPaper, _ := publishPaper(t, svc, "other-scope", []model.AssessmentQuestion{{
		QuestionType: "single_choice", Content: "y", Options: choiceOptions(), Answer: "0",
	}})
	statusSwitched, err := svc.GetStudentAssessmentStatus(a)
	if err != nil {
		t.Fatal(err)
	}
	if statusSwitched.AssessmentID != otherPaper.ID {
		t.Fatalf("current paper %d", statusSwitched.AssessmentID)
	}
	if statusSwitched.LastConfirmedDiagnosis != nil {
		t.Fatalf("different paper must not inherit level %+v", statusSwitched.LastConfirmedDiagnosis)
	}
}

func TestDeleteThenRetestKeepsHistoryAndIncrementsAttemptNo(t *testing.T) {
	svc, _ := newAssessmentService(t)
	userID := createTestUser(t, svc.Repo.DB, true)
	paper, ver := publishPaper(t, svc, "keep-history", []model.AssessmentQuestion{{
		QuestionType: "single_choice", Content: "x", Options: choiceOptions(), Answer: "0",
	}})
	qs, _ := svc.Repo.ListAllQuestions(paper.ID)
	first, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID: paper.ID, PaperVersion: ver, ClientRequestID: "del-1",
		Answers: []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.GradeSubmission(first.ID, GradeSubmissionRequest{Score: 10, RecommendedLevel: 2, Feedback: "keep"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteSubmission(first.ID); err != nil {
		t.Fatal(err)
	}

	var archived model.AssessmentSubmission
	if err := svc.Repo.DB.Unscoped().First(&archived, first.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !archived.DeletedAt.Valid {
		t.Fatal("deleted row must remain as soft-deleted history")
	}
	if archived.RecommendedLevel != 2 || archived.AttemptNo != 1 {
		t.Fatalf("must not wipe teacher grade or attempt no %+v", archived)
	}

	if err := svc.SetUserCanRetest([]uint{userID}, true); err != nil {
		t.Fatal(err)
	}
	second, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID: paper.ID, PaperVersion: ver, ClientRequestID: "del-2",
		Answers: []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.AttemptNo != 2 {
		t.Fatalf("retest after delete must use next attempt_no, got %d", second.AttemptNo)
	}
	if second.ID == first.ID {
		t.Fatal("retest must insert a new row")
	}

	var still model.AssessmentSubmission
	if err := svc.Repo.DB.Unscoped().First(&still, first.ID).Error; err != nil {
		t.Fatal(err)
	}
	if still.RecommendedLevel != 2 || !still.DeletedAt.Valid {
		t.Fatalf("history row changed %+v", still)
	}
}

func TestLegacyDiagnosisWithoutVersionKeepsHistoryDisplay(t *testing.T) {
	svc, _ := newAssessmentService(t)
	userID := createTestUser(t, svc.Repo.DB, true)
	paper, ver := publishPaper(t, svc, "legacy-ver", []model.AssessmentQuestion{{
		QuestionType: "single_choice", Content: "x", Options: choiceOptions(), Answer: "0",
	}})
	qs, _ := svc.Repo.ListAllQuestions(paper.ID)
	sub, err := svc.SubmitAssessment(userID, AssessmentSubmissionRequest{
		AssessmentID: paper.ID, PaperVersion: ver, ClientRequestID: "legacy-1",
		Answers: []model.QuestionAnswer{{QuestionID: qs[0].ID, Answer: "0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.GradeSubmission(sub.ID, GradeSubmissionRequest{Score: 10, RecommendedLevel: 3, Feedback: "old"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Repo.DB.Model(&model.AssessmentSubmission{}).Where("id = ?", sub.ID).UpdateColumn("paper_version", "").Error; err != nil {
		t.Fatal(err)
	}

	status, err := svc.GetStudentAssessmentStatus(userID)
	if err != nil {
		t.Fatal(err)
	}
	if status.LastConfirmedDiagnosis != nil {
		t.Fatalf("missing version must not count as current diagnosis %+v", status.LastConfirmedDiagnosis)
	}
	if status.StaleDiagnosis != nil {
		t.Fatalf("missing version is history, not paper-stale %+v", status.StaleDiagnosis)
	}
	if status.HistoricalDiagnosis == nil || status.HistoricalDiagnosis.RecommendedLevel != 3 {
		t.Fatalf("want historical display %+v", status.HistoricalDiagnosis)
	}
	if status.HistoricalDiagnosis.PaperVersion != "" {
		t.Fatalf("must not fabricate paperVersion %+v", status.HistoricalDiagnosis)
	}

	current, err := svc.GetConfirmedDiagnosisForUser(userID)
	if err != nil {
		t.Fatal(err)
	}
	if current != nil {
		t.Fatalf("learning-path unlock must not use versionless diagnosis %+v", current)
	}

	var stored model.AssessmentSubmission
	if err := svc.Repo.DB.First(&stored, sub.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.RecommendedLevel != 3 || stored.PaperVersion != "" {
		t.Fatalf("must keep teacher grade and empty version %+v", stored)
	}
}
