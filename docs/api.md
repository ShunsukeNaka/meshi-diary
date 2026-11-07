# API 仕様書

designDoc「7.4 API 設計」の詳細。DB のテーブル定義は [table.md](table.md) を参照。

## 共通仕様

### ベース URL

| 環境 | URL |
| --- | --- |
| ローカル | `http://localhost:8080` |
| 本番 | `https://<サーバーのドメイン>/meshi-diary-api` |

- フロントエンドからは、上の URL を直接呼ばない。フロントエンド（Vercel）の `/meshi-diary-api/*` に送り、Vercel が上の URL に中継する（`web/next.config.ts` の `rewrites`）
  - 例: フロントエンドの `/meshi-diary-api/login` → API の `/login`
  - 中継する先は、フロントエンドの環境変数 `API_BASE_URL` で決める（ローカルは `http://localhost:8080`、本番は上の URL）
- フロントエンドと API は同じサイトのドメインに置けないため、中継してブラウザからは同じオリジンへのリクエストにしている（理由は「セッション Cookie」を参照）

### 認証

- 方式: フロントエンドが Google でログインして ID トークンを受け取り、`POST /signup` か `POST /login` に送る。API は ID トークンを検証し、セッション Cookie を発行する
- `/signup`・`/login`・`/logout`・`/health` 以外は、セッション Cookie が必要
- 未ログインでアクセスした場合は `401 UNAUTHORIZED` を返す
- 自分以外の日記・写真を指定した場合は `404 NOT_FOUND` を返す（存在するかどうかを知られないようにするため）

#### セッション Cookie

| 項目 | 値 |
| --- | --- |
| 名前 | `session` |
| 値 | `user_id` と有効期限を HMAC-SHA256 で署名した文字列。DB にはセッションを保存しない |
| 属性 | `HttpOnly; Secure; SameSite=Lax; Path=/`（本番では Apache が `Path=/meshi-diary-api/` に書き換える） |
| 有効期限 | 30 日（`Max-Age=2592000`） |

- **前提**: フロントエンドから API へのリクエストは、Vercel の `/meshi-diary-api/*` を通して中継する（「ベース URL」を参照）
  - ブラウザからは同じオリジンへのリクエストになるので、Cookie はフロントエンドのドメインの Cookie として保存され、`SameSite=Lax` のまま送られる。フロントエンドと API のサイトが違うまま直接呼ぶと、Cookie がサードパーティ Cookie 扱いになり、Safari などでは送られない
  - 本番で Apache が書き換える Path（`/meshi-diary-api/`）と、中継のパス（`/meshi-diary-api`）がそろっているので、Cookie は API へのリクエストにだけ付く
- 同じオリジンになるので CORS は設定しない

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
| `GET` | `/me` | ログイン中のユーザー | 要 |
| `POST` | `/logout` | ログアウト | 不要 |
| `POST` | `/diaries` | 日記を作る | 要 |
| `GET` | `/diaries/{id}` | 日記の詳細 | 要 |
| `PATCH` | `/diaries/{id}` | 日記を編集する | 要 |
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

### `GET /me` — ログイン中のユーザー

ログインしているかどうかと、ログイン中のユーザーを返す。フロントエンドが画面を開いたときに呼び、`401` ならログイン画面に移動する。

- 認証: 要

#### パスパラメータ

なし

#### クエリパラメータ

なし

#### リクエストボディ

なし

#### レスポンス

`200 OK`

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

共通のエラー（401・500）以外に返すものはない。`401 UNAUTHORIZED` は次のときに返す。

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 401 | `UNAUTHORIZED` | セッション Cookie がない・正しくない・期限切れ |
| 401 | `UNAUTHORIZED` | セッション Cookie は正しいが、その `user_id` のユーザーが users にいない |

#### 備考

- セッション Cookie の `user_id` で users を探す
- ユーザーがいないとき（DB から消した場合など）は、ログインしていないのと同じ扱いにする。そのときはセッション Cookie も消す（`Max-Age=0`）ので、次からは Cookie なしの `401` になる

---

### `POST /logout` — ログアウト

