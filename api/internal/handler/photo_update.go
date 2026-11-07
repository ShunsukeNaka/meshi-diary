package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

type updatePhotoRequest struct {
	Alt optional[string] `json:"alt"`
}

// UpdatePhoto は写真の説明を変える（docs/api.md の PATCH /diaries/{id}/photos/{photoId}）。Sessions.RequireSession で包んで使う。
func UpdatePhoto(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diary, ok := findOwnDiary(w, r, d.Diaries)
		if !ok {
			return
		}
		photoID, ok := parsePhotoID(w, r)
		if !ok {
			return
		}

		var req updatePhotoRequest
		if !decodeDiaryJSON(w, r, &req) {
			return
		}
		if !req.Alt.Set {
			writeValidationError(w, "alt を指定してください")
			return
		}
		var alt *string
		if !req.Alt.Null {
			alt = normalizeInfo(&req.Alt.Value)
		}

		photo, err := d.Photos.UpdateAlt(r.Context(), diary.ID, photoID, alt)
		if errors.Is(err, postgres.ErrNotFound) {
			writePhotoNotFound(w)
			return
		}
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}
		res, err := newPhotoResponses(r.Context(), d.Storage, []postgres.Photo{photo})
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]photoResponse{"photo": res[0]})
	}
}

// DeletePhoto は写真を1枚削除する（docs/api.md の DELETE /diaries/{id}/photos/{photoId}）。Sessions.RequireSession で包んで使う。
func DeletePhoto(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diary, ok := findOwnDiary(w, r, d.Diaries)
		if !ok {
			return
		}
		photoID, ok := parsePhotoID(w, r)
		if !ok {
			return
		}
		photo, err := d.Photos.Delete(r.Context(), diary.ID, photoID)
		if errors.Is(err, postgres.ErrNotFound) {
			writePhotoNotFound(w)
			return
		}
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}
		deletePhotoFile(r.Context(), d.Storage, photo.ObjectKey)
		w.WriteHeader(http.StatusNoContent)
	}
}

type reorderPhotosRequest struct {
	PhotoIDs *[]uuid.UUID `json:"photo_ids"`
}

// ReorderPhotos は写真の表示順を、送った id の順にする（docs/api.md の PUT /diaries/{id}/photos/order）。
// Sessions.RequireSession で包んで使う。
func ReorderPhotos(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diary, ok := findOwnDiary(w, r, d.Diaries)
		if !ok {
			return
		}
		var req reorderPhotosRequest
		if !decodeDiaryJSON(w, r, &req) {
			return
		}
		if req.PhotoIDs == nil {
			writeValidationError(w, "photo_ids を指定してください")
			return
		}

		err := d.Photos.Reorder(r.Context(), diary.ID, *req.PhotoIDs)
		switch {
		case errors.Is(err, postgres.ErrInvalidOrder):
			writeValidationError(w, "photo_ids には、日記の写真すべてを1回ずつ入れてください")
			return
		case errors.Is(err, postgres.ErrNotFound):
			writeDiaryNotFound(w)
			return
		case err != nil:
			httpx.WriteInternalError(w, err)
			return
		}

		photos, err := d.Photos.ListByDiary(r.Context(), diary.ID)
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}
		res, err := newPhotoResponses(r.Context(), d.Storage, photos)
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string][]photoResponse{"photos": res})
	}
}

// parsePhotoID はパスの {photoId} を読む。UUID でないときは 404 を書いて false を返す。
func parsePhotoID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("photoId"))
	if err != nil {
		writePhotoNotFound(w)
		return uuid.Nil, false
	}
	return id, true
}

func writePhotoNotFound(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "写真が見つかりません")
}
