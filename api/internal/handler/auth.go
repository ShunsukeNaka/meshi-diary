package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"gorm.io/gorm"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
)

// maxAuthBodyBytes は /signup・/login のリクエストボディの上限。ID トークンは数 KB なので十分。
const maxAuthBodyBytes = 64 << 10

type authRequest struct {
	IDToken string `json:"id_token"`
}

// Signup は Google アカウントでユーザーを登録する（docs/api.md の POST /signup）。
func Signup(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := verifyIDToken(w, r, d.IDTokens)
		if !ok {
			return
		}

		// 許可するかどうかの判定にだけメールアドレスを使い、保存はしない
		if !identity.EmailVerified || !slices.Contains(d.AllowedEmails, strings.ToLower(identity.Email)) {
			httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden, "このアカウントは登録できません")
			return
		}

		user := postgres.User{GoogleSub: identity.Sub, Name: identity.Name}
		if err := d.Users.Create(r.Context(), &user); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict, "このアカウントはすでに登録されています")
				return
			}
			httpx.WriteInternalError(w, err)
			return
		}

		d.Sessions.Issue(w, user.ID)
		httpx.WriteJSON(w, http.StatusCreated, map[string]userResponse{"user": newUserResponse(user)})
	}
}

// Login は登録ずみの Google アカウントでログインする（docs/api.md の POST /login）。
func Login(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := verifyIDToken(w, r, d.IDTokens)
		if !ok {
			return
		}

		user, err := d.Users.FindByGoogleSub(r.Context(), identity.Sub)
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "このアカウントは登録されていません")
			return
		}
		if err != nil {
			httpx.WriteInternalError(w, err)
			return
		}

		d.Sessions.Issue(w, user.ID)
		httpx.WriteJSON(w, http.StatusOK, map[string]userResponse{"user": newUserResponse(user)})
	}
}

// verifyIDToken はリクエストボディの id_token を検証する。
// 失敗したときはエラーレスポンスを書いて false を返す。
func verifyIDToken(w http.ResponseWriter, r *http.Request, verifier auth.IDTokenVerifier) (auth.GoogleIdentity, bool) {
	var req authRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)).Decode(&req); err != nil || req.IDToken == "" {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationError, "id_token を指定してください")
		return auth.GoogleIdentity{}, false
	}

	identity, err := verifier.Verify(r.Context(), req.IDToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidIDToken) {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "ID トークンが正しくありません")
			return auth.GoogleIdentity{}, false
		}
		httpx.WriteInternalError(w, err)
		return auth.GoogleIdentity{}, false
	}
	return identity, true
}
