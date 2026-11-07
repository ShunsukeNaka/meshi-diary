package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MaxPhotosPerDiary は1つの日記に登録できる写真の数。DB の制約ではなく、ここで確かめる（docs/table.md）。
const MaxPhotosPerDiary = 10

// ErrTooManyPhotos は、追加すると日記の写真が MaxPhotosPerDiary を超えるときに返す。
var ErrTooManyPhotos = errors.New("too many photos")

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

// AddMany は日記の今ある写真のうしろに、photos を渡した順に追加する。
// photos の Position はここで入れる。ID と ObjectKey は呼ぶ側で入れておく。
// 追加すると MaxPhotosPerDiary を超えるときは ErrTooManyPhotos を返し、1枚も追加しない。
func (p *Photos) AddMany(ctx context.Context, diaryID uuid.UUID, photos []Photo) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 同じ日記への追加が同時に来ても枚数を数えまちがえないように、日記の行をロックする
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&Diary{}, "id = ?", diaryID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		var count int64
		if err := tx.Model(&Photo{}).Where("diary_id = ?", diaryID).Count(&count).Error; err != nil {
			return err
		}
		if int(count)+len(photos) > MaxPhotosPerDiary {
			return ErrTooManyPhotos
		}
		for i := range photos {
			photos[i].DiaryID = diaryID
			photos[i].Position = int(count) + i
		}
		return tx.Create(&photos).Error
	})
}
