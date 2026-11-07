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
	// Update は diary を保存し直し、updated_at を入れる。ないときは postgres.ErrNotFound を返す。
	Update(ctx context.Context, diary *postgres.Diary) error
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
		if !decodeDiaryJSON(w, r, &req) {
			return
		}
		eatenOn, ok := parseEatenOn(req.EatenOn)
		if !ok {
			writeValidationError(w, msgInvalidEatenOn)
			return
		}
		status, ok := parseStatus(req.Status)
		if !ok {
			writeValidationError(w, msgInvalidStatus)
			return
		}
		if !validBody(req.Body) {
			writeValidationError(w, msgInvalidBody)
			return
		}

		diary := postgres.Diary{UserID: userID, EatenOn: eatenOn, Status: status, Info: normalizeInfo(req.Info), Body: req.Body}
		if err := d.Diaries.Create(r.Context(), &diary); err != nil {
			httpx.WriteInternalError(w, err)
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, map[string]diaryResponse{"diary": newDiaryResponse(diary)})
	}
}

// updateDiaryRequest は PATCH /diaries/{id} のリクエスト。省略したフィールドは変えない。
type updateDiaryRequest struct {
	EatenOn optional[string] `json:"eaten_on"`
	Status  optional[string] `json:"status"`
	Info    optional[string] `json:"info"`
	Body    optional[string] `json:"body"`
}

// UpdateDiary はログイン中のユーザーの日記を、送ったフィールドだけ変える（docs/api.md の PATCH /diaries/{id}）。
// Sessions.RequireSession で包んで使う。
func UpdateDiary(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diary, ok := findOwnDiary(w, r, d.Diaries)
		if !ok {
			return
		}

		var req updateDiaryRequest
		if !decodeDiaryJSON(w, r, &req) {
			return
		}
		if req.EatenOn.Set {
			eatenOn, ok := parseEatenOn(req.EatenOn.Value)
			if req.EatenOn.Null || !ok {
				writeValidationError(w, msgInvalidEatenOn)
				return
			}
			diary.EatenOn = eatenOn
		}
		if req.Status.Set {
			// null・空文字は OTHER（作るときと同じ）
			status, ok := parseStatus(req.Status.Value)
			if !ok {
				writeValidationError(w, msgInvalidStatus)
				return
			}
			diary.Status = status
		}
		if req.Info.Set {
			if req.Info.Null {
				diary.Info = nil
			} else {
				diary.Info = normalizeInfo(&req.Info.Value)
			}
		}
		if req.Body.Set {
			if req.Body.Null || !validBody(req.Body.Value) {
				writeValidationError(w, msgInvalidBody)
				return
			}
			diary.Body = req.Body.Value
		}

		if err := d.Diaries.Update(r.Context(), &diary); err != nil {
			if errors.Is(err, postgres.ErrNotFound) {
				writeDiaryNotFound(w)
				return
			}
			httpx.WriteInternalError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]diaryResponse{"diary": newDiaryResponse(diary)})
	}
}

// optional は JSON のフィールドが「省略された」「null」「値がある」のどれかを見分ける。
type optional[T any] struct {
	Set   bool // フィールドがあった（null も含む）
	Null  bool // null だった
	Value T
}

// UnmarshalJSON はフィールドがあるときだけ呼ばれる（null のときも呼ばれる）。
func (o *optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Null = true
		return nil
	}
	return json.Unmarshal(b, &o.Value)
}

const (
	msgInvalidEatenOn = "eaten_on は YYYY-MM-DD の形の日付にしてください"
	msgInvalidStatus  = "status は BREAKFAST・LUNCH・DINNER・OTHER のどれかにしてください"
	msgInvalidBody    = "body を1文字以上入力してください"
)

// decodeDiaryJSON はリクエストボディの JSON を v に読み込む。
// 読めないときは 400、大きすぎるときは 413 を書いて false を返す。
func decodeDiaryJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDiaryBodyBytes)).Decode(v); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge, "日記が大きすぎます")
			return false
		}
		writeValidationError(w, "リクエストを JSON として読めません")
		return false
	}
	return true
}

func writeValidationError(w http.ResponseWriter, message string) {
	httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationError, message)
}

// parseEatenOn は YYYY-MM-DD の日付を読む。
func parseEatenOn(s string) (time.Time, bool) {
	t, err := time.Parse(time.DateOnly, s)
	return t, err == nil
}

// parseStatus は食事の区分を読む。選び忘れても困らないので、空のときは OTHER にする。
func parseStatus(s string) (postgres.MealType, bool) {
	if s == "" {
		return postgres.MealTypeOther, true
	}
	m := postgres.MealType(s)
	return m, validMealType(m)
}

// validBody は本文に空白以外の文字があるかを確かめる。TrimSpace は全角スペース・改行・タブも取り除く。
func validBody(s string) bool {
	return strings.TrimSpace(s) != ""
}

// normalizeInfo は空文字の一言メモを nil（DB では NULL）にそろえる。
func normalizeInfo(info *string) *string {
	if info == nil || *info == "" {
		return nil
	}
	return info
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
