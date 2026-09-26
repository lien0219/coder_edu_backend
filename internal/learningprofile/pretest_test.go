package learningprofile

import "testing"

func TestRequirePretestWaveBlocksPosttest(t *testing.T) {
	if err := RequirePretestWave(WavePretest); err != nil {
		t.Fatalf("pretest: %v", err)
	}
	if err := RequirePretestWave(WavePosttest); err != ErrPosttestBlocked {
		t.Fatalf("posttest err=%v", err)
	}
	if err := RequirePretestWave(""); err != ErrInvalidWave {
		t.Fatalf("empty wave err=%v", err)
	}
	if err := RequirePretestWave("PRETEST"); err != ErrInvalidWave {
		t.Fatalf("mixed-case wave err=%v", err)
	}
}

func TestRequireWaveAllowsPretestAndPosttest(t *testing.T) {
	if err := RequireWave(WavePretest); err != nil {
		t.Fatal(err)
	}
	if err := RequireWave(WavePosttest); err != nil {
		t.Fatal(err)
	}
	if err := RequireWave("midterm"); err != ErrInvalidWave {
		t.Fatalf("midterm err=%v", err)
	}
	if err := RequireWave(""); err != ErrInvalidWave {
		t.Fatalf("empty err=%v", err)
	}
}

func TestOfficialInstrumentGate(t *testing.T) {
	if !IsOfficialInstrument(InstrumentDL) || !IsOfficialInstrument(InstrumentSDL) {
		t.Fatal("official instruments rejected")
	}
	if IsOfficialInstrument("DL-C56-v2") || IsOfficialInstrument("") {
		t.Fatal("unknown instrument accepted")
	}
}

func TestValidateDraftAnswersAllowsPartial(t *testing.T) {
	if err := ValidateDraftAnswers(InstrumentDL, nil); err != nil {
		t.Fatalf("empty draft: %v", err)
	}
	if err := ValidateDraftAnswers(InstrumentDL, map[string]int{"CT1": 3, "PS1": 5}); err != nil {
		t.Fatalf("partial draft: %v", err)
	}
	if err := ValidateDraftAnswers(InstrumentSDL, map[string]int{"SDL1": 7}); err != nil {
		t.Fatalf("sdl draft: %v", err)
	}
}

func TestValidateDraftAnswersRejectsBadCodesAndScales(t *testing.T) {
	if err := ValidateDraftAnswers(InstrumentDL, map[string]int{"SDL1": 4}); err == nil {
		t.Fatal("cross-instrument item accepted")
	}
	if err := ValidateDraftAnswers(InstrumentDL, map[string]int{"CT1": 6}); err == nil {
		t.Fatal("DL value 6 accepted")
	}
	if err := ValidateDraftAnswers(InstrumentSDL, map[string]int{"SDL1": 0}); err == nil {
		t.Fatal("SDL value 0 accepted")
	}
	if err := ValidateDraftAnswers("other", map[string]int{"CT1": 1}); err != ErrUnknownInstrument {
		t.Fatalf("unknown instrument err=%v", err)
	}
}

func TestValidateOfficialAnswersIncompleteDLReportsMissingCT3(t *testing.T) {
	err := ValidateOfficialAnswers(InstrumentDL, map[string]int{"CT1": 3, "CT2": 4})
	if err == nil {
		t.Fatal("two-item DL submit accepted")
	}
	if err.Error() != "missing CT3" {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateOfficialAnswersRequiresCompleteSet(t *testing.T) {
	if err := ValidateOfficialAnswers(InstrumentDL, map[string]int{"CT1": 1}); err == nil {
		t.Fatal("incomplete DL submit accepted")
	}
	dl := completeAnswers(InstrumentDL, 4)
	if err := ValidateOfficialAnswers(InstrumentDL, dl); err != nil {
		t.Fatal(err)
	}
	dl["EXTRA"] = 1
	if err := ValidateOfficialAnswers(InstrumentDL, dl); err == nil {
		t.Fatal("extra DL answer accepted")
	}
	sdl := completeAnswers(InstrumentSDL, 4)
	if err := ValidateOfficialAnswers(InstrumentSDL, sdl); err != nil {
		t.Fatal(err)
	}
	delete(sdl, "SDL20")
	if err := ValidateOfficialAnswers(InstrumentSDL, sdl); err == nil {
		t.Fatal("incomplete SDL submit accepted")
	}
}

func TestMatchOfficialItemCodes(t *testing.T) {
	var codes []string
	for _, item := range ItemsByInstrument(InstrumentDL) {
		codes = append(codes, item.ItemCode)
	}
	if err := MatchOfficialItemCodes(InstrumentDL, codes); err != nil {
		t.Fatal(err)
	}
	if err := MatchOfficialItemCodes(InstrumentDL, codes[:55]); err == nil {
		t.Fatal("missing item accepted")
	}
	codes[0] = "ZZ1"
	if err := MatchOfficialItemCodes(InstrumentDL, codes); err == nil {
		t.Fatal("renamed item accepted")
	}
}

func TestPretestRejectsOverwriteOfSubmittedStatus(t *testing.T) {
	if err := CanTransitionToSubmitted(StatusDraft); err != nil {
		t.Fatal(err)
	}
	if err := CanTransitionToSubmitted(""); err != nil {
		t.Fatal(err)
	}
	if err := CanTransitionToSubmitted(StatusSubmitted); err == nil {
		t.Fatal("submitted overwrite allowed")
	}
}

func TestAdministrationStatus(t *testing.T) {
	if got := AdministrationStatus(false, ""); got != "not_started" {
		t.Fatalf("got %s", got)
	}
	if got := AdministrationStatus(true, StatusDraft); got != StatusDraft {
		t.Fatalf("got %s", got)
	}
	if got := AdministrationStatus(true, StatusSubmitted); got != StatusSubmitted {
		t.Fatalf("got %s", got)
	}
}
