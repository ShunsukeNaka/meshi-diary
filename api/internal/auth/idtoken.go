package auth

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/api/idtoken"
)

// ErrInvalidIDToken は ID トークンの検証に失敗したときに返す。
var ErrInvalidIDToken = errors.New("invalid id token")

// GoogleIdentity は検証ずみの ID トークンから取り出した、ユーザーの情報。
type GoogleIdentity struct {
	Sub           string
	Email         string
	EmailVerified bool
	Name          string
}

// IDTokenVerifier は Google の ID トークンを検証する。テストでは偽物に差し替える。
type IDTokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (GoogleIdentity, error)
}

// GoogleVerifier は idtoken.Validate で ID トークンを検証する。
// 署名・iss・aud（クライアント ID）・exp を確かめる。
type GoogleVerifier struct {
	ClientID string
}

func (v GoogleVerifier) Verify(ctx context.Context, rawToken string) (GoogleIdentity, error) {
	p, err := idtoken.Validate(ctx, rawToken, v.ClientID)
	if err != nil {
		return GoogleIdentity{}, fmt.Errorf("%w: %v", ErrInvalidIDToken, err)
	}
	if p.Subject == "" {
		return GoogleIdentity{}, fmt.Errorf("%w: sub is empty", ErrInvalidIDToken)
	}
	email, _ := p.Claims["email"].(string)
	verified, _ := p.Claims["email_verified"].(bool)
	name, _ := p.Claims["name"].(string)
	return GoogleIdentity{Sub: p.Subject, Email: email, EmailVerified: verified, Name: name}, nil
}
