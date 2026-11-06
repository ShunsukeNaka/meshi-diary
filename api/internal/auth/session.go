// Package auth はログインの仕組み（Google の ID トークンの検証、セッション Cookie、認証ミドルウェア）を持つ。
// 仕様は docs/api.md の「認証」を参照。
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// SessionCookieName はセッション Cookie の名前。
	SessionCookieName = "session"
	// SessionTTL はセッションの有効期限。
	SessionTTL = 30 * 24 * time.Hour
	// MinSecretLength は署名に使う秘密鍵の最小の長さ（バイト）。
	MinSecretLength = 32
)

var (
	// ErrInvalidSession は Cookie の値の形が正しくない、または署名が合わないときに返す。
	ErrInvalidSession = errors.New("invalid session")
	// ErrSessionExpired は有効期限が切れているときに返す。
	ErrSessionExpired = errors.New("session expired")
)

// Sessions はセッション Cookie を発行・検証する。
// 値は「user_id.有効期限（Unix 秒）.署名」で、署名は前の 2 つを HMAC-SHA256 で署名したもの。DB には保存しない。
type Sessions struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewSessions は secret で署名する Sessions を作る。secret は MinSecretLength バイト以上にする。
func NewSessions(secret []byte) (*Sessions, error) {
	if len(secret) < MinSecretLength {
		return nil, fmt.Errorf("session secret must be at least %d bytes", MinSecretLength)
	}
	return &Sessions{secret: secret, ttl: SessionTTL, now: time.Now}, nil
}

// Issue は userID のセッション Cookie を w に付ける。
func (s *Sessions) Issue(w http.ResponseWriter, userID uuid.UUID) {
	http.SetCookie(w, s.newCookie(s.sign(userID, s.now().Add(s.ttl)), int(s.ttl.Seconds())))
}

// Clear はセッション Cookie を消す（Max-Age=0）。ログアウトで使う。
func (s *Sessions) Clear(w http.ResponseWriter) {
	c := s.newCookie("", 0)
	c.MaxAge = -1 // net/http では負の値で Max-Age=0 を出す
	http.SetCookie(w, c)
}

// Verify は Cookie の値を検証し、user_id を返す。
func (s *Sessions) Verify(value string) (uuid.UUID, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return uuid.Nil, ErrInvalidSession
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return uuid.Nil, ErrInvalidSession
	}
	if !hmac.Equal(sig, s.mac(parts[0]+"."+parts[1])) {
		return uuid.Nil, ErrInvalidSession
	}

	userID, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, ErrInvalidSession
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return uuid.Nil, ErrInvalidSession
	}
	if !s.now().Before(time.Unix(exp, 0)) {
		return uuid.Nil, ErrSessionExpired
	}
	return userID, nil
}

func (s *Sessions) sign(userID uuid.UUID, exp time.Time) string {
	payload := userID.String() + "." + strconv.FormatInt(exp.Unix(), 10)
	return payload + "." + base64.RawURLEncoding.EncodeToString(s.mac(payload))
}

func (s *Sessions) mac(payload string) []byte {
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(payload))
	return m.Sum(nil)
}

func (s *Sessions) newCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}
