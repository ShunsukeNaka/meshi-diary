package handler

import (
	"context"
	"io"
	"log/slog"
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
	// FirstOfDiaries は日記ごとの1枚目の写真を、日記の id をキーにして返す。写真がない日記はキーに入らない。
	FirstOfDiaries(ctx context.Context, diaryIDs []uuid.UUID) (map[uuid.UUID]postgres.Photo, error)
	// AddMany は今ある写真のうしろに photos を追加する。10枚を超えるときは postgres.ErrTooManyPhotos を返す。
	AddMany(ctx context.Context, diaryID uuid.UUID, photos []postgres.Photo) error
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

// deletePhotoFile は R2 から写真のファイルを消す。
// DB からはもう消えているので、失敗してもエラーにはせず、ログに残すだけにする（R2 にファイルが残るだけ）。
func deletePhotoFile(ctx context.Context, storage PhotoStorage, key string) {
	if err := storage.Delete(ctx, key); err != nil {
		slog.Error("delete photo file", "key", key, "err", err)
	}
}
