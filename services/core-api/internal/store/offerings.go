package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/huylqd/lms/services/core-api/internal/auth"
)

// Course is a học phần.
type Course struct {
	ID      string  `json:"-"`
	Code    string  `json:"code"`
	Name    string  `json:"name"`
	Credits float64 `json:"credits"`
}

// Term is a học kỳ.
type Term struct {
	ID       string `json:"-"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	StartsOn string `json:"-"`
	EndsOn   string `json:"-"`
}

// Offering is a lớp học phần.
type Offering struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Course Course `json:"course"`
	Term   Term   `json:"term"`
}

// StaffMember is an offering_staff entry.
type StaffMember struct {
	UserID   string `json:"-"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

// Task is a published nhiệm vụ.
type Task struct {
	ID              string     `json:"id"`
	Kind            string     `json:"kind"`
	Title           string     `json:"title"`
	Instructions    string     `json:"instructions"`
	AIPolicy        string     `json:"ai_policy"`
	AssignmentScope string     `json:"-"`
	Status          string     `json:"-"`
	OpensAt         *time.Time `json:"-"`
	DueAt           *time.Time `json:"due_at"`
	RubricID        *string    `json:"-"`
	Component       *string    `json:"-"`
}

// Material is an approved học liệu.
type Material struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Kind  string  `json:"kind"`
	URL   *string `json:"url"`
}

// OfferingDetail is the response of GET /offerings/{id}.
type OfferingDetail struct {
	Offering
	Staff     []StaffMember `json:"staff"`
	Tasks     []Task        `json:"tasks"`
	Materials []Material    `json:"materials"`
}

const offeringColumns = `
	o.id::text, o.code, o.title, o.status,
	c.id::text, c.code, c.name, c.credits::float8,
	tm.id::text, tm.code, tm.name, tm.starts_on::text, tm.ends_on::text`

const offeringFrom = `
	FROM course_offerings o
	JOIN courses c ON c.id = o.course_id
	JOIN terms tm ON tm.id = o.term_id`

func scanOffering(row pgx.Row, o *Offering) error {
	return row.Scan(&o.ID, &o.Code, &o.Title, &o.Status,
		&o.Course.ID, &o.Course.Code, &o.Course.Name, &o.Course.Credits,
		&o.Term.ID, &o.Term.Code, &o.Term.Name, &o.Term.StartsOn, &o.Term.EndsOn)
}

// ListOfferings returns offerings in the caller's scope (union of role scopes).
func (s *Store) ListOfferings(ctx context.Context, p *auth.Principal) ([]Offering, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+offeringColumns+offeringFrom+`
		WHERE $1::bool
		   OR ($2::bool AND c.faculty_id = $5::uuid)
		   OR ($3::bool AND EXISTS (SELECT 1 FROM offering_staff st WHERE st.offering_id = o.id AND st.user_id = $6::uuid))
		   OR ($4::bool AND EXISTS (SELECT 1 FROM enrollments e WHERE e.offering_id = o.id AND e.student_id = $6::uuid AND e.status = 'enrolled'))
		ORDER BY tm.starts_on DESC, o.code`,
		p.Scope.All, p.Scope.Faculty, p.Scope.Staff, p.Scope.Student, p.User.FacultyID, p.User.ID)
	if err != nil {
		return nil, fmt.Errorf("list offerings: %w", err)
	}
	defer rows.Close()
	out := []Offering{}
	for rows.Next() {
		var o Offering
		if err := scanOffering(rows, &o); err != nil {
			return nil, fmt.Errorf("scan offering: %w", err)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// OfferingFacts loads the facts for an access decision. ErrNotFound if the offering does not exist.
func (s *Store) OfferingFacts(ctx context.Context, offeringID string, user auth.User) (auth.OfferingFacts, error) {
	f := auth.OfferingFacts{UserFacultyID: user.FacultyID}
	err := s.pool.QueryRow(ctx, `
		SELECT c.faculty_id::text,
		       EXISTS (SELECT 1 FROM offering_staff st WHERE st.offering_id = o.id AND st.user_id = $2::uuid),
		       EXISTS (SELECT 1 FROM enrollments e WHERE e.offering_id = o.id AND e.student_id = $2::uuid AND e.status = 'enrolled')
		FROM course_offerings o JOIN courses c ON c.id = o.course_id
		WHERE o.id = $1::uuid`, offeringID, user.ID).Scan(&f.OfferingFacultyID, &f.IsStaff, &f.IsEnrolled)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrNotFound
	}
	if err != nil {
		return f, fmt.Errorf("offering facts: %w", err)
	}
	return f, nil
}

// OfferingDetail loads an offering with staff, published tasks and approved materials.
// When privileged is false (caller sees it only as an enrolled student), tasks with
// assignment_scope='selected' are limited to those assigned to userID.
func (s *Store) OfferingDetail(ctx context.Context, offeringID, userID string, privileged bool) (OfferingDetail, error) {
	var d OfferingDetail
	err := scanOffering(s.pool.QueryRow(ctx, `SELECT `+offeringColumns+offeringFrom+` WHERE o.id = $1::uuid`, offeringID), &d.Offering)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, fmt.Errorf("offering: %w", err)
	}

	d.Staff, err = collect(ctx, s, `
		SELECT u.id::text, u.full_name, st.role
		FROM offering_staff st JOIN users u ON u.id = st.user_id
		WHERE st.offering_id = $1::uuid
		ORDER BY CASE st.role WHEN 'lecturer' THEN 0 WHEN 'mentor' THEN 1 ELSE 2 END, u.full_name`,
		[]any{offeringID}, func(r pgx.Rows) (StaffMember, error) {
			var m StaffMember
			return m, r.Scan(&m.UserID, &m.FullName, &m.Role)
		})
	if err != nil {
		return d, err
	}

	d.Tasks, err = collect(ctx, s, `
		SELECT t.id::text, t.kind, t.title, t.instructions, t.ai_policy, t.assignment_scope, t.status,
		       t.opens_at, COALESCE(ta.due_at_override, t.due_at), t.rubric_id::text, gc.code
		FROM tasks t
		LEFT JOIN grade_components gc ON gc.id = t.component_id
		LEFT JOIN task_assignees ta ON ta.task_id = t.id AND ta.student_id = $2::uuid
		WHERE t.offering_id = $1::uuid AND t.status = 'published'
		  AND ($3::bool OR t.assignment_scope = 'all' OR ta.student_id IS NOT NULL)
		ORDER BY t.due_at NULLS LAST, t.created_at, t.title`,
		[]any{offeringID, userID, privileged}, func(r pgx.Rows) (Task, error) {
			var t Task
			return t, r.Scan(&t.ID, &t.Kind, &t.Title, &t.Instructions, &t.AIPolicy, &t.AssignmentScope, &t.Status,
				&t.OpensAt, &t.DueAt, &t.RubricID, &t.Component)
		})
	if err != nil {
		return d, err
	}

	d.Materials, err = collect(ctx, s, `
		SELECT m.id::text, m.title, m.kind, m.url
		FROM learning_materials m
		WHERE m.offering_id = $1::uuid AND m.approved_by IS NOT NULL AND m.approved_at IS NOT NULL
		ORDER BY m.created_at, m.title`,
		[]any{offeringID}, func(r pgx.Rows) (Material, error) {
			var m Material
			return m, r.Scan(&m.ID, &m.Title, &m.Kind, &m.URL)
		})
	return d, err
}

func collect[T any](ctx context.Context, s *Store, sql string, args []any, scan func(pgx.Rows) (T, error)) ([]T, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}
