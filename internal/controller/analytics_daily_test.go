package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

func TestParseDailyChallengeDays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{name: "empty defaults to 7", raw: "", want: 7},
		{name: "days 7", raw: "7", want: 7},
		{name: "days 14", raw: "14", want: 14},
		{name: "days 21", raw: "21", want: 21},
		{name: "days 0", raw: "0", wantErr: true},
		{name: "days 8", raw: "8", wantErr: true},
		{name: "days 30", raw: "30", wantErr: true},
		{name: "days abc", raw: "abc", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseDailyChallengeDays(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got days=%d", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetDailyChallengeStatsRejectsInvalidDays(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", &util.Claims{UserID: 33, Role: model.Student})
		c.Next()
	})
	ctrl := NewAnalyticsController(nil)
	router.GET("/api/analytics/challenges/daily", ctrl.GetDailyChallengeStats)

	for _, query := range []string{"?days=0", "?days=8", "?days=30", "?days=abc"} {
		query := query
		t.Run(query, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/analytics/challenges/daily"+query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}

			var body util.Response
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v body=%s", err, w.Body.String())
			}
			if body.Code != http.StatusBadRequest {
				t.Fatalf("code=%d want 400", body.Code)
			}
			if body.Message != "days must be 7, 14, or 21" {
				t.Fatalf("message=%q", body.Message)
			}
		})
	}
}
