package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
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
	t.Run("FirstOfDiaries", func(t *testing.T) {
		withPhotos := createTestDiary(t, tx, user.ID, "2026-11-08", MealTypeLunch)
		createTestPhoto(t, tx, withPhotos.ID, 1)
		first := createTestPhoto(t, tx, withPhotos.ID, 0)
		noPhotos := createTestDiary(t, tx, user.ID, "2026-11-08", MealTypeDinner)

		got, err := photos.FirstOfDiaries(ctx, []uuid.UUID{withPhotos.ID, noPhotos.ID})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[withPhotos.ID].ID != first.ID {
			t.Fatalf("got %+v", got)
		}
		if empty, err := photos.FirstOfDiaries(ctx, nil); err != nil || len(empty) != 0 {
			t.Fatalf("no ids: %v, %v", empty, err)
		}
	})
	t.Run("AddMany appends photos after the existing ones", func(t *testing.T) {
		diary := createTestDiary(t, tx, user.ID, "2026-11-09", MealTypeLunch)
		existing := createTestPhoto(t, tx, diary.ID, 0)
		alt := "説明"
		added := []Photo{newTestPhoto(diary.ID, nil), newTestPhoto(diary.ID, &alt)}
		if err := photos.AddMany(ctx, diary.ID, added); err != nil {
			t.Fatal(err)
		}
		got, err := photos.ListByDiary(ctx, diary.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 || got[0].ID != existing.ID || got[1].ID != added[0].ID || got[2].ID != added[1].ID {
			t.Fatalf("got %+v", got)
		}
		if got[2].Position != 2 || got[2].Alt == nil || *got[2].Alt != alt {
			t.Fatalf("third = %+v", got[2])
		}
	})

	t.Run("AddMany rejects more than 10 photos and adds none", func(t *testing.T) {
		diary := createTestDiary(t, tx, user.ID, "2026-11-10", MealTypeLunch)
		for i := 0; i < 8; i++ {
			createTestPhoto(t, tx, diary.ID, i)
		}
		tx.SavePoint("too_many")
		err := photos.AddMany(ctx, diary.ID, []Photo{newTestPhoto(diary.ID, nil), newTestPhoto(diary.ID, nil), newTestPhoto(diary.ID, nil)})
		tx.RollbackTo("too_many")
		if !errors.Is(err, ErrTooManyPhotos) {
			t.Fatalf("err = %v, want ErrTooManyPhotos", err)
		}
		if got, _ := photos.ListByDiary(ctx, diary.ID); len(got) != 8 {
			t.Fatalf("photos = %d, want 8", len(got))
		}
		// ちょうど10枚までは追加できる
		if err := photos.AddMany(ctx, diary.ID, []Photo{newTestPhoto(diary.ID, nil), newTestPhoto(diary.ID, nil)}); err != nil {
			t.Fatalf("add up to 10: %v", err)
		}
	})

	t.Run("AddMany to an unknown diary is ErrNotFound", func(t *testing.T) {
		id := uuid.New()
		if err := photos.AddMany(ctx, id, []Photo{newTestPhoto(id, nil)}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func newTestPhoto(diaryID uuid.UUID, alt *string) Photo {
	id := uuid.New()
	return Photo{ID: id, ObjectKey: "photos/" + diaryID.String() + "/" + id.String() + ".webp", Alt: alt}
}
