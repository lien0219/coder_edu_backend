package service

import (
	"testing"
	"time"

	"coder_edu_backend/internal/model"
)

func TestFillDailyChallengeRange(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 9, 15, 4, 0, 0, time.Local)
	today := time.Date(2026, 9, 9, 0, 0, 0, 0, time.Local)

	t.Run("pads missing days for 7", func(t *testing.T) {
		t.Parallel()
		result := fillDailyChallengeRange(now, 7, []model.ChallengeDailyData{
			{Date: "2026-09-09", CompletedChallenges: 3, AverageScore: 90},
			{Date: "2026-09-07", CompletedChallenges: 1, AverageScore: 70},
		})

		if len(result) != 7 {
			t.Fatalf("len=%d want 7", len(result))
		}
		if result[0].Date != today.AddDate(0, 0, -6).Format("2006-01-02") {
			t.Fatalf("first date=%s", result[0].Date)
		}
		if result[len(result)-1].Date != "2026-09-09" {
			t.Fatalf("last date=%s", result[len(result)-1].Date)
		}
		if result[4].CompletedChallenges != 1 || result[4].AverageScore != 70 {
			t.Fatalf("sept 7 = %+v", result[4])
		}
		if result[5].CompletedChallenges != 0 {
			t.Fatalf("sept 8 should be zero, got %+v", result[5])
		}
		if result[6].CompletedChallenges != 3 {
			t.Fatalf("today = %+v", result[6])
		}
	})

	for _, days := range []int{7, 14, 21} {
		days := days
		t.Run("length and order", func(t *testing.T) {
			t.Parallel()
			result := fillDailyChallengeRange(now, days, nil)
			if len(result) != days {
				t.Fatalf("len=%d want %d", len(result), days)
			}
			if result[0].Date != today.AddDate(0, 0, -(days-1)).Format("2006-01-02") {
				t.Fatalf("first date=%s", result[0].Date)
			}
			if result[len(result)-1].Date != "2026-09-09" {
				t.Fatalf("last date=%s", result[len(result)-1].Date)
			}
			for i := 1; i < len(result); i++ {
				if result[i].Date <= result[i-1].Date {
					t.Fatalf("dates are not ascending: %s then %s", result[i-1].Date, result[i].Date)
				}
				if result[i].CompletedChallenges != 0 || result[i].AverageScore != 0 {
					t.Fatalf("empty input should stay zero: %+v", result[i])
				}
			}
		})
	}
}
