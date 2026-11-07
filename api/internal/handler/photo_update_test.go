package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestUpdatePhoto(t *testing.T) {
	userID := uuid.New()

	t.Run("changes and clears the alt", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		p := e.addPhoto(d.ID, 0)
		path := "/diaries/" + d.ID.String() + "/photos/" + p.ID.String()

		rec := sendJSON(e.router, http.MethodPatch, path, `{"alt":"替え玉は硬めで。"}`, e.cookie(t, userID))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		var body struct {
			Photo struct {
				ID  string  `json:"id"`
				URL string  `json:"url"`
				Alt *string `json:"alt"`
			} `json:"photo"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Photo.ID != p.ID.String() || body.Photo.Alt == nil || *body.Photo.Alt != "替え玉は硬めで。" || body.Photo.URL == "" {
			t.Fatalf("photo = %+v", body.Photo)
		}

		for _, clear := range []string{`{"alt":null}`, `{"alt":""}`} {
			if rec := sendJSON(e.router, http.MethodPatch, path, clear, e.cookie(t, userID)); rec.Code != http.StatusOK {
				t.Fatalf("%s: status = %d", clear, rec.Code)
			}
			if e.photos.byID[p.ID].Alt != nil {
				t.Fatalf("%s: alt was not cleared", clear)
			}
		}
	})

	t.Run("missing alt or invalid json is 400", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		p := e.addPhoto(d.ID, 0)
		for _, body := range []string{`{}`, `{`, `{"alt":1}`} {
			if rec := sendJSON(e.router, http.MethodPatch, "/diaries/"+d.ID.String()+"/photos/"+p.ID.String(), body, e.cookie(t, userID)); rec.Code != http.StatusBadRequest {
				t.Fatalf("%s: status = %d, want 400", body, rec.Code)
			}
		}
	})

	t.Run("not found cases are 404", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		other := e.addDiary(userID)
		otherPhoto := e.addPhoto(other.ID, 0)
		othersDiary := e.addDiary(uuid.New())
		othersPhoto := e.addPhoto(othersDiary.ID, 0)
		for name, path := range map[string]string{
			"photo of another diary": "/diaries/" + d.ID.String() + "/photos/" + otherPhoto.ID.String(),
			"other user's diary":     "/diaries/" + othersDiary.ID.String() + "/photos/" + othersPhoto.ID.String(),
			"unknown photo":          "/diaries/" + d.ID.String() + "/photos/" + uuid.NewString(),
			"photo id is not a uuid": "/diaries/" + d.ID.String() + "/photos/abc",
		} {
			if rec := sendJSON(e.router, http.MethodPatch, path, `{"alt":"x"}`, e.cookie(t, userID)); rec.Code != http.StatusNotFound {
				t.Fatalf("%s: status = %d, want 404", name, rec.Code)
			}
		}
		if e.photos.byID[othersPhoto.ID].Alt != nil || e.photos.byID[otherPhoto.ID].Alt != nil {
			t.Fatal("photo was changed")
		}
	})

	t.Run("no cookie is 401", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		p := e.addPhoto(d.ID, 0)
		assertUnauthorized(t, sendJSON(e.router, http.MethodPatch, "/diaries/"+d.ID.String()+"/photos/"+p.ID.String(), `{"alt":"x"}`, nil))
	})
}
