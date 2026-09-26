package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

func TestTeacherAssessmentEndpointsRejectStudent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", &util.Claims{UserID: 9, Role: model.Student})
		c.Next()
	})
	ctrl := NewAssessmentController(nil)
	group := router.Group("/teacher/assessments")
	group.Use(ctrl.TeacherOrAdmin())
	{
		group.POST("", ctrl.CreateAssessment)
		group.GET("", ctrl.ListAssessments)
		group.POST("/questions", ctrl.CreateQuestion)
		group.GET("/questions", ctrl.ListQuestions)
		group.GET("/questions/:id", ctrl.GetQuestion)
		group.PUT("/questions/:id", ctrl.UpdateQuestion)
		group.DELETE("/questions/:id", ctrl.DeleteQuestion)
		group.GET("/submissions", ctrl.ListSubmissions)
		group.GET("/submissions/:id", ctrl.GetSubmissionDetail)
		group.POST("/submissions/:id/grade", ctrl.GradeSubmission)
		group.DELETE("/submissions/:id", ctrl.DeleteSubmission)
		group.POST("/retest", ctrl.SetUserRetest)
		group.POST("/:id/publish", ctrl.PublishAssessment)
		group.GET("/:id", ctrl.GetAssessment)
	}

	type tc struct {
		method string
		path   string
		body   string
	}
	cases := []tc{
		{http.MethodPost, "/teacher/assessments", `{"title":"x"}`},
		{http.MethodGet, "/teacher/assessments", ""},
		{http.MethodPost, "/teacher/assessments/questions", `{"questionType":"essay","content":"c"}`},
		{http.MethodGet, "/teacher/assessments/questions", ""},
		{http.MethodGet, "/teacher/assessments/questions/1", ""},
		{http.MethodPut, "/teacher/assessments/questions/1", `{"questionType":"essay","content":"c"}`},
		{http.MethodDelete, "/teacher/assessments/questions/1", ""},
		{http.MethodGet, "/teacher/assessments/submissions", ""},
		{http.MethodGet, "/teacher/assessments/submissions/1", ""},
		{http.MethodPost, "/teacher/assessments/submissions/1/grade", `{"score":1,"recommendedLevel":2}`},
		{http.MethodDelete, "/teacher/assessments/submissions/1", ""},
		{http.MethodPost, "/teacher/assessments/retest", `{"userIds":[1],"canTake":true}`},
		{http.MethodPost, "/teacher/assessments/3/publish", `{"isPublished":true}`},
		{http.MethodGet, "/teacher/assessments/3", ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			var req *http.Request
			if c.body != "" {
				req = httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(c.method, c.path, nil)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestTeacherAssessmentEndpointsAllowTeacherPastAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", &util.Claims{UserID: 2, Role: model.Teacher})
		c.Next()
	})
	ctrl := NewAssessmentController(nil)
	group := router.Group("/teacher/assessments")
	group.Use(ctrl.TeacherOrAdmin())
	group.POST("/3/publish", func(ctx *gin.Context) {
		if !requireTeacherOrAdmin(ctx) {
			return
		}
		ctx.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/teacher/assessments/3/publish", strings.NewReader(`{"isPublished":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("teacher should pass middleware, status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAssessmentConflictErrorCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		err  error
		code string
	}{
		{util.ErrAssessmentPaperStale, util.ErrCodeAssessmentPaperStale},
		{util.ErrAssessmentIdempotencyConflict, util.ErrCodeAssessmentIdempotencyConflict},
		{util.ErrAssessmentPaperUnavailable, util.ErrCodeAssessmentPaperUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			respondAssessmentError(ctx, tc.err)
			if w.Code != http.StatusConflict {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			var body util.Response
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.ErrorCode != tc.code {
				t.Fatalf("errorCode=%q want %q body=%s", body.ErrorCode, tc.code, w.Body.String())
			}
		})
	}
}

func TestAssessmentRetestDeniedIsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	respondAssessmentError(ctx, util.ErrAssessmentRetestDenied)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
