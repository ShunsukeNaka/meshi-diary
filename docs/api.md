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
| `DELETE` | `/diaries/{id}` | 日記を削除する | 要 |
| `GET` | `/diaries?date=` | 選んだ日の日記の一覧 | 要 |
| `POST` | `/diaries/{id}/photos` | 写真を追加する | 要 |
| `PATCH` | `/diaries/{id}/photos/{photoId}` | 写真の説明を変える | 要 |
| `DELETE` | `/diaries/{id}/photos/{photoId}` | 写真を削除する | 要 |
| `PUT` | `/diaries/{id}/photos/order` | 写真を並べ替える | 要 |
| `GET` | `/calendar` | カレンダーの月の日記 | 要 |
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

### `DELETE /diaries/{id}` — 日記を削除する

日記と、その写真をまとめて削除する。日記の詳細画面のメニューの「削除」で、確認したあとに呼ぶ。

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

`204 No Content`

#### エラー

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 404 | `NOT_FOUND` | 日記がない、ほかのユーザーの日記、または `id` が UUID の形ではない |

#### 備考

- 処理の順番
  1. 日記の写真の `object_key` を読んでおく
  2. 日記を削除する。photos の行は `ON DELETE CASCADE` で一緒に消える
  3. R2 から写真のファイルを削除する
- R2 のファイルの削除に失敗しても、ログに残して `204` を返す。DB からは消えているので、画面では削除できたものとして扱う（R2 にファイルが残るだけ）
- 削除したあとにもう一度呼ぶと `404`

---

### `GET /calendar` — カレンダーの月の日記

指定した月の、日記がある日と、その日の日記の一覧を返す。カレンダー画面を開いたときと、月を切り替えたときに呼ぶ。

- 認証: 要

#### パスパラメータ

なし

#### クエリパラメータ

| 名前 | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| month | `string` | ○ | 年と月（`YYYY-MM`）。例: `2026-11` |

#### リクエストボディ

なし

#### レスポンス

`200 OK`

| フィールド | 型 | 説明 |
| --- | --- | --- |
| month | `string` | 指定した年と月（`YYYY-MM`） |
| days | `day[]` | 日記がある日だけ。日付の古い順 |
| days[].date | `date` | |
| days[].diaries | `diary[]` | その日の日記。朝食・昼食・夕食・その他の順。同じ区分の中は作った順 |
| days[].diaries[].id | `uuid` | |
| days[].diaries[].info | `string \| null` | |
| days[].diaries[].body_head | `string` | 本文の最初の30文字。30文字より短いときは全文 |
| days[].diaries[].status | `enum` | |
| days[].diaries[].thumbnail_url | `string \| null` | 1枚目の写真の署名付き URL（1時間で切れる）。写真がないときは `null` |

```json
{
  "month": "2026-11",
  "days": [
    {
      "date": "2026-11-02",
      "diaries": [
        {
          "id": "8d4e2b1a-6c3f-4a7e-9b2d-1e0f3a4b5c6d",
          "info": null,
          "body_head": "家でカレーを作った。今日は、本格的なスパイスから作ってみたら",
          "status": "DINNER",
          "thumbnail_url": null
        }
      ]
    },
    {
      "date": "2026-11-06",
      "diaries": [
        {
          "id": "5c2a9f7e-1b3d-4e6f-8a9b-0c1d2e3f4a5b",
          "info": "駅前のラーメン屋",
          "body_head": "研究室のみんなで新しくできたお店へ。濃厚な豚骨スープで、替え",
          "status": "LUNCH",
          "thumbnail_url": "https://<R2 の署名付き URL>"
        },
        {
          "id": "2f7b9c0d-4e1a-4b3c-8d5e-6f7a8b9c0d1e",
          "info": null,
          "body_head": "コンビニのおにぎりの梅干し入り。時間がなかったので、研究室で",
          "status": "OTHER",
          "thumbnail_url": null
        }
      ]
    }
  ]
}
```

#### エラー

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | `month` がない、または `YYYY-MM` の形ではない |

#### 備考

- 日記がない日は `days` に入れない。フロントエンドは `days` にない日を、日記がない日として表示する
- `diaries` は、カレンダー画面で日付を選んだときの一覧に使う。一覧に出す分だけを返す（全文は `GET /diaries/{id}`）
- カレンダーのマスのサムネイルは、フロントエンドが `diaries` の順に見て、最初に `thumbnail_url` がある日記のものを使う
- `body_head` の文字数は、バイトではなく文字で数える

---

### `GET /diaries` — 選んだ日の日記の一覧

