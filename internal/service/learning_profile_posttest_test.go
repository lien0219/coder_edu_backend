package service

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"coder_edu_backend/internal/learningprofile"
	"coder_edu_backend/internal/util"
)

func TestPosttestBlockedWhenEitherPretestMissing(t *testing.T) {
	svc := NewLearningProfileService(nil)
	t1 := time.Date(2026, 9, 23, 20, 16, 25, 0, time.Local)
	svc.pretestTimesFn = func(uint) (*time.Time, *time.Time, error) {
		return &t1, nil, nil
	}
	svc.now = func() time.Time { return t1.Add(8 * 24 * time.Hour) }

	if _, err := svc.GetPaper(5, learningprofile.InstrumentDL, learningprofile.WavePosttest); !errors.Is(err, learningprofile.ErrPosttestBlocked) {
		t.Fatalf("missing SDL GetPaper err=%v", err)
	}
	if _, err := svc.SaveDraft(5, learningprofile.InstrumentSDL, learningprofile.WavePosttest, map[string]int{"SDL1": 4}); !errors.Is(err, learningprofile.ErrPosttestBlocked) {
		t.Fatalf("missing SDL SaveDraft err=%v", err)
	}
	if _, err := svc.Submit(5, learningprofile.InstrumentDL, learningprofile.WavePosttest, map[string]int{"CT1": 3}); !errors.Is(err, learningprofile.ErrPosttestBlocked) {
		t.Fatalf("missing SDL Submit err=%v", err)
	}
}

func TestPosttestBlockedWhenOnlyDraftPretest(t *testing.T) {
	svc := NewLearningProfileService(nil)
	svc.pretestTimesFn = func(uint) (*time.Time, *time.Time, error) {
		return nil, nil, nil
	}
	if _, err := svc.SaveDraft(7, learningprofile.InstrumentDL, learningprofile.WavePosttest, map[string]int{"CT1": 3}); !errors.Is(err, learningprofile.ErrPosttestBlocked) {
		t.Fatalf("draft pretest err=%v", err)
	}
}

func TestPosttestOpenBoundaryUsesInjectedNow(t *testing.T) {
	dl := time.Date(2026, 9, 23, 20, 16, 25, 0, time.Local)
	sdl := time.Date(2026, 9, 23, 20, 17, 25, 0, time.Local)
	openAt := sdl.Add(7 * 24 * time.Hour)

	svc := NewLearningProfileService(nil)
	svc.pretestTimesFn = func(uint) (*time.Time, *time.Time, error) {
		d, s := dl, sdl
		return &d, &s, nil
	}

	svc.now = func() time.Time { return openAt.Add(-time.Second) }
	if _, err := svc.GetPaper(5, learningprofile.InstrumentDL, learningprofile.WavePosttest); !errors.Is(err, learningprofile.ErrPosttestBlocked) {
		t.Fatalf("before open GetPaper err=%v", err)
	}
	if _, err := svc.SaveDraft(5, learningprofile.InstrumentDL, learningprofile.WavePosttest, map[string]int{"CT1": 3}); !errors.Is(err, learningprofile.ErrPosttestBlocked) {
		t.Fatalf("before open SaveDraft err=%v", err)
	}
	if _, err := svc.Submit(5, learningprofile.InstrumentSDL, learningprofile.WavePosttest, map[string]int{"SDL1": 4}); !errors.Is(err, learningprofile.ErrPosttestBlocked) {
		t.Fatalf("before open Submit err=%v", err)
	}

	svc.now = func() time.Time { return openAt }
	_, err := svc.GetPaper(5, learningprofile.InstrumentDL, learningprofile.WavePosttest)
	if errors.Is(err, learningprofile.ErrPosttestBlocked) {
		t.Fatal("openAt GetPaper still blocked")
	}
	if !errors.Is(err, learningprofile.ErrCatalogUnavailable) {
		t.Fatalf("openAt GetPaper should reach catalog/db, err=%v", err)
	}
	_, err = svc.SaveDraft(5, learningprofile.InstrumentDL, learningprofile.WavePosttest, map[string]int{"CT1": 3})
	if errors.Is(err, learningprofile.ErrPosttestBlocked) {
		t.Fatal("openAt SaveDraft still blocked")
	}
	if !errors.Is(err, learningprofile.ErrCatalogUnavailable) {
		t.Fatalf("openAt SaveDraft should reach writeWave/requireDB, err=%v", err)
	}
}

func TestPosttestIncompleteSubmitReturnsMissingItem(t *testing.T) {
	dl := time.Date(2026, 9, 16, 10, 0, 0, 0, time.Local)
	openAt := dl.Add(7 * 24 * time.Hour)
	svc := NewLearningProfileService(nil)
	svc.pretestTimesFn = func(uint) (*time.Time, *time.Time, error) {
		d := dl
		return &d, &d, nil
	}
	svc.now = func() time.Time { return openAt }
	_, err := svc.Submit(5, learningprofile.InstrumentDL, learningprofile.WavePosttest, map[string]int{
		"CT1": 3,
		"CT2": 4,
	})
	if err == nil {
		t.Fatal("incomplete posttest accepted")
	}
	if errors.Is(err, learningprofile.ErrPosttestBlocked) {
		t.Fatal("open posttest should validate answers")
	}
	if errors.Is(err, learningprofile.ErrCatalogUnavailable) {
		t.Fatal("incomplete posttest reached writeWave")
	}
	if err.Error() != "missing CT3" {
		t.Fatalf("err=%v", err)
	}
}

func TestPosttestLockNameDiffersFromPretest(t *testing.T) {
	pre := learningProfileLockName(5, learningprofile.InstrumentDL, learningprofile.WavePretest)
	post := learningProfileLockName(5, learningprofile.InstrumentDL, learningprofile.WavePosttest)
	if pre == post {
		t.Fatal("lock names must include wave")
	}
	if pre != "lp-5-DL-C56-v1-pretest" || post != "lp-5-DL-C56-v1-posttest" {
		t.Fatalf("pre=%s post=%s", pre, post)
	}
}

func TestGetMeJSONKeepsPretestInstrumentsAndAddsPosttestFields(t *testing.T) {
	me := LearningProfileMe{
		Program:      "platform_self_assessment",
		Instruments:  []LearningProfileInstrumentStatus{{Code: "DL-C56-v1", Status: "submitted"}},
		PosttestOpen: false,
		PosttestInstruments: []LearningProfileInstrumentStatus{
			{Code: "DL-C56-v1", Status: "not_started"},
		},
	}
	raw, err := json.Marshal(me)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, `"instruments"`) || !strings.Contains(body, `"posttestOpen":false`) {
		t.Fatalf("compatible fields missing: %s", body)
	}
	if !strings.Contains(body, `"posttestInstruments"`) {
		t.Fatalf("posttestInstruments missing: %s", body)
	}
}

func TestPosttestIdentityRejectsZeroUserAndInvalidWave(t *testing.T) {
	svc := NewLearningProfileService(nil)
	if _, err := svc.GetPaper(0, learningprofile.InstrumentDL, learningprofile.WavePosttest); !errors.Is(err, util.ErrUnauthorized) {
		t.Fatalf("zero user err=%v", err)
	}
	if _, err := svc.SaveDraft(5, learningprofile.InstrumentDL, "midterm", map[string]int{"CT1": 3}); !errors.Is(err, learningprofile.ErrInvalidWave) {
		t.Fatalf("invalid wave err=%v", err)
	}
}
