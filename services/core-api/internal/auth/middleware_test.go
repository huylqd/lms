package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/huylqd/lms/services/core-api/internal/audit"
)

const (
	testIssuer   = "http://lms.example.test/idp/realms/lms"
	testAudience = "lms-api"
)

type fakeProvisioner struct {
	err  error
	got  Identity
	call int
}

func (f *fakeProvisioner) Provision(_ context.Context, id Identity, _ audit.Meta) (User, error) {
	f.call++
	f.got = id
	if f.err != nil {
		return User{}, f.err
	}
	return User{ID: "00000000-0000-0000-0000-000000000001", Username: id.Username, FullName: id.Name, Status: "active"}, nil
}

type testIDP struct {
	key    *rsa.PrivateKey
	signer jose.Signer
	jwks   *httptest.Server
}

func newTestIDP(t *testing.T) *testIDP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const kid = "test-key"
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: jose.JSONWebKey{Key: key, KeyID: kid}},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		t.Fatal(err)
	}
	set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(set)
	}))
	t.Cleanup(srv.Close)
	return &testIDP{key: key, signer: signer, jwks: srv}
}

type claims struct {
	jwt.Claims
	PreferredUsername string `json:"preferred_username,omitempty"`
	Email             string `json:"email,omitempty"`
	Name              string `json:"name,omitempty"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

func (idp *testIDP) token(t *testing.T, mutate func(*claims)) string {
	t.Helper()
	now := time.Now()
	c := claims{
		Claims: jwt.Claims{
			Issuer:   testIssuer,
			Subject:  "kc-sub-1",
			Audience: jwt.Audience{testAudience, "account"},
			IssuedAt: jwt.NewNumericDate(now),
			Expiry:   jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
		PreferredUsername: "sv01",
		Email:             "sv01@example.test",
		Name:              "Nguyễn Văn An",
	}
	c.RealmAccess.Roles = []string{"student", "offline_access"}
	if mutate != nil {
		mutate(&c)
	}
	raw, err := jwt.Signed(idp.signer).Claims(c).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func runMiddleware(t *testing.T, idp *testIDP, prov Provisioner, authHeader string) (*httptest.ResponseRecorder, *Principal) {
	t.Helper()
	v := NewOIDCVerifier(testIssuer, idp.jwks.URL, testAudience)
	var seen *Principal
	h := Middleware(v, prov)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = PrincipalFrom(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec, seen
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, status, rec.Body.String())
	}
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %s", rec.Body.String())
	}
	if body.Error.Code != code {
		t.Fatalf("error code = %q, want %q", body.Error.Code, code)
	}
}

func TestMiddlewareValidToken(t *testing.T) {
	idp := newTestIDP(t)
	prov := &fakeProvisioner{}
	rec, p := runMiddleware(t, idp, prov, "Bearer "+idp.token(t, nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if p == nil || p.User.Username != "sv01" || !HasRole(p.Roles, RoleStudent) || !p.Scope.Student {
		t.Fatalf("principal %+v", p)
	}
	if prov.got.Subject != "kc-sub-1" || prov.got.Email != "sv01@example.test" || prov.got.Name != "Nguyễn Văn An" {
		t.Fatalf("identity %+v", prov.got)
	}
}

func TestMiddlewareRejects(t *testing.T) {
	idp := newTestIDP(t)
	other := newTestIDP(t) // different signing key, not in idp's JWKS

	cases := map[string]string{
		"missing header":   "",
		"wrong scheme":     "Basic abc",
		"garbage":          "Bearer not-a-jwt",
		"wrong issuer":     "Bearer " + idp.token(t, func(c *claims) { c.Issuer = "http://evil.test/realms/lms" }),
		"missing audience": "Bearer " + idp.token(t, func(c *claims) { c.Audience = jwt.Audience{"account"} }),
		"expired": "Bearer " + idp.token(t, func(c *claims) {
			c.IssuedAt = jwt.NewNumericDate(time.Now().Add(-2 * time.Hour))
			c.Expiry = jwt.NewNumericDate(time.Now().Add(-time.Hour))
		}),
		"unknown key":      "Bearer " + other.token(t, nil),
		"missing username": "Bearer " + idp.token(t, func(c *claims) { c.PreferredUsername = "" }),
	}
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			prov := &fakeProvisioner{}
			rec, p := runMiddleware(t, idp, prov, header)
			if header == "" || name == "wrong scheme" {
				assertErrorCode(t, rec, http.StatusUnauthorized, "unauthorized")
			} else {
				assertErrorCode(t, rec, http.StatusUnauthorized, "invalid_token")
			}
			if p != nil || prov.call != 0 {
				t.Fatal("handler or provisioner must not run")
			}
		})
	}
}

func TestMiddlewareSubjectMismatch(t *testing.T) {
	idp := newTestIDP(t)
	rec, _ := runMiddleware(t, idp, &fakeProvisioner{err: ErrSubjectMismatch}, "Bearer "+idp.token(t, nil))
	assertErrorCode(t, rec, http.StatusForbidden, "account_conflict")
}
