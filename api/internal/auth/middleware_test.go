package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
)

func TestRequireSession(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	s := newTestSessions(t, now)
	userID := uuid.New()
	valid := issue(t, s, userID)

	// テスト用のルート。通ったら、取り出した user_id を本文に書く。
	handler := s.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := UserID(r.Context())
		if !ok {
			t.Error("UserID not found in context")
		}
		_, _ = w.Write([]byte(id.String()))
	}))

	serve := func(cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	t.Run("valid cookie passes", func(t *testing.T) {
		rec := serve(valid)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Body.String(); got != userID.String() {
			t.Fatalf("body = %q, want %q", got, userID)
		}
	})

	unauthorized := map[string]*http.Cookie{
		"no cookie":       nil,
		"invalid cookie":  {Name: SessionCookieName, Value: "garbage"},
		"tampered cookie": {Name: SessionCookieName, Value: uuid.New().String() + valid.Value[36:]},
	}
	for name, c := range unauthorized {
		t.Run(name+" is 401", func(t *testing.T) {
			assertUnauthorized(t, serve(c))
		})
	}

	t.Run("expired cookie is 401", func(t *testing.T) {
		s.now = func() time.Time { return now.Add(SessionTTL + time.Second) }
		defer func() { s.now = func() time.Time { return now } }()
		assertUnauthorized(t, serve(valid))
	})
}

func assertUnauthorized(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error.Code != httpx.CodeUnauthorized {
		t.Fatalf("code = %q, want %q", body.Error.Code, httpx.CodeUnauthorized)
	}
}
