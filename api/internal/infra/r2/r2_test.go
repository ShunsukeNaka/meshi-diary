package r2

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/config"
)

// 署名付き URL はローカルで計算するので、R2 につながなくても確かめられる。
func TestPresignGet(t *testing.T) {
	c, err := New(context.Background(), config.R2Config{
		AccountID:       "test-account",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Bucket:          "test-bucket",
	})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := c.PresignGet(context.Background(), "photos/d/p.webp", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(u.Host, "test-account.r2.cloudflarestorage.com") {
		t.Errorf("host = %s", u.Host)
	}
	if !strings.HasSuffix(u.Path, "photos/d/p.webp") {
		t.Errorf("path = %s", u.Path)
	}
	q := u.Query()
	if q.Get("X-Amz-Expires") != "300" {
		t.Errorf("X-Amz-Expires = %q, want 300", q.Get("X-Amz-Expires"))
	}
	for k := range q {
		if strings.HasPrefix(strings.ToLower(k), "x-amz-checksum") {
			t.Errorf("unexpected checksum parameter %s", k)
		}
	}
}
