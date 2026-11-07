package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

// calendarDay はカレンダーの1日（docs/api.md の GET /calendar の days[]）。
type calendarDay struct {
	Date         string  `json:"date"`
	DiaryCount   int     `json:"diary_count"`
	ThumbnailURL *string `json:"thumbnail_url"`
}

type calendarResponse struct {
	Month string        `json:"month"`
	Days  []calendarDay `json:"days"`
}

// Calendar は指定した月の、日記がある日とサムネイルを返す（docs/api.md の GET /calendar）。Sessions.RequireSession で包んで使う。
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

		// diaries は日付の古い順・同じ日は作った順なので、順に見ていけば日ごとにまとまる
		days := []calendarDay{}
		for _, diary := range diaries {
			date := diary.EatenOn.Format(time.DateOnly)
			if len(days) == 0 || days[len(days)-1].Date != date {
				days = append(days, calendarDay{Date: date})
			}
			day := &days[len(days)-1]
			day.DiaryCount++
			// 作った順に見て、写真がある最初の日記の1枚目をサムネイルにする
			if url, ok := thumbnails[diary.ID]; ok && day.ThumbnailURL == nil {
				day.ThumbnailURL = &url
			}
		}

		httpx.WriteJSON(w, http.StatusOK, calendarResponse{Month: month.Format("2006-01"), Days: days})
	}
}

// thumbnailURLs は日記ごとの1枚目の写真の表示用 URL を、日記の id をキーにして返す。写真がない日記はキーに入らない。
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
		url, err := d.Storage.PresignGet(r.Context(), photo.ObjectKey, photoURLExpiry)
		if err != nil {
			return nil, err
		}
		urls[diaryID] = url
	}
	return urls, nil
}
