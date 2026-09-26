package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"coder_edu_backend/internal/learningprofile"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

type stubLearningProfileService struct {
	lastUserID uint
	lastCode   string
	lastWave   string
	me         *service.LearningProfileMe
	paper      *service.LearningProfilePaper
	write      *service.LearningProfileWriteResult
	sdlProfile *service.SdlLearningProfile
	err        error
}

func (s *stubLearningProfileService) GetMe(userID uint) (*service.LearningProfileMe, error) {
	s.lastUserID = userID
	return s.me, s.err
}

func (s *stubLearningProfileService) GetPaper(userID uint, instrumentCode, wave string) (*service.LearningProfilePaper, error) {
	s.lastUserID = userID
	s.lastCode = instrumentCode
	s.lastWave = wave
	return s.paper, s.err
}

func (s *stubLearningProfileService) SaveDraft(userID uint, instrumentCode, wave string, answers map[string]int) (*service.LearningProfileWriteResult, error) {
	s.lastUserID = userID
	s.lastCode = instrumentCode
	s.lastWave = wave
	return s.write, s.err
}

func (s *stubLearningProfileService) Submit(userID uint, instrumentCode, wave string, answers map[string]int) (*service.LearningProfileWriteResult, error) {
	s.lastUserID = userID
	s.lastCode = instrumentCode
	s.lastWave = wave
	return s.write, s.err
}

func (s *stubLearningProfileService) GetSdlProfile(userID uint) (*service.SdlLearningProfile, error) {
	s.lastUserID = userID
	return s.sdlProfile, s.err
}

func newLearningProfileRouter(t *testing.T, stub *stubLearningProfileService, userID uint) *gin.Engine {
	t.Helper()
	return newLearningProfileRouterWithAPI(t, stub, userID)
}

func newLearningProfileRouterWithAPI(t *testing.T, api learningProfileAPI, userID uint) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID > 0 {
			c.Set("user", &util.Claims{UserID: userID, Role: model.Student})
		}
		c.Next()
	})
	ctrl := NewLearningProfileController(api)
	apiGroup := r.Group("/api")
	g := apiGroup.Group("/learning-profile")
	g.GET("/me", ctrl.GetMe)
	g.GET("/instruments/:code/:wave", ctrl.GetPaper)
	g.PUT("/instruments/:code/:wave/draft", ctrl.SaveDraft)
	g.POST("/instruments/:code/:wave/submit", ctrl.Submit)
	apiGroup.GET("/analytics/sdl-profile", ctrl.GetSdlProfile)
	return r
}

func TestLearningProfileMeUsesJWTUserNotQuery(t *testing.T) {
	stub := &stubLearningProfileService{me: &service.LearningProfileMe{Program: "platform_self_assessment"}}
	r := newLearningProfileRouter(t, stub, 41)
	req := httptest.NewRequest(http.MethodGet, "/api/learning-profile/me?userId=99&teacherId=2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if stub.lastUserID != 41 {
		t.Fatalf("used user %d", stub.lastUserID)
	}
}

func TestLearningProfileUnauthorizedWithoutJWT(t *testing.T) {
	stub := &stubLearningProfileService{}
	r := newLearningProfileRouter(t, stub, 0)
	req := httptest.NewRequest(http.MethodGet, "/api/learning-profile/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if stub.lastUserID != 0 {
		t.Fatal("service called without JWT")
	}
}

func TestLearningProfilePosttestReachesServiceAndCanBeBlocked(t *testing.T) {
	stub := &stubLearningProfileService{err: learningprofile.ErrPosttestBlocked}
	r := newLearningProfileRouter(t, stub, 41)
	paths := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/learning-profile/instruments/DL-C56-v1/posttest", ""},
		{http.MethodPut, "/api/learning-profile/instruments/SDL-C20-v1/posttest/draft", `{"answers":{"SDL1":4},"userId":99,"isOpen":true}`},
		{http.MethodPost, "/api/learning-profile/instruments/DL-C56-v1/posttest/submit", `{"answers":{},"submittedAt":"2020-01-01T00:00:00Z"}`},
	}
	for _, tc := range paths {
		stub.lastWave = ""
		stub.lastUserID = 0
		var req *http.Request
		if tc.body != "" {
			req = httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(tc.method, tc.path, nil)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s %s status=%d body=%s", tc.method, tc.path, w.Code, w.Body.String())
		}
		var body util.Response
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.ErrorCode != "POSTTEST_BLOCKED" {
			t.Fatalf("errorCode=%s", body.ErrorCode)
		}
		if stub.lastWave != "posttest" || stub.lastUserID != 41 {
			t.Fatalf("service not called with JWT posttest: %+v", stub)
		}
	}
}

