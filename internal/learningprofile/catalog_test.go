package learningprofile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateCatalog(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogCountsAndUniqueness(t *testing.T) {
	items := OfficialItems()
	if len(items) != 76 {
		t.Fatalf("got %d", len(items))
	}
	if len(ItemsByInstrument(InstrumentDL)) != 56 {
		t.Fatal("DL count")
	}
	if len(ItemsByInstrument(InstrumentSDL)) != 20 {
		t.Fatal("SDL count")
	}
	if Instruments()[0].ItemCount != 56 || Instruments()[1].ItemCount != 20 {
		t.Fatal("instrument item_count")
	}
}

func TestRadarDimensionSpecs(t *testing.T) {
	specs := RadarDimensionSpecs()
	if len(specs) != 7 {
		t.Fatalf("got %d specs", len(specs))
	}
	want := []RadarDimensionSpec{
		{Code: DimCriticalThinking, InstrumentCode: InstrumentDL, ItemCount: 26, ScaleMin: 1, ScaleMax: 5},
		{Code: DimProblemSolving, InstrumentCode: InstrumentDL, ItemCount: 18, ScaleMin: 1, ScaleMax: 5},
		{Code: DimKnowledgeTransfer, InstrumentCode: InstrumentDL, ItemCount: 12, ScaleMin: 1, ScaleMax: 5},
		{Code: DimLearningMotivation, InstrumentCode: InstrumentSDL, ItemCount: 6, ScaleMin: 1, ScaleMax: 7},
		{Code: DimPlanningAndImplementation, InstrumentCode: InstrumentSDL, ItemCount: 6, ScaleMin: 1, ScaleMax: 7},
		{Code: DimSelfManagement, InstrumentCode: InstrumentSDL, ItemCount: 4, ScaleMin: 1, ScaleMax: 7},
		{Code: DimInterpersonalCommunication, InstrumentCode: InstrumentSDL, ItemCount: 4, ScaleMin: 1, ScaleMax: 7},
	}
	for i, spec := range specs {
		if spec != want[i] {
			t.Fatalf("index %d got %+v want %+v", i, spec, want[i])
		}
		if spec.Code != DimensionOrder()[i] {
			t.Fatalf("order %s", spec.Code)
		}
	}
}

func TestCatalogHasNoDemographicsOrAppendix(t *testing.T) {
	for _, item := range OfficialItems() {
		joined := item.Stem + item.ItemCode + item.SecondaryDimension
		for _, bad := range []string{"性别：", "KMO", "Bartlett", "EFA", "CFA必须"} {
			if strings.Contains(joined, bad) {
				t.Fatalf("%s contains %q", item.ItemCode, bad)
			}
		}
	}
}

func TestValidateOfficialAnswers(t *testing.T) {
	answers := map[string]int{}
	for _, item := range ItemsByInstrument(InstrumentDL) {
		answers[item.ItemCode] = item.ScaleMin
	}
	if err := ValidateOfficialAnswers(InstrumentDL, answers); err != nil {
		t.Fatal(err)
	}
	delete(answers, "CT1")
	if err := ValidateOfficialAnswers(InstrumentDL, answers); err == nil {
		t.Fatal("missing item should fail")
	}
	answers["CT1"] = 1
	answers["CT1"] = 9
	if err := ValidateOfficialAnswers(InstrumentDL, answers); err == nil {
		t.Fatal("out of range should fail")
	}
}

func TestCanTransitionToSubmitted(t *testing.T) {
	if err := CanTransitionToSubmitted(StatusDraft); err != nil {
		t.Fatal(err)
	}
	if err := CanTransitionToSubmitted(StatusSubmitted); err == nil {
		t.Fatal("submitted cannot resubmit")
	}
}

func TestDisplayScore(t *testing.T) {
	got, err := DisplayScore(1, 1, 5)
	if err != nil || got != 0 {
		t.Fatalf("DL min: %v %v", got, err)
	}
	got, err = DisplayScore(5, 1, 5)
	if err != nil || got != 100 {
		t.Fatalf("DL max: %v %v", got, err)
	}
	got, err = DisplayScore(1, 1, 7)
	if err != nil || got != 0 {
		t.Fatalf("SDL min: %v %v", got, err)
	}
	got, err = DisplayScore(7, 1, 7)
	if err != nil || got != 100 {
		t.Fatalf("SDL max: %v %v", got, err)
	}
	got, err = DisplayScore(3, 1, 5)
	if err != nil || got != 50 {
		t.Fatalf("DL mid: %v %v", got, err)
	}
	if _, err := DisplayScore(0, 1, 5); err == nil {
		t.Fatal("below min")
	}
}

func TestIsStoredDisplayScore(t *testing.T) {
	if !IsStoredDisplayScore(0) || !IsStoredDisplayScore(50) || !IsStoredDisplayScore(100) {
		t.Fatal("0–100 should be accepted")
	}
	if IsStoredDisplayScore(-0.01) || IsStoredDisplayScore(100.01) {
		t.Fatal("outside 0–100 should be rejected")
	}
}

func TestScoreInstrumentDL(t *testing.T) {
	answers := map[string]int{}
	for _, item := range ItemsByInstrument(InstrumentDL) {
		answers[item.ItemCode] = 5
	}
	scores, err := ScoreInstrument(InstrumentDL, answers)
	if err != nil {
		t.Fatal(err)
	}
	if len(scores) != 3 {
		t.Fatalf("got %d dims", len(scores))
	}
	if scores[0].DimensionCode != DimCriticalThinking || scores[0].ItemCount != 26 {
		t.Fatalf("%+v", scores[0])
	}
	if scores[0].RawMean != 5 || scores[0].DisplayScore != 100 {
		t.Fatalf("mean %+v", scores[0])
	}
	if scores[1].ItemCount != 18 || scores[2].ItemCount != 12 {
		t.Fatalf("counts %+v", scores)
	}
}

func TestPosttestOpenAt(t *testing.T) {
	t1 := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 9, 2, 9, 0, 0, 0, time.UTC)
	if PosttestOpenAt(&t1, nil) != nil {
		t.Fatal("one pretest must not start countdown")
	}
	open := PosttestOpenAt(&t1, &t2)
	if open == nil {
		t.Fatal("expected open at")
	}
	want := t2.Add(7 * 24 * time.Hour)
	if !open.Equal(want) {
		t.Fatalf("got %s want %s", open, want)
	}
	if PosttestIsOpen(want.Add(-time.Second), &t1, &t2) {
		t.Fatal("before boundary")
	}
	if !PosttestIsOpen(want, &t1, &t2) {
		t.Fatal("boundary should allow")
	}
	if err := CheckPosttestOpen(want.Add(-time.Second), &t1, &t2); err != ErrPosttestBlocked {
		t.Fatalf("before boundary err=%v", err)
	}
	if err := CheckPosttestOpen(want, &t1, &t2); err != nil {
		t.Fatalf("boundary err=%v", err)
	}
	if err := CheckPosttestOpen(want, &t1, nil); err != ErrPosttestBlocked {
		t.Fatalf("missing SDL err=%v", err)
	}
}

func TestSDL10Code(t *testing.T) {
	item, ok := ItemByCode("SDL10")
	if !ok || item.SortOrder != 10 || item.SourceItemNo != "10" {
		t.Fatalf("%+v ok=%v", item, ok)
	}
}

func TestSeedSQLMatchesCatalog(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", "20260923_learning_profile_seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "性别") || strings.Contains(text, "KMO") {
		t.Fatal("seed must not include demographics or appendix")
	}
	for _, item := range OfficialItems() {
		if !strings.Contains(text, item.ItemCode) {
			t.Fatalf("seed missing code %s", item.ItemCode)
		}
		if !strings.Contains(text, item.Stem) {
			t.Fatalf("seed missing stem for %s", item.ItemCode)
		}
	}
}
