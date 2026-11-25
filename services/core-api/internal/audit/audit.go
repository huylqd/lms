// Package audit writes append-only audit_log rows and carries request metadata
// (request ID, client IP) needed for them.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type ctxKey struct{}

// WithRequestID stores the request ID in ctx.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// RequestID returns the request ID stored in ctx, or "".
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// ClientIP returns the client address. Behind Caddy the first X-Forwarded-For
// entry is the client (Caddy overwrites untrusted incoming values); otherwise
// the TCP peer is used. Returns "" if nothing parses as an IP.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first := strings.TrimSpace(strings.Split(xff, ",")[0])
		if ip := net.ParseIP(first); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return ""
}

// Meta is request metadata recorded with each audit entry.
type Meta struct {
	RequestID string
	IP        string
}

// Entry is one audit_log row.
type Entry struct {
	ActorID       *string
	ActorUsername string
	Action        string
	Entity        string
	EntityID      string
	Before        any
	After         any
	Meta          Meta
}

// Insert writes e inside tx. Callers must use the same transaction as the
// business change being audited.
func Insert(ctx context.Context, tx pgx.Tx, e Entry) error {
	before, err := marshalNullable(e.Before)
	if err != nil {
		return err
	}
	after, err := marshalNullable(e.After)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_log (actor_id, actor_username, action, entity, entity_id, before, after, request_id, ip)
		VALUES ($1::uuid, NULLIF($2, ''), $3, $4, $5, $6::jsonb, $7::jsonb, NULLIF($8, ''), NULLIF($9, '')::inet)`,
		e.ActorID, e.ActorUsername, e.Action, e.Entity, e.EntityID, before, after, e.Meta.RequestID, e.Meta.IP)
	if err != nil {
		return fmt.Errorf("insert audit_log: %w", err)
	}
	return nil
}

func marshalNullable(v any) (*string, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal audit payload: %w", err)
	}
	s := string(b)
	return &s, nil
}
