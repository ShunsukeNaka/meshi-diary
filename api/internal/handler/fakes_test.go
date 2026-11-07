package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

var testTime = time.Date(2026, 11, 6, 12, 30, 0, 0, time.FixedZone("JST", 9*60*60))

// fakeDiaries はメモリ上の diaries。postgres.Diaries と同じエラーを返す。
type fakeDiaries struct {
	byID    map[uuid.UUID]postgres.Diary
	created []postgres.Diary
	err     error // 設定すると、すべての操作でこのエラーを返す
	// onDelete は日記を消したときに呼ばれる（写真の ON DELETE CASCADE の再現に使う）
	onDelete func(id uuid.UUID)
}

func (f *fakeDiaries) Create(_ context.Context, d *postgres.Diary) error {
	if f.err != nil {
		return f.err
	}
	d.ID = uuid.New()
	d.CreatedAt = testTime
	d.UpdatedAt = d.CreatedAt
	f.created = append(f.created, *d)
	f.add(*d)
	return nil
}

func (f *fakeDiaries) FindByID(_ context.Context, userID, id uuid.UUID) (postgres.Diary, error) {
	if f.err != nil {
		return postgres.Diary{}, f.err
	}
	d, ok := f.byID[id]
	if !ok || d.UserID != userID {
		return postgres.Diary{}, postgres.ErrNotFound
	}
	return d, nil
}

func (f *fakeDiaries) Update(_ context.Context, d *postgres.Diary) error {
	if f.err != nil {
		return f.err
	}
	old, ok := f.byID[d.ID]
	if !ok || old.UserID != d.UserID {
		return postgres.ErrNotFound
	}
	d.UpdatedAt = testTime.Add(time.Hour)
	f.byID[d.ID] = *d
	return nil
}

// Delete は日記を消す。写真の行を消す（CASCADE）のは fakePhotos 側で cascade を呼んで再現する。
func (f *fakeDiaries) Delete(_ context.Context, userID, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	d, ok := f.byID[id]
	if !ok || d.UserID != userID {
		return postgres.ErrNotFound
	}
	delete(f.byID, id)
	if f.onDelete != nil {
		f.onDelete(id)
	}
	return nil
}

func (f *fakeDiaries) ListBetween(_ context.Context, userID uuid.UUID, from, to time.Time) ([]postgres.Diary, error) {
	if f.err != nil {
		return nil, f.err
	}
	diaries := []postgres.Diary{}
	for _, d := range f.byID {
		if d.UserID == userID && !d.EatenOn.Before(from) && d.EatenOn.Before(to) {
			diaries = append(diaries, d)
		}
	}
	sort.Slice(diaries, func(i, j int) bool {
		a, b := diaries[i], diaries[j]
		if !a.EatenOn.Equal(b.EatenOn) {
			return a.EatenOn.Before(b.EatenOn)
		}
		return a.CreatedAt.Before(b.CreatedAt)
	})
	return diaries, nil
}

// mealOrder は enum meal_type の定義順。ListByDate の並び順の再現に使う。
var mealOrder = map[postgres.MealType]int{
	postgres.MealTypeBreakfast: 0, postgres.MealTypeLunch: 1, postgres.MealTypeDinner: 2, postgres.MealTypeOther: 3,
}

func (f *fakeDiaries) ListByDate(_ context.Context, userID uuid.UUID, date time.Time) ([]postgres.Diary, error) {
	if f.err != nil {
		return nil, f.err
	}
	diaries := []postgres.Diary{}
	for _, d := range f.byID {
		if d.UserID == userID && d.EatenOn.Equal(date) {
			diaries = append(diaries, d)
		}
	}
	sort.Slice(diaries, func(i, j int) bool {
		a, b := diaries[i], diaries[j]
		if a.Status != b.Status {
			return mealOrder[a.Status] < mealOrder[b.Status]
		}
		return a.CreatedAt.Before(b.CreatedAt)
	})
	return diaries, nil
}

func (f *fakeDiaries) add(d postgres.Diary) {
	if f.byID == nil {
		f.byID = map[uuid.UUID]postgres.Diary{}
	}
	f.byID[d.ID] = d
}

// fakePhotos はメモリ上の photos。
type fakePhotos struct {
	byID   map[uuid.UUID]postgres.Photo
	err    error
	addErr error // AddMany だけで返すエラー
}

func (f *fakePhotos) ListByDiary(_ context.Context, diaryID uuid.UUID) ([]postgres.Photo, error) {
	if f.err != nil {
		return nil, f.err
	}
	photos := []postgres.Photo{}
	for _, p := range f.byID {
		if p.DiaryID == diaryID {
			photos = append(photos, p)
		}
	}
	sort.Slice(photos, func(i, j int) bool { return photos[i].Position < photos[j].Position })
	return photos, nil
}

func (f *fakePhotos) FirstOfDiaries(_ context.Context, diaryIDs []uuid.UUID) (map[uuid.UUID]postgres.Photo, error) {
	if f.err != nil {
		return nil, f.err
	}
	first := map[uuid.UUID]postgres.Photo{}
	for _, id := range diaryIDs {
		for _, p := range f.byID {
			if p.DiaryID == id && p.Position == 0 {
				first[id] = p
			}
		}
	}
	return first, nil
}

