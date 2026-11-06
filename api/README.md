# meshi-diary API

Go で書いた API サーバー。仕様は [docs/api.md](../docs/api.md)、テーブル定義は [docs/table.md](../docs/table.md) を参照。

## 構成

```
api/
├── cmd/server/          エントリーポイント（設定の読み込み、DB・R2 の準備、HTTP サーバーの起動）
└── internal/
    ├── config/          環境変数の読み込み
    ├── handler/         ルーティングと HTTP ハンドラー
    └── infra/
        ├── postgres/    PostgreSQL（Neon）への接続と GORM のモデル
        └── r2/          Cloudflare R2（S3 互換 API）の操作
```

- ルーティングは標準の `net/http`（Go 1.22 以降の `"GET /health"` のような書き方）を使う
- DB のスキーマは [db/migrations](../db/migrations) の SQL が正。GORM の AutoMigrate は使わない

## セットアップ

必要なもの: Go 1.25 以降（依存ライブラリが要求するため。古い Go でも `GOTOOLCHAIN=auto` なら自動で取得される）

```sh
cd api
cp .env.example .env   # 値を入れる
make migrate-up        # db/migrations を DB に適用する
make run               # http://127.0.0.1:8080 で起動
curl -i http://127.0.0.1:8080/health
```

## make のコマンド

| コマンド | 内容 |
| --- | --- |
| `make run` | `.env` を読み込んでサーバーを起動する |
| `make build` | `bin/server` を作る |
| `make test` | テストを実行する（DB のテストはスキップされる） |
| `make test-db` | DB への書き込み・読み出しを確かめる。トランザクションをロールバックするのでデータは残らない。`TEST_DATABASE_URL` がなければ `DATABASE_URL` を使う |
| `make migrate-up` | マイグレーションを適用する |
| `make migrate-down` | マイグレーションを1つ戻す |

マイグレーションは [golang-migrate](https://github.com/golang-migrate/migrate) を `go run` で実行するので、別にインストールしなくてよい。

## 環境変数

| 名前 | 必須 | 説明 |
| --- | --- | --- |
| `LISTEN_ADDR` | | 待ち受けるアドレス。既定は `127.0.0.1:8080`（Apache から転送されたリクエストだけを受ける） |
| `DATABASE_URL` | ○ | Neon の接続文字列（`?sslmode=require` を付ける） |
| `R2_ACCOUNT_ID` | ○ | Cloudflare のアカウント ID |
| `R2_ACCESS_KEY_ID` | ○ | R2 の API トークンのアクセスキー ID |
| `R2_SECRET_ACCESS_KEY` | ○ | R2 の API トークンのシークレット |
| `R2_BUCKET` | ○ | 写真を保存するバケット名 |
