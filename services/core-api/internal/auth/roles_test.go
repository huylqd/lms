package auth

import "testing"

func TestTodoRolePriority(t *testing.T) {
	cases := []struct {
		roles []string
		want  string
	}{
		{nil, ""},
		{[]string{"partner", "offline_access"}, ""},
		{[]string{"student"}, TodoStudent},
		{[]string{"advisor"}, TodoLecturer},
		{[]string{"student", "lecturer"}, TodoLecturer},
		{[]string{"lecturer", "faculty_manager"}, TodoManager},
		{[]string{"student", "school_manager"}, TodoManager},
		{[]string{"admin"}, TodoManager},
		{[]string{"advisor", "student", "admin"}, TodoManager},
	}
	for _, c := range cases {
		if got := TodoRole(c.roles); got != c.want {
			t.Errorf("TodoRole(%v) = %q, want %q", c.roles, got, c.want)
		}
	}
}

func TestScopeFor(t *testing.T) {
	s := ScopeFor([]string{"student", "advisor"})
	if s.All || s.Faculty || !s.Staff || !s.Student {
		t.Fatalf("got %+v", s)
	}
	if s := ScopeFor([]string{"school_manager"}); !s.All {
		t.Fatalf("school_manager should see all: %+v", s)
	}
	if s := ScopeFor([]string{"partner"}); s != (Scope{}) {
		t.Fatalf("partner should see nothing: %+v", s)
	}
}

func ptr(s string) *string { return &s }

func TestScopeDecisions(t *testing.T) {
	fac := "f1"
	cases := []struct {
		name           string
		roles          []string
		facts          OfferingFacts
		view, privileg bool
	}{
		{"admin sees everything", []string{"admin"}, OfferingFacts{OfferingFacultyID: fac}, true, true},
		{"school manager sees everything", []string{"school_manager"}, OfferingFacts{OfferingFacultyID: fac}, true, true},
		{"faculty manager own faculty", []string{"faculty_manager"}, OfferingFacts{OfferingFacultyID: fac, UserFacultyID: ptr(fac)}, true, true},
		{"faculty manager other faculty", []string{"faculty_manager"}, OfferingFacts{OfferingFacultyID: fac, UserFacultyID: ptr("f2")}, false, false},
		{"faculty manager without faculty", []string{"faculty_manager"}, OfferingFacts{OfferingFacultyID: fac}, false, false},
		{"lecturer assigned", []string{"lecturer"}, OfferingFacts{OfferingFacultyID: fac, IsStaff: true}, true, true},
		{"lecturer not assigned", []string{"lecturer"}, OfferingFacts{OfferingFacultyID: fac, UserFacultyID: ptr(fac)}, false, false},
		{"advisor assigned", []string{"advisor"}, OfferingFacts{OfferingFacultyID: fac, IsStaff: true}, true, true},
		{"student enrolled", []string{"student"}, OfferingFacts{OfferingFacultyID: fac, IsEnrolled: true}, true, false},
		{"student not enrolled", []string{"student"}, OfferingFacts{OfferingFacultyID: fac, UserFacultyID: ptr(fac)}, false, false},
		{"student listed as staff without staff role", []string{"student"}, OfferingFacts{OfferingFacultyID: fac, IsStaff: true}, false, false},
		{"enrolled user without student role", []string{"partner"}, OfferingFacts{OfferingFacultyID: fac, IsEnrolled: true}, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := ScopeFor(c.roles)
			if got := s.CanView(c.facts); got != c.view {
				t.Errorf("CanView = %v, want %v", got, c.view)
			}
			if got := s.Privileged(c.facts); got != c.privileg {
				t.Errorf("Privileged = %v, want %v", got, c.privileg)
			}
		})
	}
}
