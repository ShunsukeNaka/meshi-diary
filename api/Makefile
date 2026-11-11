# .env を読み込んでからコマンドを実行する
ENV := set -a; [ -f .env ] && . ./.env; set +a;
MIGRATE := go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1 -path ../db/migrations

.PHONY: run build test test-db migrate-up migrate-down backfill-thumbnails

run:
	$(ENV) go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

# DB への書き込み・読み出しを確かめる（データはロールバックされて残らない）
test-db:
	$(ENV) TEST_DATABASE_URL="$${TEST_DATABASE_URL:-$$DATABASE_URL}" go test -count=1 -run TestWriteAndRead -v ./internal/infra/postgres

migrate-up:
	$(ENV) $(MIGRATE) -database "$$DATABASE_URL" up

migrate-down:
	$(ENV) $(MIGRATE) -database "$$DATABASE_URL" down 1

# サムネイルがまだない写真のサムネイルを作る（#103）。ARGS=-dry-run で数えるだけにできる
backfill-thumbnails:
	$(ENV) go run ./cmd/backfill-thumbnails $(ARGS)
