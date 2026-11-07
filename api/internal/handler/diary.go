package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

// maxDiaryBodyBytes は日記のリクエストボディの上限。本文に文字数の制限はないが、1 MB あれば十分。
const maxDiaryBodyBytes = 1 << 20

// DiaryRepository はハンドラーが使う diaries の操作。本番は postgres.Diaries、テストでは偽物に差し替える。
type DiaryRepository interface {
	// Create は diary を保存し、id・created_at などを diary に入れる。
	Create(ctx context.Context, diary *postgres.Diary) error
	// FindByID は userID のユーザーの、id の日記を返す。ないとき・ほかのユーザーの日記のときは postgres.ErrNotFound を返す。
	FindByID(ctx context.Context, userID, id uuid.UUID) (postgres.Diary, error)
}

type createDiaryRequest struct {
	EatenOn string  `json:"eaten_on"`
	Status  string  `json:"status"`
	Info    *string `json:"info"`
	Body    string  `json:"body"`
}

// diaryResponse はレスポンスの diary（docs/api.md の POST /diaries）。写真は含まない。
type diaryResponse struct {
	ID        uuid.UUID         `json:"id"`
	EatenOn   string            `json:"eaten_on"`
	Status    postgres.MealType `json:"status"`
	Info      *string           `json:"info"`
	Body      string            `json:"body"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func newDiaryResponse(d postgres.Diary) diaryResponse {
	return diaryResponse{
		ID:        d.ID,
		EatenOn:   d.EatenOn.Format(time.DateOnly),
		Status:    d.Status,
		Info:      d.Info,
		Body:      d.Body,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// CreateDiary はログイン中のユーザーの日記を作る（docs/api.md の POST /diaries）。Sessions.RequireSession で包んで使う。
func CreateDiary(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserID(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "ログインしてください")
			return
		}

		var req createDiaryRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDiaryBodyBytes)).Decode(&req); err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge, "日記が大きすぎます")
				return
			}
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationError, "リクエストを JSON として読めません")
			return
		}

		eatenOn, err := time.Parse(time.DateOnly, req.EatenOn)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationError, "eaten_on は YYYY-MM-DD の形の日付にしてください")
			return
		}
		// 選び忘れても困らないので、省略したときは OTHER にする
		status := postgres.MealTypeOther
		if req.Status != "" {
			status = postgres.MealType(req.Status)
		}
		if !validMealType(status) {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationError, "status は BREAKFAST・LUNCH・DINNER・OTHER のどれかにしてください")
			return
		}
		// TrimSpace は全角スペース・改行・タブも取り除く
		if strings.TrimSpace(req.Body) == "" {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationError, "body を1文字以上入力してください")
			return
		}
		info := req.Info
		if info != nil && *info == "" {
			info = nil
		}

		diary := postgres.Diary{UserID: userID, EatenOn: eatenOn, Status: status, Info: info, Body: req.Body}
		if err := d.Diaries.Create(r.Context(), &diary); err != nil {
			httpx.WriteInternalError(w, err)
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, map[string]diaryResponse{"diary": newDiaryResponse(diary)})
	}
}

// diaryDetailResponse は写真も含めた diary（docs/api.md の GET /diaries/{id}）。
type diaryDetailResponse struct {
	diaryResponse
	Photos []photoResponse `json:"photos"`
}

// GetDiary はログイン中のユーザーの日記を、写真と一緒に返す（docs/api.md の GET /diaries/{id}）。Sessions.RequireSession で包んで使う。
func GetDiary(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diary, ok := findOwnDiary(w, r, d.Diaries)
		if !ok {
			return
		}
		photos, err := d.Photos.ListByDiary(r.Context(), diary.ID)
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}
		photoRes, err := newPhotoResponses(r.Context(), d.Storage, photos)
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]diaryDetailResponse{
			"diary": {diaryResponse: newDiaryResponse(diary), Photos: photoRes},
		})
	}
}

// findOwnDiary はパスの {id} の日記を、ログイン中のユーザーのものから探す。
// 見つからないとき（id が UUID でない・ない・ほかのユーザーの日記）は 404 を書いて false を返す。
func findOwnDiary(w http.ResponseWriter, r *http.Request, diaries DiaryRepository) (postgres.Diary, bool) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "ログインしてください")
		return postgres.Diary{}, false
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeDiaryNotFound(w)
		return postgres.Diary{}, false
	}
	diary, err := diaries.FindByID(r.Context(), userID, id)
	if errors.Is(err, postgres.ErrNotFound) {
		writeDiaryNotFound(w)
		return postgres.Diary{}, false
	}
	if err != nil {
		httpx.WriteInternalError(w, err)
		return postgres.Diary{}, false
	}
	return diary, true
}

func writeDiaryNotFound(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "日記が見つかりません")
}

func validMealType(m postgres.MealType) bool {
	switch m {
	case postgres.MealTypeBreakfast, postgres.MealTypeLunch, postgres.MealTypeDinner, postgres.MealTypeOther:
		return true
	}
	return false
}
