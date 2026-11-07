package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

type calendarBody struct {
	Month string `json:"month"`
	Days  []struct {
		Date         string  `json:"date"`
		DiaryCount   int     `json:"diary_count"`
		ThumbnailURL *string `json:"thumbnail_url"`
	} `json:"days"`
}

func TestCalendar(t *testing.T) {
	userID := uuid.New()
	at := func(h int) time.Time { return testTime.Add(time.Duration(h) * time.Hour) }

	t.Run("returns days with diaries and their thumbnails", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		e.addDiaryOn(userID, "2026-10-31", postgres.MealTypeDinner, at(0))
		e.addDiaryOn(userID, "2026-11-02", postgres.MealTypeLunch, at(0))
		// 11/6: 先に作った日記には写真がなく、あとの日記に写真がある
		e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeLunch, at(1))
		withPhoto := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeDinner, at(2))
		photo := e.addPhoto(withPhoto.ID, 0)
		e.addPhoto(withPhoto.ID, 1)
		// 11/7: 2つとも写真があるときは、先に作った日記の写真
		first := e.addDiaryOn(userID, "2026-11-07", postgres.MealTypeDinner, at(3))
		second := e.addDiaryOn(userID, "2026-11-07", postgres.MealTypeBreakfast, at(4))
		firstPhoto := e.addPhoto(first.ID, 0)
		e.addPhoto(second.ID, 0)
		e.addDiaryOn(userID, "2026-12-01", postgres.MealTypeOther, at(0))
		e.addDiaryOn(uuid.New(), "2026-11-03", postgres.MealTypeOther, at(0))

		rec := serve(e.router, http.MethodGet, "/calendar?month=2026-11", e.cookie(t, userID))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		var body calendarBody
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Month != "2026-11" || len(body.Days) != 3 {
			t.Fatalf("body = %+v", body)
		}
		d2, d6, d7 := body.Days[0], body.Days[1], body.Days[2]
		if d2.Date != "2026-11-02" || d2.DiaryCount != 1 || d2.ThumbnailURL != nil {
			t.Fatalf("11/02 = %+v", d2)
		}
		if d6.Date != "2026-11-06" || d6.DiaryCount != 2 || d6.ThumbnailURL == nil || *d6.ThumbnailURL != "https://r2.test/"+photo.ObjectKey+"?expires=3600" {
			t.Fatalf("11/06 = %+v", d6)
		}
		if d7.Date != "2026-11-07" || d7.ThumbnailURL == nil || *d7.ThumbnailURL != "https://r2.test/"+firstPhoto.ObjectKey+"?expires=3600" {
			t.Fatalf("11/07 = %+v", d7)
		}
	})

	t.Run("month without diaries returns an empty days array", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		rec := serve(e.router, http.MethodGet, "/calendar?month=2026-02", e.cookie(t, userID))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if string(body["days"]) != "[]" {
			t.Fatalf("days = %s, want []", body["days"])
		}
	})

	t.Run("invalid month is 400", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		for _, q := range []string{"", "?month=", "?month=2026-13", "?month=2026-1", "?month=2026/11", "?month=2026-11-01"} {
			rec := serve(e.router, http.MethodGet, "/calendar"+q, e.cookie(t, userID))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("%q: status = %d, want 400", q, rec.Code)
			}
		}
	})

	t.Run("no cookie is 401", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		assertUnauthorized(t, serve(e.router, http.MethodGet, "/calendar?month=2026-11", nil))
	})

	t.Run("DB error is 500", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		e.diaries.err = errors.New("db down")
		if rec := serve(e.router, http.MethodGet, "/calendar?month=2026-11", e.cookie(t, userID)); rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}
