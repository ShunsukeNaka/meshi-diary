package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
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

// fakeAuthUsers は google_sub で引くメモリ上の users。postgres.Users と同じエラーを返す。
type fakeAuthUsers struct {
	bySub map[string]postgres.User
	err   error // 設定すると、すべての操作でこのエラーを返す
}

func (f *fakeAuthUsers) FindByID(_ context.Context, id uuid.UUID) (postgres.User, error) {
	if f.err != nil {
		return postgres.User{}, f.err
	}
	for _, u := range f.bySub {
		if u.ID == id {
			return u, nil
		}
	}
	return postgres.User{}, postgres.ErrNotFound
}

func (f *fakeAuthUsers) Create(_ context.Context, u *postgres.User) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.bySub[u.GoogleSub]; ok {
		return gorm.ErrDuplicatedKey
	}
	u.ID = uuid.New()
	u.CreatedAt = time.Date(2026, 11, 6, 12, 30, 0, 0, time.UTC)
	u.UpdatedAt = u.CreatedAt
	f.bySub[u.GoogleSub] = *u
	return nil
}

func (f *fakeAuthUsers) FindByGoogleSub(_ context.Context, sub string) (postgres.User, error) {
	if f.err != nil {
		return postgres.User{}, f.err
	}
	u, ok := f.bySub[sub]
	if !ok {
		return postgres.User{}, postgres.ErrNotFound
	}
	return u, nil
}

var (
	allowed      = auth.GoogleIdentity{Sub: "sub-allowed", Email: "Me@Example.com", EmailVerified: true, Name: "山田 太郎"}
	notAllowed   = auth.GoogleIdentity{Sub: "sub-other", Email: "other@example.com", EmailVerified: true, Name: "他人"}
	notVerified  = auth.GoogleIdentity{Sub: "sub-unverified", Email: "me@example.com", EmailVerified: false, Name: "未確認"}
	testVerifier = fakeVerifier{"tok-allowed": allowed, "tok-other": notAllowed, "tok-unverified": notVerified}
)

func newTestRouter(t *testing.T, users *fakeAuthUsers) (http.Handler, *auth.Sessions) {
	t.Helper()
	sessions, err := auth.NewSessions([]byte(strings.Repeat("s", auth.MinSecretLength)))
	if err != nil {
		t.Fatal(err)
	}
	return NewRouter(Deps{
		Users:         users,
		Sessions:      sessions,
		IDTokens:      testVerifier,
		AllowedEmails: []string{"me@example.com"},
	}), sessions
}

func post(router http.Handler, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v (%q)", err, rec.Body.String())
	}
	return body.Error.Code
}

// sessionUserID はレスポンスのセッション Cookie を検証し、user_id を返す。Cookie がなければ uuid.Nil。
func sessionUserID(t *testing.T, rec *httptest.ResponseRecorder, sessions *auth.Sessions) uuid.UUID {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			id, err := sessions.Verify(c.Value)
			if err != nil {
				t.Fatalf("verify session cookie: %v", err)
			}
			return id
		}
	}
	return uuid.Nil
}

func decodeUser(t *testing.T, rec *httptest.ResponseRecorder) userResponse {
	t.Helper()
	var body map[string]userResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode user body: %v", err)
	}
	return body["user"]
}

func TestLogin(t *testing.T) {
	registered := postgres.User{
		ID:        uuid.New(),
		GoogleSub: allowed.Sub,
		Name:      "登録時の名前",
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	t.Run("登録ずみなら 200 で Cookie を発行する", func(t *testing.T) {
		router, sessions := newTestRouter(t, &fakeAuthUsers{bySub: map[string]postgres.User{registered.GoogleSub: registered}})

		rec := post(router, "/login", `{"id_token":"tok-allowed"}`)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
		}
		got := decodeUser(t, rec)
		// name はログインのたびには更新しないので、登録時の名前のまま
		if got.ID != registered.ID || got.Name != registered.Name || !got.CreatedAt.Equal(registered.CreatedAt) {
			t.Fatalf("response user = %+v, want %+v", got, registered)
		}
		if id := sessionUserID(t, rec, sessions); id != registered.ID {
			t.Fatalf("session user_id = %v, want %v", id, registered.ID)
		}
	})

	t.Run("ALLOWED_EMAILS は見ない（登録ずみならログインできる）", func(t *testing.T) {
		other := postgres.User{ID: uuid.New(), GoogleSub: notAllowed.Sub, Name: notAllowed.Name}
		router, _ := newTestRouter(t, &fakeAuthUsers{bySub: map[string]postgres.User{other.GoogleSub: other}})

		rec := post(router, "/login", `{"id_token":"tok-other"}`)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	cases := []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{"id_token がないと 400", `{"id_token":""}`, http.StatusBadRequest, httpx.CodeValidationError},
		{"ID トークンが正しくないと 401", `{"id_token":"tok-bad"}`, http.StatusUnauthorized, httpx.CodeUnauthorized},
		{"登録されていないと 404", `{"id_token":"tok-allowed"}`, http.StatusNotFound, httpx.CodeNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router, sessions := newTestRouter(t, &fakeAuthUsers{bySub: map[string]postgres.User{}})

			rec := post(router, "/login", tc.body)

			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tc.status, rec.Body.String())
			}
			if code := errorCode(t, rec); code != tc.code {
				t.Fatalf("code = %q, want %q", code, tc.code)
			}
			if id := sessionUserID(t, rec, sessions); id != uuid.Nil {
				t.Fatal("session cookie should not be issued")
			}
		})
	}

	t.Run("DB のエラーは 500", func(t *testing.T) {
		router, _ := newTestRouter(t, &fakeAuthUsers{bySub: map[string]postgres.User{}, err: errors.New("db down")})

		rec := post(router, "/login", `{"id_token":"tok-allowed"}`)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
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
