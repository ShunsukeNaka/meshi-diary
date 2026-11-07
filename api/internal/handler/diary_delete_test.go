package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestDeleteDiary(t *testing.T) {
	userID := uuid.New()

	t.Run("deletes the diary, its photos and the files", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		p0 := e.addPhoto(d.ID, 0)
		p1 := e.addPhoto(d.ID, 1)
		keep := e.addPhoto(e.addDiary(userID).ID, 0)

		rec := serve(e.router, http.MethodDelete, "/diaries/"+d.ID.String(), e.cookie(t, userID))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204 (%s)", rec.Code, rec.Body.String())
		}
		if _, ok := e.diaries.byID[d.ID]; ok {
			t.Fatal("diary was not deleted")
		}
		for _, p := range []string{p0.ObjectKey, p1.ObjectKey} {
			if _, ok := e.storage.objects[p]; ok {
				t.Fatalf("file %s was not deleted", p)
			}
		}
		if _, ok := e.storage.objects[keep.ObjectKey]; !ok {
			t.Fatal("another diary's file was deleted")
		}
		if rec := serve(e.router, http.MethodGet, "/diaries/"+d.ID.String(), e.cookie(t, userID)); rec.Code != http.StatusNotFound {
			t.Fatalf("GET after delete: status = %d, want 404", rec.Code)
		}
		if rec := serve(e.router, http.MethodDelete, "/diaries/"+d.ID.String(), e.cookie(t, userID)); rec.Code != http.StatusNotFound {
			t.Fatalf("delete again: status = %d, want 404", rec.Code)
		}
	})

	t.Run("R2 failure still returns 204", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		e.addPhoto(d.ID, 0)
		e.storage.deleteErr = errors.New("r2 down")
		if rec := serve(e.router, http.MethodDelete, "/diaries/"+d.ID.String(), e.cookie(t, userID)); rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if _, ok := e.diaries.byID[d.ID]; ok {
			t.Fatal("diary was not deleted")
		}
	})

	t.Run("other user's or unknown diary is 404", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		others := e.addDiary(uuid.New())
		for _, id := range []string{others.ID.String(), uuid.NewString(), "abc"} {
			if rec := serve(e.router, http.MethodDelete, "/diaries/"+id, e.cookie(t, userID)); rec.Code != http.StatusNotFound {
				t.Fatalf("%s: status = %d, want 404", id, rec.Code)
			}
		}
		if _, ok := e.diaries.byID[others.ID]; !ok {
			t.Fatal("other user's diary was deleted")
		}
	})

	t.Run("no cookie is 401", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		assertUnauthorized(t, serve(e.router, http.MethodDelete, "/diaries/"+d.ID.String(), nil))
	})
}
