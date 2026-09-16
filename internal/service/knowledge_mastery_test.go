package service

import (
	"encoding/json"
	"strings"
	"testing"

	"coder_edu_backend/internal/model"
)

func kpMeta(id, title string, order, passing int) KnowledgePointMasteryMeta {
	return KnowledgePointMasteryMeta{ID: id, Title: title, Order: order, CompletionScore: passing}
}

func objItem(id uint, index, awarded, max int, result string, kps ...string) model.AssessmentItemResult {
	return model.AssessmentItemResult{
		QuestionID:        id,
		QuestionIndex:     index,
		QuestionType:      "single_choice",
		AutoResult:        result,
		PointsAwarded:     awarded,
		MaxPoints:         max,
		KnowledgePointIDs: kps,
	}
}

func TestKnowledgeMasteryAllCorrectIsMastered(t *testing.T) {
	got := AggregateKnowledgeMastery(
		[]model.AssessmentItemResult{
			objItem(1, 1, 10, 10, model.AutoResultCorrect, "kp-var"),
			objItem(2, 2, 10, 10, model.AutoResultCorrect, "kp-var"),
		},
		map[string]KnowledgePointMasteryMeta{"kp-var": kpMeta("kp-var", "变量与数据类型", 1, 60)},
		nil,
	)
	if len(got) != 1 {
		t.Fatalf("%+v", got)
	}
	row := got[0]
	if row.EarnedPoints != 20 || row.MaxPoints != 20 || row.Percent != 100 || row.Status != KnowledgeMasteryStatusMastered {
		t.Fatalf("%+v", row)
	}
	if row.PassingScore != 60 || row.Title != "变量与数据类型" {
		t.Fatalf("%+v", row)
	}
}

func TestKnowledgeMasteryHalfScoreNeedsReview(t *testing.T) {
	got := AggregateKnowledgeMastery(
		[]model.AssessmentItemResult{
			objItem(1, 3, 10, 10, model.AutoResultCorrect, "kp-if"),
			objItem(2, 4, 0, 10, model.AutoResultWrong, "kp-if"),
		},
		map[string]KnowledgePointMasteryMeta{"kp-if": kpMeta("kp-if", "条件分支", 2, 60)},
		nil,
	)
	if len(got) != 1 || got[0].Percent != 50 || got[0].Status != KnowledgeMasteryStatusNeedsReview {
		t.Fatalf("%+v", got)
	}
	if len(got[0].IncorrectQuestionOrders) != 1 || got[0].IncorrectQuestionOrders[0] != 4 {
		t.Fatalf("%+v", got[0].IncorrectQuestionOrders)
	}
}

func TestKnowledgeMasteryAllWrongIsZero(t *testing.T) {
	got := AggregateKnowledgeMastery(
		[]model.AssessmentItemResult{
			objItem(3, 3, 0, 10, model.AutoResultWrong, "kp-if"),
			objItem(4, 4, 0, 10, model.AutoResultWrong, "kp-if"),
		},
		map[string]KnowledgePointMasteryMeta{"kp-if": kpMeta("kp-if", "条件分支", 2, 0)},
		nil,
	)
	if len(got) != 1 || got[0].Percent != 0 || got[0].PassingScore != DefaultKnowledgeMasteryPassingScore {
		t.Fatalf("invalid passing must default to 60, got %+v", got)
	}
	if got[0].Status != KnowledgeMasteryStatusNeedsReview {
		t.Fatalf("%+v", got)
	}
}

func TestKnowledgeMasterySameTotalDifferentWeakPoints(t *testing.T) {
	studentA := AggregateKnowledgeMastery(
		[]model.AssessmentItemResult{
			objItem(1, 1, 20, 20, model.AutoResultCorrect, "kp-var"),
			objItem(2, 2, 20, 20, model.AutoResultCorrect, "kp-loop"),
			objItem(3, 3, 0, 10, model.AutoResultWrong, "kp-if"),
			objItem(4, 4, 0, 10, model.AutoResultWrong, "kp-if"),
		},
		map[string]KnowledgePointMasteryMeta{
			"kp-var":  kpMeta("kp-var", "变量与数据类型", 1, 60),
			"kp-if":   kpMeta("kp-if", "条件分支", 2, 60),
			"kp-loop": kpMeta("kp-loop", "循环结构", 3, 60),
		},
		nil,
	)
	studentB := AggregateKnowledgeMastery(
		[]model.AssessmentItemResult{
			objItem(1, 1, 20, 20, model.AutoResultCorrect, "kp-var"),
			objItem(2, 2, 20, 20, model.AutoResultCorrect, "kp-if"),
			objItem(5, 5, 0, 10, model.AutoResultWrong, "kp-loop"),
			objItem(6, 6, 0, 10, model.AutoResultWrong, "kp-loop"),
		},
		map[string]KnowledgePointMasteryMeta{
			"kp-var":  kpMeta("kp-var", "变量与数据类型", 1, 60),
			"kp-if":   kpMeta("kp-if", "条件分支", 2, 60),
			"kp-loop": kpMeta("kp-loop", "循环结构", 3, 60),
		},
		nil,
	)
	aTotal, bTotal := 0, 0
	var aWeak, bWeak KnowledgeMastery
	for _, row := range studentA {
		aTotal += row.EarnedPoints
		if row.KnowledgePointID == "kp-if" {
			aWeak = row
		}
	}
	for _, row := range studentB {
		bTotal += row.EarnedPoints
		if row.KnowledgePointID == "kp-loop" {
			bWeak = row
		}
	}
	if aTotal != 40 || bTotal != 40 {
		t.Fatalf("same total expected 40, got A=%d B=%d", aTotal, bTotal)
	}
	if aWeak.Percent != 0 || bWeak.Percent != 0 {
		t.Fatalf("weak percents A=%+v B=%+v", aWeak, bWeak)
	}
	if aWeak.Title == bWeak.Title {
		t.Fatal("weak knowledge points must differ")
	}
}

