package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

var testSecret = []byte(strings.Repeat("s", MinSecretLength))

func newTestSessions(t *testing.T, now time.Time) *Sessions {
	t.Helper()
	s, err := NewSessions(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return now }
	return s
}

// issue は Issue で発行した Cookie を返す。
func issue(t *testing.T, s *Sessions, userID uuid.UUID) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	s.Issue(rec, userID)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	return cookies[0]
}

func TestNewSessionsRejectsShortSecret(t *testing.T) {
	if _, err := NewSessions([]byte(strings.Repeat("s", MinSecretLength-1))); err == nil {
		t.Fatal("want error for short secret")
	}
}

func TestIssueSetsCookieAttributes(t *testing.T) {
	s := newTestSessions(t, time.Now())
	rec := httptest.NewRecorder()
	s.Issue(rec, uuid.New())

	header := rec.Header().Get("Set-Cookie")
	for _, want := range []string{"session=", "Path=/", "Max-Age=2592000", "HttpOnly", "Secure", "SameSite=Lax"} {
		if !strings.Contains(header, want) {
			t.Errorf("Set-Cookie = %q, want it to contain %q", header, want)
		}
	}
}

func TestVerify(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	s := newTestSessions(t, now)
	value := issue(t, s, userID).Value

	t.Run("valid", func(t *testing.T) {
		got, err := s.Verify(value)
		if err != nil {
			t.Fatal(err)
		}
		if got != userID {
			t.Fatalf("user id = %s, want %s", got, userID)
		}
	})

	t.Run("just before expiry", func(t *testing.T) {
		s.now = func() time.Time { return now.Add(SessionTTL - time.Second) }
		defer func() { s.now = func() time.Time { return now } }()
		if _, err := s.Verify(value); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("expired", func(t *testing.T) {
		s.now = func() time.Time { return now.Add(SessionTTL) }
		defer func() { s.now = func() time.Time { return now } }()
		if _, err := s.Verify(value); err != ErrSessionExpired {
			t.Fatalf("err = %v, want ErrSessionExpired", err)
		}
	})

	parts := strings.Split(value, ".")
	other, err := NewSessions([]byte(strings.Repeat("x", MinSecretLength)))
	if err != nil {
		t.Fatal(err)
	}
	other.now = s.now

	tampered := map[string]string{
		"other user id":       uuid.New().String() + "." + parts[1] + "." + parts[2],
		"extended expiry":     parts[0] + "." + "9999999999" + "." + parts[2],
		"broken signature":    parts[0] + "." + parts[1] + "." + flipFirst(parts[2]),
		"signed by other key": issue(t, other, userID).Value,
		"empty":               "",
		"no signature":        parts[0] + "." + parts[1],
		"not base64":          parts[0] + "." + parts[1] + ".!!!",
	}
	for name, v := range tampered {
		t.Run("tampered: "+name, func(t *testing.T) {
			if _, err := s.Verify(v); err != ErrInvalidSession {
				t.Fatalf("err = %v, want ErrInvalidSession", err)
			}
		})
	}
}

func TestClear(t *testing.T) {
	s := newTestSessions(t, time.Now())
	rec := httptest.NewRecorder()
	s.Clear(rec)

	header := rec.Header().Get("Set-Cookie")
	for _, want := range []string{"session=", "Path=/", "Max-Age=0", "HttpOnly", "Secure", "SameSite=Lax"} {
		if !strings.Contains(header, want) {
			t.Errorf("Set-Cookie = %q, want it to contain %q", header, want)
		}
	}
}

// flipFirst は先頭の 1 文字を別の base64url の文字に変える。
func flipFirst(s string) string {
	if s[0] == 'A' {
		return "B" + s[1:]
	}
	return "A" + s[1:]
}
