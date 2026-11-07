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
		if err := lockDiary(tx, diaryID); err != nil {
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

// UpdateAlt は日記 diaryID の写真 photoID の説明を変え、変えたあとの写真を返す。
// 写真がない・その日記の写真ではないときは ErrNotFound を返す。
func (p *Photos) UpdateAlt(ctx context.Context, diaryID, photoID uuid.UUID, alt *string) (Photo, error) {
	res := p.db.WithContext(ctx).Model(&Photo{}).
		Where("id = ? AND diary_id = ?", photoID, diaryID).
		Updates(map[string]any{"alt": alt, "updated_at": gorm.Expr("now()")})
	if res.Error != nil {
		return Photo{}, res.Error
	}
	if res.RowsAffected == 0 {
		return Photo{}, ErrNotFound
	}
	var photo Photo
	err := p.db.WithContext(ctx).First(&photo, "id = ?", photoID).Error
	return photo, err
}

// Delete は日記 diaryID の写真 photoID を削除し、うしろの写真の position を1つずつ詰める。削除した写真を返す。
// 写真がない・その日記の写真ではないときは ErrNotFound を返す。R2 のファイルは消さない。
func (p *Photos) Delete(ctx context.Context, diaryID, photoID uuid.UUID) (Photo, error) {
	var photo Photo
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockDiary(tx, diaryID); err != nil {
			return err
		}
		if err := tx.First(&photo, "id = ? AND diary_id = ?", photoID, diaryID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if err := tx.Delete(&Photo{}, "id = ?", photoID).Error; err != nil {
			return err
		}
		// (diary_id, position) の UNIQUE はコミット時に検査されるので、途中で重なってもよい
		return tx.Model(&Photo{}).
			Where("diary_id = ? AND position > ?", diaryID, photo.Position).
			Updates(map[string]any{"position": gorm.Expr("position - 1"), "updated_at": gorm.Expr("now()")}).Error
	})
	return photo, err
}

// lockDiary は日記の行を SELECT ... FOR UPDATE でロックする。同じ日記の写真を同時に変えたときに、
// 枚数や position を数えまちがえないようにするため。日記がないときは ErrNotFound を返す。
func lockDiary(tx *gorm.DB, diaryID uuid.UUID) error {
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&Diary{}, "id = ?", diaryID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
