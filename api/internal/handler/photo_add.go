package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"runtime"
	"sync"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/imageconv"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

const (
	// maxPhotoBytes は写真1枚の大きさの上限（送られたファイルと、WebP にしたあとの両方）。
	maxPhotoBytes = 10 << 20
	// maxUploadBytes はリクエスト全体の上限。10枚ぶんに、multipart の区切りなどの分を足す。
	maxUploadBytes = postgres.MaxPhotosPerDiary*maxPhotoBytes + 1<<20
	// multipartMemory はメモリに置く分。超えた分は一時ファイルになる。
	multipartMemory = 32 << 20
)

// AddPhotos は日記に写真をまとめて追加する（docs/api.md の POST /diaries/{id}/photos）。Sessions.RequireSession で包んで使う。
// 1枚でもエラーになったときは、1枚も追加しない。
func AddPhotos(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diary, ok := findOwnDiary(w, r, d.Diaries)
		if !ok {
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
		if err := r.ParseMultipartForm(multipartMemory); err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge, "写真が大きすぎます")
				return
			}
			writeValidationError(w, "multipart/form-data で写真を送ってください")
			return
		}
		defer r.MultipartForm.RemoveAll()

		files := r.MultipartForm.File["photos"]
		alts := r.MultipartForm.Value["alts"]
		if len(files) == 0 {
			writeValidationError(w, "photos に写真を1枚以上入れてください")
			return
		}
		if len(alts) != 0 && len(alts) != len(files) {
			writeValidationError(w, "alts は photos と同じ数だけ送ってください")
			return
		}
		for _, fh := range files {
			if fh.Size > maxPhotoBytes {
				httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge, "10 MB を超える写真は追加できません")
				return
			}
		}
		// 変換は時間がかかるので、枚数が多すぎるときは先に断る（最後に AddMany でもう一度確かめる）
		existing, err := d.Photos.ListByDiary(r.Context(), diary.ID)
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}
		if len(existing)+len(files) > postgres.MaxPhotosPerDiary {
			writeTooManyPhotos(w)
			return
		}

		converted, err := convertAll(files)
		if err != nil {
			switch {
			case errors.Is(err, imageconv.ErrUnsupportedFormat), errors.Is(err, imageconv.ErrTooLarge):
				httpx.WriteError(w, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType, "JPEG・PNG・WebP の写真を選んでください")
			case errors.Is(err, errConvertedTooLarge):
				httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge, "10 MB を超える写真は追加できません")
			default:
				httpx.WriteInternalError(w, err)
			}
			return
		}

		photos := make([]postgres.Photo, len(converted))
		for i := range converted {
			id := uuid.New()
			photos[i] = postgres.Photo{ID: id, ObjectKey: "photos/" + diary.ID.String() + "/" + id.String() + ".webp"}
			if len(alts) != 0 {
				photos[i].Alt = normalizeInfo(&alts[i])
			}
		}

		if err := savePhotos(r.Context(), d, diary.ID, photos, converted); err != nil {
			switch {
			case errors.Is(err, postgres.ErrTooManyPhotos):
				writeTooManyPhotos(w)
			case errors.Is(err, postgres.ErrNotFound):
				writeDiaryNotFound(w)
			default:
				httpx.WriteInternalError(w, err)
			}
			return
		}

		all, err := d.Photos.ListByDiary(r.Context(), diary.ID)
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}
		res, err := newPhotoResponses(r.Context(), d.Storage, all)
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, map[string][]photoResponse{"photos": res})
	}
}

func writeTooManyPhotos(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict, "写真は1つの日記に10枚までです")
}

var errConvertedTooLarge = errors.New("converted photo is too large")

// convertAll は写真をすべて WebP にする。CPU の数だけ並べて変換する。どれか1枚でも失敗したらエラーを返す。
func convertAll(files []*multipart.FileHeader) ([][]byte, error) {
	out := make([][]byte, len(files))
	errs := make([]error, len(files))
	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup
	for i, fh := range files {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i], errs[i] = convertOne(fh)
		}()
	}
	wg.Wait()
	return out, errors.Join(errs...)
}

func convertOne(fh *multipart.FileHeader) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	webp, err := imageconv.ToWebP(data)
	if err != nil {
		return nil, err
	}
	if len(webp) > maxPhotoBytes {
		return nil, errConvertedTooLarge
	}
	return webp, nil
}

// savePhotos は WebP を R2 に保存してから、photos の行をまとめて追加する。
// 途中で失敗したときは、R2 に保存したファイルを消してエラーを返す。
func savePhotos(ctx context.Context, d Deps, diaryID uuid.UUID, photos []postgres.Photo, files [][]byte) error {
	var saved []string
	cleanup := func() {
		for _, key := range saved {
			deletePhotoFile(ctx, d.Storage, key)
		}
	}
	for i, p := range photos {
		if err := d.Storage.Put(ctx, p.ObjectKey, "image/webp", bytes.NewReader(files[i])); err != nil {
			cleanup()
			return err
		}
		saved = append(saved, p.ObjectKey)
	}
	if err := d.Photos.AddMany(ctx, diaryID, photos); err != nil {
		cleanup()
		return err
	}
	return nil
}
