# テーブル定義

designDoc「7.3 データモデル」用。ドメインモデルは [domain-model.svg](domain-model.svg)（元データは [domain-model.dio](domain-model.dio)）を参照。
ER図は [er-diagram.dio](er-diagram.dio)（draw.io）。

## users

| カラム | 型 | 制約 | 説明 | 理由 |
| --- | --- | --- | --- | --- |
| id | uuid | PK, default `gen_random_uuid()` | | |
| google_sub | text | NOT NULL, UNIQUE | Google アカウントを一意に表す ID | メールアドレスは変更できるため、変わらない `sub` で本人を判定する |
| name | text | NOT NULL | 表示名 | |
| created_at | timestamptz | NOT NULL, default `now()` | | |
| updated_at | timestamptz | NOT NULL, default `now()` | | |

## diaries

| カラム | 型 | 制約 | 説明 | 理由 |
| --- | --- | --- | --- | --- |
| id | uuid | PK, default `gen_random_uuid()` | | |
| user_id | uuid | NOT NULL, FK → users.id | | |
| eaten_on | date | NOT NULL | 食事をした日（カレンダーの日付） | 日付（`YYYY-MM-DD`）だけ表せればよい。`timestamptz` だと UTC への変換で日付がずれることがあるため `date` にする |
| status | enum | NOT NULL, default `'OTHER'`<br>値: `BREAKFAST` / `LUNCH` / `DINNER` / `OTHER` | 朝食・昼食・夕食・その他のどれか | `text` + CHECK 制約だと ORM で string 型になるため enum にする。値を今後増やす予定もない |
| body | text | NOT NULL, default `''` | 本文 | |
| info | text | | 食事をした場所や食べたものの一言メモ | |
| created_at | timestamptz | NOT NULL, default `now()` | | |
| updated_at | timestamptz | NOT NULL, default `now()` | | |

- **インデックス**: `(user_id, eaten_on)` — カレンダー表示で月ごとに取得するため
- 同じ日に複数の日記を書ける（昼と夜など）。1日1件に制限する場合は `(user_id, eaten_on)` を UNIQUE にする

## photos

| カラム | 型 | 制約 | 説明 | 理由 |
| --- | --- | --- | --- | --- |
| id | uuid | PK, default `gen_random_uuid()` | | |
| diary_id | uuid | NOT NULL, FK → diaries.id, ON DELETE CASCADE | | |
| object_key | text | NOT NULL, UNIQUE | R2 のオブジェクトキー。例: `photos/{diary_id}/{photo_id}.webp` | URL ではなくキーを保存し、表示のたびに署名付き URL を発行する。アップロードされた画像は webp に変換して保存する |
| thumbnail_key | text | UNIQUE | サムネイルの R2 のオブジェクトキー。例: `thumbnails/{diary_id}/{photo_id}.webp` | カレンダーと一覧のマスは小さいので、元の写真ではなく短辺 192 px に縮めたものを表示する。NULL のときは元の写真を表示する（サムネイルを作る前に保存した写真） |
| position | int | NOT NULL | 表示順（0 始まり） | |
| alt | text | | 写真の説明・メモ | |
| created_at | timestamptz | NOT NULL, default `now()` | | |
| updated_at | timestamptz | NOT NULL, default `now()` | | |

- **UNIQUE**: `(diary_id, position)` DEFERRABLE INITIALLY DEFERRED — 同じ日記の中で表示順が重複しないようにする
  - DEFERRABLE にする理由: 写真を削除して position を詰め直すとき、PostgreSQL は通常の UNIQUE を1行ずつ検査するため、1回の UPDATE の途中で一時的に重複してエラーになる。DEFERRED にすればコミット時にまとめて検査される
  - この UNIQUE のインデックスは `diary_id` だけでの検索にも使えるので、`diary_id` 単独のインデックスは作らない

## 設計メモ

- **サムネイルにする写真**: カレンダーに出す「その日の写真の1枚目」は、その日の日記を `created_at` の古い順に見て、写真がある最初の日記の `position = 0` の写真とする。専用のカラムは持たず、一覧取得のクエリで求める
- **サムネイルの画像**: 写真を保存するときに、短辺 192 px の WebP を作って `thumbnail_key` に保存する。前からある写真は `thumbnail_key` が NULL なので、API は元の写真の URL を返す。前からある写真のサムネイルは `api/cmd/backfill-thumbnails` で作る（手順は api/README.md）
- **署名付き URL は保存しない**: 有効期限があるため。DB には `object_key`・`thumbnail_key` だけを持つ
- **写真の削除**: 日記を削除すると photos の行は CASCADE で消えるが、R2 のオブジェクトは残る。そのため API 側で R2 のオブジェクト（写真とサムネイル）も削除する
- **users テーブルを持つ理由**: 利用者は自分だけだが、ログインした人が自分かどうか（`google_sub` が一致するか）をこのテーブルで確認する
- **updated_at の更新**: `default now()` が効くのは INSERT のときだけ。UPDATE のときはアプリ（またはトリガー）で `now()` を入れる
- **DB では持たないルール**: 写真は10枚までなどのルールは、DB の制約ではなくドメインモデル（[domain-model.svg](domain-model.svg)）の側で確かめる