セッション Cookie を消す。ログアウトのボタンから呼び、そのあとフロントエンドはログイン画面に移動する。

- 認証: 不要

#### パスパラメータ

なし

#### クエリパラメータ

なし

#### リクエストボディ

なし

#### レスポンス

`204 No Content`。`Set-Cookie` でセッション Cookie を消す（`Max-Age=0`）。

#### エラー

なし

#### 備考

- セッション Cookie がない・正しくない・期限切れのときも、エラーにせず `204` を返す。何度呼んでも結果は同じ（冪等）なので、フロントエンドは結果を見ずにログイン画面に移動してよい
- セッションは DB に保存していないので、消すのはブラウザの Cookie だけ。Cookie の値を控えていれば、有効期限まではその値で使えてしまう（利用者が自分だけなので許容する）

---

### `POST /diaries` — 日記を作る

ログイン中のユーザーの日記を1件作る。日記の編集画面で、新しく書いた日記を保存するときに呼ぶ。

- 認証: 要

#### パスパラメータ

なし

#### クエリパラメータ

なし

#### リクエストボディ

| フィールド | 型 | 必須 | 説明・制約 |
| --- | --- | --- | --- |
| eaten_on | `date` | ○ | 食事をした日 |
| status | `enum` | | 食事の区分。`BREAKFAST`（朝食）・`LUNCH`（昼食）・`DINNER`（夕食）・`OTHER`（その他）。省略・`null`・空文字のときは `OTHER` |
| info | `string \| null` | | 一言メモ（食事をした場所や食べたもの）。省略・`null`・空文字のときは `null` として保存する |
| body | `string` | ○ | 本文。空白以外の文字が1文字以上 |

```json
{
  "eaten_on": "2026-11-06",
  "status": "LUNCH",
  "info": "駅前のラーメン屋",
  "body": "久しぶりに会って、仕事の話をした。"
}
```

#### レスポンス

`201 Created`

| フィールド | 型 | 説明 |
| --- | --- | --- |
| diary.id | `uuid` | |
| diary.eaten_on | `date` | |
| diary.status | `enum` | |
| diary.info | `string \| null` | |
| diary.body | `string` | |
| diary.created_at | `datetime` | |
| diary.updated_at | `datetime` | |

```json
{
  "diary": {
    "id": "5c2a9f7e-1b3d-4e6f-8a9b-0c1d2e3f4a5b",
    "eaten_on": "2026-11-06",
    "status": "LUNCH",
    "info": "駅前のラーメン屋",
    "body": "久しぶりに会って、仕事の話をした。",
    "created_at": "2026-11-06T12:30:00+09:00",
    "updated_at": "2026-11-06T12:30:00+09:00"
  }
}
```

#### エラー

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | JSON として読めない |
| 400 | `VALIDATION_ERROR` | `eaten_on` がない、または `YYYY-MM-DD` の形の正しい日付ではない |
| 400 | `VALIDATION_ERROR` | `status` が決まった値のどれでもない |
| 400 | `VALIDATION_ERROR` | `body` がない、または空白だけ |
| 413 | `PAYLOAD_TOO_LARGE` | リクエストボディが 1 MB を超える |

#### 備考

- `status` を選び忘れても困らないので、省略できるようにする（DB の既定値と同じ `OTHER`）
- 本文の「空白」は、全角スペース・改行・タブも含む
- 本文に文字数の制限はないが、リクエストボディは 1 MB までにする
- 同じ日に何件でも作れる（昼と夜など）
- 写真は、日記を作ったあとに別のエンドポイントで追加する

---

### `GET /diaries/{id}` — 日記の詳細

日記1件と、その写真を返す。日記の詳細画面と、編集画面を開いたときに呼ぶ。

- 認証: 要

#### パスパラメータ

| 名前 | 型 | 説明 |
| --- | --- | --- |
| id | `uuid` | 日記の id |

#### クエリパラメータ

なし

#### リクエストボディ

なし

#### レスポンス

`200 OK`

