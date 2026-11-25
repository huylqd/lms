package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/huylqd/lms/services/core-api/internal/audit"
)

// TutorTarget is the task context needed for an AI tutor call.
type TutorTarget struct {
	TaskID       string
	OfferingID   string
	CourseName   string
	TaskTitle    string
	Instructions string
	AIPolicy     string
}

// TutorTarget loads a published task that the student is enrolled in (and assigned to,
// for assignment_scope='selected'). ErrNotFound otherwise.
func (s *Store) TutorTarget(ctx context.Context, studentID, taskID string) (TutorTarget, error) {
	var t TutorTarget
	err := s.pool.QueryRow(ctx, `
		SELECT t.id::text, t.offering_id::text, c.name, t.title, t.instructions, t.ai_policy
		FROM tasks t
		JOIN course_offerings o ON o.id = t.offering_id
		JOIN courses c ON c.id = o.course_id
		JOIN enrollments e ON e.offering_id = o.id AND e.student_id = $2::uuid AND e.status = 'enrolled'
		LEFT JOIN task_assignees ta ON ta.task_id = t.id AND ta.student_id = $2::uuid
		WHERE t.id = $1::uuid AND t.status = 'published'
		  AND (t.assignment_scope = 'all' OR ta.student_id IS NOT NULL)`,
		taskID, studentID).Scan(&t.TaskID, &t.OfferingID, &t.CourseName, &t.TaskTitle, &t.Instructions, &t.AIPolicy)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, fmt.Errorf("tutor target: %w", err)
	}
	return t, nil
}

// TutorRecord describes a completed tutor interaction. The prompt text is never stored.
type TutorRecord struct {
	UserID        string
	Username      string
	TaskID        string
	Policy        string
	Provider      string
	Model         string
	PromptChars   int
	ResponseChars int
	Meta          audit.Meta
}

// RecordTutor inserts ai_interactions and audit_log in one transaction.
func (s *Store) RecordTutor(ctx context.Context, rec TutorRecord) error {
	return withTx(ctx, s.pool, func(tx pgx.Tx) error {
		var interactionID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO ai_interactions (user_id, task_id, feature, provider, model, prompt_chars, response_chars)
			VALUES ($1::uuid, $2::uuid, 'tutor', $3, $4, $5, $6)
			RETURNING id`,
			rec.UserID, rec.TaskID, rec.Provider, rec.Model, rec.PromptChars, rec.ResponseChars).Scan(&interactionID); err != nil {
			return fmt.Errorf("insert ai_interactions: %w", err)
		}
		return audit.Insert(ctx, tx, audit.Entry{
			ActorID:       &rec.UserID,
			ActorUsername: rec.Username,
			Action:        "ai.tutor",
			Entity:        "task",
			EntityID:      rec.TaskID,
			After: map[string]any{
				"ai_interaction_id": interactionID,
				"policy":            rec.Policy,
				"provider":          rec.Provider,
				"model":             rec.Model,
				"prompt_chars":      rec.PromptChars,
				"response_chars":    rec.ResponseChars,
			},
			Meta: rec.Meta,
		})
	})
}
