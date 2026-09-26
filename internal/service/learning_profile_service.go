package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"coder_edu_backend/internal/learningprofile"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const learningProfileStatusNotStarted = "not_started"

type LearningProfileMe struct {
	Program             string                            `json:"program"`
	Instruments         []LearningProfileInstrumentStatus `json:"instruments"`
	PosttestOpenAt      *time.Time                        `json:"posttestOpenAt,omitempty"`
	PosttestOpen        bool                              `json:"posttestOpen"`
	PosttestInstruments []LearningProfileInstrumentStatus `json:"posttestInstruments"`
}

type LearningProfileInstrumentStatus struct {
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Version       string     `json:"version"`
	ItemCount     int        `json:"itemCount"`
	ScaleMin      int        `json:"scaleMin"`
	ScaleMax      int        `json:"scaleMax"`
	Status        string     `json:"status"`
	AnsweredCount int        `json:"answeredCount"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	SubmittedAt   *time.Time `json:"submittedAt,omitempty"`
}

type LearningProfilePaper struct {
	Code        string                     `json:"code"`
	Name        string                     `json:"name"`
	Version     string                     `json:"version"`
	WaveType    string                     `json:"waveType"`
	Program     string                     `json:"program"`
	Status      string                     `json:"status"`
	ScaleMin    int                        `json:"scaleMin"`
	ScaleMax    int                        `json:"scaleMax"`
	ItemCount   int                        `json:"itemCount"`
	StartedAt   *time.Time                 `json:"startedAt,omitempty"`
	SubmittedAt *time.Time                 `json:"submittedAt,omitempty"`
	Items       []LearningProfilePaperItem `json:"items"`
}

type LearningProfilePaperItem struct {
	ItemCode         string                         `json:"itemCode"`
	Stem             string                         `json:"stem"`
	PrimaryDimension string                         `json:"primaryDimension"`
	SortOrder        int                            `json:"sortOrder"`
	ScaleMin         int                            `json:"scaleMin"`
	ScaleMax         int                            `json:"scaleMax"`
	Options          []learningprofile.LikertOption `json:"options"`
	Value            *int                           `json:"value,omitempty"`
}

type LearningProfileWriteResult struct {
	Code          string     `json:"code"`
	WaveType      string     `json:"waveType"`
	Status        string     `json:"status"`
	AnsweredCount int        `json:"answeredCount"`
	ItemCount     int        `json:"itemCount"`
	SubmittedAt   *time.Time `json:"submittedAt,omitempty"`
}

type LearningProfileService struct {
	db             *gorm.DB
	now            func() time.Time
	pretestTimesFn func(userID uint) (dl, sdl *time.Time, err error)
}

func NewLearningProfileService(db *gorm.DB) *LearningProfileService {
	return &LearningProfileService{db: db, now: time.Now}
}

func (s *LearningProfileService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *LearningProfileService) GetMe(userID uint) (*LearningProfileMe, error) {
	if userID == 0 {
		return nil, util.ErrUnauthorized
	}
	if err := s.requireDB(); err != nil {
		return nil, err
	}

	out := &LearningProfileMe{
		Program:             model.LearningProfileProgramSelfAssessment,
		PosttestInstruments: []LearningProfileInstrumentStatus{},
	}
	var dlSubmitted, sdlSubmitted *time.Time
	for _, code := range []string{model.LearningProfileInstrumentDL, model.LearningProfileInstrumentSDL} {
		st, err := s.instrumentStatus(userID, code, learningprofile.WavePretest)
		if err != nil {
			return nil, err
		}
		out.Instruments = append(out.Instruments, st)
		if st.Status == model.LearningProfileStatusSubmitted && st.SubmittedAt != nil {
			if st.Code == model.LearningProfileInstrumentDL {
				dlSubmitted = st.SubmittedAt
			}
			if st.Code == model.LearningProfileInstrumentSDL {
				sdlSubmitted = st.SubmittedAt
			}
		}
		post, err := s.instrumentStatus(userID, code, learningprofile.WavePosttest)
		if err != nil {
			return nil, err
		}
		out.PosttestInstruments = append(out.PosttestInstruments, post)
	}
	out.PosttestOpenAt = learningprofile.PosttestOpenAt(dlSubmitted, sdlSubmitted)
	out.PosttestOpen = learningprofile.PosttestIsOpen(s.currentTime(), dlSubmitted, sdlSubmitted)
	return out, nil
}

func (s *LearningProfileService) GetPaper(userID uint, instrumentCode, wave string) (*LearningProfilePaper, error) {
	if err := learningProfileIdentity(userID, instrumentCode, wave); err != nil {
		return nil, err
	}
	if err := s.ensureWaveAllowed(s.db, userID, wave); err != nil {
		return nil, err
	}
	if err := s.requireDB(); err != nil {
		return nil, err
	}

	loaded, err := s.loadOfficialPaper(instrumentCode)
	if err != nil {
		return nil, err
	}

	admin, err := s.findAdministration(s.db, userID, loaded.instrument.ID, wave)
	if err != nil {
		return nil, err
	}

	status := learningProfileStatusNotStarted
	var startedAt, submittedAt *time.Time
	answers := map[string]int{}
	if admin != nil {
		status = learningprofile.AdministrationStatus(true, admin.Status)
		startedAt = admin.StartedAt
		submittedAt = admin.SubmittedAt
		if admin.Status == model.LearningProfileStatusSubmitted && len(admin.ItemSnapshot) > 0 {
			paper, err := paperFromSnapshot(admin)
			if err != nil {
				return nil, err
			}
			return paper, nil
		}
		stored, err := s.loadAnswers(s.db, admin.ID)
		if err != nil {
			return nil, err
		}
		answers = stored
	}

	return buildLivePaper(loaded, wave, status, startedAt, submittedAt, answers), nil
}

func (s *LearningProfileService) SaveDraft(userID uint, instrumentCode, wave string, answers map[string]int) (*LearningProfileWriteResult, error) {
	if err := learningProfileIdentity(userID, instrumentCode, wave); err != nil {
		return nil, err
	}
	if err := s.ensureWaveAllowed(s.db, userID, wave); err != nil {
		return nil, err
	}
	if answers == nil {
		answers = map[string]int{}
	}
	if err := learningprofile.ValidateDraftAnswers(instrumentCode, answers); err != nil {
		return nil, err
	}
	return s.writeWave(userID, instrumentCode, wave, answers, false)
}

func (s *LearningProfileService) Submit(userID uint, instrumentCode, wave string, answers map[string]int) (*LearningProfileWriteResult, error) {
	if err := learningProfileIdentity(userID, instrumentCode, wave); err != nil {
		return nil, err
	}
	if err := s.ensureWaveAllowed(s.db, userID, wave); err != nil {
		return nil, err
	}
	if answers == nil {
		answers = map[string]int{}
	}
	if err := learningprofile.ValidateOfficialAnswers(instrumentCode, answers); err != nil {
		return nil, err
	}
	return s.writeWave(userID, instrumentCode, wave, answers, true)
}

func (s *LearningProfileService) writeWave(userID uint, instrumentCode, wave string, answers map[string]int, submit bool) (*LearningProfileWriteResult, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	loaded, err := s.loadOfficialPaper(instrumentCode)
	if err != nil {
		return nil, err
	}

	var result *LearningProfileWriteResult
	err = s.db.Transaction(func(tx *gorm.DB) error {
		lockName := learningProfileLockName(userID, instrumentCode, wave)
		locked, lockErr := acquireNamedMySQLLock(tx, lockName)
		if lockErr != nil {
			return lockErr
		}
		defer releaseNamedMySQLLock(tx, lockName, locked)

		if err := s.ensureWaveAllowed(tx, userID, wave); err != nil {
			return err
		}

		admin, err := s.getOrCreateDraftAdmin(tx, userID, loaded.instrument.ID, loaded.instrument.Code, wave)
		if err != nil {
			return err
		}
		if admin.WaveType != wave {
			return fmt.Errorf("%w: administration wave %s", learningprofile.ErrCatalogMismatch, admin.WaveType)
		}
		if admin.Status == model.LearningProfileStatusSubmitted {
			return learningprofile.ErrAlreadySubmitted
		}
		if err := learningprofile.CanTransitionToSubmitted(admin.Status); err != nil && submit {
			return learningprofile.ErrAlreadySubmitted
		}

		itemByCode := map[string]model.LearningProfileItem{}
		for _, item := range loaded.items {
			itemByCode[item.ItemCode] = item
		}
		now := s.currentTime()
		for code, value := range answers {
			item, ok := itemByCode[code]
			if !ok {
				return fmt.Errorf("%w: unexpected item %s", learningprofile.ErrCatalogMismatch, code)
			}
			if err := upsertLearningProfileAnswer(tx, admin.ID, item, value, now); err != nil {
				return err
			}
		}

		if !submit {
			if err := tx.Model(admin).Updates(map[string]interface{}{
				"status":     model.LearningProfileStatusDraft,
				"updated_at": now,
			}).Error; err != nil {
				return err
			}
			stored, err := s.loadAnswers(tx, admin.ID)
			if err != nil {
				return err
			}
			result = &LearningProfileWriteResult{
				Code:          instrumentCode,
				WaveType:      wave,
				Status:        model.LearningProfileStatusDraft,
				AnsweredCount: len(stored),
				ItemCount:     loaded.instrument.ItemCount,
			}
			return nil
		}

		stored, err := s.loadAnswers(tx, admin.ID)
		if err != nil {
			return err
		}
		for code, value := range answers {
			stored[code] = value
		}
		if err := learningprofile.ValidateOfficialAnswers(instrumentCode, stored); err != nil {
			return err
		}

		spec := officialSpecFromModel(loaded.instrument)
		paper, err := learningprofile.FreezeShownPaper(spec, loaded.shown, stored)
		if err != nil {
			return err
		}
		snapshot, err := json.Marshal(paper)
		if err != nil {
			return err
		}
		scores, err := learningprofile.ScoreInstrument(instrumentCode, stored)
		if err != nil {
			return err
		}

		submittedAt := now
		if err := tx.Model(admin).Updates(map[string]interface{}{
			"status":        model.LearningProfileStatusSubmitted,
			"item_snapshot": snapshot,
			"submitted_at":  submittedAt,
			"paper_version": loaded.instrument.Code,
			"updated_at":    now,
		}).Error; err != nil {
			return err
		}

		for _, score := range scores {
			row := model.LearningProfileDimensionScore{
				AdministrationID: admin.ID,
				DimensionCode:    score.DimensionCode,
				RawMean:          score.RawMean,
				ScaleMin:         score.ScaleMin,
				ScaleMax:         score.ScaleMax,
				DisplayScore:     score.DisplayScore,
				ItemCount:        score.ItemCount,
				ComputedAt:       submittedAt,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		result = &LearningProfileWriteResult{
			Code:          instrumentCode,
			WaveType:      wave,
			Status:        model.LearningProfileStatusSubmitted,
			AnsweredCount: len(stored),
			ItemCount:     loaded.instrument.ItemCount,
			SubmittedAt:   &submittedAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

type officialPaper struct {
	instrument model.LearningProfileInstrument
	items      []model.LearningProfileItem
	shown      []learningprofile.ShownItem
}

func (s *LearningProfileService) loadOfficialPaper(instrumentCode string) (*officialPaper, error) {
	var inst model.LearningProfileInstrument
	err := s.db.Where("code = ? AND program = ? AND status = ?", instrumentCode, model.LearningProfileProgramSelfAssessment, "active").
		First(&inst).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, learningprofile.ErrCatalogUnavailable
	}
	if err != nil {
		return nil, err
	}
	if inst.ItemCount <= 0 {
		return nil, learningprofile.ErrCatalogMismatch
	}

	var items []model.LearningProfileItem
	if err := s.db.Where("instrument_id = ?", inst.ID).Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(items))
	shown := make([]learningprofile.ShownItem, 0, len(items))
	for _, item := range items {
		codes = append(codes, item.ItemCode)
		opts, err := parseLearningProfileOptions(item.OptionsJSON)
		if err != nil {
			return nil, err
		}
		shown = append(shown, learningprofile.ShownItem{
			Item:    itemSpecFromModel(instrumentCode, item),
			Options: opts,
		})
	}
	if err := learningprofile.MatchOfficialItemCodes(instrumentCode, codes); err != nil {
		return nil, err
	}
	if inst.ItemCount != len(items) {
		return nil, fmt.Errorf("%w: item_count %d rows %d", learningprofile.ErrCatalogMismatch, inst.ItemCount, len(items))
	}
	return &officialPaper{instrument: inst, items: items, shown: shown}, nil
}

func (s *LearningProfileService) instrumentStatus(userID uint, code, wave string) (LearningProfileInstrumentStatus, error) {
	loaded, err := s.loadOfficialPaper(code)
	if err != nil {
		return LearningProfileInstrumentStatus{}, err
	}
	st := LearningProfileInstrumentStatus{
		Code:      loaded.instrument.Code,
		Name:      loaded.instrument.Name,
		Version:   loaded.instrument.Version,
		ItemCount: loaded.instrument.ItemCount,
		ScaleMin:  loaded.instrument.ScaleMin,
		ScaleMax:  loaded.instrument.ScaleMax,
		Status:    learningProfileStatusNotStarted,
	}
	admin, err := s.findAdministration(s.db, userID, loaded.instrument.ID, wave)
	if err != nil {
		return LearningProfileInstrumentStatus{}, err
	}
	if admin == nil {
		return st, nil
	}
	st.Status = learningprofile.AdministrationStatus(true, admin.Status)
	st.StartedAt = admin.StartedAt
	st.SubmittedAt = admin.SubmittedAt
	answers, err := s.loadAnswers(s.db, admin.ID)
	if err != nil {
		return LearningProfileInstrumentStatus{}, err
	}
	st.AnsweredCount = len(answers)
	return st, nil
}

func (s *LearningProfileService) findAdministration(db *gorm.DB, userID, instrumentID uint, wave string) (*model.LearningProfileAdministration, error) {
	var admin model.LearningProfileAdministration
	err := db.Where(
		"user_id = ? AND instrument_id = ? AND wave_type = ? AND program = ?",
		userID, instrumentID, wave, model.LearningProfileProgramSelfAssessment,
	).First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

func (s *LearningProfileService) getOrCreateDraftAdmin(tx *gorm.DB, userID, instrumentID uint, paperVersion, wave string) (*model.LearningProfileAdministration, error) {
	if err := learningprofile.RequireWave(wave); err != nil {
		return nil, err
	}
	var admin model.LearningProfileAdministration
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(
		"user_id = ? AND instrument_id = ? AND wave_type = ? AND program = ?",
		userID, instrumentID, wave, model.LearningProfileProgramSelfAssessment,
	).First(&admin).Error
	if err == nil {
		return &admin, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	now := s.currentTime()
	admin = model.LearningProfileAdministration{
		UserID:       userID,
		InstrumentID: instrumentID,
		WaveType:     wave,
		Program:      model.LearningProfileProgramSelfAssessment,
		Status:       model.LearningProfileStatusDraft,
		PaperVersion: paperVersion,
		StartedAt:    &now,
	}
	if err := tx.Create(&admin).Error; err != nil {
		if isLearningProfileAdminConflict(err) {
			if reloadErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(
				"user_id = ? AND instrument_id = ? AND wave_type = ? AND program = ?",
				userID, instrumentID, wave, model.LearningProfileProgramSelfAssessment,
			).First(&admin).Error; reloadErr != nil {
				return nil, reloadErr
			}
			return &admin, nil
		}
		return nil, err
	}
	return &admin, nil
}

func (s *LearningProfileService) loadAnswers(db *gorm.DB, administrationID uint) (map[string]int, error) {
	var rows []model.LearningProfileAnswer
	if err := db.Where("administration_id = ?", administrationID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.ItemCode] = row.RawValue
	}
	return out, nil
}

func (s *LearningProfileService) requireDB() error {
	if s == nil || s.db == nil {
		return learningprofile.ErrCatalogUnavailable
	}
	return nil
}

func learningProfileIdentity(userID uint, instrumentCode, wave string) error {
	if userID == 0 {
		return util.ErrUnauthorized
	}
	if err := learningprofile.RequireOfficialInstrument(instrumentCode); err != nil {
		return err
	}
	return learningprofile.RequireWave(wave)
}

func (s *LearningProfileService) ensureWaveAllowed(db *gorm.DB, userID uint, wave string) error {
	if wave == learningprofile.WavePretest {
		return nil
	}
	if wave != learningprofile.WavePosttest {
		return learningprofile.ErrInvalidWave
	}
	return s.requirePosttestOpen(db, userID)
}

func (s *LearningProfileService) requirePosttestOpen(db *gorm.DB, userID uint) error {
	dl, sdl, err := s.loadPretestSubmittedAt(db, userID)
	if err != nil {
		return err
	}
	return learningprofile.CheckPosttestOpen(s.currentTime(), dl, sdl)
}

func (s *LearningProfileService) loadPretestSubmittedAt(db *gorm.DB, userID uint) (dl, sdl *time.Time, err error) {
	if s != nil && s.pretestTimesFn != nil {
		return s.pretestTimesFn(userID)
	}
	if db == nil {
		return nil, nil, learningprofile.ErrCatalogUnavailable
	}
	for _, code := range []string{model.LearningProfileInstrumentDL, model.LearningProfileInstrumentSDL} {
		inst, findErr := s.findActiveInstrument(code)
		if findErr != nil {
			return nil, nil, findErr
		}
		if inst == nil {
			return nil, nil, nil
		}
		admin, findErr := s.findAdministration(db, userID, inst.ID, learningprofile.WavePretest)
		if findErr != nil {
			return nil, nil, findErr
		}
		if admin == nil || admin.Status != model.LearningProfileStatusSubmitted || admin.SubmittedAt == nil {
			continue
		}
		if code == model.LearningProfileInstrumentDL {
			dl = admin.SubmittedAt
		} else {
			sdl = admin.SubmittedAt
		}
	}
	return dl, sdl, nil
}

func learningProfileLockName(userID uint, instrumentCode, wave string) string {
	return fmt.Sprintf("lp-%d-%s-%s", userID, instrumentCode, wave)
}

func upsertLearningProfileAnswer(tx *gorm.DB, administrationID uint, item model.LearningProfileItem, value int, at time.Time) error {
	var existing model.LearningProfileAnswer
	err := tx.Where("administration_id = ? AND item_id = ?", administrationID, item.ID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row := model.LearningProfileAnswer{
			AdministrationID: administrationID,
			ItemID:           item.ID,
			ItemCode:         item.ItemCode,
			RawValue:         value,
			AnsweredAt:       at,
		}
		if err := tx.Create(&row).Error; err != nil {
			if isLearningProfileAnswerConflict(err) {
				return tx.Model(&model.LearningProfileAnswer{}).
					Where("administration_id = ? AND item_id = ?", administrationID, item.ID).
					Updates(map[string]interface{}{
						"item_code":   item.ItemCode,
						"raw_value":   value,
						"answered_at": at,
					}).Error
			}
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	return tx.Model(&existing).Updates(map[string]interface{}{
		"item_code":   item.ItemCode,
		"raw_value":   value,
		"answered_at": at,
	}).Error
}

func isLearningProfileAdminConflict(err error) bool {
	return isLearningProfileUniqueConflict(err, "uk_lp_admin_user_inst_wave_program")
}

func isLearningProfileAnswerConflict(err error) bool {
	return isLearningProfileUniqueConflict(err, "uk_lp_answers_admin_item")
}

func isLearningProfileUniqueConflict(err error, indexName string) bool {
	if err == nil || indexName == "" {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062 && strings.Contains(mysqlErr.Message, indexName)
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, strings.ToLower(indexName)) &&
		(strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique"))
}

func parseLearningProfileOptions(raw json.RawMessage) ([]learningprofile.LikertOption, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty options")
	}
	var opts []learningprofile.LikertOption
	if err := json.Unmarshal(raw, &opts); err != nil {
		return nil, err
	}
	if len(opts) == 0 {
		return nil, fmt.Errorf("empty options")
	}
	return opts, nil
}

func itemSpecFromModel(instrumentCode string, item model.LearningProfileItem) learningprofile.ItemSpec {
	return learningprofile.ItemSpec{
		InstrumentCode:     instrumentCode,
		ItemCode:           item.ItemCode,
		SourceItemNo:       item.SourceItemNo,
		Stem:               item.Stem,
		PrimaryDimension:   item.PrimaryDimension,
		SecondaryDimension: item.SecondaryDimension,
		ScaleMin:           item.ScaleMin,
		ScaleMax:           item.ScaleMax,
		ScoringDirection:   item.ScoringDirection,
		SortOrder:          item.SortOrder,
	}
}

func officialSpecFromModel(inst model.LearningProfileInstrument) learningprofile.InstrumentSpec {
	spec, _ := learningprofile.InstrumentByCode(inst.Code)
	spec.Code = inst.Code
	spec.Name = inst.Name
	spec.Version = inst.Version
	spec.Program = inst.Program
	spec.ScaleMin = inst.ScaleMin
	spec.ScaleMax = inst.ScaleMax
	spec.ItemCount = inst.ItemCount
	return spec
}

func buildLivePaper(loaded *officialPaper, wave, status string, startedAt, submittedAt *time.Time, answers map[string]int) *LearningProfilePaper {
	paper := &LearningProfilePaper{
		Code:        loaded.instrument.Code,
		Name:        loaded.instrument.Name,
		Version:     loaded.instrument.Version,
		WaveType:    wave,
		Program:     loaded.instrument.Program,
		Status:      status,
		ScaleMin:    loaded.instrument.ScaleMin,
		ScaleMax:    loaded.instrument.ScaleMax,
		ItemCount:   loaded.instrument.ItemCount,
		StartedAt:   startedAt,
		SubmittedAt: submittedAt,
	}
	for _, shown := range loaded.shown {
		item := LearningProfilePaperItem{
			ItemCode:         shown.Item.ItemCode,
			Stem:             shown.Item.Stem,
			PrimaryDimension: shown.Item.PrimaryDimension,
			SortOrder:        shown.Item.SortOrder,
			ScaleMin:         shown.Item.ScaleMin,
			ScaleMax:         shown.Item.ScaleMax,
			Options:          shown.Options,
		}
		if val, ok := answers[shown.Item.ItemCode]; ok {
			v := val
			item.Value = &v
		}
		paper.Items = append(paper.Items, item)
	}
	return paper
}

func paperFromSnapshot(admin *model.LearningProfileAdministration) (*LearningProfilePaper, error) {
	var frozen learningprofile.FrozenPaper
	if err := json.Unmarshal(admin.ItemSnapshot, &frozen); err != nil {
		return nil, err
	}
	paper := &LearningProfilePaper{
		Code:        frozen.InstrumentCode,
		Name:        frozen.InstrumentName,
		Version:     frozen.InstrumentVersion,
		WaveType:    admin.WaveType,
		Program:     frozen.Program,
		Status:      model.LearningProfileStatusSubmitted,
		ScaleMin:    frozen.ScaleMin,
		ScaleMax:    frozen.ScaleMax,
		ItemCount:   len(frozen.Items),
		StartedAt:   admin.StartedAt,
		SubmittedAt: admin.SubmittedAt,
	}
	for _, item := range frozen.Items {
		val := item.RawValue
		paper.Items = append(paper.Items, LearningProfilePaperItem{
			ItemCode:         item.ItemCode,
			Stem:             item.Stem,
			PrimaryDimension: item.PrimaryDimension,
			SortOrder:        item.SortOrder,
			ScaleMin:         item.ScaleMin,
			ScaleMax:         item.ScaleMax,
			Options:          item.Options,
			Value:            &val,
		})
	}
	return paper, nil
}
