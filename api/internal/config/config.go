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
}

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
	}

	required := []struct{ name, value string }{
		{"DATABASE_URL", cfg.DatabaseURL},
		{"R2_ACCOUNT_ID", cfg.R2.AccountID},
		{"R2_ACCESS_KEY_ID", cfg.R2.AccessKeyID},
		{"R2_SECRET_ACCESS_KEY", cfg.R2.SecretAccessKey},
		{"R2_BUCKET", cfg.R2.Bucket},
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
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
