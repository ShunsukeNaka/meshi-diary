package handler

import "net/http"

// Health は API プロセスが応答できるかだけを返す。
// Neon のコンピュートを起こさないように DB には接続しない（docs/api.md の GET /health を参照）。
func Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
}
