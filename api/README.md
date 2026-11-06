# meshi-diary API

Go で書いた API サーバー。仕様は [docs/api.md](../docs/api.md)、テーブル定義は [docs/table.md](../docs/table.md) を参照。

## 構成

```
api/
├── cmd/server/          エントリーポイント（設定の読み込み、DB・R2 の準備、HTTP サーバーの起動）
└── internal/
    ├── auth/            ログイン（ID トークンの検証、セッション Cookie、認証ミドルウェア）
    ├── config/          環境変数の読み込み
    ├── handler/         ルーティングと HTTP ハンドラー
    ├── httpx/           JSON のレスポンスと共通のエラーレスポンス
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
| `GOOGLE_CLIENT_ID` | ○ | Google の OAuth クライアント ID。ID トークンの `aud` の検証に使う |
| `SESSION_SECRET` | ○ | セッション Cookie の署名に使う秘密鍵。32 バイト以上（`openssl rand -base64 32` で作る）。足りないと起動しない |
| `ALLOWED_EMAILS` | ○ | `/signup` できるメールアドレス（カンマ区切り） |

## デプロイ

master に `api/` の変更が入ると、GitHub Actions（[.github/workflows/deploy-api.yml](../.github/workflows/deploy-api.yml)）が次を行う。

1. `go vet` と `go test` を実行する（失敗したらデプロイしない）
2. 静的バイナリをビルドする（`CGO_ENABLED=0`。サーバーに Go は要らない）
3. SSH でサーバーの `~/services/meshi-diary/server.new` に送る
4. [deploy/remote-deploy.sh](deploy/remote-deploy.sh) をサーバーで実行し、`server` に置き換えて再起動する。`/health` が 15 秒以内に通らなければ、元のバイナリに戻して失敗にする

GitHub の Actions のページから手動でも実行できる（`workflow_dispatch`）。

| 項目 | 値 |
| --- | --- |
| ユーザー | `shunsuke`（SSH の接続も API の実行もこのユーザー） |
| 置き場所 | `~/services/meshi-diary/`（`server` と `.env`） |
| 起動 | systemd のサービス `meshi-diary-api`（[deploy/meshi-diary-api.service](deploy/meshi-diary-api.service)）。`/etc/systemd/system/` に置き、`shunsuke` で動かす |

`.env` は CI から送らない。サーバーに手で置く。マイグレーションも CI では行わないので、手元で `make migrate-up` を実行する。

### サーバーの初期設定（1回だけ）

サーバーに `shunsuke` でログインして行う。

```sh
# 1. 置き場所を作り、.env を置く（中身は .env.example を参照）
mkdir -p ~/services/meshi-diary
chmod 600 ~/services/meshi-diary/.env

# 2. サービスを登録する（meshi-diary-api.service はリポジトリの api/deploy/ からコピーする）
sudo cp meshi-diary-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable meshi-diary-api

# 3. CI がパスワードなしで再起動できるようにする（中身は api/deploy/sudoers-meshi-diary-api を貼る）
sudo visudo -f /etc/sudoers.d/meshi-diary-api
sudo -n systemctl restart meshi-diary-api   # パスワードを聞かれなければよい（バイナリがまだ無いので起動は失敗してよい）
```

CI が sudo で実行できるのは、[deploy/sudoers-meshi-diary-api](deploy/sudoers-meshi-diary-api) に書いた、引数まで決まったコマンドだけ。デプロイ用の鍵が漏れても、ほかのコマンドを root で実行されないようにしている。

最初のバイナリは CI のデプロイで送られ、そのときにサービスが起動する。

API は `https://<サーバーのドメイン>/meshi-diary-api/` で公開する（例: `/meshi-diary-api/health`）。Apache から転送する設定は [deploy/apache-meshi-diary-api.conf](deploy/apache-meshi-diary-api.conf)。<サーバーのドメイン> の HTTPS の VirtualHost に読み込ませる。

```sh
sudo cp apache-meshi-diary-api.conf /etc/apache2/meshi-diary-api.conf
sudo apache2ctl -S        # <サーバーのドメイン> の *:443 がどのファイルに書かれているかを見る
# そのファイルの <VirtualHost *:443> の中に、次の1行を足す
#   Include /etc/apache2/meshi-diary-api.conf
sudo a2enmod proxy proxy_http
sudo apache2ctl configtest
sudo systemctl reload apache2
curl -i https://<サーバーのドメイン>/meshi-diary-api/health
```

### デプロイ用の SSH 鍵と Secrets

手元で、デプロイ専用の鍵を作る（パスフレーズなし。ほかの用途には使わない）。

```sh
ssh-keygen -t ed25519 -N '' -C 'meshi-diary deploy' -f ./meshi-diary-deploy
# 公開鍵をサーバーの ~/.ssh/authorized_keys に足す
ssh-copy-id -i ./meshi-diary-deploy.pub shunsuke@<サーバー>
# サーバーのホスト鍵を取る（ポートが 22 以外なら -p <ポート> を付ける）
ssh-keyscan <サーバー>
```

GitHub のリポジトリの Settings > Secrets and variables > Actions に登録する。登録したら、手元の秘密鍵（`meshi-diary-deploy`）は消してよい。

| 名前 | 内容 |
| --- | --- |
| `SSH_HOST` | サーバーのホスト名か IP アドレス |
| `SSH_PORT` | SSH のポート。22 なら登録しなくてよい |
| `SSH_PRIVATE_KEY` | `meshi-diary-deploy`（秘密鍵）の中身 |
| `SSH_KNOWN_HOSTS` | `ssh-keyscan` の出力 |

### 状態を見る・手で操作する

```sh
systemctl status meshi-diary-api
journalctl -u meshi-diary-api -f             # ログ（見えなければ sudo を付ける）
sudo systemctl restart meshi-diary-api       # .env を書き換えたあとなど
```
