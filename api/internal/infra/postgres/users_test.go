package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TestUsers は実際の DB で Users を確かめる。
// TEST_DATABASE_URL がないときはスキップする。トランザクションの中で行い、最後にロールバックする。
func TestUsers(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })
	users := NewUsers(tx)
	ctx := context.Background()

	sub := "test-" + uuid.NewString()
	user := User{GoogleSub: sub, Name: "テスト"}
	if err := users.Create(ctx, &user); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := users.FindByGoogleSub(ctx, sub)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("id = %v, want %v", got.ID, user.ID)
	}

	if _, err := users.FindByGoogleSub(ctx, "no-such-sub-"+uuid.NewString()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("find unknown: err = %v, want ErrNotFound", err)
	}

	byID, err := users.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if byID.GoogleSub != sub {
		t.Fatalf("google_sub = %q, want %q", byID.GoogleSub, sub)
	}

	// SAVEPOINT の中で重複させ、トランザクション全体が失敗しないようにする
	tx.SavePoint("dup")
	err = users.Create(ctx, &User{GoogleSub: sub, Name: "重複"})
	tx.RollbackTo("dup")
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("create duplicate: err = %v, want gorm.ErrDuplicatedKey", err)
	}
}
