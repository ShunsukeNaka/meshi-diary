package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestDiaries は実際の DB で Diaries を確かめる。
// TEST_DATABASE_URL がないときはスキップする。トランザクションの中で行い、最後にロールバックする。
func TestDiaries(t *testing.T) {
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
	ctx := context.Background()

	user := User{GoogleSub: "test-" + uuid.NewString(), Name: "テスト"}
	if err := NewUsers(tx).Create(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	info := "駅前のラーメン屋"
	diary := Diary{
		UserID:  user.ID,
		EatenOn: time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC),
		Status:  MealTypeLunch,
		Body:    "本文",
		Info:    &info,
	}
	if err := NewDiaries(tx).Create(ctx, &diary); err != nil {
		t.Fatalf("create diary: %v", err)
	}
	if diary.ID == uuid.Nil || diary.CreatedAt.IsZero() {
		t.Fatalf("id or created_at was not filled: %+v", diary)
	}

	var got Diary
	if err := tx.First(&got, "id = ?", diary.ID).Error; err != nil {
		t.Fatalf("find diary: %v", err)
	}
	if got.UserID != user.ID || got.Status != MealTypeLunch || got.Body != "本文" || got.Info == nil || *got.Info != info {
		t.Fatalf("got %+v", got)
	}
	if d := got.EatenOn.Format(time.DateOnly); d != "2026-11-06" {
		t.Fatalf("eaten_on = %s, want 2026-11-06", d)
	}
}
