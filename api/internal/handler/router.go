// Package handler は HTTP のルーティングとハンドラーを持つ。仕様は docs/api.md を参照。
package handler

import (
	"net/http"

	"gorm.io/gorm"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/r2"
)

// Deps はハンドラーが使う依存。日記などのエンドポイントを足すときに使う。
type Deps struct {
	DB      *gorm.DB
	Storage *r2.Client
}

func NewRouter(_ Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", Health)
	return mux
}
