package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Identity is what core-api trusts from a verified access token.
type Identity struct {
	Subject  string
	Username string
	Email    string
	Name     string
	Roles    []string
}

// TokenVerifier verifies a raw bearer token.
type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (Identity, error)
}

// OIDCVerifier verifies Keycloak access tokens: signature via the internal
// JWKS URL, issuer = public issuer URL, audience contains OIDC_AUDIENCE, expiry.
type OIDCVerifier struct {
	v *oidc.IDTokenVerifier
}

// NewOIDCVerifier builds a verifier. Keys are fetched lazily from jwksURL and
// cached by go-oidc; no network call happens here.
func NewOIDCVerifier(issuer, jwksURL, audience string) *OIDCVerifier {
	ctx := oidc.ClientContext(context.Background(), &http.Client{Timeout: 10 * time.Second})
	keySet := oidc.NewRemoteKeySet(ctx, jwksURL)
	return &OIDCVerifier{v: oidc.NewVerifier(issuer, keySet, &oidc.Config{ClientID: audience})}
}

type tokenClaims struct {
	Subject           string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

// Verify implements TokenVerifier.
func (o *OIDCVerifier) Verify(ctx context.Context, raw string) (Identity, error) {
	tok, err := o.v.Verify(ctx, raw)
	if err != nil {
		return Identity{}, err
	}
	var c tokenClaims
	if err := tok.Claims(&c); err != nil {
		return Identity{}, fmt.Errorf("decode claims: %w", err)
	}
	if c.Subject == "" || c.PreferredUsername == "" {
		return Identity{}, errors.New("token lacks sub or preferred_username")
	}
	return Identity{
		Subject:  c.Subject,
		Username: c.PreferredUsername,
		Email:    c.Email,
		Name:     c.Name,
		Roles:    c.RealmAccess.Roles,
	}, nil
}
