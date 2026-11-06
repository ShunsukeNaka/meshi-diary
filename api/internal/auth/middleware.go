package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
)

type contextKey struct{}

// RequireSession はセッション Cookie を確かめるミドルウェア。
// Cookie がない・正しくない・期限切れのときは 401 UNAUTHORIZED を返し、next は呼ばない。
// 通ったときは UserID でログイン中のユーザーの id を取り出せる。
func (s *Sessions) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(SessionCookieName)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "ログインしてください")
			return
		}
		userID, err := s.Verify(c.Value)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "ログインしてください")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, userID)))
	})
}

// UserID は RequireSession を通ったリクエストから、ログイン中のユーザーの id を取り出す。
func UserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(contextKey{}).(uuid.UUID)
	return id, ok
}
