package auth

// Scope describes which course offerings a caller may see. Multiple roles
// combine as a union.
type Scope struct {
	All     bool // school_manager, admin
	Faculty bool // faculty_manager: offerings whose course belongs to users.faculty_id
	Staff   bool // lecturer, advisor: offerings listed in offering_staff
	Student bool // student: offerings with an active enrollment
}

// ScopeFor derives the scope from realm roles.
func ScopeFor(roles []string) Scope {
	return Scope{
		All:     HasRole(roles, RoleAdmin, RoleSchoolManager),
		Faculty: HasRole(roles, RoleFacultyManager),
		Staff:   HasRole(roles, RoleLecturer, RoleAdvisor),
		Student: HasRole(roles, RoleStudent),
	}
}

// OfferingFacts are the per-offering facts needed for an access decision.
type OfferingFacts struct {
	OfferingFacultyID string  // faculty of the offering's course
	UserFacultyID     *string // users.faculty_id of the caller
	IsStaff           bool    // caller is in offering_staff for the offering
	IsEnrolled        bool    // caller has an 'enrolled' enrollment
}

// CanView reports whether the caller may see the offering at all.
func (s Scope) CanView(f OfferingFacts) bool {
	return s.Privileged(f) || (s.Student && f.IsEnrolled)
}

// Privileged reports whether the caller sees the offering as staff/manager
// (all published tasks, including tasks assigned to selected students only).
// A caller who only sees it as an enrolled student is not privileged.
func (s Scope) Privileged(f OfferingFacts) bool {
	if s.All {
		return true
	}
	if s.Faculty && f.UserFacultyID != nil && *f.UserFacultyID == f.OfferingFacultyID {
		return true
	}
	return s.Staff && f.IsStaff
}
