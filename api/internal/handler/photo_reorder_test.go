package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func idsJSON(ids ...uuid.UUID) string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = `"` + id.String() + `"`
	}
	return `{"photo_ids":[` + strings.Join(s, ",") + `]}`
}

func TestReorderPhotos(t *testing.T) {
	userID := uuid.New()

	t.Run("reorders and returns all photos", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		p0, p1, p2 := e.addPhoto(d.ID, 0), e.addPhoto(d.ID, 1), e.addPhoto(d.ID, 2)
		rec := sendJSON(e.router, http.MethodPut, "/diaries/"+d.ID.String()+"/photos/order", idsJSON(p2.ID, p0.ID, p1.ID), e.cookie(t, userID))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		var body photosBody
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Photos) != 3 || body.Photos[0].ID != p2.ID.String() || body.Photos[1].ID != p0.ID.String() || body.Photos[2].Position != 2 {
			t.Fatalf("photos = %+v", body.Photos)
		}
	})

	t.Run("empty list for a diary without photos", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		if rec := sendJSON(e.router, http.MethodPut, "/diaries/"+d.ID.String()+"/photos/order", `{"photo_ids":[]}`, e.cookie(t, userID)); rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("invalid ids are 400 and nothing changes", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		p0, p1 := e.addPhoto(d.ID, 0), e.addPhoto(d.ID, 1)
		other := e.addPhoto(e.addDiary(userID).ID, 0)
		for name, body := range map[string]string{
			"no photo_ids":    `{}`,
			"null":            `{"photo_ids":null}`,
			"not json":        `{`,
			"not uuid":        `{"photo_ids":["abc"]}`,
			"missing":         idsJSON(p1.ID),
			"duplicate":       idsJSON(p1.ID, p1.ID),
			"another diary's": idsJSON(p1.ID, other.ID),
			"extra":           idsJSON(p1.ID, p0.ID, uuid.New()),
		} {
			rec := sendJSON(e.router, http.MethodPut, "/diaries/"+d.ID.String()+"/photos/order", body, e.cookie(t, userID))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("%s: status = %d, want 400", name, rec.Code)
			}
		}
		if e.photos.byID[p0.ID].Position != 0 || e.photos.byID[p1.ID].Position != 1 {
			t.Fatal("order changed")
		}
	})

	t.Run("other user's diary is 404", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(uuid.New())
		p := e.addPhoto(d.ID, 0)
		if rec := sendJSON(e.router, http.MethodPut, "/diaries/"+d.ID.String()+"/photos/order", idsJSON(p.ID), e.cookie(t, userID)); rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("no cookie is 401", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		assertUnauthorized(t, sendJSON(e.router, http.MethodPut, "/diaries/"+d.ID.String()+"/photos/order", `{"photo_ids":[]}`, nil))
	})
}
