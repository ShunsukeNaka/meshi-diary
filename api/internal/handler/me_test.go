package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

// fakeUsers は UserRepository の偽物。users にいる id だけ見つかる。
type fakeUsers map[uuid.UUID]postgres.User

func (f fakeUsers) FindByID(_ context.Context, id uuid.UUID) (postgres.User, error) {
	u, ok := f[id]
	if !ok {
		return postgres.User{}, postgres.ErrNotFound
	}
	return u, nil
}

func newTestDeps(t *testing.T, users fakeUsers) Deps {
	t.Helper()
	sessions, err := auth.NewSessions([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	return Deps{Sessions: sessions, Users: users}
}

// sessionCookie は userID のセッション Cookie を発行して返す。
func sessionCookie(t *testing.T, s *auth.Sessions, userID uuid.UUID) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	s.Issue(rec, userID)
	return findSessionCookie(t, rec)
}

func findSessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			return c
		}
	}
	t.Fatal("session cookie not set")
	return nil
}

func serve(router http.Handler, method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertUnauthorized(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != httpx.CodeUnauthorized {
		t.Fatalf("code = %q, want %q", body.Error.Code, httpx.CodeUnauthorized)
	}
}

func TestMe(t *testing.T) {
	user := postgres.User{
		ID:        uuid.New(),
		Name:      "山田 太郎",
		CreatedAt: time.Date(2026, 11, 6, 12, 30, 0, 0, time.FixedZone("JST", 9*60*60)),
	}
	d := newTestDeps(t, fakeUsers{user.ID: user})
	router := NewRouter(d)

	t.Run("logged in returns the user", func(t *testing.T) {
		rec := serve(router, http.MethodGet, "/me", sessionCookie(t, d.Sessions, user.ID))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var body struct {
			User struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				CreatedAt string `json:"created_at"`
			} `json:"user"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.User.ID != user.ID.String() || body.User.Name != user.Name {
			t.Fatalf("user = %+v, want %+v", body.User, user)
		}
		if body.User.CreatedAt != "2026-11-06T12:30:00+09:00" {
			t.Fatalf("created_at = %q", body.User.CreatedAt)
		}
	})

	t.Run("no cookie is 401", func(t *testing.T) {
		assertUnauthorized(t, serve(router, http.MethodGet, "/me", nil))
	})

	t.Run("invalid cookie is 401", func(t *testing.T) {
		assertUnauthorized(t, serve(router, http.MethodGet, "/me", &http.Cookie{Name: auth.SessionCookieName, Value: "garbage"}))
	})

	t.Run("deleted user is 401 and the cookie is cleared", func(t *testing.T) {
		rec := serve(router, http.MethodGet, "/me", sessionCookie(t, d.Sessions, uuid.New()))
		if c := findSessionCookie(t, rec); c.MaxAge >= 0 || c.Value != "" {
			t.Fatalf("cookie not cleared: %+v", c)
		}
		assertUnauthorized(t, rec)
	})
}

func TestLogout(t *testing.T) {
	user := postgres.User{ID: uuid.New(), Name: "山田 太郎"}
	d := newTestDeps(t, fakeUsers{user.ID: user})
	router := NewRouter(d)

	t.Run("clears the cookie and /me becomes 401", func(t *testing.T) {
		rec := serve(router, http.MethodPost, "/logout", sessionCookie(t, d.Sessions, user.ID))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		cleared := findSessionCookie(t, rec)
		// http.Cookie.MaxAge が負なら Set-Cookie は Max-Age=0
		if cleared.MaxAge >= 0 || cleared.Value != "" {
			t.Fatalf("cookie not cleared: %+v", cleared)
		}
		if got := rec.Header().Get("Set-Cookie"); !strings.Contains(got, "Max-Age=0") {
			t.Fatalf("Set-Cookie = %q, want Max-Age=0", got)
		}

		// ブラウザは Max-Age=0 の Cookie を消すので、次のリクエストには Cookie が付かない
		assertUnauthorized(t, serve(router, http.MethodGet, "/me", nil))
		// 消した値（空）が送られてきても 401
		assertUnauthorized(t, serve(router, http.MethodGet, "/me", &http.Cookie{Name: auth.SessionCookieName, Value: cleared.Value}))
	})

	t.Run("no cookie is also 204", func(t *testing.T) {
		rec := serve(router, http.MethodPost, "/logout", nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		findSessionCookie(t, rec)
	})

	t.Run("GET is not allowed", func(t *testing.T) {
		if rec := serve(router, http.MethodGet, "/logout", nil); rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})
}
