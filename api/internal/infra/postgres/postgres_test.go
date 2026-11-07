package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/gorm"
)

// TestWriteAndRead は実際の DB に書き込み、読み出せるかを確かめる。
// TEST_DATABASE_URL がないときはスキップする。マイグレーション適用ずみの DB が必要。
// すべてトランザクションの中で行い、最後にロールバックするので DB にデータは残らない。
func TestWriteAndRead(t *testing.T) {
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

	// users: 書き込み → 読み出し
	user := User{GoogleSub: "test-" + uuid.NewString(), Name: "テスト"}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID == uuid.Nil {
		t.Fatal("user.ID was not filled by gen_random_uuid()")
	}
	var gotUser User
	if err := tx.First(&gotUser, "google_sub = ?", user.GoogleSub).Error; err != nil {
		t.Fatalf("find user: %v", err)
	}
	if gotUser.ID != user.ID || gotUser.Name != user.Name {
		t.Fatalf("got %+v, want %+v", gotUser, user)
	}

	// diaries: 既定値（status = OTHER, body = ''）と date 型
	eatenOn := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	diary := Diary{UserID: user.ID, EatenOn: eatenOn}
	if err := tx.Create(&diary).Error; err != nil {
		t.Fatalf("create diary: %v", err)
	}
	var gotDiary Diary
	if err := tx.First(&gotDiary, "id = ?", diary.ID).Error; err != nil {
		t.Fatalf("find diary: %v", err)
	}
	if gotDiary.Status != MealTypeOther || gotDiary.Body != "" || gotDiary.Info != nil {
		t.Fatalf("unexpected defaults: %+v", gotDiary)
	}
	if got := gotDiary.EatenOn.Format(time.DateOnly); got != "2026-11-06" {
		t.Fatalf("eaten_on = %s, want 2026-11-06", got)
	}

	// photos: position = 0 も保存できる
	photo := Photo{DiaryID: diary.ID, ObjectKey: "photos/" + diary.ID.String() + "/test.webp", Position: 0}
	if err := tx.Create(&photo).Error; err != nil {
		t.Fatalf("create photo: %v", err)
	}
	var count int64
	if err := tx.Model(&Photo{}).Where("diary_id = ?", diary.ID).Count(&count).Error; err != nil {
		t.Fatalf("count photos: %v", err)
	}
	if count != 1 {
		t.Fatalf("photos = %d, want 1", count)
	}

	// UNIQUE 違反が gorm.ErrDuplicatedKey になる。エラーでトランザクションが使えなくなるので最後に行う。
	err = tx.Create(&User{GoogleSub: user.GoogleSub, Name: "重複"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate google_sub: got %v, want gorm.ErrDuplicatedKey", err)
	}
}

func TestShouldPing(t *testing.T) {
	for _, tc := range []struct {
		idle time.Duration
		want bool
	}{
		{time.Second, false},
		{3 * time.Minute, false},
		{4 * time.Minute, false},
		{4*time.Minute + time.Second, true},
		{10 * time.Minute, true},
	} {
		if got := shouldPing(context.Background(), stdlib.ShouldPingParams{IdleDuration: tc.idle}); got != tc.want {
			t.Errorf("idle %v: got %v, want %v", tc.idle, got, tc.want)
		}
	}
}
