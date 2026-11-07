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

// FirstOfDiaries は日記ごとの1枚目（position = 0）の写真を、日記の id をキーにして返す。
// 写真がない日記はキーに入らない。
func (p *Photos) FirstOfDiaries(ctx context.Context, diaryIDs []uuid.UUID) (map[uuid.UUID]Photo, error) {
	first := map[uuid.UUID]Photo{}
	if len(diaryIDs) == 0 {
		return first, nil
	}
	var photos []Photo
	if err := p.db.WithContext(ctx).Where("diary_id IN ? AND position = 0", diaryIDs).Find(&photos).Error; err != nil {
		return nil, err
	}
	for _, ph := range photos {
		first[ph.DiaryID] = ph
	}
	return first, nil
}
