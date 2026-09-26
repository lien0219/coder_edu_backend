package service

import (
	"errors"
	"time"

	"coder_edu_backend/internal/learningprofile"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"

	"gorm.io/gorm"
)

type SdlLearningProfile struct {
	Program  string          `json:"program"`
	Pretest  *SdlProfileWave `json:"pretest"`
	Posttest *SdlProfileWave `json:"posttest"`
}

type SdlProfileWave struct {
	WaveType    string                 `json:"waveType"`
	SubmittedAt *time.Time             `json:"submittedAt,omitempty"`
	Instruments []SdlProfileInstrument `json:"instruments"`
	Dimensions  []SdlProfileDimension  `json:"dimensions"`
}

type SdlProfileInstrument struct {
	Code        string     `json:"code"`
	Version     string     `json:"version"`
	WaveType    string     `json:"waveType"`
	SubmittedAt *time.Time `json:"submittedAt,omitempty"`
}

type SdlProfileDimension struct {
	Code              string     `json:"code"`
	RawMean           float64    `json:"rawMean"`
	ScaleMin          int        `json:"scaleMin"`
	ScaleMax          int        `json:"scaleMax"`
	DisplayScore      float64    `json:"displayScore"`
	ItemCount         int        `json:"itemCount"`
	InstrumentCode    string     `json:"instrumentCode"`
	InstrumentVersion string     `json:"instrumentVersion"`
	WaveType          string     `json:"waveType"`
	SubmittedAt       *time.Time `json:"submittedAt,omitempty"`
}

func (s *LearningProfileService) GetSdlProfile(userID uint) (*SdlLearningProfile, error) {
	if userID == 0 {
		return nil, util.ErrUnauthorized
	}
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	pretest, err := s.loadSdlProfileWave(userID, learningprofile.WavePretest)
	if err != nil {
		return nil, err
	}
	posttest, err := s.loadSdlProfileWave(userID, learningprofile.WavePosttest)
	if err != nil {
		return nil, err
	}
	return &SdlLearningProfile{
		Program:  model.LearningProfileProgramSelfAssessment,
		Pretest:  pretest,
		Posttest: posttest,
	}, nil
}

func (s *LearningProfileService) loadSdlProfileWave(userID uint, wave string) (*SdlProfileWave, error) {
	dlInst, err := s.findActiveInstrument(model.LearningProfileInstrumentDL)
	if err != nil {
		return nil, err
	}
	sdlInst, err := s.findActiveInstrument(model.LearningProfileInstrumentSDL)
	if err != nil {
		return nil, err
	}
	if dlInst == nil || sdlInst == nil {
		return nil, nil
	}
	dlAdmin, err := s.findSubmittedAdministration(userID, dlInst.ID, wave)
	if err != nil {
		return nil, err
	}
	sdlAdmin, err := s.findSubmittedAdministration(userID, sdlInst.ID, wave)
	if err != nil {
		return nil, err
	}
	if dlAdmin == nil || sdlAdmin == nil {
		return nil, nil
	}
	var scores []model.LearningProfileDimensionScore
	if err := s.db.Where("administration_id IN ?", []uint{dlAdmin.ID, sdlAdmin.ID}).Find(&scores).Error; err != nil {
		return nil, err
	}
	return assembleSdlProfileWave(wave, dlInst, sdlInst, dlAdmin, sdlAdmin, scores), nil
}