func (f *fakePhotos) AddMany(_ context.Context, diaryID uuid.UUID, photos []postgres.Photo) error {
	if f.err != nil {
		return f.err
	}
	if f.addErr != nil {
		return f.addErr
	}
	existing, _ := f.ListByDiary(context.Background(), diaryID)
	if len(existing)+len(photos) > postgres.MaxPhotosPerDiary {
		return postgres.ErrTooManyPhotos
	}
	for i := range photos {
		photos[i].DiaryID = diaryID
		photos[i].Position = len(existing) + i
		f.add(photos[i])
	}
	return nil
}

func (f *fakePhotos) UpdateAlt(_ context.Context, diaryID, photoID uuid.UUID, alt *string) (postgres.Photo, error) {
	if f.err != nil {
		return postgres.Photo{}, f.err
	}
	p, ok := f.byID[photoID]
	if !ok || p.DiaryID != diaryID {
		return postgres.Photo{}, postgres.ErrNotFound
	}
	p.Alt = alt
	f.byID[photoID] = p
	return p, nil
}

func (f *fakePhotos) add(p postgres.Photo) {
	if f.byID == nil {
		f.byID = map[uuid.UUID]postgres.Photo{}
	}
	f.byID[p.ID] = p
}

// fakeStorage はメモリ上の R2。署名付き URL は https://r2.test/<key> を返す。
type fakeStorage struct {
	objects   map[string][]byte
	deleted   []string
	putErr    error
	deleteErr error
}

func (f *fakeStorage) Put(_ context.Context, key, _ string, body io.Reader) error {
	if f.putErr != nil {
		return f.putErr
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if f.objects == nil {
		f.objects = map[string][]byte{}
	}
	f.objects[key] = b
	return nil
}

func (f *fakeStorage) PresignGet(_ context.Context, key string, expires time.Duration) (string, error) {
	return fmt.Sprintf("https://r2.test/%s?expires=%d", key, int(expires.Seconds())), nil
}

func (f *fakeStorage) Delete(_ context.Context, key string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.objects, key)
	f.deleted = append(f.deleted, key)
	return nil
}

// diaryTestEnv は日記・写真のハンドラーのテストに使う、偽物をつないだルーター。
type diaryTestEnv struct {
	router   http.Handler
	sessions *auth.Sessions
	diaries  *fakeDiaries
	photos   *fakePhotos
	storage  *fakeStorage
}

func newDiaryTestEnv(t *testing.T) *diaryTestEnv {
	t.Helper()
	sessions, err := auth.NewSessions([]byte(strings.Repeat("s", auth.MinSecretLength)))
	if err != nil {
		t.Fatal(err)
	}
	e := &diaryTestEnv{sessions: sessions, diaries: &fakeDiaries{}, photos: &fakePhotos{}, storage: &fakeStorage{}}
	e.diaries.onDelete = func(diaryID uuid.UUID) {
		for id, p := range e.photos.byID {
			if p.DiaryID == diaryID {
				delete(e.photos.byID, id)
			}
		}
	}
	e.router = NewRouter(Deps{Sessions: sessions, Diaries: e.diaries, Photos: e.photos, Storage: e.storage})
	return e
}

// addDiary は userID の日記を1件入れて返す。
func (e *diaryTestEnv) addDiary(userID uuid.UUID) postgres.Diary {
	info := "駅前のラーメン屋"
	d := postgres.Diary{
		ID:        uuid.New(),
		UserID:    userID,
		EatenOn:   time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC),
		Status:    postgres.MealTypeLunch,
		Info:      &info,
		Body:      "本文",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	e.diaries.add(d)
	return d
}

// addDiaryOn は userID の、date（YYYY-MM-DD）の日記を、created の時刻に作ったものとして入れる。
func (e *diaryTestEnv) addDiaryOn(userID uuid.UUID, date string, status postgres.MealType, created time.Time) postgres.Diary {
	d := e.addDiary(userID)
	d.EatenOn, _ = time.Parse(time.DateOnly, date)
	d.Status = status
	d.CreatedAt = created
	e.diaries.add(d)
	return d
}

// addPhoto は日記に position の写真を入れて返す。R2 にもファイルを置く。
func (e *diaryTestEnv) addPhoto(diaryID uuid.UUID, position int) postgres.Photo {
	id := uuid.New()
	p := postgres.Photo{ID: id, DiaryID: diaryID, ObjectKey: "photos/" + diaryID.String() + "/" + id.String() + ".webp", Position: position, CreatedAt: testTime, UpdatedAt: testTime}
	e.photos.add(p)
	_ = e.storage.Put(context.Background(), p.ObjectKey, "image/webp", strings.NewReader("webp"))
	return p
}

func (e *diaryTestEnv) cookie(t *testing.T, userID uuid.UUID) *http.Cookie {
	t.Helper()
	return sessionCookie(t, e.sessions, userID)
}
