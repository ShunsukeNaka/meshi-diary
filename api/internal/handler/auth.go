package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/httpx"
)

// maxAuthBodyBytes は /signup・/login のリクエストボディの上限。ID トークンは数 KB なので十分。
const maxAuthBodyBytes = 64 << 10

type authRequest struct {
	IDToken string `json:"id_token"`
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
