-- カレンダーと一覧に出す、写真を小さくしたサムネイルの R2 のキー。定義と理由は docs/table.md を参照。
-- サムネイルがまだない写真（このマイグレーションより前に保存した写真）は NULL で、そのときは元の写真を表示する。

ALTER TABLE photos ADD COLUMN thumbnail_key text UNIQUE;
