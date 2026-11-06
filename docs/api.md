# API 仕様書

designDoc「7.4 API 設計」の詳細。DB のテーブル定義は [table.md](table.md) を参照。

## 共通仕様

### ベース URL

| 環境 | URL |
| --- | --- |
| ローカル | `http://localhost:8080` |
| 本番 | `https://<サーバーのドメイン>/meshi-diary-api` |

### 認証

- 方式: フロントエンドが Google でログインして ID トークンを受け取り、`POST /signup` か `POST /login` に送る。API は ID トークンを検証し、セッション Cookie を発行する
- `/signup`・`/login`・`/health` 以外は、セッション Cookie が必要
- 未ログインでアクセスした場合は `401 UNAUTHORIZED` を返す
- 自分以外の日記・写真を指定した場合は `404 NOT_FOUND` を返す（存在するかどうかを知られないようにするため）

#### セッション Cookie

| 項目 | 値 |
| --- | --- |
| 名前 | `session` |
| 値 | `user_id` と有効期限を HMAC-SHA256 で署名した文字列。DB にはセッションを保存しない |
| 属性 | `HttpOnly; Secure; SameSite=Lax; Path=/`（本番では Apache が `Path=/meshi-diary-api/` に書き換える） |
| 有効期限 | 30 日（`Max-Age=2592000`） |

- **前提**: フロントエンドと API を同じサイトのドメインに置く（例: `meshi.example.com` と `api.meshi.example.com`）。ドメインのサイトが違うと Cookie がサードパーティ Cookie 扱いになり、ブラウザによっては送られない
- オリジンは違うので CORS は必要。`Access-Control-Allow-Origin` にフロントエンドのオリジンを指定し、`Access-Control-Allow-Credentials: true` を返す

#### 登録できるユーザー

利用者は自分だけなので、環境変数 `ALLOWED_EMAILS`（カンマ区切り）に書いたメールアドレスだけが `/signup` できる。登録したあとの本人確認には、メールアドレスではなく `google_sub` を使う（メールアドレスは変更できるため）。

### リクエスト・レスポンスの形式

- `Content-Type: application/json`、文字コードは UTF-8（写真のアップロードは各エンドポイントに書く）
- JSON のキーは snake_case（DB のカラム名に合わせる）
- 値がないフィールドは省略せず `null` を返す

### 型の書き方

| 表記 | 意味 | 例 |
| --- | --- | --- |
| `string` | 文字列 | `"ラーメン"` |
| `integer` | 整数 | `0` |
| `boolean` | 真偽値 | `true` |
| `uuid` | UUID の文字列 | `"0b7e4c1a-3f2d-4c8e-9a1b-2d3e4f5a6b7c"` |
| `date` | 日付（`YYYY-MM-DD`） | `"2026-11-06"` |
| `datetime` | 日時（RFC 3339、タイムゾーン付き） | `"2026-11-06T12:30:00+09:00"` |
| `enum` | 決まった値のどれか。値は説明欄に書く | `"LUNCH"` |
| `T[]` | T の配列 | `string[]` |
| `T \| null` | T または null | `string \| null` |

### エラーレスポンス

エラーのときは、どのエンドポイントも次の形で返す。

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "eaten_on は今日以前の日付にしてください"
  }
}
```

| HTTP ステータス | code | 使う場面 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | 入力値がルールに合わない |
| 401 | `UNAUTHORIZED` | ログインしていない、または ID トークンが正しくない |
| 403 | `FORBIDDEN` | 登録が許可されていないアカウント |
| 404 | `NOT_FOUND` | 対象がない、または自分のものではない |
| 409 | `CONFLICT` | 現在の状態では実行できない（例: 写真がすでに上限の枚数） |
| 413 | `PAYLOAD_TOO_LARGE` | ファイルが大きすぎる |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | 受け付けない形式のファイル |
| 500 | `INTERNAL_ERROR` | サーバー内部のエラー |

## エンドポイント一覧

| メソッド | パス | 概要 | 認証 |
| --- | --- | --- | --- |
| `POST` | `/signup` | ユーザー登録（初回だけ） | 不要 |
| `POST` | `/login` | ログイン | 不要 |
| `GET` | `/health` | 死活確認 | 不要 |

## エンドポイント詳細

<!-- 下の「テンプレート」をコピーして、エンドポイントごとに 1 つずつ追加する -->

### `POST /signup` — ユーザー登録

Google アカウントを users に登録し、そのままログインした状態にする。初めて使うときに1回だけ呼ぶ。

- 認証: 不要

#### パスパラメータ

なし

#### クエリパラメータ

なし

#### リクエストボディ

| フィールド | 型 | 必須 | 説明・制約 |
| --- | --- | --- | --- |
| id_token | `string` | ○ | Google から受け取った ID トークン（JWT） |

```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6Ij..."
}
```

#### レスポンス

`201 Created`。`Set-Cookie` でセッション Cookie を返す。

| フィールド | 型 | 説明 |
| --- | --- | --- |
| user.id | `uuid` | |
| user.name | `string` | ID トークンの `name` |
| user.created_at | `datetime` | |

```json
{
  "user": {
    "id": "0b7e4c1a-3f2d-4c8e-9a1b-2d3e4f5a6b7c",
    "name": "山田 太郎",
    "created_at": "2026-11-06T12:30:00+09:00"
  }
}
```

#### エラー

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | `id_token` がない |
| 401 | `UNAUTHORIZED` | ID トークンの検証に失敗した |
| 403 | `FORBIDDEN` | メールアドレスが `ALLOWED_EMAILS` にない、または `email_verified` が `false` |
| 409 | `CONFLICT` | 同じ `google_sub` のユーザーがすでに登録されている |

#### 備考

- 処理の順番
  1. ID トークンを検証する（署名、`iss` が `https://accounts.google.com`、`aud` が自分のクライアント ID、`exp` が切れていない）。Go では `google.golang.org/api/idtoken` の `idtoken.Validate` を使う
  2. `email_verified` が `true` で、`email` が `ALLOWED_EMAILS` にあるかを確認する
  3. `sub` を `google_sub`、`name` を `name` として users に INSERT する。`google_sub` の UNIQUE 違反になったら 409 を返す
  4. セッション Cookie を発行する
