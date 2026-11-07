// Package postgres は PostgreSQL（Neon）への接続と、テーブルに対応する GORM のモデルを持つ。
//
// スキーマは db/migrations の SQL が正。GORM の AutoMigrate は使わない。
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// pingAfterIdle より長く使っていない接続だけ、再利用の前に生存確認（Ping）をする。
// Neon は5分使わないと止まり、接続も切れるので、それより短くする。
const pingAfterIdle = 4 * time.Minute

// Open は DB に接続し、疎通を確認する（gorm.Open が Ping する）。
func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:          dsn,
		OptionOpenDB: []stdlib.OptionOpenDB{stdlib.OptionShouldPing(shouldPing)},
	}), &gorm.Config{
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

// shouldPing は、プールの接続を再利用する前に生存確認をするかを決める。
// pgx の既定は「1秒以上使っていなければ Ping」で、Neon との1往復（約 200 ms）が SQL ごとに増えていた。
// 生存確認を省いた接続が切れていても、SQL を送る前に失敗するので database/sql が別の接続でやり直す。
func shouldPing(_ context.Context, p stdlib.ShouldPingParams) bool {
	return p.IdleDuration > pingAfterIdle
}
