package handler

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

// photoURLExpiry は写真の表示用 URL（署名付き URL）の有効期限。
const photoURLExpiry = time.Hour

// PhotoStorage はハンドラーが使う写真のファイルの操作。本番は r2.Client、テストでは偽物に差し替える。
type PhotoStorage interface {
	Put(ctx context.Context, key, contentType string, body io.Reader) error
	PresignGet(ctx context.Context, key string, expires time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

// PhotoRepository はハンドラーが使う photos の操作。本番は postgres.Photos、テストでは偽物に差し替える。
type PhotoRepository interface {
	// ListByDiary は日記の写真を表示順に返す。写真がないときは空のスライスを返す。
	ListByDiary(ctx context.Context, diaryID uuid.UUID) ([]postgres.Photo, error)
}

// photoResponse はレスポンスの写真（docs/api.md の photos[]）。
type photoResponse struct {
	ID       uuid.UUID `json:"id"`
	URL      string    `json:"url"`
	Alt      *string   `json:"alt"`
	Position int       `json:"position"`
}

// newPhotoResponses は写真ごとに表示用 URL を発行して、レスポンスの形にする。
func newPhotoResponses(ctx context.Context, storage PhotoStorage, photos []postgres.Photo) ([]photoResponse, error) {
	res := make([]photoResponse, 0, len(photos))
	for _, p := range photos {
		url, err := storage.PresignGet(ctx, p.ObjectKey, photoURLExpiry)
		if err != nil {
			return nil, err
		}
		res = append(res, photoResponse{ID: p.ID, URL: url, Alt: p.Alt, Position: p.Position})
	}
	return res, nil
}
