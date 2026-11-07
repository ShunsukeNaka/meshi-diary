// Package handler は HTTP のルーティングとハンドラーを持つ。仕様は docs/api.md を参照。
package handler

import (
	"net/http"

	"gorm.io/gorm"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
)

// Deps はハンドラーが使う依存。日記などのエンドポイントを足すときに使う。
// ログインが必要なルートは Sessions.RequireSession で包む。
type Deps struct {
	DB       *gorm.DB
	Storage  PhotoStorage
	Sessions *auth.Sessions
	IDTokens auth.IDTokenVerifier
	Users    UserRepository
	Diaries  DiaryRepository
	Photos   PhotoRepository
	// AllowedEmails は /signup できるメールアドレス（小文字）。
	AllowedEmails []string
}

func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", Health)
	mux.HandleFunc("POST /signup", Signup(d))
	mux.HandleFunc("POST /login", Login(d))
	mux.Handle("GET /me", d.Sessions.RequireSession(Me(d)))
	mux.HandleFunc("POST /logout", Logout(d))
	mux.Handle("POST /diaries", d.Sessions.RequireSession(CreateDiary(d)))
	mux.Handle("GET /diaries", d.Sessions.RequireSession(ListDiaries(d)))
	mux.Handle("GET /diaries/{id}", d.Sessions.RequireSession(GetDiary(d)))
	mux.Handle("PATCH /diaries/{id}", d.Sessions.RequireSession(UpdateDiary(d)))
	mux.Handle("DELETE /diaries/{id}", d.Sessions.RequireSession(DeleteDiary(d)))
	mux.Handle("GET /calendar", d.Sessions.RequireSession(Calendar(d)))
	mux.Handle("POST /diaries/{id}/photos", d.Sessions.RequireSession(AddPhotos(d)))
	return mux
}
