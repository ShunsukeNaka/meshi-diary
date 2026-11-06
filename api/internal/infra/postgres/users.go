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
