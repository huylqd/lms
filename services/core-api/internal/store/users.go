package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/huylqd/lms/services/core-api/internal/audit"
	"github.com/huylqd/lms/services/core-api/internal/auth"
)

const selectUserByUsername = `
	SELECT u.id::text, u.username, u.kc_subject, u.email, u.full_name,
	       u.faculty_id::text, f.code, f.name, u.status
	FROM users u LEFT JOIN faculties f ON f.id = u.faculty_id
	WHERE u.username = $1`

func lookupUser(ctx context.Context, q querier, username string) (auth.User, *string, error) {
	var u auth.User
	var kcSubject *string
	err := q.QueryRow(ctx, selectUserByUsername, username).Scan(
		&u.ID, &u.Username, &kcSubject, &u.Email, &u.FullName,
		&u.FacultyID, &u.FacultyCode, &u.FacultyName, &u.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, nil, ErrNotFound
	}
	if err != nil {
		return auth.User{}, nil, fmt.Errorf("lookup user: %w", err)
	}
	return u, kcSubject, nil
}

func checkLinked(u auth.User, kcSubject *string, sub string) (auth.User, error) {
	if kcSubject == nil || *kcSubject != sub {
		return auth.User{}, auth.ErrSubjectMismatch
	}
	if u.Status != "active" {
		return auth.User{}, auth.ErrInactive
	}
	return u, nil
}

// Provision implements auth.Provisioner: find users by username; link kc_subject
// on first login; create the row if missing. A username already linked to another
// subject (or a subject linked to another username) yields auth.ErrSubjectMismatch.
func (s *Store) Provision(ctx context.Context, id auth.Identity, meta audit.Meta) (auth.User, error) {
	// Fast path: already linked.
	u, kc, err := lookupUser(ctx, s.pool, id.Username)
	if err == nil && kc != nil {
		return checkLinked(u, kc, id.Subject)
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return auth.User{}, err
	}

	var result auth.User
	txErr := withTx(ctx, s.pool, func(tx pgx.Tx) error {
		u, kc, err := lookupUser(ctx, tx, id.Username)
		switch {
		case err == nil && kc != nil:
			result, err = checkLinked(u, kc, id.Subject)
			return err

		case err == nil: // exists, not linked yet
			tag, err := tx.Exec(ctx, `
				UPDATE users SET kc_subject = $2, email = COALESCE(email, NULLIF($3, '')), updated_at = now()
				WHERE id = $1::uuid AND kc_subject IS NULL`, u.ID, id.Subject, id.Email)
			if isUniqueViolation(err) {
				return auth.ErrSubjectMismatch
			}
			if err != nil {
				return fmt.Errorf("link user: %w", err)
			}
			if tag.RowsAffected() == 0 {
				return errRetry
			}
			if err := audit.Insert(ctx, tx, audit.Entry{
				ActorID: &u.ID, ActorUsername: u.Username, Action: "user.link", Entity: "user", EntityID: u.ID,
				After: map[string]string{"kc_subject": id.Subject}, Meta: meta,
			}); err != nil {
				return err
			}
			if u.Email == nil && id.Email != "" {
				e := id.Email
				u.Email = &e
			}
			if u.Status != "active" {
				return auth.ErrInactive
			}
			result = u
			return nil

		case errors.Is(err, ErrNotFound):
			fullName := id.Name
			if fullName == "" {
				fullName = id.Username
			}
			var newID string
			err := tx.QueryRow(ctx, `
				INSERT INTO users (username, kc_subject, email, full_name)
				VALUES ($1, $2, NULLIF($3, ''), $4)
				ON CONFLICT (username) DO NOTHING
				RETURNING id::text`, id.Username, id.Subject, id.Email, fullName).Scan(&newID)
			if errors.Is(err, pgx.ErrNoRows) {
				return errRetry
			}
			if isUniqueViolation(err) {
				return auth.ErrSubjectMismatch
			}
			if err != nil {
				return fmt.Errorf("create user: %w", err)
			}
			if err := audit.Insert(ctx, tx, audit.Entry{
				ActorID: &newID, ActorUsername: id.Username, Action: "user.provision", Entity: "user", EntityID: newID,
				After: map[string]string{"username": id.Username, "kc_subject": id.Subject, "full_name": fullName}, Meta: meta,
			}); err != nil {
				return err
			}
			result = auth.User{ID: newID, Username: id.Username, FullName: fullName, Status: "active"}
			if id.Email != "" {
				e := id.Email
				result.Email = &e
			}
			return nil

		default:
			return err
		}
	})
	if errors.Is(txErr, errRetry) {
		// A concurrent request linked/created the row first; re-read it.
		u, kc, err := lookupUser(ctx, s.pool, id.Username)
		if err != nil {
			return auth.User{}, err
		}
		return checkLinked(u, kc, id.Subject)
	}
	if txErr != nil {
		return auth.User{}, txErr
	}
	return result, nil
}

var errRetry = errors.New("concurrent provisioning, retry")