- メールアドレスは users に保存しない（許可するかどうかの判定にだけ使う）

---

### `POST /login` — ログイン

登録ずみの Google アカウントでログインし、セッション Cookie を発行する。

- 認証: 不要

#### パスパラメータ

なし

#### クエリパラメータ

なし

#### リクエストボディ

| フィールド | 型 | 必須 | 説明・制約 |
| --- | --- | --- | --- |
| id_token | `string` | ○ | Google から受け取った ID トークン（JWT） |

```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6Ij..."
}
```

#### レスポンス

`200 OK`。`Set-Cookie` でセッション Cookie を返す。

| フィールド | 型 | 説明 |
| --- | --- | --- |
| user.id | `uuid` | |
| user.name | `string` | |
| user.created_at | `datetime` | |

```json
{
  "user": {
    "id": "0b7e4c1a-3f2d-4c8e-9a1b-2d3e4f5a6b7c",
    "name": "山田 太郎",
    "created_at": "2026-11-06T12:30:00+09:00"
  }
}
```

#### エラー

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | `id_token` がない |
| 401 | `UNAUTHORIZED` | ID トークンの検証に失敗した |
| 404 | `NOT_FOUND` | その `google_sub` のユーザーが登録されていない（フロントエンドは `/signup` に案内する） |

#### 備考

- ID トークンの検証は `/signup` と同じ
- `google_sub` で users を探す。見つかったら、その `id` でセッション Cookie を発行する
- users の `name` はログインのたびには更新しない

---

### `GET /health` — 死活確認

API のプロセスが動いていてリクエストに応答できるかを確かめる。死活監視や、デプロイ後の確認に使う。

- 認証: 不要

#### パスパラメータ

なし

#### クエリパラメータ

なし

#### リクエストボディ

なし

#### レスポンス

`200 OK`。`Cache-Control: no-store` を付ける（プロキシにキャッシュされると、止まっていても 200 が返るため）。

| フィールド | 型 | 説明 |
| --- | --- | --- |
| status | `string` | 常に `"ok"` |

```json
{
  "status": "ok"
}
```

#### エラー

なし（プロセスが止まっているときは応答がなく、Apache が `502` / `503` を返す）

#### 備考

- **DB には接続しない**: Neon は一定時間アクセスがないとコンピュートを止める。監視のたびに DB に問い合わせると止まらなくなり、無料枠の稼働時間を使ってしまう。また、DB だけが落ちたときに API プロセスまで落ちていると判断されないようにするため
- バージョンや環境の情報は返さない（認証なしで誰でも見られるため）

---

## テンプレート

新しいエンドポイントを書くときは、このブロックをコピーして「エンドポイント詳細」に貼る。使わない項目は「なし」と書く。

````markdown
### `METHOD /path/{id}` — 概要を一言で

何をするエンドポイントか。どの画面から呼ぶか。

- 認証: 要 / 不要

#### パスパラメータ

| 名前 | 型 | 説明 |
| --- | --- | --- |
| id | `uuid` | |

#### クエリパラメータ

| 名前 | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| | | | |

#### リクエストボディ

| フィールド | 型 | 必須 | 説明・制約 |
| --- | --- | --- | --- |
| | | | |

```json
{
}
```

#### レスポンス

`200 OK`

| フィールド | 型 | 説明 |
| --- | --- | --- |
| | | |

```json
{
}
```

#### エラー

共通のエラー（401・500）以外に返すもの。

| ステータス | code | 返す条件 |
| --- | --- | --- |
| | | |

#### 備考

- 処理の順番、R2 との連携、冪等かどうかなど
````
