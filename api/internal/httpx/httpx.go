// Package httpx は JSON のレスポンスと、docs/api.md の共通のエラーレスポンスを書き出す。
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// エラーレスポンスの code。docs/api.md の「エラーレスポンス」を参照。
const (
	CodeValidationError      = "VALIDATION_ERROR"
	CodeUnauthorized         = "UNAUTHORIZED"
	CodeForbidden            = "FORBIDDEN"
	CodeNotFound             = "NOT_FOUND"
	CodeConflict             = "CONFLICT"
	CodePayloadTooLarge      = "PAYLOAD_TOO_LARGE"
	CodeUnsupportedMediaType = "UNSUPPORTED_MEDIA_TYPE"
	CodeInternalError        = "INTERNAL_ERROR"
)

// ErrorBody は {"error": {"code", "message"}} の形を表す。
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON は v を JSON にして status で返す。
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json response", "err", err)
	}
}

// WriteError は共通の形のエラーレスポンスを返す。
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

// WriteInternalError は原因をログに残し、利用者には詳細を見せずに 500 を返す。
func WriteInternalError(w http.ResponseWriter, err error) {
	slog.Error("internal error", "err", err)
	WriteError(w, http.StatusInternalServerError, CodeInternalError, "サーバーでエラーが発生しました")
}