func TestLearningProfileRejectsUnknownWave(t *testing.T) {
	stub := &stubLearningProfileService{}
	r := newLearningProfileRouter(t, stub, 41)
	req := httptest.NewRequest(http.MethodGet, "/api/learning-profile/instruments/DL-C56-v1/midterm", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if stub.lastWave != "" {
		t.Fatal("service called for invalid wave")
	}
}

func TestLearningProfilePosttestAlreadySubmittedConflict(t *testing.T) {
	stub := &stubLearningProfileService{err: learningprofile.ErrAlreadySubmitted}
	r := newLearningProfileRouter(t, stub, 41)
	req := httptest.NewRequest(http.MethodPost, "/api/learning-profile/instruments/SDL-C20-v1/posttest/submit", bytes.NewBufferString(`{"answers":{"SDL1":4}}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body util.Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ErrorCode != "ALREADY_SUBMITTED" {
		t.Fatalf("errorCode=%s", body.ErrorCode)
	}
}

func TestLearningProfileRejectsUnknownInstrument(t *testing.T) {
	stub := &stubLearningProfileService{}
	r := newLearningProfileRouter(t, stub, 41)
	req := httptest.NewRequest(http.MethodGet, "/api/learning-profile/instruments/OTHER/pretest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if stub.lastCode != "" {
		t.Fatal("service called for unknown instrument")
	}
}

func TestLearningProfileSubmitIncompleteDLReturns400MissingCT3(t *testing.T) {
	svc := service.NewLearningProfileService(nil)
	r := newLearningProfileRouterWithAPI(t, svc, 5)
	body := `{"answers":{"CT1":3,"CT2":4},"userId":99}`
	req := httptest.NewRequest(http.MethodPost, "/api/learning-profile/instruments/DL-C56-v1/pretest/submit", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp util.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("code=%d", resp.Code)
	}
	if resp.Message != "missing CT3" {
		t.Fatalf("message=%q", resp.Message)
	}
	if resp.ErrorCode != "" {
		t.Fatalf("errorCode=%s", resp.ErrorCode)
	}
}

func TestLearningProfileDraftIgnoresClientIdentityFields(t *testing.T) {
	stub := &stubLearningProfileService{write: &service.LearningProfileWriteResult{Status: "draft"}}
	r := newLearningProfileRouter(t, stub, 41)
	body := `{"answers":{"CT1":3},"userId":99,"waveType":"posttest","submittedAt":"2020-01-01T00:00:00Z","isOpen":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/learning-profile/instruments/DL-C56-v1/pretest/draft", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if stub.lastUserID != 41 || stub.lastWave != "pretest" || stub.lastCode != "DL-C56-v1" {
		t.Fatalf("identity %+v", stub)
	}
}

func TestLearningProfileSdlProfileUsesJWTUserNotQuery(t *testing.T) {
	stub := &stubLearningProfileService{sdlProfile: &service.SdlLearningProfile{Program: "platform_self_assessment"}}
	r := newLearningProfileRouter(t, stub, 41)
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/sdl-profile?userId=99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if stub.lastUserID != 41 {
		t.Fatalf("used user %d", stub.lastUserID)
	}
	var body util.Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(body.Data)
	if strings.Contains(string(raw), `"userId"`) || strings.Contains(string(raw), "item_snapshot") || strings.Contains(string(raw), "answers") {
		t.Fatalf("payload leaked protected fields: %s", raw)
	}
}

func TestLearningProfileSdlProfileUnauthorizedWithoutJWT(t *testing.T) {
	stub := &stubLearningProfileService{sdlProfile: &service.SdlLearningProfile{}}
	r := newLearningProfileRouter(t, stub, 0)
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/sdl-profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if stub.lastUserID != 0 {
		t.Fatal("service called without JWT")
	}
}

func TestLearningProfileSdlProfileJSONNullPosttestOmitsAnswers(t *testing.T) {
	submitted := time.Date(2026, 9, 23, 20, 16, 25, 0, time.Local)
	stub := &stubLearningProfileService{
		sdlProfile: &service.SdlLearningProfile{
			Program: "platform_self_assessment",
			Pretest: &service.SdlProfileWave{
				WaveType:    "pretest",
				SubmittedAt: &submitted,
				Instruments: []service.SdlProfileInstrument{
					{Code: "DL-C56-v1", Version: "v1", WaveType: "pretest", SubmittedAt: &submitted},
					{Code: "SDL-C20-v1", Version: "v1", WaveType: "pretest", SubmittedAt: &submitted},
				},
				Dimensions: []service.SdlProfileDimension{
					{
						Code: "critical_thinking_disposition", RawMean: 3, ScaleMin: 1, ScaleMax: 5,
						DisplayScore: 50, ItemCount: 26, InstrumentCode: "DL-C56-v1", InstrumentVersion: "v1",
						WaveType: "pretest", SubmittedAt: &submitted,
					},
				},
			},
			Posttest: nil,
		},
	}
	r := newLearningProfileRouter(t, stub, 5)
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/sdl-profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"posttest":null`) {
		t.Fatalf("posttest must be JSON null: %s", body)
	}
	if strings.Contains(body, `"userId"`) || strings.Contains(body, "item_snapshot") || strings.Contains(body, `"answers"`) {
		t.Fatalf("payload leaked protected fields: %s", body)
	}
	if !strings.Contains(body, "20:16:25") {
		t.Fatalf("submittedAt wall clock rewritten: %s", body)
	}
}