| フィールド | 型 | 説明 |
| --- | --- | --- |
| diary.id | `uuid` | |
| diary.eaten_on | `date` | |
| diary.status | `enum` | `BREAKFAST`・`LUNCH`・`DINNER`・`OTHER` |
| diary.info | `string \| null` | |
| diary.body | `string` | |
| diary.created_at | `datetime` | |
| diary.updated_at | `datetime` | |
| diary.photos | `photo[]` | 表示順（`position` の小さい順）。写真がないときは `[]` |
| diary.photos[].id | `uuid` | |
| diary.photos[].url | `string` | 表示用の署名付き URL。1時間で切れる |
| diary.photos[].alt | `string \| null` | 写真の説明 |
| diary.photos[].position | `integer` | 表示順（0 始まり） |

```json
{
  "diary": {
    "id": "5c2a9f7e-1b3d-4e6f-8a9b-0c1d2e3f4a5b",
    "eaten_on": "2026-11-06",
    "status": "LUNCH",
    "info": "駅前のラーメン屋",
    "body": "久しぶりに会って、仕事の話をした。",
    "created_at": "2026-11-06T12:30:00+09:00",
    "updated_at": "2026-11-06T12:30:00+09:00",
    "photos": [
      {
        "id": "9d8c7b6a-5f4e-4d3c-2b1a-0f9e8d7c6b5a",
        "url": "https://<R2 の署名付き URL>",
        "alt": "替え玉は硬めで。",
        "position": 0
      }
    ]
  }
}
```

#### エラー

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 404 | `NOT_FOUND` | 日記がない、ほかのユーザーの日記、または `id` が UUID の形ではない |

#### 備考

- 写真の `url` は R2 の署名付き URL。期限が切れたら、もう一度このエンドポイントを呼んで取り直す
- `url` は呼ぶたびに変わるので、フロントエンドは写真を `id` で見分ける

---

### `PATCH /diaries/{id}` — 日記を編集する

日記の食事をした日・食事の区分・一言メモ・本文を変える。送ったフィールドだけを変える。日記の編集画面の「保存」で呼ぶ。

- 認証: 要

#### パスパラメータ

| 名前 | 型 | 説明 |
| --- | --- | --- |
| id | `uuid` | 日記の id |

#### クエリパラメータ

なし

#### リクエストボディ

どのフィールドも省略できる。省略したフィールドは変えない。

| フィールド | 型 | 必須 | 説明・制約 |
| --- | --- | --- | --- |
| eaten_on | `date` | | 食事をした日。`null` は不可 |
| status | `enum` | | 食事の区分。`null`・空文字のときは `OTHER` |
| info | `string \| null` | | 一言メモ。`null`・空文字を送ると消す |
| body | `string` | | 本文。空白以外の文字が1文字以上。`null` は不可 |

```json
{
  "info": null,
  "body": "久しぶりに会って、仕事の話をした。次は鍋に行く。"
}
```

#### レスポンス

`200 OK`。変えたあとの日記を返す。形は `POST /diaries` のレスポンスと同じ（写真は含まない）。

```json
{
  "diary": {
    "id": "5c2a9f7e-1b3d-4e6f-8a9b-0c1d2e3f4a5b",
    "eaten_on": "2026-11-06",
    "status": "LUNCH",
    "info": null,
    "body": "久しぶりに会って、仕事の話をした。次は鍋に行く。",
    "created_at": "2026-11-06T12:30:00+09:00",
    "updated_at": "2026-11-07T09:00:00+09:00"
  }
}
```

#### エラー

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | JSON として読めない |
| 400 | `VALIDATION_ERROR` | `eaten_on` が `null`、または `YYYY-MM-DD` の形の正しい日付ではない |
| 400 | `VALIDATION_ERROR` | `status` が決まった値のどれでもない |
| 400 | `VALIDATION_ERROR` | `body` が `null`、または空白だけ |
| 404 | `NOT_FOUND` | 日記がない、ほかのユーザーの日記、または `id` が UUID の形ではない |
| 413 | `PAYLOAD_TOO_LARGE` | リクエストボディが 1 MB を超える |

#### 備考

- 入力のルールは `POST /diaries` と同じ
- 写真の変更は、写真のエンドポイントで行う
- `updated_at` は、送ったフィールドの値が変わらなくても更新する

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
