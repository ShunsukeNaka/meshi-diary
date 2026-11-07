package handler

import (
	"errors"
	"net/http"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

// Me はログイン中のユーザーを返す（docs/api.md の GET /me）。Sessions.RequireSession で包んで使う。
func Me(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserID(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "ログインしてください")
			return
		}

		user, err := d.Users.FindByID(r.Context(), userID)
		if errors.Is(err, postgres.ErrNotFound) {
			// Cookie は正しいがユーザーが消えている。ログインしていないのと同じ扱いにし、Cookie も消す
			d.Sessions.Clear(w)
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "ログインしてください")
			return
		}
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]userResponse{"user": newUserResponse(user)})
	}
}

// Logout はセッション Cookie を消す（docs/api.md の POST /logout）。
// Cookie がない・正しくないときも 204 を返す（冪等）。
func Logout(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		d.Sessions.Clear(w)
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNoContent)
	}
}