func (s *LearningProfileService) findActiveInstrument(code string) (*model.LearningProfileInstrument, error) {
	var inst model.LearningProfileInstrument
	err := s.db.Where("code = ? AND program = ? AND status = ?", code, model.LearningProfileProgramSelfAssessment, "active").
		First(&inst).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

func (s *LearningProfileService) findSubmittedAdministration(userID, instrumentID uint, wave string) (*model.LearningProfileAdministration, error) {
	admin, err := s.findAdministration(s.db, userID, instrumentID, wave)
	if err != nil {
		return nil, err
	}
	if admin == nil || admin.Status != model.LearningProfileStatusSubmitted || admin.SubmittedAt == nil {
		return nil, nil
	}
	return admin, nil
}

func assembleSdlProfileWave(
	wave string,
	dlInst, sdlInst *model.LearningProfileInstrument,
	dlAdmin, sdlAdmin *model.LearningProfileAdministration,
	scores []model.LearningProfileDimensionScore,
) *SdlProfileWave {
	if wave != learningprofile.WavePretest && wave != learningprofile.WavePosttest {
		return nil
	}
	if dlInst == nil || sdlInst == nil || dlAdmin == nil || sdlAdmin == nil {
		return nil
	}
	if dlAdmin.Status != model.LearningProfileStatusSubmitted || sdlAdmin.Status != model.LearningProfileStatusSubmitted {
		return nil
	}
	if dlAdmin.SubmittedAt == nil || sdlAdmin.SubmittedAt == nil {
		return nil
	}
	if dlAdmin.Program != model.LearningProfileProgramSelfAssessment || sdlAdmin.Program != model.LearningProfileProgramSelfAssessment {
		return nil
	}
	if dlAdmin.WaveType != wave || sdlAdmin.WaveType != wave {
		return nil
	}

	instByAdmin := map[uint]*model.LearningProfileInstrument{
		dlAdmin.ID:  dlInst,
		sdlAdmin.ID: sdlInst,
	}
	adminByID := map[uint]*model.LearningProfileAdministration{
		dlAdmin.ID:  dlAdmin,
		sdlAdmin.ID: sdlAdmin,
	}
	byCode := map[string]model.LearningProfileDimensionScore{}
	for _, row := range scores {
		if _, ok := instByAdmin[row.AdministrationID]; !ok {
			return nil
		}
		if _, dup := byCode[row.DimensionCode]; dup {
			return nil
		}
		byCode[row.DimensionCode] = row
	}
	if len(byCode) != len(learningprofile.RadarDimensionSpecs()) {
		return nil
	}

	dims := make([]SdlProfileDimension, 0, len(learningprofile.RadarDimensionSpecs()))
	for _, spec := range learningprofile.RadarDimensionSpecs() {
		row, ok := byCode[spec.Code]
		if !ok {
			return nil
		}
		inst := instByAdmin[row.AdministrationID]
		admin := adminByID[row.AdministrationID]
		if inst == nil || admin == nil || inst.Code != spec.InstrumentCode {
			return nil
		}
		if row.ItemCount != spec.ItemCount || row.ScaleMin != spec.ScaleMin || row.ScaleMax != spec.ScaleMax {
			return nil
		}
		if row.RawMean < float64(spec.ScaleMin) || row.RawMean > float64(spec.ScaleMax) {
			return nil
		}
		if !learningprofile.IsStoredDisplayScore(row.DisplayScore) {
			return nil
		}
		dims = append(dims, SdlProfileDimension{
			Code:              spec.Code,
			RawMean:           row.RawMean,
			ScaleMin:          row.ScaleMin,
			ScaleMax:          row.ScaleMax,
			DisplayScore:      row.DisplayScore,
			ItemCount:         row.ItemCount,
			InstrumentCode:    inst.Code,
			InstrumentVersion: inst.Version,
			WaveType:          wave,
			SubmittedAt:       admin.SubmittedAt,
		})
	}

	waveSubmitted := dlAdmin.SubmittedAt
	if sdlAdmin.SubmittedAt.After(*waveSubmitted) {
		waveSubmitted = sdlAdmin.SubmittedAt
	}
	return &SdlProfileWave{
		WaveType:    wave,
		SubmittedAt: waveSubmitted,
		Instruments: []SdlProfileInstrument{
			{Code: dlInst.Code, Version: dlInst.Version, WaveType: wave, SubmittedAt: dlAdmin.SubmittedAt},
			{Code: sdlInst.Code, Version: sdlInst.Version, WaveType: wave, SubmittedAt: sdlAdmin.SubmittedAt},
		},
		Dimensions: dims,
	}
}