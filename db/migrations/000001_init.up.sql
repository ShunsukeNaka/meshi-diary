-- 初期テーブル。定義と理由は docs/table.md を参照。
-- gen_random_uuid() は PostgreSQL 13 以降の標準機能なので拡張は不要。

BEGIN;

CREATE TYPE meal_type AS ENUM ('BREAKFAST', 'LUNCH', 'DINNER', 'OTHER');

CREATE TABLE users (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub  text        NOT NULL UNIQUE,
    name        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE diaries (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid        NOT NULL REFERENCES users (id),
    eaten_on    date        NOT NULL,
    status      meal_type   NOT NULL DEFAULT 'OTHER',
    body        text        NOT NULL DEFAULT '',
    info        text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- カレンダー表示で月ごとに取得するため
CREATE INDEX diaries_user_id_eaten_on_idx ON diaries (user_id, eaten_on);

CREATE TABLE photos (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    diary_id    uuid        NOT NULL REFERENCES diaries (id) ON DELETE CASCADE,
    object_key  text        NOT NULL UNIQUE,
    position    int         NOT NULL,
    alt         text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),

    -- position を詰め直す UPDATE の途中で一時的に重複するため、検査をコミット時まで遅らせる。
    -- このインデックスは diary_id だけの検索にも使えるので、diary_id 単独のインデックスは作らない。
    CONSTRAINT photos_diary_id_position_key
        UNIQUE (diary_id, position) DEFERRABLE INITIALLY DEFERRED
);

COMMIT;
