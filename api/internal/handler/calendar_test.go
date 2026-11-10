package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

type calendarBody struct {
	Month string `json:"month"`
	Days  []struct {
		Date    string `json:"date"`
		Diaries []struct {
			ID           uuid.UUID         `json:"id"`
			Info         *string           `json:"info"`
			BodyHead     string            `json:"body_head"`
			Status       postgres.MealType `json:"status"`
			ThumbnailURL *string           `json:"thumbnail_url"`
		} `json:"diaries"`
	} `json:"days"`
}

// getCalendar は GET /calendar?month= を呼び、200 であることを確かめてレスポンスを返す。
func getCalendar(t *testing.T, e *diaryTestEnv, userID uuid.UUID, month string) calendarBody {
	t.Helper()
	rec := serve(e.router, http.MethodGet, "/calendar?month="+month, e.cookie(t, userID))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	var body calendarBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestCalendar(t *testing.T) {
	userID := uuid.New()
	at := func(h int) time.Time { return testTime.Add(time.Duration(h) * time.Hour) }

	t.Run("returns days of the month with their diaries", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		e.addDiaryOn(userID, "2026-10-31", postgres.MealTypeDinner, at(0))
		nov2 := e.addDiaryOn(userID, "2026-11-02", postgres.MealTypeLunch, at(0))
		noPhoto := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeLunch, at(1))
		withPhoto := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeDinner, at(2))
		photo := e.addPhoto(withPhoto.ID, 0)
		e.addPhoto(withPhoto.ID, 1)
		e.addDiaryOn(userID, "2026-12-01", postgres.MealTypeOther, at(0))
		e.addDiaryOn(uuid.New(), "2026-11-03", postgres.MealTypeOther, at(0))

		body := getCalendar(t, e, userID, "2026-11")
		if body.Month != "2026-11" || len(body.Days) != 2 {
			t.Fatalf("body = %+v", body)
		}
		d2, d6 := body.Days[0], body.Days[1]
		if d2.Date != "2026-11-02" || len(d2.Diaries) != 1 || d2.Diaries[0].ID != nov2.ID || d2.Diaries[0].ThumbnailURL != nil {
			t.Fatalf("11/02 = %+v", d2)
		}
		if d6.Date != "2026-11-06" || len(d6.Diaries) != 2 || d6.Diaries[0].ID != noPhoto.ID || d6.Diaries[1].ID != withPhoto.ID {
			t.Fatalf("11/06 = %+v", d6)
		}
		// サムネイルは日記の1枚目の写真のサムネイル
		if d6.Diaries[0].ThumbnailURL != nil || d6.Diaries[1].ThumbnailURL == nil || *d6.Diaries[1].ThumbnailURL != "https://r2.test/"+*photo.ThumbnailKey+"?expires=3600" {
			t.Fatalf("11/06 thumbnails = %+v", d6.Diaries)
		}
	})

	t.Run("photo without a thumbnail uses the photo itself", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeLunch, at(0))
		photo := e.addPhotoWithoutThumbnail(d.ID, 0)

		body := getCalendar(t, e, userID, "2026-11")
		if got := body.Days[0].Diaries[0].ThumbnailURL; got == nil || *got != "https://r2.test/"+photo.ObjectKey+"?expires=3600" {
			t.Fatalf("thumbnail = %v", got)
		}
	})

	t.Run("returns diaries of each day in meal order", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		dinner := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeDinner, at(0))
		lunch1 := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeLunch, at(1))
		breakfast := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeBreakfast, at(2))
		lunch2 := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeLunch, at(3))
		lunch2.Info = nil
		e.diaries.add(lunch2)
		photo := e.addPhoto(lunch1.ID, 0)

		body := getCalendar(t, e, userID, "2026-11")
		if len(body.Days) != 1 {
			t.Fatalf("days = %+v", body.Days)
		}
		got := body.Days[0].Diaries
		want := []uuid.UUID{breakfast.ID, lunch1.ID, lunch2.ID, dinner.ID}
		if len(got) != len(want) {
			t.Fatalf("diaries = %+v", got)
		}
		for i := range want {
			if got[i].ID != want[i] {
				t.Fatalf("diaries[%d] = %v, want %v", i, got[i].ID, want[i])
			}
		}
		l1 := got[1]
		if l1.Status != postgres.MealTypeLunch || l1.Info == nil || *l1.Info != "駅前のラーメン屋" || l1.BodyHead != "本文" ||
			l1.ThumbnailURL == nil || *l1.ThumbnailURL != "https://r2.test/"+*photo.ThumbnailKey+"?expires=3600" {
			t.Fatalf("lunch1 = %+v", l1)
		}
		if got[2].Info != nil || got[2].ThumbnailURL != nil {
			t.Fatalf("lunch2 = %+v", got[2])
		}
	})

	t.Run("body_head is the first 30 characters of the body", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		long := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeBreakfast, at(0))
		long.Body = "研究室のみんなで新しくできたお店へ。濃厚な豚骨スープで、替え玉までしてしまった。"
		e.diaries.add(long)
		exact := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeLunch, at(1))
		exact.Body = strings.Repeat("あ", 30)
		e.diaries.add(exact)
		empty := e.addDiaryOn(userID, "2026-11-06", postgres.MealTypeDinner, at(2))
		empty.Body = ""
		e.diaries.add(empty)

		got := getCalendar(t, e, userID, "2026-11").Days[0].Diaries
		if got[0].BodyHead != "研究室のみんなで新しくできたお店へ。濃厚な豚骨スープで、替え" {
			t.Errorf("long = %q", got[0].BodyHead)
		}
		if got[1].BodyHead != strings.Repeat("あ", 30) {
			t.Errorf("exact = %q", got[1].BodyHead)
		}
		if got[2].BodyHead != "" {
			t.Errorf("empty = %q", got[2].BodyHead)
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
