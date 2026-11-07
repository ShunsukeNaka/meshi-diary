package postgres

import (
	"context"
	"testing"
)

// TestPhotos は実際の DB で Photos を確かめる。TEST_DATABASE_URL がないときはスキップする。
func TestPhotos(t *testing.T) {
	tx := openTestTx(t)
	ctx := context.Background()
	photos := NewPhotos(tx)
	user := createTestUser(t, tx)

	t.Run("ListByDiary returns photos in position order", func(t *testing.T) {
		diary := createTestDiary(t, tx, user.ID, "2026-11-06", MealTypeLunch)
		p2 := createTestPhoto(t, tx, diary.ID, 2)
		p0 := createTestPhoto(t, tx, diary.ID, 0)
		p1 := createTestPhoto(t, tx, diary.ID, 1)

		got, err := photos.ListByDiary(ctx, diary.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 || got[0].ID != p0.ID || got[1].ID != p1.ID || got[2].ID != p2.ID {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("ListByDiary returns an empty slice when there are no photos", func(t *testing.T) {
		diary := createTestDiary(t, tx, user.ID, "2026-11-07", MealTypeOther)
		got, err := photos.ListByDiary(ctx, diary.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("got %#v, want empty slice", got)
		}
	})
}
