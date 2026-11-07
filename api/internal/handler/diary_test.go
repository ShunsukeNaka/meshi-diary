package handler

import (
	"encoding/json"
	"errors"
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

// postJSON は cookie を付けて body を POST する。
func postJSON(router http.Handler, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeDiary(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body struct {
		Diary map[string]any `json:"diary"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode diary: %v", err)
	}
	return body.Diary
}

func TestCreateDiary(t *testing.T) {
	userID := uuid.New()

	t.Run("creates the diary of the logged-in user", func(t *testing.T) {
		diaries := &fakeDiaries{}
		router, sessions := newDiaryTestRouter(t, diaries)
		rec := postJSON(router, "/diaries",
			`{"eaten_on":"2026-11-06","status":"LUNCH","info":"駅前のラーメン屋","body":"久しぶりに会った。"}`,
			sessionCookie(t, sessions, userID))

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (%s)", rec.Code, rec.Body.String())
		}
		if len(diaries.created) != 1 {
			t.Fatalf("created = %d, want 1", len(diaries.created))
		}
		saved := diaries.created[0]
		if saved.UserID != userID || saved.Status != postgres.MealTypeLunch || saved.Body != "久しぶりに会った。" {
			t.Fatalf("saved = %+v", saved)
		}
		if got := saved.EatenOn.Format(time.DateOnly); got != "2026-11-06" {
			t.Fatalf("saved eaten_on = %s", got)
		}

		got := decodeDiary(t, rec)
		want := map[string]any{
			"id":         saved.ID.String(),
			"eaten_on":   "2026-11-06",
			"status":     "LUNCH",
			"info":       "駅前のラーメン屋",
			"body":       "久しぶりに会った。",
			"created_at": "2026-11-06T12:30:00+09:00",
			"updated_at": "2026-11-06T12:30:00+09:00",
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s = %v, want %v", k, got[k], v)
			}
		}
	})

	t.Run("info that is omitted, null or empty is saved as null", func(t *testing.T) {
		for _, info := range []string{``, `,"info":null`, `,"info":""`} {
			diaries := &fakeDiaries{}
			router, sessions := newDiaryTestRouter(t, diaries)
			rec := postJSON(router, "/diaries",
				`{"eaten_on":"2026-11-06","status":"OTHER","body":"本文"`+info+`}`,
				sessionCookie(t, sessions, userID))
			if rec.Code != http.StatusCreated {
				t.Fatalf("info %q: status = %d, want 201", info, rec.Code)
			}
			if diaries.created[0].Info != nil {
				t.Fatalf("info %q: saved info = %q, want nil", info, *diaries.created[0].Info)
			}
			if got, ok := decodeDiary(t, rec)["info"]; !ok || got != nil {
				t.Fatalf("info %q: response info = %v (present %v), want null", info, got, ok)
			}
		}
	})

	t.Run("status that is omitted, null or empty is saved as OTHER", func(t *testing.T) {
		for _, status := range []string{``, `,"status":null`, `,"status":""`} {
			diaries := &fakeDiaries{}
			router, sessions := newDiaryTestRouter(t, diaries)
			rec := postJSON(router, "/diaries",
				`{"eaten_on":"2026-11-06","body":"本文"`+status+`}`,
				sessionCookie(t, sessions, userID))
			if rec.Code != http.StatusCreated {
				t.Fatalf("status %q: code = %d, want 201", status, rec.Code)
			}
			if got := diaries.created[0].Status; got != postgres.MealTypeOther {
				t.Fatalf("status %q: saved status = %q, want OTHER", status, got)
			}
			if got := decodeDiary(t, rec)["status"]; got != "OTHER" {
				t.Fatalf("status %q: response status = %v, want OTHER", status, got)
			}
		}
	})

	t.Run("invalid input is 400", func(t *testing.T) {
		cases := map[string]string{
			"not json":            `{`,
			"wrong type":          `{"eaten_on":"2026-11-06","status":1,"body":"本文"}`,
			"no eaten_on":         `{"status":"LUNCH","body":"本文"}`,
			"eaten_on slash":      `{"eaten_on":"2026/11/06","status":"LUNCH","body":"本文"}`,
			"eaten_on not a date": `{"eaten_on":"2026-02-30","status":"LUNCH","body":"本文"}`,
			"lowercase status":    `{"eaten_on":"2026-11-06","status":"lunch","body":"本文"}`,
			"no body":             `{"eaten_on":"2026-11-06","status":"LUNCH"}`,
			"blank body":          `{"eaten_on":"2026-11-06","status":"LUNCH","body":"　 \n\t"}`,
		}
		for name, body := range cases {
			t.Run(name, func(t *testing.T) {
				diaries := &fakeDiaries{}
				router, sessions := newDiaryTestRouter(t, diaries)
				rec := postJSON(router, "/diaries", body, sessionCookie(t, sessions, userID))
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400", rec.Code)
				}
				if code := errorCode(t, rec); code != httpx.CodeValidationError {
					t.Fatalf("code = %q", code)
				}
				if len(diaries.created) != 0 {
					t.Fatal("diary was saved")
				}
			})
		}
	})

	t.Run("too large body is 413", func(t *testing.T) {
		router, sessions := newDiaryTestRouter(t, &fakeDiaries{})
		body := `{"eaten_on":"2026-11-06","status":"LUNCH","body":"` + strings.Repeat("a", maxDiaryBodyBytes) + `"}`
		rec := postJSON(router, "/diaries", body, sessionCookie(t, sessions, userID))
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", rec.Code)
		}
		if code := errorCode(t, rec); code != httpx.CodePayloadTooLarge {
			t.Fatalf("code = %q", code)
		}
	})

	t.Run("no cookie is 401", func(t *testing.T) {
		diaries := &fakeDiaries{}
		router, _ := newDiaryTestRouter(t, diaries)
		assertUnauthorized(t, postJSON(router, "/diaries", `{"eaten_on":"2026-11-06","status":"LUNCH","body":"本文"}`, nil))
		if len(diaries.created) != 0 {
			t.Fatal("diary was saved")
		}
	})

	t.Run("DB error is 500", func(t *testing.T) {
		router, sessions := newDiaryTestRouter(t, &fakeDiaries{err: errors.New("db down")})
		rec := postJSON(router, "/diaries", `{"eaten_on":"2026-11-06","status":"LUNCH","body":"本文"}`, sessionCookie(t, sessions, userID))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}

func newDiaryTestRouter(t *testing.T, diaries *fakeDiaries) (http.Handler, *auth.Sessions) {
	t.Helper()
	e := newDiaryTestEnv(t)
	e.diaries = diaries
	e.router = NewRouter(Deps{Sessions: e.sessions, Diaries: diaries, Photos: e.photos, Storage: e.storage})
	return e.router, e.sessions
}
