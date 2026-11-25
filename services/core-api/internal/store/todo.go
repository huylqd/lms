package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// StudentTodoItem is one task for a student.
type StudentTodoItem struct {
	TaskID        string     `json:"task_id"`
	OfferingID    string     `json:"offering_id"`
	OfferingTitle string     `json:"offering_title"`
	Title         string     `json:"title"`
	Kind          string     `json:"kind"`
	DueAt         *time.Time `json:"due_at"`
	Status        string     `json:"status"` // not_submitted | submitted | overdue
}

// LecturerTodoItem summarises an offering for lecturers/advisors.
type LecturerTodoItem struct {
	OfferingID         string `json:"offering_id"`
	OfferingTitle      string `json:"offering_title"`
	PendingAssessments int64  `json:"pending_assessments"`
	OpenAlerts         int64  `json:"open_alerts"`
}

// ManagerTodoItem summarises an offering for managers/admins.
type ManagerTodoItem struct {
	OfferingID         string `json:"offering_id"`
	OfferingTitle      string `json:"offering_title"`
	Students           int64  `json:"students"`
	OverdueSubmissions int64  `json:"overdue_submissions"`
	OpenAlerts         int64  `json:"open_alerts"`
}

// "Submitted" = any submission version exists (or, for quizzes, a submitted attempt).
const submittedByEnrollee = `(EXISTS (SELECT 1 FROM submissions s WHERE s.task_id = t.id AND s.student_id = e.student_id)
	OR EXISTS (SELECT 1 FROM quiz_attempts qa WHERE qa.task_id = t.id AND qa.student_id = e.student_id AND qa.submitted_at IS NOT NULL))`

// StudentTodo lists published tasks assigned to the student in enrolled offerings.
func (s *Store) StudentTodo(ctx context.Context, studentID string) ([]StudentTodoItem, error) {
	return collect(ctx, s, `
		SELECT t.id::text, o.id::text, o.title, t.title, t.kind, x.due_at,
		       CASE WHEN `+submittedByEnrollee+` THEN 'submitted'
		            WHEN x.due_at < now() THEN 'overdue'
		            ELSE 'not_submitted' END
		FROM enrollments e
		JOIN course_offerings o ON o.id = e.offering_id
		JOIN tasks t ON t.offering_id = o.id AND t.status = 'published'
		LEFT JOIN task_assignees ta ON ta.task_id = t.id AND ta.student_id = e.student_id
		CROSS JOIN LATERAL (SELECT COALESCE(ta.due_at_override, t.due_at) AS due_at) x
		WHERE e.student_id = $1::uuid AND e.status = 'enrolled'
		  AND (t.assignment_scope = 'all' OR ta.student_id IS NOT NULL)
		ORDER BY x.due_at NULLS LAST, o.title, t.title`,
		[]any{studentID}, func(r pgx.Rows) (StudentTodoItem, error) {
			var it StudentTodoItem
			return it, r.Scan(&it.TaskID, &it.OfferingID, &it.OfferingTitle, &it.Title, &it.Kind, &it.DueAt, &it.Status)
		})
}

// LecturerTodo summarises offerings where the user is in offering_staff.
// pending_assessments = latest submission versions without a submitted/approved assessment.
func (s *Store) LecturerTodo(ctx context.Context, userID string) ([]LecturerTodoItem, error) {
	return collect(ctx, s, `
		SELECT o.id::text, o.title,
		  (SELECT count(*) FROM (
		      SELECT DISTINCT ON (sb.task_id, sb.student_id) sb.id
		      FROM submissions sb JOIN tasks t ON t.id = sb.task_id
		      WHERE t.offering_id = o.id
		      ORDER BY sb.task_id, sb.student_id, sb.version DESC
		   ) latest
		   WHERE NOT EXISTS (SELECT 1 FROM assessments a
		                     WHERE a.submission_id = latest.id AND a.status IN ('submitted', 'approved'))),
		  (SELECT count(*) FROM alerts al WHERE al.offering_id = o.id AND al.status IN ('open', 'assigned'))
		FROM course_offerings o
		WHERE EXISTS (SELECT 1 FROM offering_staff st WHERE st.offering_id = o.id AND st.user_id = $1::uuid)
		ORDER BY o.title`,
		[]any{userID}, func(r pgx.Rows) (LecturerTodoItem, error) {
			var it LecturerTodoItem
			return it, r.Scan(&it.OfferingID, &it.OfferingTitle, &it.PendingAssessments, &it.OpenAlerts)
		})
}

// ManagerTodo summarises offerings for managers: all offerings when all is true,
// otherwise those whose course belongs to facultyID (nil → none).
func (s *Store) ManagerTodo(ctx context.Context, all bool, facultyID *string) ([]ManagerTodoItem, error) {
	return collect(ctx, s, `
		SELECT o.id::text, o.title,
		  (SELECT count(*) FROM enrollments e WHERE e.offering_id = o.id AND e.status = 'enrolled'),
		  (SELECT count(*)
		     FROM tasks t
		     JOIN enrollments e ON e.offering_id = t.offering_id AND e.status = 'enrolled'
		     LEFT JOIN task_assignees ta ON ta.task_id = t.id AND ta.student_id = e.student_id
		     WHERE t.offering_id = o.id AND t.status = 'published'
		       AND (t.assignment_scope = 'all' OR ta.student_id IS NOT NULL)
		       AND COALESCE(ta.due_at_override, t.due_at) < now()
		       AND NOT `+submittedByEnrollee+`),
		  (SELECT count(*) FROM alerts al WHERE al.offering_id = o.id AND al.status IN ('open', 'assigned'))
		FROM course_offerings o JOIN courses c ON c.id = o.course_id
		WHERE $1::bool OR c.faculty_id = $2::uuid
		ORDER BY o.title`,
		[]any{all, facultyID}, func(r pgx.Rows) (ManagerTodoItem, error) {
			var it ManagerTodoItem
			return it, r.Scan(&it.OfferingID, &it.OfferingTitle, &it.Students, &it.OverdueSubmissions, &it.OpenAlerts)
		})
}
