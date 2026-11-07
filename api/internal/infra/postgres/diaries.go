package postgres

import (
	"context"
	"errors"

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
