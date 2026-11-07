package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestDeletePhoto(t *testing.T) {
	userID := uuid.New()

	t.Run("deletes the photo and its file and closes the gap", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		p0 := e.addPhoto(d.ID, 0)
		p1 := e.addPhoto(d.ID, 1)
		p2 := e.addPhoto(d.ID, 2)

		rec := serve(e.router, http.MethodDelete, "/diaries/"+d.ID.String()+"/photos/"+p1.ID.String(), e.cookie(t, userID))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		if _, ok := e.storage.objects[p1.ObjectKey]; ok {
			t.Fatal("file was not deleted")
		}
		got, _ := e.photos.ListByDiary(t.Context(), d.ID)
		if len(got) != 2 || got[0].ID != p0.ID || got[1].ID != p2.ID || got[1].Position != 1 {
			t.Fatalf("photos = %+v", got)
		}
	})

	t.Run("R2 failure still returns 204", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		p := e.addPhoto(d.ID, 0)
		e.storage.deleteErr = errors.New("r2 down")
		if rec := serve(e.router, http.MethodDelete, "/diaries/"+d.ID.String()+"/photos/"+p.ID.String(), e.cookie(t, userID)); rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
	})

	t.Run("not found cases are 404", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		otherPhoto := e.addPhoto(e.addDiary(userID).ID, 0)
		othersDiary := e.addDiary(uuid.New())
		othersPhoto := e.addPhoto(othersDiary.ID, 0)
		for name, path := range map[string]string{
			"photo of another diary": "/diaries/" + d.ID.String() + "/photos/" + otherPhoto.ID.String(),
			"other user's diary":     "/diaries/" + othersDiary.ID.String() + "/photos/" + othersPhoto.ID.String(),
			"unknown photo":          "/diaries/" + d.ID.String() + "/photos/" + uuid.NewString(),
			"not a uuid":             "/diaries/" + d.ID.String() + "/photos/abc",
		} {
			if rec := serve(e.router, http.MethodDelete, path, e.cookie(t, userID)); rec.Code != http.StatusNotFound {
				t.Fatalf("%s: status = %d, want 404", name, rec.Code)
			}
		}
		if len(e.storage.deleted) != 0 {
			t.Fatalf("files were deleted: %v", e.storage.deleted)
		}
	})

	t.Run("no cookie is 401", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		p := e.addPhoto(d.ID, 0)
		assertUnauthorized(t, serve(e.router, http.MethodDelete, "/diaries/"+d.ID.String()+"/photos/"+p.ID.String(), nil))
	})
}
