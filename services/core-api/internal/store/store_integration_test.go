package store

// Integration test against a DISPOSABLE database that already has migrations and the
// demo seed applied (it writes rows). Skipped unless LMS_TEST_DATABASE_URL is set.

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/huylqd/lms/services/core-api/internal/audit"
	"github.com/huylqd/lms/services/core-api/internal/auth"
)

func TestIntegrationSeededDB(t *testing.T) {
	dsn := os.Getenv("LMS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("LMS_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	s := New(pool)
	meta := audit.Meta{RequestID: "it-test", IP: "127.0.0.1"}

	login := func(username, sub string, roles ...string) *auth.Principal {
		t.Helper()
		u, err := s.Provision(ctx, auth.Identity{Subject: sub, Username: username, Roles: roles}, meta)
		if err != nil {
			t.Fatalf("provision %s: %v", username, err)
		}
		return &auth.Principal{User: u, Roles: roles, Scope: auth.ScopeFor(roles)}
	}

	sv01 := login("sv01", "sub-sv01", "student")
	if _, err := s.Provision(ctx, auth.Identity{Subject: "other", Username: "sv01"}, meta); !errors.Is(err, auth.ErrSubjectMismatch) {
		t.Fatalf("expected mismatch, got %v", err)
	}
	newbie := login("newuser01", "sub-new", "student")
	if newbie.User.FullName != "newuser01" {
		t.Fatalf("new user %+v", newbie.User)
	}

	items, err := s.StudentTodo(ctx, sv01.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[string]string{}
	for _, it := range items {
		statuses[it.Title] = it.Status
	}
	t.Logf("sv01 todo: %v", statuses)
	if len(items) != 4 || statuses["Phân tích chân dung khách hàng mục tiêu"] != "submitted" ||
		statuses["Lập kế hoạch nội dung tuần 1"] != "overdue" || statuses["Kiểm tra trước buổi 1"] != "not_submitted" {
		t.Fatalf("unexpected student todo %v", statuses)
	}

	gv01 := login("gv01", "sub-gv01", "lecturer")
	lt, err := s.LecturerTodo(ctx, gv01.User.ID)
	if err != nil || len(lt) != 1 || lt[0].PendingAssessments != 2 || lt[0].OpenAlerts != 1 {
		t.Fatalf("lecturer todo %+v %v", lt, err)
	}

	qlk := login("qlk01", "sub-qlk01", "faculty_manager")
	mt, err := s.ManagerTodo(ctx, false, qlk.User.FacultyID)
	if err != nil || len(mt) != 2 {
		t.Fatalf("manager todo %+v %v", mt, err)
	}
	t.Logf("manager todo: %+v", mt)

	sv06 := login("sv06", "sub-sv06", "student")
	offs, err := s.ListOfferings(ctx, sv06)
	if err != nil || len(offs) != 1 || offs[0].Code != "MKT201-01" {
		t.Fatalf("sv06 offerings %+v %v", offs, err)
	}
	if offs, _ := s.ListOfferings(ctx, newbie); len(offs) != 0 {
		t.Fatalf("newbie sees %d offerings", len(offs))
	}
	admin := login("admin01", "sub-admin01", "admin")
	if offs, _ := s.ListOfferings(ctx, admin); len(offs) != 2 {
		t.Fatalf("admin sees %d offerings", len(offs))
	}

	mkt := offs[0].ID
	facts, err := s.OfferingFacts(ctx, mkt, sv06.User)
	if err != nil || !sv06.Scope.CanView(facts) || sv06.Scope.Privileged(facts) {
		t.Fatalf("facts %+v %v", facts, err)
	}
	d, err := s.OfferingDetail(ctx, mkt, sv06.User.ID, false)
	if err != nil || len(d.Tasks) != 3 || len(d.Materials) != 2 || len(d.Staff) != 2 {
		t.Fatalf("detail tasks=%d materials=%d staff=%d err=%v", len(d.Tasks), len(d.Materials), len(d.Staff), err)
	}

	var assignID string
	for _, tk := range d.Tasks {
		if tk.AIPolicy == "explain_only" {
			assignID = tk.ID
		}
	}
	tgt, err := s.TutorTarget(ctx, sv06.User.ID, assignID)
	if err != nil || tgt.CourseName != "Marketing số" {
		t.Fatalf("tutor target %+v %v", tgt, err)
	}
	if _, err := s.TutorTarget(ctx, newbie.User.ID, assignID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-enrolled tutor target: %v", err)
	}
	if err := s.RecordTutor(ctx, TutorRecord{UserID: sv06.User.ID, Username: "sv06", TaskID: assignID, Policy: "explain_only",
		Provider: "mock", Model: "m", PromptChars: 10, ResponseChars: 20, Meta: meta}); err != nil {
		t.Fatal(err)
	}
}
