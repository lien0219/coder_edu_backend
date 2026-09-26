package service

import (
	"errors"
	"strings"
	"testing"

	"coder_edu_backend/internal/learningprofile"
)

func TestSubmitIncompleteDLPretestRejectsBeforeDatabase(t *testing.T) {
	svc := NewLearningProfileService(nil)
	got, err := svc.Submit(5, learningprofile.InstrumentDL, learningprofile.WavePretest, map[string]int{
		"CT1": 3,
		"CT2": 4,
	})
	if got != nil {
		t.Fatalf("result=%+v", got)
	}
	if err == nil {
		t.Fatal("incomplete submit accepted")
	}
	if errors.Is(err, learningprofile.ErrCatalogUnavailable) {
		t.Fatal("reached writeWave/requireDB")
	}
	if err.Error() != "missing CT3" {
		t.Fatalf("err=%v", err)
	}
}

func TestSubmitIncompleteDLDoesNotFillFromDraftOrDefaults(t *testing.T) {
	svc := NewLearningProfileService(nil)
	_, err := svc.Submit(5, learningprofile.InstrumentDL, learningprofile.WavePretest, map[string]int{
		"CT1": 3,
		"CT2": 4,
	})
	if err == nil {
		t.Fatal("incomplete submit accepted")
	}
	msg := err.Error()
	if !strings.Contains(msg, "missing CT3") {
		t.Fatalf("err=%v", err)
	}
	if strings.Contains(msg, "missing CT1") || strings.Contains(msg, "missing CT2") {
		t.Fatalf("CT1/CT2 treated as missing: %v", err)
	}
}
