package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/middleware"
	"coder_edu_backend/internal/model"

	"github.com/gin-gonic/gin"
)

func mountTeacherManagementRoutes(r *gin.Engine, extras ...gin.HandlerFunc) {
	g := r.Group("/api")
	g.Use(extras...)
	teacher := g.Group("/teacher")
	teacher.Use(middleware.RoleMiddleware(model.Teacher, model.Admin))
	{
		teacher.POST("/levels", okHandler)
		teacher.PUT("/levels/:id", okHandler)
		teacher.DELETE("/levels/:id", okHandler)
		teacher.POST("/levels/:id/publish", okHandler)
		teacher.POST("/tasks/weekly", okHandler)
		teacher.PUT("/assessments/questions/:id", okHandler)
		teacher.POST("/assessments/submissions/:id/grade", okHandler)
		teacher.POST("/levels/:id/attempts/:attemptId/grade", okHandler)
		teacher.POST("/knowledge-points/reward", okHandler)
		teacher.POST("/post-class-tests/submissions/reset", okHandler)
		teacher.GET("/students/progress", okHandler)
		teacher.GET("/tasks/weekly/current", okHandler)
	}
}

func mountSharedCurrentWeekTaskRoute(r *gin.Engine, extras ...gin.HandlerFunc) {
	g := r.Group("/api")
	g.Use(extras...)
	g.GET("/tasks/weekly/current", okHandler)
	g.GET("/levels/student", okHandler)
	g.POST("/levels/:id/attempts/start", okHandler)
	g.GET("/learning/pre-class", okHandler)
	g.GET("/learning/in-class", okHandler)
	g.GET("/learning/post-class", okHandler)
	g.GET("/knowledge-points/student", okHandler)
	g.GET("/student/post-class-tests/published", okHandler)
	g.GET("/c-programming/resources", okHandler)
}

func TestTeacherManagementRoutesRejectStudent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mountTeacherManagementRoutes(router, injectClaims(model.Student))

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/teacher/levels", `{"title":"x"}`},
		{http.MethodPut, "/api/teacher/levels/1", `{"title":"x"}`},
		{http.MethodDelete, "/api/teacher/levels/1", ""},
		{http.MethodPost, "/api/teacher/levels/1/publish", ""},
		{http.MethodPost, "/api/teacher/tasks/weekly", `{"resourceModuleId":1}`},
		{http.MethodPut, "/api/teacher/assessments/questions/1", `{"title":"x"}`},
		{http.MethodPost, "/api/teacher/assessments/submissions/1/grade", `{"score":1}`},
		{http.MethodPost, "/api/teacher/levels/1/attempts/1/grade", `{"score":1}`},
		{http.MethodPost, "/api/teacher/knowledge-points/reward", `{"userIds":[1]}`},
		{http.MethodPost, "/api/teacher/post-class-tests/submissions/reset", `{"userId":1}`},
		{http.MethodGet, "/api/teacher/students/progress", ""},
		{http.MethodGet, "/api/teacher/tasks/weekly/current", ""},
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

func TestTeacherManagementRoutesAllowTeacher(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mountTeacherManagementRoutes(router, injectClaims(model.Teacher))

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/teacher/levels", `{"title":"x"}`},
		{http.MethodPost, "/api/teacher/levels/1/publish", ""},
		{http.MethodPost, "/api/teacher/tasks/weekly", `{"resourceModuleId":1}`},
		{http.MethodGet, "/api/teacher/tasks/weekly/current", ""},
		{http.MethodPut, "/api/teacher/assessments/questions/1", `{"title":"x"}`},
		{http.MethodPost, "/api/teacher/assessments/submissions/1/grade", `{"score":1}`},
		{http.MethodGet, "/api/teacher/students/progress", ""},
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
			if w.Code != http.StatusNoContent {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestTeacherManagementRoutesUnauthorizedWithoutLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-teacher-route-auth"}}
	mountTeacherManagementRoutes(router, middleware.AuthMiddleware(cfg))

	req := httptest.NewRequest(http.MethodGet, "/api/teacher/students/progress", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated teacher route status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestCurrentWeekTaskRouteAllowsStudentAndTeacher(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, role := range []model.UserRole{model.Student, model.Teacher, model.Admin} {
		router := gin.New()
		mountSharedCurrentWeekTaskRoute(router, injectClaims(role))
		req := httptest.NewRequest(http.MethodGet, "/api/tasks/weekly/current", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("role %s GET current week status=%d body=%s", role, w.Code, w.Body.String())
		}
	}
}

func TestCurrentWeekTaskRouteUnauthorizedWithoutLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-current-week-auth"}}
	mountSharedCurrentWeekTaskRoute(router, middleware.AuthMiddleware(cfg))

	for _, path := range []string{
		"/api/tasks/weekly/current",
		"/api/levels/student",
		"/api/learning/pre-class",
		"/api/knowledge-points/student",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated %s status=%d body=%s", path, w.Code, w.Body.String())
		}
	}
}

func TestStudentLearningRoutesRemainAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mountStudentOnlyRoutes(router, injectClaims(model.Student))
	mountSharedCurrentWeekTaskRoute(router, injectClaims(model.Student))

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/tasks/weekly/current"},
		{http.MethodGet, "/api/levels/student"},
		{http.MethodPost, "/api/levels/1/attempts/start"},
		{http.MethodGet, "/api/learning/pre-class"},
		{http.MethodGet, "/api/learning/in-class"},
		{http.MethodGet, "/api/learning/post-class"},
		{http.MethodGet, "/api/knowledge-points/student"},
		{http.MethodGet, "/api/student/post-class-tests/published"},
		{http.MethodGet, "/api/c-programming/resources"},
		{http.MethodGet, "/api/assessments/questions"},
		{http.MethodGet, "/api/learning-path/student"},
		{http.MethodGet, "/api/learning-profile/me"},
		{http.MethodGet, "/api/analytics/sdl-profile"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusNoContent {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
