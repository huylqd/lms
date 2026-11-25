package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/huylqd/lms/services/core-api/internal/audit"
	"github.com/huylqd/lms/services/core-api/internal/respond"
)

// User is the local users row of the caller.
type User struct {
	ID          string
	Username    string
	FullName    string
	Email       *string
	FacultyID   *string
	FacultyCode *string
	FacultyName *string
	Status      string
}

// Principal is the authenticated caller stored in the request context.
type Principal struct {
	User  User
	Roles []string
	Scope Scope
}

// Provisioning errors.
var (
	// ErrSubjectMismatch: a users row with this username is linked to a different Keycloak subject
	// (or the subject is linked to a different username). Never relinked silently.
	ErrSubjectMismatch = errors.New("account is linked to a different identity")
	// ErrInactive: users.status = 'inactive'.
	ErrInactive = errors.New("account is inactive")
)

// Provisioner finds or creates (JIT) the local user for a verified identity.
type Provisioner interface {
	Provision(ctx context.Context, id Identity, meta audit.Meta) (User, error)
}

type principalKey struct{}

// WithPrincipal stores p in ctx.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom returns the authenticated caller, or nil.
func PrincipalFrom(ctx context.Context) *Principal {
	p, _ := ctx.Value(principalKey{}).(*Principal)
	return p
}

// Middleware authenticates the bearer token, provisions the user and stores
// the Principal in the request context.
func Middleware(v TokenVerifier, prov Provisioner) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := bearerToken(r)
			if !ok {
				respond.Unauthorized(w, "unauthorized", "missing bearer token")
				return
			}
			id, err := v.Verify(r.Context(), raw)
			if err != nil {
				slog.InfoContext(r.Context(), "token rejected", "request_id", audit.RequestID(r.Context()), "reason", err.Error())
				respond.Unauthorized(w, "invalid_token", "invalid or expired token")
				return
			}
			user, err := prov.Provision(r.Context(), id, audit.Meta{RequestID: audit.RequestID(r.Context()), IP: audit.ClientIP(r)})
			switch {
			case errors.Is(err, ErrSubjectMismatch):
				slog.WarnContext(r.Context(), "identity conflict", "request_id", audit.RequestID(r.Context()), "username", id.Username)
				respond.Forbidden(w, "account_conflict", "account is linked to a different identity")
				return
			case errors.Is(err, ErrInactive):
				respond.Forbidden(w, "account_inactive", "account is inactive")
				return
			case err != nil:
				slog.ErrorContext(r.Context(), "provision user failed", "request_id", audit.RequestID(r.Context()), "err", err)
				respond.Internal(w)
				return
			}
			p := &Principal{User: user, Roles: id.Roles, Scope: ScopeFor(id.Roles)}
			if p.Roles == nil {
				p.Roles = []string{}
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	scheme, tok, found := strings.Cut(h, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	tok = strings.TrimSpace(tok)
	return tok, tok != ""
}
