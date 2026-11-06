package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
)

// fakeVerifier は ID トークンの文字列から GoogleIdentity を返す偽物。
type fakeVerifier map[string]auth.GoogleIdentity

func (f fakeVerifier) Verify(_ context.Context, raw string) (auth.GoogleIdentity, error) {
	id, ok := f[raw]
	if !ok {
		return auth.GoogleIdentity{}, auth.ErrInvalidIDToken
	}
	return id, nil
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v (%q)", err, rec.Body.String())
	}
	return body.Error.Code
}

func TestVerifyIDToken(t *testing.T) {
	want := auth.GoogleIdentity{Sub: "sub-1", Email: "me@example.com", EmailVerified: true, Name: "山田 太郎"}
	verifier := fakeVerifier{"tok-ok": want}

	verify := func(body string) (auth.GoogleIdentity, bool, *httptest.ResponseRecorder) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		identity, ok := verifyIDToken(rec, req, verifier)
		return identity, ok, rec
	}

	t.Run("正しい ID トークンなら GoogleIdentity を返す", func(t *testing.T) {
		identity, ok, rec := verify(`{"id_token":"tok-ok"}`)
		if !ok {
			t.Fatalf("ok = false (%d %s)", rec.Code, rec.Body.String())
		}
		if identity != want {
			t.Fatalf("identity = %+v, want %+v", identity, want)
		}
	})

	cases := []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{"id_token がないと 400", `{}`, http.StatusBadRequest, httpx.CodeValidationError},
		{"id_token が空だと 400", `{"id_token":""}`, http.StatusBadRequest, httpx.CodeValidationError},
		{"JSON でないと 400", `not json`, http.StatusBadRequest, httpx.CodeValidationError},
		{"ID トークンが正しくないと 401", `{"id_token":"tok-bad"}`, http.StatusUnauthorized, httpx.CodeUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok, rec := verify(tc.body)
			if ok {
				t.Fatal("ok = true, want false")
			}
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
			if code := errorCode(t, rec); code != tc.code {
				t.Fatalf("code = %q, want %q", code, tc.code)
			}
		})
	}
}
