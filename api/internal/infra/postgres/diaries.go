package postgres

import (
	"context"

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