指定した日の日記を、食事の区分の順に返す。カレンダー画面の日付ごとの一覧は `GET /calendar` の `diaries` を使うので、今は Web から呼んでいない。

- 認証: 要

#### パスパラメータ

なし

#### クエリパラメータ

| 名前 | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| date | `date` | ○ | 食事をした日（`YYYY-MM-DD`） |

#### リクエストボディ

なし

#### レスポンス

`200 OK`

| フィールド | 型 | 説明 |
| --- | --- | --- |
| date | `date` | 指定した日 |
| diaries | `diary[]` | 朝食・昼食・夕食・その他の順。同じ区分の中は作った順。日記がないときは `[]` |
| diaries[].id | `uuid` | |
| diaries[].eaten_on | `date` | |
| diaries[].status | `enum` | |
| diaries[].info | `string \| null` | |
| diaries[].body | `string` | 本文の全文。一覧での省略はフロントエンドで行う |
| diaries[].created_at | `datetime` | |
| diaries[].updated_at | `datetime` | |
| diaries[].thumbnail_url | `string \| null` | 1枚目の写真の署名付き URL（1時間で切れる）。写真がないときは `null` |

```json
{
  "date": "2026-11-06",
  "diaries": [
    {
      "id": "5c2a9f7e-1b3d-4e6f-8a9b-0c1d2e3f4a5b",
      "eaten_on": "2026-11-06",
      "status": "LUNCH",
      "info": "駅前のラーメン屋",
      "body": "研究室のみんなで新しくできたお店へ。",
      "created_at": "2026-11-06T12:30:00+09:00",
      "updated_at": "2026-11-06T12:30:00+09:00",
      "thumbnail_url": "https://<R2 の署名付き URL>"
    }
  ]
}
```

#### エラー

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | `date` がない、または `YYYY-MM-DD` の形の正しい日付ではない |

#### 備考

- 自分の日記だけを返す

---

### `POST /diaries/{id}/photos` — 写真を追加する

日記に写真を追加する。複数枚をまとめて送れる。日記の編集画面の「保存」で、新しく選んだ写真があるときに呼ぶ。

- 認証: 要

#### パスパラメータ

| 名前 | 型 | 説明 |
| --- | --- | --- |
| id | `uuid` | 日記の id |

#### クエリパラメータ

なし

#### リクエストボディ

`Content-Type: multipart/form-data`

| フィールド | 型 | 必須 | 説明・制約 |
| --- | --- | --- | --- |
| photos | ファイル | ○ | 写真。1つ以上、同じ名前で複数送れる。JPEG・PNG・WebP。1枚 10 MB まで |
| alts | `string` | | 写真の説明。送るときは `photos` と同じ数・同じ順番で送る。空文字は説明なし |

```
photos=@lunch.jpg
photos=@ramen.jpg
alts=
alts=替え玉は硬めで。
```

#### レスポンス

`201 Created`。追加したあとの、日記の写真すべてを表示順で返す。

| フィールド | 型 | 説明 |
| --- | --- | --- |
| photos | `photo[]` | 形は `GET /diaries/{id}` の `photos` と同じ |

```json
{
  "photos": [
    {
      "id": "9d8c7b6a-5f4e-4d3c-2b1a-0f9e8d7c6b5a",
      "url": "https://<R2 の署名付き URL>",
      "alt": null,
      "position": 0
    },
    {
      "id": "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
      "url": "https://<R2 の署名付き URL>",
      "alt": "替え玉は硬めで。",
      "position": 1
    }
  ]
}
```

#### エラー

どれか1枚でもエラーになったときは、1枚も追加しない。

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | `multipart/form-data` として読めない、`photos` がない、`alts` の数が `photos` と違う |
| 404 | `NOT_FOUND` | 日記がない、ほかのユーザーの日記、または `id` が UUID の形ではない |
| 409 | `CONFLICT` | 追加すると、日記の写真が10枚を超える |
| 413 | `PAYLOAD_TOO_LARGE` | 10 MB を超える写真がある |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | JPEG・PNG・WebP ではない、壊れている、または縦か横が 16383 px を超える写真がある |

#### 備考

- 写真は WebP にして R2 に保存する（キーは `photos/{diary_id}/{photo_id}.webp`）。縦横の大きさは変えない
  - WebP の写真は、EXIF・XMP・アニメーションがなければ、変換せずに送られたまま保存する（フロントエンドは WebP にしてから送る）
  - JPEG・PNG の写真と、EXIF・XMP・アニメーションがある WebP の写真は、WebP に変換して保存する
