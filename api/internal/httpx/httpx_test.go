package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusNotFound, CodeNotFound, "日記が見つかりません")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	want := `{"error":{"code":"NOT_FOUND","message":"日記が見つかりません"}}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWriteInternalErrorHidesCause(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteInternalError(rec, errors.New("secret detail"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	want := `{"error":{"code":"INTERNAL_ERROR","message":"サーバーでエラーが発生しました"}}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
