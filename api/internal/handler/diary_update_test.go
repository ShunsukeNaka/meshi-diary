package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

// sendJSON は method で body を送る。
func sendJSON(router http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestUpdateDiary(t *testing.T) {
	userID := uuid.New()

	t.Run("changes only the sent fields", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		rec := sendJSON(e.router, http.MethodPatch, "/diaries/"+d.ID.String(), `{"body":"書き直した"}`, e.cookie(t, userID))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
		}
		saved := e.diaries.byID[d.ID]
		if saved.Body != "書き直した" || saved.Status != d.Status || saved.Info == nil || *saved.Info != *d.Info || !saved.EatenOn.Equal(d.EatenOn) {
			t.Fatalf("saved = %+v", saved)
		}
		got := decodeDiary(t, rec)
		if got["body"] != "書き直した" || got["info"] != "駅前のラーメン屋" || got["updated_at"] != "2026-11-06T13:30:00+09:00" {
			t.Fatalf("response = %v", got)
		}
	})

	t.Run("changes all fields", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		rec := sendJSON(e.router, http.MethodPatch, "/diaries/"+d.ID.String(),
			`{"eaten_on":"2026-11-07","status":"DINNER","info":"鍋","body":"本文2"}`, e.cookie(t, userID))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		saved := e.diaries.byID[d.ID]
		if saved.EatenOn.Format("2006-01-02") != "2026-11-07" || saved.Status != postgres.MealTypeDinner || *saved.Info != "鍋" || saved.Body != "本文2" {
			t.Fatalf("saved = %+v", saved)
		}
	})

	t.Run("info null or empty clears it", func(t *testing.T) {
		for _, body := range []string{`{"info":null}`, `{"info":""}`} {
			e := newDiaryTestEnv(t)
			d := e.addDiary(userID)
			rec := sendJSON(e.router, http.MethodPatch, "/diaries/"+d.ID.String(), body, e.cookie(t, userID))
			if rec.Code != http.StatusOK {
				t.Fatalf("%s: status = %d", body, rec.Code)
			}
			if e.diaries.byID[d.ID].Info != nil {
				t.Fatalf("%s: info was not cleared", body)
			}
			if v, ok := decodeDiary(t, rec)["info"]; !ok || v != nil {
				t.Fatalf("%s: response info = %v", body, v)
			}
		}
	})

	t.Run("status null or empty becomes OTHER", func(t *testing.T) {
		for _, body := range []string{`{"status":null}`, `{"status":""}`} {
			e := newDiaryTestEnv(t)
			d := e.addDiary(userID)
			if rec := sendJSON(e.router, http.MethodPatch, "/diaries/"+d.ID.String(), body, e.cookie(t, userID)); rec.Code != http.StatusOK {
				t.Fatalf("%s: status = %d", body, rec.Code)
			}
			if got := e.diaries.byID[d.ID].Status; got != postgres.MealTypeOther {
				t.Fatalf("%s: status = %q", body, got)
			}
		}
	})

	t.Run("invalid input is 400 and nothing changes", func(t *testing.T) {
		for name, body := range map[string]string{
			"not json":         `{`,
			"eaten_on null":    `{"eaten_on":null}`,
			"eaten_on invalid": `{"eaten_on":"2026-13-01"}`,
			"status invalid":   `{"status":"lunch"}`,
			"body null":        `{"body":null}`,
			"body blank":       `{"body":"　\n"}`,
			"wrong type":       `{"body":1}`,
		} {
			e := newDiaryTestEnv(t)
			d := e.addDiary(userID)
			rec := sendJSON(e.router, http.MethodPatch, "/diaries/"+d.ID.String(), body, e.cookie(t, userID))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("%s: status = %d, want 400", name, rec.Code)
			}
			if saved := e.diaries.byID[d.ID]; saved.Body != d.Body || saved.Status != d.Status {
				t.Fatalf("%s: diary changed: %+v", name, saved)
			}
		}
	})

	t.Run("other user's or unknown diary is 404", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		others := e.addDiary(uuid.New())
		for _, id := range []string{others.ID.String(), uuid.NewString(), "abc"} {
			rec := sendJSON(e.router, http.MethodPatch, "/diaries/"+id, `{"body":"乗っ取り"}`, e.cookie(t, userID))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("%s: status = %d, want 404", id, rec.Code)
			}
		}
		if e.diaries.byID[others.ID].Body != "本文" {
			t.Fatal("other user's diary was changed")
		}
	})

	t.Run("no cookie is 401", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		assertUnauthorized(t, sendJSON(e.router, http.MethodPatch, "/diaries/"+d.ID.String(), `{"body":"x"}`, nil))
	})
}
