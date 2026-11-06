package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrNotFound は探したレコードがないときに返す。
var ErrNotFound = errors.New("record not found")

// Users は users テーブルを読み書きする。
type Users struct {
	db *gorm.DB
}

func NewUsers(db *gorm.DB) *Users {
	return &Users{db: db}
}

// FindByID は id のユーザーを返す。いないときは ErrNotFound を返す。
func (u *Users) FindByID(ctx context.Context, id uuid.UUID) (User, error) {
	var user User
	err := u.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrNotFound
	}
	return user, err
}

// FindByGoogleSub は google_sub のユーザーを返す。いないときは ErrNotFound を返す。
func (u *Users) FindByGoogleSub(ctx context.Context, sub string) (User, error) {
	var user User
	err := u.db.WithContext(ctx).First(&user, "google_sub = ?", sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrNotFound
	}
	return user, err
}

// Create は user を登録し、id・created_at などを user に入れる。
// google_sub がすでにあるときは gorm.ErrDuplicatedKey を返す（Open で TranslateError を有効にしているため）。
func (u *Users) Create(ctx context.Context, user *User) error {
	return u.db.WithContext(ctx).Create(user).Error
}
