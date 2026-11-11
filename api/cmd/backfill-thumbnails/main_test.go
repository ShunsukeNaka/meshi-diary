package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"io"
	"testing"

	"github.com/gen2brain/webp"
	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

// fakePhotos は photos テーブルの代わり。deleted の写真は、SetThumbnailKey のときにはもう消えている扱いにする。
type fakePhotos struct {
	photos  []postgres.Photo
	deleted map[uuid.UUID]bool
	setErr  error
}

func (f *fakePhotos) ListWithoutThumbnail(context.Context) ([]postgres.Photo, error) {
	var out []postgres.Photo
	for _, p := range f.photos {
		if p.ThumbnailKey == nil {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakePhotos) SetThumbnailKey(_ context.Context, id uuid.UUID, key string) (bool, error) {
	if f.setErr != nil {
		return false, f.setErr
	}
	if f.deleted[id] {
		return false, nil
	}
	for i := range f.photos {
		if f.photos[i].ID == id {
			f.photos[i].ThumbnailKey = &key
			return true, nil
		}
	}
	return false, nil
}

// fakeFiles は R2 の代わり。
type fakeFiles struct {
	objects map[string][]byte
}

func (f *fakeFiles) Get(_ context.Context, key string) ([]byte, error) {
	data, ok := f.objects[key]
	if !ok {
		return nil, errors.New("no such key")
	}
	return data, nil
}

func (f *fakeFiles) Put(_ context.Context, key, _ string, body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	f.objects[key] = data
	return nil
}

func (f *fakeFiles) Delete(_ context.Context, key string) error {
	delete(f.objects, key)
	return nil
}

func testWebP(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := webp.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func newPhoto(files *fakeFiles, data []byte) postgres.Photo {
	diaryID, id := uuid.New(), uuid.New()
	p := postgres.Photo{ID: id, DiaryID: diaryID, ObjectKey: "photos/" + diaryID.String() + "/" + id.String() + ".webp"}
	if data != nil {
		files.objects[p.ObjectKey] = data
	}
	return p
}

func thumbKey(p postgres.Photo) string {
	return "thumbnails/" + p.DiaryID.String() + "/" + p.ID.String() + ".webp"
}

func TestBackfill(t *testing.T) {
	ctx := context.Background()

	t.Run("creates thumbnails and sets thumbnail_key", func(t *testing.T) {
		files := &fakeFiles{objects: map[string][]byte{}}
		ok := newPhoto(files, testWebP(t, 800, 600))
		existingKey := "thumbnails/existing.webp"
		done := newPhoto(files, testWebP(t, 800, 600))
		done.ThumbnailKey = &existingKey
		missing := newPhoto(files, nil)             // R2 に写真がない
		broken := newPhoto(files, []byte("broken")) // 画像ではない
		deleted := newPhoto(files, testWebP(t, 800, 600))
		photos := &fakePhotos{
			photos:  []postgres.Photo{ok, done, missing, broken, deleted},
			deleted: map[uuid.UUID]bool{deleted.ID: true},
		}

		res, err := backfiller{photos: photos, files: files}.run(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if res != (result{created: 1, skipped: 1, failed: 2}) {
			t.Fatalf("result = %+v", res)
		}

		// 作ったサムネイルは短辺 192 px で、thumbnail_key がそれを指す
		got := photos.photos[0].ThumbnailKey
		if got == nil || *got != thumbKey(ok) {
			t.Fatalf("thumbnail_key = %v, want %s", got, thumbKey(ok))
		}
		img, err := webp.Decode(bytes.NewReader(files.objects[thumbKey(ok)]))
		if err != nil {
			t.Fatal(err)
		}
		if b := img.Bounds(); b.Dx() != 256 || b.Dy() != 192 {
			t.Fatalf("thumbnail size = %v, want 256x192", b)
		}
		// もうサムネイルがある写真には触らない
		if *photos.photos[1].ThumbnailKey != existingKey {
			t.Fatal("existing thumbnail_key changed")
		}
		if _, ok := files.objects[thumbKey(done)]; ok {
			t.Fatal("thumbnail was created for a photo that already had one")
		}
		// 失敗した写真は NULL のまま
		if photos.photos[2].ThumbnailKey != nil || photos.photos[3].ThumbnailKey != nil {
			t.Fatal("thumbnail_key was set for a failed photo")
		}
		// 途中で消された写真のサムネイルは、R2 に残さない
		if _, ok := files.objects[thumbKey(deleted)]; ok {
			t.Fatal("thumbnail of a deleted photo was left in R2")
		}
	})

	t.Run("running again does nothing", func(t *testing.T) {
		files := &fakeFiles{objects: map[string][]byte{}}
		photos := &fakePhotos{photos: []postgres.Photo{newPhoto(files, testWebP(t, 400, 300))}}
		b := backfiller{photos: photos, files: files}
		if res, err := b.run(ctx); err != nil || res.created != 1 {
			t.Fatalf("first run: %+v, %v", res, err)
		}
		if res, err := b.run(ctx); err != nil || res != (result{}) {
			t.Fatalf("second run: %+v, %v", res, err)
		}
	})

	t.Run("keeps the file when the DB update fails", func(t *testing.T) {
		files := &fakeFiles{objects: map[string][]byte{}}
		p := newPhoto(files, testWebP(t, 400, 300))
		photos := &fakePhotos{photos: []postgres.Photo{p}, setErr: errors.New("db is down")}
		res, err := backfiller{photos: photos, files: files}.run(ctx)
		if err != nil || res.failed != 1 {
			t.Fatalf("result = %+v, %v", res, err)
		}
		// DB に入ったか分からないので、消さない
		if _, ok := files.objects[thumbKey(p)]; !ok {
			t.Fatal("thumbnail was deleted")
		}
	})

	t.Run("stops when canceled", func(t *testing.T) {
		files := &fakeFiles{objects: map[string][]byte{}}
		photos := &fakePhotos{photos: []postgres.Photo{newPhoto(files, testWebP(t, 400, 300))}}
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := (backfiller{photos: photos, files: files}).run(canceled); !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
		if photos.photos[0].ThumbnailKey != nil {
			t.Fatal("thumbnail_key was set after cancel")
		}
	})
}
