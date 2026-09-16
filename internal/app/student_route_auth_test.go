package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/controller"
	"coder_edu_backend/internal/middleware"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

func okHandler(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func injectClaims(role model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user", &util.Claims{UserID: 1, Role: role})
		c.Next()
	}
}

func mountStudentOnlyRoutes(r *gin.Engine, extras ...gin.HandlerFunc) {
	g := r.Group("/api")
	g.Use(extras...)
	g.Use(middleware.RoleMiddleware(model.Student))
	{
		g.GET("/assessments/questions", okHandler)
		g.POST("/assessments/submit", okHandler)
		g.GET("/assessments/result", okHandler)
		g.GET("/learning-path/student", okHandler)
		g.GET("/learning-path/levels/:level/materials", okHandler)
		g.POST("/learning-path/materials/:id/learning-time", okHandler)
		g.POST("/learning-path/materials/:id/complete", okHandler)
	}
}

func mountTeacherAssessmentRoutes(r *gin.Engine, extras ...gin.HandlerFunc) {
	g := r.Group("/api/teacher/assessments")
	g.Use(extras...)
	g.Use(controller.TeacherOrAdminMiddleware())
	g.GET("", okHandler)
	g.GET("/submissions", okHandler)
}

func TestStudentAssessmentRoutesAllowStudent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mountStudentOnlyRoutes(router, injectClaims(model.Student))

	req := httptest.NewRequest(http.MethodGet, "/api/assessments/questions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("student GET questions status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStudentAssessmentRoutesRejectTeacher(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mountStudentOnlyRoutes(router, injectClaims(model.Teacher))

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/assessments/questions", ""},
		{http.MethodPost, "/api/assessments/submit", `{"answers":{}}`},
		{http.MethodGet, "/api/assessments/result", ""},
		{http.MethodGet, "/api/learning-path/student", ""},
		{http.MethodGet, "/api/learning-path/levels/1/materials", ""},
		{http.MethodPost, "/api/learning-path/materials/1/learning-time", `{"duration":1}`},
		{http.MethodPost, "/api/learning-path/materials/1/complete", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestStudentAssessmentRoutesUnauthorizedWithoutLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-student-route-auth"}}
	mountStudentOnlyRoutes(router, middleware.AuthMiddleware(cfg))

	req := httptest.NewRequest(http.MethodGet, "/api/assessments/questions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestTeacherAssessmentRoutesStillAllowTeacher(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mountTeacherAssessmentRoutes(router, injectClaims(model.Teacher))

	for _, path := range []string{"/api/teacher/assessments", "/api/teacher/assessments/submissions"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("teacher management %s status=%d body=%s", path, w.Code, w.Body.String())
		}
	}
}
