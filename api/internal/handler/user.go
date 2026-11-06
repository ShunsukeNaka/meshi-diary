package handler

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

// UserRepository はハンドラーが使う users の操作。本番は postgres.Users、テストでは偽物に差し替える。
type UserRepository interface {
	// FindByID は id のユーザーを返す。いないときは postgres.ErrNotFound を返す。
	FindByID(ctx context.Context, id uuid.UUID) (postgres.User, error)
	// FindByGoogleSub は google_sub のユーザーを返す。いないときは postgres.ErrNotFound を返す。
	FindByGoogleSub(ctx context.Context, sub string) (postgres.User, error)
	// Create は user を登録する。google_sub がすでにあるときは gorm.ErrDuplicatedKey を返す。
	Create(ctx context.Context, user *postgres.User) error
}

// userResponse はレスポンスの user（docs/api.md の /signup・/login・/me）。
type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func newUserResponse(u postgres.User) userResponse {
	return userResponse{ID: u.ID, Name: u.Name, CreatedAt: u.CreatedAt}
}
