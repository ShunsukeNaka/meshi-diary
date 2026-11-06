// Package postgres は PostgreSQL（Neon）への接続と、テーブルに対応する GORM のモデルを持つ。
//
// スキーマは db/migrations の SQL が正。GORM の AutoMigrate は使わない。
package postgres

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open は DB に接続し、疎通を確認する（gorm.Open が Ping する）。
func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// UNIQUE 違反を gorm.ErrDuplicatedKey に変換する（例: signup で 409 を返す判定に使う）
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	// 利用者は1人なので接続は少なくてよい。使っていない接続は長く持ち続けない。
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	return db, nil
}
