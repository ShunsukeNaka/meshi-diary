package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Diaries は diaries テーブルを読み書きする。
type Diaries struct {
	db *gorm.DB
}

func NewDiaries(db *gorm.DB) *Diaries {
	return &Diaries{db: db}
}

// Create は diary を保存し、id・created_at などを diary に入れる。
func (d *Diaries) Create(ctx context.Context, diary *Diary) error {
	return d.db.WithContext(ctx).Create(diary).Error
}

// FindByID は userID のユーザーの、id の日記を返す。
// ないとき・ほかのユーザーの日記のときは ErrNotFound を返す（どちらかは区別しない）。
func (d *Diaries) FindByID(ctx context.Context, userID, id uuid.UUID) (Diary, error) {
	var diary Diary
	err := d.db.WithContext(ctx).First(&diary, "id = ? AND user_id = ?", id, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Diary{}, ErrNotFound
	}
	return diary, err
}

// Update は diary の食事をした日・食事の区分・一言メモ・本文を保存し、updated_at を今の時刻にする。
// diary.ID と diary.UserID で日記を探す。ないとき・ほかのユーザーの日記のときは ErrNotFound を返す。
func (d *Diaries) Update(ctx context.Context, diary *Diary) error {
	now := time.Now()
	res := d.db.WithContext(ctx).Model(&Diary{}).
		Where("id = ? AND user_id = ?", diary.ID, diary.UserID).
		Updates(map[string]any{
			"eaten_on":   diary.EatenOn,
			"status":     diary.Status,
			"info":       diary.Info,
			"body":       diary.Body,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	diary.UpdatedAt = now
	return nil
}

// Delete は userID のユーザーの、id の日記を削除する。写真の行は ON DELETE CASCADE で一緒に消える。
// ないとき・ほかのユーザーの日記のときは ErrNotFound を返す。R2 のファイルは消さない。
func (d *Diaries) Delete(ctx context.Context, userID, id uuid.UUID) error {
	res := d.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&Diary{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListBetween は userID のユーザーの、食事をした日が from 以上 to 未満の日記を返す。
// 食事をした日の古い順、同じ日の中は作った順に並べる。
func (d *Diaries) ListBetween(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]Diary, error) {
	diaries := []Diary{}
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND eaten_on >= ? AND eaten_on < ?", userID, from, to).
		Order("eaten_on, created_at").
		Find(&diaries).Error
	return diaries, err
}

// ListByDate は userID のユーザーの、date の日記を返す。
// 食事の区分の順（enum の定義順: 朝食・昼食・夕食・その他）、同じ区分の中は作った順に並べる。
func (d *Diaries) ListByDate(ctx context.Context, userID uuid.UUID, date time.Time) ([]Diary, error) {
	diaries := []Diary{}
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND eaten_on = ?", userID, date).
		Order("status, created_at").
		Find(&diaries).Error
	return diaries, err
}
