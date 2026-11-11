// backfill-thumbnails は、サムネイルがまだない（photos.thumbnail_key が NULL の）写真のサムネイルを作る。
// R2 から写真を読み、短辺 192 px の WebP を thumbnails/{diary_id}/{photo_id}.webp に保存して、thumbnail_key に入れる。
//
// 使い方（api/ で。.env の DB と R2 を使う）:
//
//	make backfill-thumbnails ARGS=-dry-run   # 対象の枚数を数えるだけ
//	make backfill-thumbnails
//
// 何度実行してもよい。失敗した写真は thumbnail_key が NULL のままなので、もう一度実行すると作り直す。
// API を止めずに実行してよい。thumbnail_key が、ないファイルを指すことはない。
// ただし、実行中にその写真や日記を消すと、まれに R2 にサムネイルのファイルだけが残ることがある（表示には影響しない）。
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/config"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/handler"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/imageconv"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/r2"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "対象の写真を数えるだけで、サムネイルは作らない")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err := run(*dryRun); err != nil {
		slog.Error("backfill thumbnails", "err", err)
		os.Exit(1)
	}
}

func run(dryRun bool) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadStores()
	if err != nil {
		return err
	}
	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	storage, err := r2.New(ctx, cfg.R2)
	if err != nil {
		return err
	}

	b := backfiller{photos: postgres.NewPhotos(db), files: storage}
	if dryRun {
		targets, err := b.photos.ListWithoutThumbnail(ctx)
		if err != nil {
			return err
		}
		slog.Info("dry run", "photos_without_thumbnail", len(targets))
		return nil
	}
	res, err := b.run(ctx)
	slog.Info("finished", "created", res.created, "skipped", res.skipped, "failed", res.failed)
	if err != nil {
		return err
	}
	if res.failed > 0 {
		return fmt.Errorf("%d photos failed (run again to retry)", res.failed)
	}
	return nil
}

type photoStore interface {
	ListWithoutThumbnail(ctx context.Context) ([]postgres.Photo, error)
	SetThumbnailKey(ctx context.Context, photoID uuid.UUID, key string) (bool, error)
}

type fileStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key, contentType string, body io.Reader) error
	Delete(ctx context.Context, key string) error
}

type backfiller struct {
	photos photoStore
	files  fileStore
}

type result struct {
	created int // サムネイルを作って thumbnail_key に入れた
	skipped int // 作っている間に写真が消された
	failed  int // 失敗した（thumbnail_key は NULL のまま）
}

// run はサムネイルがない写真を古い順に1枚ずつ処理する。1枚が失敗しても、残りの写真は続ける。
// 写真の一覧が読めないとき・止められたとき（Ctrl-C）は、そこでやめてエラーを返す。
func (b backfiller) run(ctx context.Context) (result, error) {
	var res result
	targets, err := b.photos.ListWithoutThumbnail(ctx)
	if err != nil {
		return res, err
	}
	slog.Info("start", "photos", len(targets))
	for i, p := range targets {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		set, err := b.one(ctx, p)
		switch {
		case err != nil:
			res.failed++
			slog.Error("photo failed", "photo_id", p.ID, "object_key", p.ObjectKey, "err", err)
		case set:
			res.created++
		default:
			res.skipped++
			slog.Info("photo was deleted, skipped", "photo_id", p.ID)
		}
		if (i+1)%10 == 0 {
			slog.Info("progress", "done", i+1, "total", len(targets))
		}
	}
	return res, nil
}

// one は写真 p のサムネイルを作って保存し、thumbnail_key に入れる。入れたときは true を返す。
// 作っている間に写真が消された（thumbnail_key に入れられなかった）ときは、保存したサムネイルを消して false を返す。
func (b backfiller) one(ctx context.Context, p postgres.Photo) (bool, error) {
	data, err := b.files.Get(ctx, p.ObjectKey)
	if err != nil {
		return false, err
	}
	thumb, err := imageconv.Thumbnail(data)
	if err != nil {
		return false, err
	}
	// ファイルを先に保存してから DB に入れる。逆にすると、保存に失敗したときに、ないファイルを指してしまう
	key := handler.ThumbnailObjectKey(p.DiaryID, p.ID)
	if err := b.files.Put(ctx, key, "image/webp", bytes.NewReader(thumb)); err != nil {
		return false, err
	}
	set, err := b.photos.SetThumbnailKey(ctx, p.ID, key)
	if err != nil {
		// DB に入ったかは分からないので、ファイルは消さない（もう一度実行すると同じキーに保存し直す）
		return false, err
	}
	if !set {
		// 作っている間に写真が消された。API はサムネイルを消していない（thumbnail_key が NULL だった）ので、ここで消す
		if err := b.files.Delete(ctx, key); err != nil {
			return false, fmt.Errorf("photo was deleted, but could not delete its thumbnail: %w", err)
		}
	}
	return set, nil
}
