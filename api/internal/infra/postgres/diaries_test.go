package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestDiaries は実際の DB で Diaries を確かめる。
// TEST_DATABASE_URL がないときはスキップする。トランザクションの中で行い、最後にロールバックする。
func TestDiaries(t *testing.T) {
	tx := openTestTx(t)
	ctx := context.Background()
	diaries := NewDiaries(tx)
	user := createTestUser(t, tx)

	t.Run("Create", func(t *testing.T) {
		info := "駅前のラーメン屋"
		diary := Diary{
			UserID:  user.ID,
			EatenOn: time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC),
			Status:  MealTypeLunch,
			Body:    "本文",
			Info:    &info,
		}
		if err := diaries.Create(ctx, &diary); err != nil {
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
	})

	t.Run("FindByID", func(t *testing.T) {
		diary := createTestDiary(t, tx, user.ID, "2026-11-07", MealTypeDinner)

		got, err := diaries.FindByID(ctx, user.ID, diary.ID)
		if err != nil {
			t.Fatalf("find own diary: %v", err)
		}
		if got.ID != diary.ID || got.Status != MealTypeDinner {
			t.Fatalf("got %+v", got)
		}

		other := createTestUser(t, tx)
		if _, err := diaries.FindByID(ctx, other.ID, diary.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("other user's diary: err = %v, want ErrNotFound", err)
		}
		if _, err := diaries.FindByID(ctx, user.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
			t.Fatalf("unknown diary: err = %v, want ErrNotFound", err)
		}
	})
	t.Run("Update", func(t *testing.T) {
		created := createTestDiary(t, tx, user.ID, "2026-11-08", MealTypeLunch)
		// DB はマイクロ秒までしか持たないので、保存された値で比べる
		diary, err := diaries.FindByID(ctx, user.ID, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		before := diary.UpdatedAt

		diary.EatenOn = time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
		diary.Status = MealTypeBreakfast
		diary.Info = nil
		diary.Body = "書き直した本文"
		if err := diaries.Update(ctx, &diary); err != nil {
			t.Fatalf("update: %v", err)
		}
		got, err := diaries.FindByID(ctx, user.ID, diary.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.EatenOn.Format(time.DateOnly) != "2026-11-09" || got.Status != MealTypeBreakfast || got.Info != nil || got.Body != "書き直した本文" {
			t.Fatalf("got %+v", got)
		}
		if !got.UpdatedAt.After(before) || !got.CreatedAt.Equal(diary.CreatedAt) {
			t.Fatalf("updated_at = %v (before %v), created_at = %v", got.UpdatedAt, before, got.CreatedAt)
		}

		other := createTestUser(t, tx)
		stolen := got
		stolen.UserID = other.ID
		if err := diaries.Update(ctx, &stolen); !errors.Is(err, ErrNotFound) {
			t.Fatalf("other user's diary: err = %v, want ErrNotFound", err)
		}
		missing := Diary{ID: uuid.New(), UserID: user.ID, EatenOn: got.EatenOn, Status: MealTypeOther, Body: "x"}
		if err := diaries.Update(ctx, &missing); !errors.Is(err, ErrNotFound) {
			t.Fatalf("unknown diary: err = %v, want ErrNotFound", err)
		}
	})
	t.Run("Delete removes the diary and its photos", func(t *testing.T) {
		diary := createTestDiary(t, tx, user.ID, "2026-11-10", MealTypeLunch)
		createTestPhoto(t, tx, diary.ID, 0)
		createTestPhoto(t, tx, diary.ID, 1)

		other := createTestUser(t, tx)
		if err := diaries.Delete(ctx, other.ID, diary.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("other user's diary: err = %v, want ErrNotFound", err)
		}

		if err := diaries.Delete(ctx, user.ID, diary.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}
		if _, err := diaries.FindByID(ctx, user.ID, diary.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("after delete: err = %v, want ErrNotFound", err)
		}
		photos, err := NewPhotos(tx).ListByDiary(ctx, diary.ID)
		if err != nil || len(photos) != 0 {
			t.Fatalf("photos after delete = %v, %v", photos, err)
		}
		if err := diaries.Delete(ctx, user.ID, diary.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("delete again: err = %v, want ErrNotFound", err)
		}
	})
}
