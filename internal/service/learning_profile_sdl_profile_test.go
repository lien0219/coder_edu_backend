package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"coder_edu_backend/internal/learningprofile"
	"coder_edu_backend/internal/model"
)

func TestAssembleSdlProfileWaveCompletePretest(t *testing.T) {
	dlAdmin, sdlAdmin, dlInst, sdlInst, scores := completeSdlWaveFixture(learningprofile.WavePretest)
	wave := assembleSdlProfileWave(learningprofile.WavePretest, dlInst, sdlInst, dlAdmin, sdlAdmin, scores)
	if wave == nil {
		t.Fatal("complete pretest returned nil")
	}
	if wave.WaveType != learningprofile.WavePretest {
		t.Fatalf("wave=%s", wave.WaveType)
	}
	if len(wave.Instruments) != 2 || wave.Instruments[0].Code != learningprofile.InstrumentDL || wave.Instruments[1].Code != learningprofile.InstrumentSDL {
		t.Fatalf("instruments %+v", wave.Instruments)
	}
	if len(wave.Dimensions) != 7 {
		t.Fatalf("dims=%d", len(wave.Dimensions))
	}
	for i, spec := range learningprofile.RadarDimensionSpecs() {
		row := wave.Dimensions[i]
		if row.Code != spec.Code || row.InstrumentCode != spec.InstrumentCode || row.ItemCount != spec.ItemCount {
			t.Fatalf("dim %d %+v", i, row)
		}
		if row.DisplayScore < 0 || row.DisplayScore > 100 {
			t.Fatalf("display %v", row.DisplayScore)
		}
		if row.WaveType != learningprofile.WavePretest {
			t.Fatalf("dim wave %s", row.WaveType)
		}
	}
	if wave.SubmittedAt == nil || !wave.SubmittedAt.Equal(*sdlAdmin.SubmittedAt) {
		t.Fatal("wave submittedAt should be the later SDL time")
	}
	raw, err := json.Marshal(wave)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, forbidden := range []string{`"userId"`, "item_snapshot", `"answers"`, "itemSnapshot"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("payload contains %s: %s", forbidden, body)
		}
	}
}

func TestAssembleSdlProfileWaveMissingInstrumentIsNull(t *testing.T) {
	dlAdmin, _, dlInst, sdlInst, scores := completeSdlWaveFixture(learningprofile.WavePretest)
	if assembleSdlProfileWave(learningprofile.WavePretest, dlInst, sdlInst, dlAdmin, nil, scores) != nil {
		t.Fatal("missing SDL admin should be null")
	}
}

func TestAssembleSdlProfileWaveDraftIsNull(t *testing.T) {
	dlAdmin, sdlAdmin, dlInst, sdlInst, scores := completeSdlWaveFixture(learningprofile.WavePretest)
	dlAdmin.Status = model.LearningProfileStatusDraft
	if assembleSdlProfileWave(learningprofile.WavePretest, dlInst, sdlInst, dlAdmin, sdlAdmin, scores) != nil {
		t.Fatal("draft should be null")
	}
}

func TestAssembleSdlProfileWaveMissingDimensionIsNull(t *testing.T) {
	dlAdmin, sdlAdmin, dlInst, sdlInst, scores := completeSdlWaveFixture(learningprofile.WavePretest)
	if assembleSdlProfileWave(learningprofile.WavePretest, dlInst, sdlInst, dlAdmin, sdlAdmin, scores[:6]) != nil {
		t.Fatal("missing dimension should be null")
	}
}

func TestAssembleSdlProfileWaveDuplicateDimensionIsNull(t *testing.T) {
	dlAdmin, sdlAdmin, dlInst, sdlInst, scores := completeSdlWaveFixture(learningprofile.WavePretest)
	scores = append(scores, scores[0])
	if assembleSdlProfileWave(learningprofile.WavePretest, dlInst, sdlInst, dlAdmin, sdlAdmin, scores) != nil {
		t.Fatal("duplicate dimension should be null")
	}
}

func TestAssembleSdlProfileWaveExtraDimensionIsNull(t *testing.T) {
	dlAdmin, sdlAdmin, dlInst, sdlInst, scores := completeSdlWaveFixture(learningprofile.WavePretest)
	scores = append(scores, model.LearningProfileDimensionScore{
		AdministrationID: dlAdmin.ID,
		DimensionCode:    "unexpected_dimension",
		RawMean:          3,
		ScaleMin:         1,
		ScaleMax:         5,
		DisplayScore:     50,
		ItemCount:        1,
	})
	if assembleSdlProfileWave(learningprofile.WavePretest, dlInst, sdlInst, dlAdmin, sdlAdmin, scores) != nil {
		t.Fatal("extra dimension should be null")
	}
}

func TestAssembleSdlProfileWaveWrongItemCountOrScaleIsNull(t *testing.T) {
	dlAdmin, sdlAdmin, dlInst, sdlInst, scores := completeSdlWaveFixture(learningprofile.WavePretest)
	scores[0].ItemCount = 99
	if assembleSdlProfileWave(learningprofile.WavePretest, dlInst, sdlInst, dlAdmin, sdlAdmin, scores) != nil {
		t.Fatal("wrong item count should be null")
	}
	_, _, _, _, scores = completeSdlWaveFixture(learningprofile.WavePretest)
	scores[0].ScaleMax = 7
	if assembleSdlProfileWave(learningprofile.WavePretest, dlInst, sdlInst, dlAdmin, sdlAdmin, scores) != nil {
		t.Fatal("wrong scale should be null")
	}
}

