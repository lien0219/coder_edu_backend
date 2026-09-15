package service

import (
	"testing"

	"coder_edu_backend/internal/model"
)

func TestRecommendObjectiveLevelBoundaries(t *testing.T) {
	cases := []struct {
		earned int
		max    int
		level  int
		ok     bool
	}{
		{0, 100, 1, true},
		{59, 100, 1, true},
		{60, 100, 2, true},
		{74, 100, 2, true},
		{75, 100, 3, true},
		{89, 100, 3, true},
		{90, 100, 4, true},
		{100, 100, 4, true},
		{40, 60, 2, true},
		{0, 0, 0, false},
		{10, -1, 0, false},
		{-5, 100, 1, true},
		{150, 100, 4, true},
	}
	for _, tc := range cases {
		got, ok := RecommendObjectiveLevel(tc.earned, tc.max)
		if got != tc.level || ok != tc.ok {
			t.Fatalf("earned=%d max=%d want %d/%v got %d/%v", tc.earned, tc.max, tc.level, tc.ok, got, ok)
		}
	}
}

func TestRecommendObjectiveLevelUsesPercentNotRawScore(t *testing.T) {
	level, ok := RecommendObjectiveLevel(40, 60)
	if !ok || level != 2 {
		t.Fatalf("40/60 must be level 2, got %d ok=%v", level, ok)
	}
	if level == 1 {
		t.Fatal("must not treat 40 as an integer cutoff for basic")
	}
}

func TestCanAutoCompleteAssessmentRules(t *testing.T) {
	objective := []model.AssessmentQuestion{
		{QuestionType: "single_choice"},
		{QuestionType: "fill_blank"},
	}
	okResults := []model.AssessmentItemResult{
		{AutoResult: model.AutoResultCorrect},
		{AutoResult: model.AutoResultUnanswered},
	}
	if !CanAutoCompleteAssessment(objective, okResults, 20) {
		t.Fatal("unanswered objective items must still auto-complete")
	}
	if CanAutoCompleteAssessment(nil, nil, 10) {
		t.Fatal("empty paper must not auto-complete")
	}
	if CanAutoCompleteAssessment(objective, okResults, 0) {
		t.Fatal("objectiveMax<=0 must not auto-complete")
	}
	mixed := []model.AssessmentQuestion{
		{QuestionType: "single_choice"},
		{QuestionType: "essay"},
	}
	mixedResults := []model.AssessmentItemResult{
		{AutoResult: model.AutoResultCorrect},
		{AutoResult: model.AutoResultPendingManual},
	}
	if CanAutoCompleteAssessment(mixed, mixedResults, 10) {
		t.Fatal("essay paper must not auto-complete")
	}
	ambiguous := []model.AssessmentItemResult{
		{AutoResult: model.AutoResultCorrect},
		{AutoResult: model.AutoResultUnscoredAmbiguous},
	}
	if CanAutoCompleteAssessment(objective, ambiguous, 10) {
		t.Fatal("ambiguous auto-score must not auto-complete")
	}
	unknown := []model.AssessmentQuestion{{QuestionType: "code"}}
	codeResults := []model.AssessmentItemResult{{AutoResult: model.AutoResultPendingManual}}
	if CanAutoCompleteAssessment(unknown, codeResults, 10) {
		t.Fatal("code questions must not auto-complete")
	}
}
