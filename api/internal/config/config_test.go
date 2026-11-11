package config

import (
	"reflect"
	"strings"
	"testing"
)

func setRequired(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"DATABASE_URL":         "postgresql://example",
		"R2_ACCOUNT_ID":        "account",
		"R2_ACCESS_KEY_ID":     "key",
		"R2_SECRET_ACCESS_KEY": "secret",
		"R2_BUCKET":            "bucket",
		"GOOGLE_CLIENT_ID":     "client.apps.googleusercontent.com",
		"SESSION_SECRET":       strings.Repeat("s", 32),
		"ALLOWED_EMAILS":       " Me@Example.com , ,other@example.com",
	} {
		t.Setenv(k, v)
	}
}

func TestLoadAuth(t *testing.T) {
	setRequired(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"me@example.com", "other@example.com"}; !reflect.DeepEqual(cfg.Auth.AllowedEmails, want) {
		t.Fatalf("AllowedEmails = %q, want %q", cfg.Auth.AllowedEmails, want)
	}
}

func TestLoadRejectsShortSessionSecret(t *testing.T) {
	setRequired(t)
	t.Setenv("SESSION_SECRET", strings.Repeat("s", 31))
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "SESSION_SECRET") {
		t.Fatalf("err = %v, want SESSION_SECRET error", err)
	}
}

func TestLoadReportsMissingAuthVariables(t *testing.T) {
	setRequired(t)
	for _, k := range []string{"GOOGLE_CLIENT_ID", "SESSION_SECRET", "ALLOWED_EMAILS"} {
		t.Setenv(k, "")
	}
	t.Setenv("ALLOWED_EMAILS", " , ")
	_, err := Load()
	if err == nil {
		t.Fatal("want error")
	}
	for _, k := range []string{"GOOGLE_CLIENT_ID", "SESSION_SECRET", "ALLOWED_EMAILS"} {
		if !strings.Contains(err.Error(), k) {
			t.Errorf("err = %v, want it to mention %s", err, k)
		}
	}
}

func TestLoadStoresNeedsOnlyDBAndR2(t *testing.T) {
	setRequired(t)
	for _, k := range []string{"GOOGLE_CLIENT_ID", "SESSION_SECRET", "ALLOWED_EMAILS"} {
		t.Setenv(k, "")
	}
	cfg, err := LoadStores()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgresql://example" || cfg.R2.Bucket != "bucket" {
		t.Fatalf("cfg = %+v", cfg)
	}

	t.Setenv("R2_BUCKET", "")
	if _, err := LoadStores(); err == nil || !strings.Contains(err.Error(), "R2_BUCKET") {
		t.Fatalf("err = %v, want R2_BUCKET error", err)
	}
}