func TestAssembleSdlProfileWaveDisplayScoreOutOfRangeIsNull(t *testing.T) {
	dlAdmin, sdlAdmin, dlInst, sdlInst, scores := completeSdlWaveFixture(learningprofile.WavePretest)
	scores[0].DisplayScore = 101
	if assembleSdlProfileWave(learningprofile.WavePretest, dlInst, sdlInst, dlAdmin, sdlAdmin, scores) != nil {
		t.Fatal("displayScore 101 should be null")
	}
}

func TestAssembleSdlProfileWavePosttestIndependent(t *testing.T) {
	if assembleSdlProfileWave(learningprofile.WavePosttest, nil, nil, nil, nil, nil) != nil {
		t.Fatal("empty posttest should be null")
	}
	dlAdmin, sdlAdmin, dlInst, sdlInst, scores := completeSdlWaveFixture(learningprofile.WavePosttest)
	wave := assembleSdlProfileWave(learningprofile.WavePosttest, dlInst, sdlInst, dlAdmin, sdlAdmin, scores)
	if wave == nil || wave.WaveType != learningprofile.WavePosttest {
		t.Fatal("complete posttest should assemble")
	}
}

func TestGetSdlProfileRejectsMissingUser(t *testing.T) {
	svc := NewLearningProfileService(nil)
	got, err := svc.GetSdlProfile(0)
	if got != nil || err == nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestSdlLearningProfileJSONNullPosttestKeepsLocalWallClock(t *testing.T) {
	dlAdmin, sdlAdmin, dlInst, sdlInst, scores := completeSdlWaveFixture(learningprofile.WavePretest)
	pretest := assembleSdlProfileWave(learningprofile.WavePretest, dlInst, sdlInst, dlAdmin, sdlAdmin, scores)
	if pretest == nil {
		t.Fatal("expected complete pretest")
	}
	profile := &SdlLearningProfile{
		Program:  model.LearningProfileProgramSelfAssessment,
		Pretest:  pretest,
		Posttest: nil,
	}
	raw, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, `"posttest":null`) {
		t.Fatalf("posttest must be JSON null: %s", body)
	}
	if !strings.Contains(body, `"pretest":{`) {
		t.Fatalf("pretest missing: %s", body)
	}
	if !strings.Contains(body, "20:16:25") || !strings.Contains(body, "20:17:25") {
		t.Fatalf("submittedAt wall clock rewritten: %s", body)
	}
	for _, forbidden := range []string{`"userId"`, "item_snapshot", `"answers"`, "itemSnapshot"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("payload contains %s: %s", forbidden, body)
		}
	}
}

func completeSdlWaveFixture(wave string) (*model.LearningProfileAdministration, *model.LearningProfileAdministration, *model.LearningProfileInstrument, *model.LearningProfileInstrument, []model.LearningProfileDimensionScore) {
	dlAt := time.Date(2026, 9, 23, 20, 16, 25, 0, time.Local)
	sdlAt := time.Date(2026, 9, 23, 20, 17, 25, 0, time.Local)
	dlInst := &model.LearningProfileInstrument{Code: learningprofile.InstrumentDL, Version: "v1", Program: model.LearningProfileProgramSelfAssessment, ScaleMin: 1, ScaleMax: 5, ItemCount: 56, Status: "active"}
	sdlInst := &model.LearningProfileInstrument{Code: learningprofile.InstrumentSDL, Version: "v1", Program: model.LearningProfileProgramSelfAssessment, ScaleMin: 1, ScaleMax: 7, ItemCount: 20, Status: "active"}
	dlAdmin := &model.LearningProfileAdministration{
		ID: 1, UserID: 5, InstrumentID: 4, WaveType: wave, Program: model.LearningProfileProgramSelfAssessment,
		Status: model.LearningProfileStatusSubmitted, SubmittedAt: &dlAt,
	}
	sdlAdmin := &model.LearningProfileAdministration{
		ID: 2, UserID: 5, InstrumentID: 5, WaveType: wave, Program: model.LearningProfileProgramSelfAssessment,
		Status: model.LearningProfileStatusSubmitted, SubmittedAt: &sdlAt,
	}
	var scores []model.LearningProfileDimensionScore
	for _, spec := range learningprofile.RadarDimensionSpecs() {
		adminID := dlAdmin.ID
		if spec.InstrumentCode == learningprofile.InstrumentSDL {
			adminID = sdlAdmin.ID
		}
		raw := float64(spec.ScaleMin+spec.ScaleMax) / 2
		display, err := learningprofile.DisplayScore(raw, spec.ScaleMin, spec.ScaleMax)
		if err != nil {
			panic(err)
		}
		scores = append(scores, model.LearningProfileDimensionScore{
			AdministrationID: adminID,
			DimensionCode:    spec.Code,
			RawMean:          raw,
			ScaleMin:         spec.ScaleMin,
			ScaleMax:         spec.ScaleMax,
			DisplayScore:     display,
			ItemCount:        spec.ItemCount,
		})
	}
	return dlAdmin, sdlAdmin, dlInst, sdlInst, scores
}