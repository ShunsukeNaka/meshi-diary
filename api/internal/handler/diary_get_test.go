package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestGetDiary(t *testing.T) {
	userID := uuid.New()

	t.Run("returns the diary with photos in position order", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		p1 := e.addPhoto(d.ID, 1)
		p0 := e.addPhoto(d.ID, 0)
		alt := "替え玉は硬めで。"
		p0.Alt = &alt
		e.photos.add(p0)

		rec := serve(e.router, http.MethodGet, "/diaries/"+d.ID.String(), e.cookie(t, userID))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
		}
		var body struct {
			Diary struct {
				ID      string `json:"id"`
				EatenOn string `json:"eaten_on"`
				Status  string `json:"status"`
				Info    string `json:"info"`
				Body    string `json:"body"`
				Photos  []struct {
					ID       string  `json:"id"`
					URL      string  `json:"url"`
					Alt      *string `json:"alt"`
					Position int     `json:"position"`
				} `json:"photos"`
			} `json:"diary"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		got := body.Diary
		if got.ID != d.ID.String() || got.EatenOn != "2026-11-06" || got.Status != "LUNCH" || got.Info != "駅前のラーメン屋" || got.Body != "本文" {
			t.Fatalf("diary = %+v", got)
		}
		if len(got.Photos) != 2 || got.Photos[0].ID != p0.ID.String() || got.Photos[1].ID != p1.ID.String() {
			t.Fatalf("photos = %+v", got.Photos)
		}
		if got.Photos[0].Alt == nil || *got.Photos[0].Alt != alt || got.Photos[1].Alt != nil {
			t.Fatalf("alt = %v, %v", got.Photos[0].Alt, got.Photos[1].Alt)
		}
		if want := "https://r2.test/" + p0.ObjectKey + "?expires=3600"; got.Photos[0].URL != want {
			t.Fatalf("url = %q, want %q", got.Photos[0].URL, want)
		}
	})

	t.Run("photos is an empty array when there are no photos", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		rec := serve(e.router, http.MethodGet, "/diaries/"+d.ID.String(), e.cookie(t, userID))
		var body struct {
			Diary map[string]json.RawMessage `json:"diary"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := string(body.Diary["photos"]); got != "[]" {
			t.Fatalf("photos = %s, want []", got)
		}
	})

	t.Run("not found cases are 404", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		others := e.addDiary(uuid.New())
		for name, path := range map[string]string{
			"other user's diary": "/diaries/" + others.ID.String(),
			"unknown diary":      "/diaries/" + uuid.NewString(),
			"not a uuid":         "/diaries/abc",
		} {
			rec := serve(e.router, http.MethodGet, path, e.cookie(t, userID))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("%s: status = %d, want 404", name, rec.Code)
			}
			if code := errorCode(t, rec); code != "NOT_FOUND" {
				t.Fatalf("%s: code = %q", name, code)
			}
		}
	})

	t.Run("no cookie is 401", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		assertUnauthorized(t, serve(e.router, http.MethodGet, "/diaries/"+d.ID.String(), nil))
	})

	t.Run("DB error is 500", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		e.photos.err = errors.New("db down")
		if rec := serve(e.router, http.MethodGet, "/diaries/"+d.ID.String(), e.cookie(t, userID)); rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}