func TestKnowledgeMasteryIgnoresUnlinkedAndManualItems(t *testing.T) {
	got := AggregateKnowledgeMastery(
		[]model.AssessmentItemResult{
			objItem(1, 1, 10, 10, model.AutoResultCorrect),
			{
				QuestionID: 2, QuestionIndex: 2, QuestionType: "essay",
				AutoResult: model.AutoResultPendingManual, KnowledgePointIDs: []string{"kp-if"},
			},
			objItem(3, 3, 0, 10, model.AutoResultWrong, "kp-if"),
		},
		map[string]KnowledgePointMasteryMeta{"kp-if": kpMeta("kp-if", "条件分支", 1, 60)},
		nil,
	)
	if len(got) != 1 || got[0].MaxPoints != 10 || got[0].EarnedPoints != 0 {
		t.Fatalf("unlinked and essay must be ignored, got %+v", got)
	}
}

func TestKnowledgeMasteryMultiKnowledgePointSharesFullScore(t *testing.T) {
	got := AggregateKnowledgeMastery(
		[]model.AssessmentItemResult{
			objItem(1, 1, 10, 10, model.AutoResultCorrect, "kp-a", "kp-b"),
		},
		map[string]KnowledgePointMasteryMeta{
			"kp-a": kpMeta("kp-a", "A", 2, 60),
			"kp-b": kpMeta("kp-b", "B", 1, 60),
		},
		nil,
	)
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
	if got[0].KnowledgePointID != "kp-b" || got[1].KnowledgePointID != "kp-a" {
		t.Fatalf("order by knowledge point order, got %+v", got)
	}
	for _, row := range got {
		if row.EarnedPoints != 10 || row.MaxPoints != 10 || row.Percent != 100 {
			t.Fatalf("full score counted on each KP: %+v", row)
		}
	}
}

func TestKnowledgeMasteryZeroMaxDoesNotPanic(t *testing.T) {
	got := AggregateKnowledgeMastery(
		[]model.AssessmentItemResult{
			objItem(1, 1, 0, 0, model.AutoResultWrong, "kp-zero"),
		},
		map[string]KnowledgePointMasteryMeta{"kp-zero": kpMeta("kp-zero", "空分值", 1, 60)},
		nil,
	)
	if got == nil {
		t.Fatal("must return empty slice, not nil panic")
	}
	if len(got) != 0 {
		t.Fatalf("maxPoints<=0 must be omitted, got %+v", got)
	}
}

func TestKnowledgeMasteryStableOrderByOrderThenTitleThenID(t *testing.T) {
	got := AggregateKnowledgeMastery(
		[]model.AssessmentItemResult{
			objItem(1, 1, 10, 10, model.AutoResultCorrect, "kp-c", "kp-a", "kp-b"),
		},
		map[string]KnowledgePointMasteryMeta{
			"kp-c": kpMeta("kp-c", "C标题", 5, 60),
			"kp-a": kpMeta("kp-a", "B标题", 1, 60),
			"kp-b": kpMeta("kp-b", "A标题", 1, 60),
		},
		nil,
	)
	ids := []string{got[0].KnowledgePointID, got[1].KnowledgePointID, got[2].KnowledgePointID}
	want := []string{"kp-b", "kp-a", "kp-c"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("got %v want %v", ids, want)
		}
	}
}

func TestKnowledgeMasteryJSONOmitsAnswerKeys(t *testing.T) {
	got := AggregateKnowledgeMastery(
		[]model.AssessmentItemResult{
			objItem(1, 1, 10, 10, model.AutoResultCorrect, "kp-var"),
		},
		map[string]KnowledgePointMasteryMeta{"kp-var": kpMeta("kp-var", "变量", 1, 60)},
		nil,
	)
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ToLower(string(raw))
	for _, banned := range []string{"\"answer\"", "explanation", "correctoption", "standardanswer"} {
		if strings.Contains(s, banned) {
			t.Fatalf("must not leak %s: %s", banned, raw)
		}
	}
}

func TestResolveKnowledgeMasteryPassingScore(t *testing.T) {
	if ResolveKnowledgeMasteryPassingScore(80) != 80 {
		t.Fatal("valid completionScore")
	}
	if ResolveKnowledgeMasteryPassingScore(0) != DefaultKnowledgeMasteryPassingScore {
		t.Fatal("zero is invalid")
	}
	if ResolveKnowledgeMasteryPassingScore(-1) != DefaultKnowledgeMasteryPassingScore {
		t.Fatal("negative is invalid")
	}
	if ResolveKnowledgeMasteryPassingScore(101) != DefaultKnowledgeMasteryPassingScore {
		t.Fatal(">100 is invalid")
	}
}
