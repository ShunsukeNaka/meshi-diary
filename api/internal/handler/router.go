// Package handler は HTTP のルーティングとハンドラーを持つ。仕様は docs/api.md を参照。
package handler

import (
	"net/http"

	"gorm.io/gorm"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/r2"
)

// Deps はハンドラーが使う依存。日記などのエンドポイントを足すときに使う。
// ログインが必要なルートは Sessions.RequireSession で包む。
type Deps struct {
	DB       *gorm.DB
	Storage  *r2.Client
	Sessions *auth.Sessions
	IDTokens auth.IDTokenVerifier
	Users    UserRepository
	// AllowedEmails は /signup できるメールアドレス（小文字）。
	AllowedEmails []string
}

func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", Health)
	mux.Handle("GET /me", d.Sessions.RequireSession(Me(d)))
	mux.HandleFunc("POST /logout", Logout(d))
	return mux
}
