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

func TestListDiaries(t *testing.T) {
	userID := uuid.New()
	at := func(h int) time.Time { return testTime.Add(time.Duration(h) * time.Hour) }

	t.Run("returns the day's diaries in meal order with thumbnails", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		dinner := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeDinner, at(0))
		lunch := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeLunch, at(1))
		photo := e.addPhoto(lunch.ID, 0)
		e.addDiaryOn(userID, "2026-11-07", postgres.MealTypeBreakfast, at(0))
		e.addDiaryOn(uuid.New(), "2026-11-06", postgres.MealTypeBreakfast, at(0))

		rec := serve(e.router, http.MethodGet, "/diaries?date=2026-11-06", e.cookie(t, userID))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		var body struct {
			Date    string `json:"date"`
			Diaries []struct {
				ID           string  `json:"id"`
				Status       string  `json:"status"`
				Body         string  `json:"body"`
				ThumbnailURL *string `json:"thumbnail_url"`
			} `json:"diaries"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Date != "2026-11-06" || len(body.Diaries) != 2 {
			t.Fatalf("body = %+v", body)
		}
		l, d := body.Diaries[0], body.Diaries[1]
		if l.ID != lunch.ID.String() || l.ThumbnailURL == nil || *l.ThumbnailURL != "https://r2.test/"+photo.ObjectKey+"?expires=3600" {
			t.Fatalf("lunch = %+v", l)
		}
		if d.ID != dinner.ID.String() || d.ThumbnailURL != nil || d.Body != "本文" {
			t.Fatalf("dinner = %+v", d)
		}
	})

	t.Run("day without diaries returns an empty array", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		rec := serve(e.router, http.MethodGet, "/diaries?date=2026-11-06", e.cookie(t, userID))
		var body map[string]json.RawMessage
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if string(body["diaries"]) != "[]" {
			t.Fatalf("diaries = %s, want []", body["diaries"])
		}
	})

	t.Run("invalid date is 400", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		for _, q := range []string{"", "?date=", "?date=2026-11", "?date=2026-02-30", "?date=2026/11/06"} {
			if rec := serve(e.router, http.MethodGet, "/diaries"+q, e.cookie(t, userID)); rec.Code != http.StatusBadRequest {
				t.Fatalf("%q: status = %d, want 400", q, rec.Code)
			}
		}
	})

	t.Run("no cookie is 401", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		assertUnauthorized(t, serve(e.router, http.MethodGet, "/diaries?date=2026-11-06", nil))
	})

	t.Run("DB error is 500", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		e.photos.err = errors.New("db down")
		e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeLunch, at(0))
		if rec := serve(e.router, http.MethodGet, "/diaries?date=2026-11-06", e.cookie(t, userID)); rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}
