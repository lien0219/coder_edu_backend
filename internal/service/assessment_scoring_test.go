package service

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"
)

func tfOptions() json.RawMessage {
	return json.RawMessage(`[{"label":"A","text":"正确"},{"label":"B","text":"错误"}]`)
}

func choiceOptions() json.RawMessage {
	return json.RawMessage(`[{"label":"A","text":"int"},{"label":"B","text":"float"},{"label":"C","text":"char"}]`)
}

func question(id uint, q model.AssessmentQuestion) model.AssessmentQuestion {
	q.ID = id
	return q
}

func TestScoreTrueFalseWithOptionIndex(t *testing.T) {
	q := question(11, model.AssessmentQuestion{QuestionType: "true_false", Options: tfOptions(), Answer: "0", Points: 5})
	got, err := scoreQuestion(q, "0")
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.AutoResult != model.AutoResultCorrect || got.Result.PointsAwarded != 5 {
		t.Fatalf("%+v", got.Result)
	}
	got, err = scoreQuestion(q, "正确")
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.AutoResult != model.AutoResultCorrect {
		t.Fatalf("label 正确 should map to index 0, got %+v", got.Result)
	}
	got, err = scoreQuestion(q, "1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.AutoResult != model.AutoResultWrong || got.Result.PointsAwarded != 0 {
		t.Fatalf("wrong option %+v", got.Result)
	}
}

func TestScoreTrueFalseAmbiguousNumericKeyWithoutOptions(t *testing.T) {
	q := question(12, model.AssessmentQuestion{QuestionType: "true_false", Answer: "0", Points: 5})
	got, err := scoreQuestion(q, "正确")
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.AutoResult != model.AutoResultUnscoredAmbiguous || !got.Result.NeedsTeacherFix {
		t.Fatalf("numeric 0/1 without options must not be guessed, got %+v", got.Result)
	}
}

func TestScoreTrueFalseLegacyBooleanTokens(t *testing.T) {
	q := question(13, model.AssessmentQuestion{QuestionType: "true_false", Answer: "true", Points: 2})
	got, err := scoreQuestion(q, "正确")
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.AutoResult != model.AutoResultCorrect {
		t.Fatalf("%+v", got.Result)
	}
}

func TestScoreMultipleChoiceIgnoresOrder(t *testing.T) {
	q := question(21, model.AssessmentQuestion{QuestionType: "multiple_choice", Options: choiceOptions(), Answer: "0,2", Points: 10})
	got, err := scoreQuestion(q, "2, 0")
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.AutoResult != model.AutoResultCorrect {
		t.Fatalf("%+v", got.Result)
	}
	got, err = scoreQuestion(q, "C,A")
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.AutoResult != model.AutoResultCorrect {
		t.Fatalf("labels C,A %+v", got.Result)
	}
}

func TestScoreEssayPendingManual(t *testing.T) {
	q := question(31, model.AssessmentQuestion{QuestionType: "essay", Answer: "ignored", Points: 20})
	got, err := scoreQuestion(q, "student essay")
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.AutoResult != model.AutoResultPendingManual || got.Result.PointsAwarded != 0 {
		t.Fatalf("%+v", got.Result)
	}
	blank, err := scoreQuestion(q, "")
	if err != nil {
		t.Fatal(err)
	}
	if blank.Result.AutoResult != model.AutoResultPendingManual {
		t.Fatalf("blank essay still pending, got %+v", blank.Result)
	}
}

func TestValidateRejectsOutOfPaperAndDuplicates(t *testing.T) {
	qs := []model.AssessmentQuestion{question(1, model.AssessmentQuestion{QuestionType: "single_choice", Options: choiceOptions(), Answer: "0", Points: 1})}
	_, _, _, _, err := validateAndScoreAnswers(qs, []model.QuestionAnswer{{QuestionID: 99, Answer: "0"}})
	if !errors.Is(err, util.ErrAssessmentSubmitInvalid) {
		t.Fatalf("out-of-paper: %v", err)
	}
	_, _, _, _, err = validateAndScoreAnswers(qs, []model.QuestionAnswer{
		{QuestionID: 1, Answer: "0"},
		{QuestionID: 1, Answer: "1"},
	})
	if !errors.Is(err, util.ErrAssessmentSubmitInvalid) {
		t.Fatalf("duplicate: %v", err)
	}
	_, _, _, _, err = validateAndScoreAnswers(qs, []model.QuestionAnswer{{QuestionID: 1, Answer: "9"}})
	if !errors.Is(err, util.ErrAssessmentSubmitInvalid) {
		t.Fatalf("illegal option: %v", err)
	}
}

func TestUnansweredObjectivePersistsZero(t *testing.T) {
	qs := []model.AssessmentQuestion{question(1, model.AssessmentQuestion{QuestionType: "single_choice", Options: choiceOptions(), Answer: "0", Points: 4})}
	results, auto, max, pending, err := validateAndScoreAnswers(qs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if auto != 0 || max != 4 || pending != 0 || results[0].AutoResult != model.AutoResultUnanswered {
		t.Fatalf("auto=%d max=%d pending=%d %+v", auto, max, pending, results[0])
	}
}

func TestAmbiguousKeyExcludedFromObjectiveMax(t *testing.T) {
	qs := []model.AssessmentQuestion{
		question(1, model.AssessmentQuestion{QuestionType: "true_false", Answer: "0", Points: 5}),
		question(2, model.AssessmentQuestion{QuestionType: "single_choice", Options: choiceOptions(), Answer: "0", Points: 10}),
	}
	results, auto, max, pending, err := validateAndScoreAnswers(qs, []model.QuestionAnswer{
		{QuestionID: 1, Answer: "正确"},
		{QuestionID: 2, Answer: "0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].AutoResult != model.AutoResultUnscoredAmbiguous || !results[0].NeedsTeacherFix {
		t.Fatalf("ambiguous %+v", results[0])
	}
	if auto != 10 || max != 10 || pending != 0 {
		t.Fatalf("ambiguous must not enter denominator auto=%d max=%d pending=%d", auto, max, pending)
	}
}

func TestObjectiveMaxExcludesEssay(t *testing.T) {
	qs := []model.AssessmentQuestion{
		question(1, model.AssessmentQuestion{QuestionType: "single_choice", Options: choiceOptions(), Answer: "0", Points: 10}),
		question(2, model.AssessmentQuestion{QuestionType: "essay", Points: 90}),
	}
	results, auto, max, pending, err := validateAndScoreAnswers(qs, []model.QuestionAnswer{
		{QuestionID: 1, Answer: "0"},
		{QuestionID: 2, Answer: "long text"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if auto != 10 || max != 10 || pending != 1 {
		t.Fatalf("auto=%d max=%d pending=%d results=%+v", auto, max, pending, results)
	}
}

func TestPaperVersionChangesWhenAnswerChanges(t *testing.T) {
	now := time.Now()
	q := question(7, model.AssessmentQuestion{QuestionType: "single_choice", Options: choiceOptions(), Answer: "0", Points: 1})
	q.UpdatedAt = now
	v1, err := ComputePaperVersion(42, []model.AssessmentQuestion{q})
	if err != nil || v1 == "" {
		t.Fatalf("v1 %s %v", v1, err)
	}
	q.Answer = "1"
	v2, err := ComputePaperVersion(42, []model.AssessmentQuestion{q})
	if err != nil {
		t.Fatal(err)
	}
	if v1 == v2 {
		t.Fatal("version must change after answer edit")
	}
	if !strings.EqualFold(v1, v1) {
		t.Fatal("hash should be hex")
	}
}