- JPEG の EXIF の向きは、画素を回転させて反映する。EXIF・XMP（撮影場所なども含む）は保存しない
- 追加した写真は、今ある写真のうしろに、送った順に並べる
- 処理の順番
  1. 枚数・大きさ・形式を確かめる
  2. すべての写真を WebP にして（そのまま保存できる WebP は変換しない）、R2 に保存する
  3. photos に1つのトランザクションでまとめて INSERT する
  4. 2 か 3 で失敗したら、R2 に保存したファイルを消してエラーを返す
- フロントエンドからは Vercel の中継を通るので、リクエスト全体が 120 秒以内に終わる必要がある（Vercel の上限）

---

### `PATCH /diaries/{id}/photos/{photoId}` — 写真の説明を変える

写真の説明を変える。日記の編集画面の「保存」で、説明を変えた写真があるときに呼ぶ。

- 認証: 要

#### パスパラメータ

| 名前 | 型 | 説明 |
| --- | --- | --- |
| id | `uuid` | 日記の id |
| photoId | `uuid` | 写真の id |

#### クエリパラメータ

なし

#### リクエストボディ

| フィールド | 型 | 必須 | 説明・制約 |
| --- | --- | --- | --- |
| alt | `string \| null` | ○ | 写真の説明。`null`・空文字を送ると消す |

```json
{
  "alt": "替え玉は硬めで。"
}
```

#### レスポンス

`200 OK`。変えたあとの写真を返す。

| フィールド | 型 | 説明 |
| --- | --- | --- |
| photo | `photo` | 形は `GET /diaries/{id}` の `photos[]` と同じ |

```json
{
  "photo": {
    "id": "9d8c7b6a-5f4e-4d3c-2b1a-0f9e8d7c6b5a",
    "url": "https://<R2 の署名付き URL>",
    "alt": "替え玉は硬めで。",
    "position": 0
  }
}
```

#### エラー

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | JSON として読めない、または `alt` がない |
| 404 | `NOT_FOUND` | 日記・写真がない、ほかのユーザーの日記、写真がその日記のものではない、または id が UUID の形ではない |

#### 備考

- 写真のファイルは変えない

---

### `DELETE /diaries/{id}/photos/{photoId}` — 写真を削除する

日記の写真を1枚削除し、残りの写真の表示順を詰める。日記の編集画面の「保存」で、削除した写真があるときに呼ぶ。

- 認証: 要

#### パスパラメータ

| 名前 | 型 | 説明 |
| --- | --- | --- |
| id | `uuid` | 日記の id |
| photoId | `uuid` | 写真の id |

#### クエリパラメータ

なし

#### リクエストボディ

なし

#### レスポンス

`204 No Content`

#### エラー

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 404 | `NOT_FOUND` | 日記・写真がない、ほかのユーザーの日記、写真がその日記のものではない、または id が UUID の形ではない |

#### 備考

- 残った写真の `position` は 0 から順に詰め直す（例: 0・1・2 の 1 を消すと、2 が 1 になる）
- R2 のファイルの削除に失敗しても、ログに残して `204` を返す（`DELETE /diaries/{id}` と同じ）

---

### `PUT /diaries/{id}/photos/order` — 写真を並べ替える

日記の写真の表示順を、送った順番にする。日記の編集画面の「保存」で、写真を並べ替えたときに呼ぶ。

- 認証: 要

#### パスパラメータ

| 名前 | 型 | 説明 |
| --- | --- | --- |
| id | `uuid` | 日記の id |

#### クエリパラメータ

なし

#### リクエストボディ

| フィールド | 型 | 必須 | 説明・制約 |
| --- | --- | --- | --- |
| photo_ids | `uuid[]` | ○ | 日記の写真の id を、新しい順番に並べたもの。日記の写真すべてを、1回ずつ入れる |

```json
{
  "photo_ids": [
    "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
    "9d8c7b6a-5f4e-4d3c-2b1a-0f9e8d7c6b5a"
  ]
}
```

#### レスポンス

`200 OK`。並べ替えたあとの写真すべてを返す。形は `POST /diaries/{id}/photos` のレスポンスと同じ。

#### エラー

| ステータス | code | 返す条件 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | JSON として読めない、`photo_ids` がない、日記の写真が足りない・多い・重なっている、ほかの日記の写真が入っている |
| 404 | `NOT_FOUND` | 日記がない、ほかのユーザーの日記、または `id` が UUID の形ではない |

#### 備考

- `photo_ids` の順に `position` を 0 から入れ直す（1つのトランザクション）
- 写真がない日記には `"photo_ids": []` を送ると `200` になる

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
