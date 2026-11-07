package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Photos は photos テーブルを読み書きする。
type Photos struct {
	db *gorm.DB
}

func NewPhotos(db *gorm.DB) *Photos {
	return &Photos{db: db}
}

// ListByDiary は日記の写真を表示順（position の小さい順）に返す。写真がないときは空のスライスを返す。
func (p *Photos) ListByDiary(ctx context.Context, diaryID uuid.UUID) ([]Photo, error) {
	photos := []Photo{}
	err := p.db.WithContext(ctx).Where("diary_id = ?", diaryID).Order("position").Find(&photos).Error
	return photos, err
}
