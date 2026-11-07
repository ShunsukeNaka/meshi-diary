package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// openTestTx は TEST_DATABASE_URL の DB でトランザクションを始め、テストの最後にロールバックする。
// TEST_DATABASE_URL がないときはテストをスキップする。
func openTestTx(t *testing.T) *gorm.DB {
	t.Helper()
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
	return tx
}

func createTestUser(t *testing.T, tx *gorm.DB) User {
	t.Helper()
	user := User{GoogleSub: "test-" + uuid.NewString(), Name: "テスト"}
	if err := NewUsers(tx).Create(context.Background(), &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

// createTestDiary は eatenOn（YYYY-MM-DD）の日記を作る。
func createTestDiary(t *testing.T, tx *gorm.DB, userID uuid.UUID, eatenOn string, status MealType) Diary {
	t.Helper()
	day, err := time.Parse(time.DateOnly, eatenOn)
	if err != nil {
		t.Fatal(err)
	}
	diary := Diary{UserID: userID, EatenOn: day, Status: status, Body: "本文"}
	if err := NewDiaries(tx).Create(context.Background(), &diary); err != nil {
		t.Fatalf("create diary: %v", err)
	}
	return diary
}

// createTestPhoto は日記に position の写真を作る。
func createTestPhoto(t *testing.T, tx *gorm.DB, diaryID uuid.UUID, position int) Photo {
	t.Helper()
	id := uuid.New()
	photo := Photo{ID: id, DiaryID: diaryID, ObjectKey: "photos/" + diaryID.String() + "/" + id.String() + ".webp", Position: position}
	if err := tx.Create(&photo).Error; err != nil {
		t.Fatalf("create photo: %v", err)
	}
	return photo
}
