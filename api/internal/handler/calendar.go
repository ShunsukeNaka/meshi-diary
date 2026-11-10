package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

// bodyHeadLength は一覧に出す本文の冒頭の文字数（docs/api.md の GET /calendar の body_head）。
const bodyHeadLength = 30

// calendarDiary はカレンダーの日の日記（docs/api.md の GET /calendar の days[].diaries[]）。一覧の表示に使う分だけ返す。
type calendarDiary struct {
	ID           uuid.UUID         `json:"id"`
	Info         *string           `json:"info"`
	BodyHead     string            `json:"body_head"`
	Status       postgres.MealType `json:"status"`
	ThumbnailURL *string           `json:"thumbnail_url"`
}

// calendarDay はカレンダーの1日（docs/api.md の GET /calendar の days[]）。
type calendarDay struct {
	Date    string          `json:"date"`
	Diaries []calendarDiary `json:"diaries"`
}

type calendarResponse struct {
	Month string        `json:"month"`
	Days  []calendarDay `json:"days"`
}

// Calendar は指定した月の、日記がある日と、その日の日記の一覧を返す（docs/api.md の GET /calendar）。Sessions.RequireSession で包んで使う。
func Calendar(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserID(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "ログインしてください")
			return
		}
		month, err := time.Parse("2006-01", r.URL.Query().Get("month"))
		if err != nil {
			writeValidationError(w, "month は YYYY-MM の形にしてください")
			return
		}

		diaries, err := d.Diaries.ListBetween(r.Context(), userID, month, month.AddDate(0, 1, 0))
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}
		thumbnails, err := thumbnailURLs(r, d, diaries)
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}

		// diaries は日付の古い順・同じ日は食事の区分の順なので、順に見ていけば日ごとにまとまる
		days := []calendarDay{}
		for _, diary := range diaries {
			date := diary.EatenOn.Format(time.DateOnly)
			if len(days) == 0 || days[len(days)-1].Date != date {
				days = append(days, calendarDay{Date: date, Diaries: []calendarDiary{}})
			}
			day := &days[len(days)-1]
			item := calendarDiary{ID: diary.ID, Info: diary.Info, BodyHead: bodyHead(diary.Body), Status: diary.Status}
			if url, ok := thumbnails[diary.ID]; ok {
				item.ThumbnailURL = &url
			}
			day.Diaries = append(day.Diaries, item)
		}

		httpx.WriteJSON(w, http.StatusOK, calendarResponse{Month: month.Format("2006-01"), Days: days})
	}
}

// bodyHead は本文の最初の bodyHeadLength 文字を返す。日本語が途中で切れないように、バイトではなく文字（rune）で数える。
func bodyHead(body string) string {
	n := 0
	for i := range body {
		if n == bodyHeadLength {
			return body[:i]
		}
		n++
	}
	return body
}

// thumbnailURLs は日記ごとの1枚目の写真のサムネイルの表示用 URL を、日記の id をキーにして返す。写真がない日記はキーに入らない。
// サムネイルがまだない写真（サムネイルを作る前に保存した写真）は、写真そのものの URL を返す。
func thumbnailURLs(r *http.Request, d Deps, diaries []postgres.Diary) (map[uuid.UUID]string, error) {
	ids := make([]uuid.UUID, len(diaries))
	for i, diary := range diaries {
		ids[i] = diary.ID
	}
	first, err := d.Photos.FirstOfDiaries(r.Context(), ids)
	if err != nil {
		return nil, err
	}
	urls := make(map[uuid.UUID]string, len(first))
	for diaryID, photo := range first {
		key := photo.ObjectKey
		if photo.ThumbnailKey != nil {
			key = *photo.ThumbnailKey
		}
		url, err := d.Storage.PresignGet(r.Context(), key, photoURLExpiry)
		if err != nil {
			return nil, err
		}
		urls[diaryID] = url
	}
	return urls, nil
}
