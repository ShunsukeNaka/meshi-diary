// Package config は環境変数から設定を読み込む。
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	// ListenAddr は API が待ち受けるアドレス。Apache から転送されたリクエストだけを受けるため、既定は 127.0.0.1。
	ListenAddr  string
	DatabaseURL string
	R2          R2Config
	Auth        AuthConfig
}

type AuthConfig struct {
	// GoogleClientID は ID トークンの aud の検証に使う OAuth クライアント ID。
	GoogleClientID string
	// SessionSecret はセッション Cookie の署名に使う秘密鍵。32 バイト以上。
	SessionSecret []byte
	// AllowedEmails は /signup できるメールアドレス。小文字にそろえてある。
	AllowedEmails []string
}

// minSessionSecretLength は SESSION_SECRET の最小の長さ（バイト）。auth.MinSecretLength と同じ値。
const minSessionSecretLength = 32

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

// Load は環境変数を読み込む。必須の値がないときは、足りない名前をすべて並べたエラーを返す。
func Load() (Config, error) {
	cfg := Config{
		ListenAddr:  getenv("LISTEN_ADDR", "127.0.0.1:8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		R2: R2Config{
			AccountID:       os.Getenv("R2_ACCOUNT_ID"),
			AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
			Bucket:          os.Getenv("R2_BUCKET"),
		},
		Auth: AuthConfig{
			GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),
			SessionSecret:  []byte(os.Getenv("SESSION_SECRET")),
			AllowedEmails:  splitEmails(os.Getenv("ALLOWED_EMAILS")),
		},
	}

	required := []struct{ name, value string }{
		{"DATABASE_URL", cfg.DatabaseURL},
		{"R2_ACCOUNT_ID", cfg.R2.AccountID},
		{"R2_ACCESS_KEY_ID", cfg.R2.AccessKeyID},
		{"R2_SECRET_ACCESS_KEY", cfg.R2.SecretAccessKey},
		{"R2_BUCKET", cfg.R2.Bucket},
		{"GOOGLE_CLIENT_ID", cfg.Auth.GoogleClientID},
		{"SESSION_SECRET", string(cfg.Auth.SessionSecret)},
		{"ALLOWED_EMAILS", strings.Join(cfg.Auth.AllowedEmails, ",")},
	}
	var missing []string
	for _, r := range required {
		if r.value == "" {
			missing = append(missing, r.name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	if len(cfg.Auth.SessionSecret) < minSessionSecretLength {
		return Config{}, fmt.Errorf("SESSION_SECRET must be at least %d bytes", minSessionSecretLength)
	}
	return cfg, nil
}

// splitEmails はカンマ区切りのメールアドレスを、前後の空白を除き小文字にそろえて返す。空の要素は捨てる。
func splitEmails(s string) []string {
	var emails []string
	for _, e := range strings.Split(s, ",") {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			emails = append(emails, e)
		}
	}
	return emails
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
