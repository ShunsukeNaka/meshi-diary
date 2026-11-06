package postgres

import (
	"time"

	"github.com/google/uuid"
)

// テーブル定義は docs/table.md を参照。
// 制約（NOT NULL・UNIQUE・FK）は DB 側にあるので、タグには型と既定値だけを書く。
// CreatedAt / UpdatedAt は GORM が作成・更新のたびに自動で入れる。

// MealType は DB の enum 型 meal_type に対応する。
type MealType string

const (
	MealTypeBreakfast MealType = "BREAKFAST"
	MealTypeLunch     MealType = "LUNCH"
	MealTypeDinner    MealType = "DINNER"
	MealTypeOther     MealType = "OTHER"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	GoogleSub string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Diary struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	EatenOn   time.Time `gorm:"type:date"`
	Status    MealType  `gorm:"type:meal_type;default:OTHER"`
	Body      string    `gorm:"default:''"`
	Info      *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Photo struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DiaryID   uuid.UUID `gorm:"type:uuid"`
	ObjectKey string
	Position  int
	Alt       *string
	CreatedAt time.Time
	UpdatedAt time.Time
}
