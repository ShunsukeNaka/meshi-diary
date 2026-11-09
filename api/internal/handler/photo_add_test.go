package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gen2brain/webp"
	"github.com/google/uuid"
)

func testJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 32, 24)), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func testWebP(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := webp.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 20, 10))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// uploadPhotos は files を photos、alts を alts として multipart で送る。
func uploadPhotos(t *testing.T, e *diaryTestEnv, path string, cookie *http.Cookie, files [][]byte, alts []string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for i, f := range files {
		part, err := mw.CreateFormFile("photos", "photo"+string(rune('a'+i))+".jpg")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(f)
	}
	for _, a := range alts {
		mw.WriteField("alts", a)
	}
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

type photosBody struct {
	Photos []struct {
		ID       string  `json:"id"`
		URL      string  `json:"url"`
		Alt      *string `json:"alt"`
		Position int     `json:"position"`
	} `json:"photos"`
}

func TestAddPhotos(t *testing.T) {
	userID := uuid.New()

	t.Run("converts to webp, saves to R2 and appends after existing photos", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		existing := e.addPhoto(d.ID, 0)
		rec := uploadPhotos(t, e, "/diaries/"+d.ID.String()+"/photos", e.cookie(t, userID),
			[][]byte{testJPEG(t), testPNG(t)}, []string{"", "替え玉は硬めで。"})
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		var body photosBody
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Photos) != 3 || body.Photos[0].ID != existing.ID.String() {
			t.Fatalf("photos = %+v", body.Photos)
		}
		if body.Photos[1].Alt != nil || body.Photos[2].Alt == nil || *body.Photos[2].Alt != "替え玉は硬めで。" || body.Photos[2].Position != 2 {
			t.Fatalf("added = %+v", body.Photos[1:])
		}
		for _, p := range body.Photos[1:] {
			key := "photos/" + d.ID.String() + "/" + p.ID + ".webp"
			data, ok := e.storage.objects[key]
			if !ok {
				t.Fatalf("file %s was not saved", key)
			}
			if !bytes.HasPrefix(data, []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WEBP")) {
				t.Fatalf("file %s is not webp", key)
			}
			if !strings.Contains(p.URL, key) {
				t.Fatalf("url = %q", p.URL)
			}
		}
	})

	t.Run("saves WebP as sent", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		sent := testWebP(t)
		rec := uploadPhotos(t, e, "/diaries/"+d.ID.String()+"/photos", e.cookie(t, userID), [][]byte{sent}, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		var body photosBody
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		key := "photos/" + d.ID.String() + "/" + body.Photos[0].ID + ".webp"
		if !bytes.Equal(e.storage.objects[key], sent) {
			t.Fatalf("file %s is not the sent webp", key)
		}
	})

	t.Run("without alts", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		rec := uploadPhotos(t, e, "/diaries/"+d.ID.String()+"/photos", e.cookie(t, userID), [][]byte{testJPEG(t)}, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("errors add nothing", func(t *testing.T) {
		big := append(testJPEG(t), make([]byte, maxPhotoBytes)...)
		cases := []struct {
			name     string
			existing int
			files    [][]byte
			alts     []string
			status   int
			code     string
		}{
			{"no photos", 0, nil, nil, 400, "VALIDATION_ERROR"},
			{"alts count mismatch", 0, [][]byte{testJPEG(t), testJPEG(t)}, []string{"a"}, 400, "VALIDATION_ERROR"},
			{"more than 10", 9, [][]byte{testJPEG(t), testJPEG(t)}, nil, 409, "CONFLICT"},
			{"over 10MB", 0, [][]byte{testJPEG(t), big}, nil, 413, "PAYLOAD_TOO_LARGE"},
			{"not an image", 0, [][]byte{testJPEG(t), []byte("hello")}, nil, 415, "UNSUPPORTED_MEDIA_TYPE"},
			{"gif", 0, [][]byte{[]byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")}, nil, 415, "UNSUPPORTED_MEDIA_TYPE"},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				e := newDiaryTestEnv(t)
				d := e.addDiary(userID)
				for i := 0; i < c.existing; i++ {
					e.addPhoto(d.ID, i)
				}
				before := len(e.storage.objects)
				rec := uploadPhotos(t, e, "/diaries/"+d.ID.String()+"/photos", e.cookie(t, userID), c.files, c.alts)
				if rec.Code != c.status {
					t.Fatalf("status = %d, want %d (%s)", rec.Code, c.status, rec.Body.String())
				}
				if code := errorCode(t, rec); code != c.code {
					t.Fatalf("code = %q, want %q", code, c.code)
				}
				if got, _ := e.photos.ListByDiary(t.Context(), d.ID); len(got) != c.existing {
					t.Fatalf("photos = %d, want %d", len(got), c.existing)
				}
				if len(e.storage.objects) != before {
					t.Fatalf("files = %d, want %d", len(e.storage.objects), before)
				}
			})
		}
	})

	t.Run("DB failure removes the saved files", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		e.photos.addErr = errors.New("db down")
		rec := uploadPhotos(t, e, "/diaries/"+d.ID.String()+"/photos", e.cookie(t, userID), [][]byte{testJPEG(t), testPNG(t)}, nil)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
		if len(e.storage.objects) != 0 || len(e.storage.deleted) != 2 {
			t.Fatalf("objects = %d, deleted = %d", len(e.storage.objects), len(e.storage.deleted))
		}
	})

	t.Run("R2 failure is 500 and adds nothing", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		e.storage.putErr = errors.New("r2 down")
		rec := uploadPhotos(t, e, "/diaries/"+d.ID.String()+"/photos", e.cookie(t, userID), [][]byte{testJPEG(t)}, nil)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
		if got, _ := e.photos.ListByDiary(t.Context(), d.ID); len(got) != 0 {
			t.Fatal("photo row was added")
		}
	})

	t.Run("other user's diary is 404", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(uuid.New())
		rec := uploadPhotos(t, e, "/diaries/"+d.ID.String()+"/photos", e.cookie(t, userID), [][]byte{testJPEG(t)}, nil)
		if rec.Code != http.StatusNotFound || len(e.storage.objects) != 0 {
			t.Fatalf("status = %d, files = %d", rec.Code, len(e.storage.objects))
		}
	})

	t.Run("not multipart is 400", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		if rec := sendJSON(e.router, http.MethodPost, "/diaries/"+d.ID.String()+"/photos", `{}`, e.cookie(t, userID)); rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("no cookie is 401", func(t *testing.T) {
		e := newDiaryTestEnv(t)
		d := e.addDiary(userID)
		assertUnauthorized(t, uploadPhotos(t, e, "/diaries/"+d.ID.String()+"/photos", nil, [][]byte{testJPEG(t)}, nil))
	})
}
