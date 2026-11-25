// Package auth verifies Keycloak access tokens, provisions users just in time
// and implements role-based scoping decisions.
package auth

import "slices"

// Realm roles (read from the token, never stored in the DB).
const (
	RoleStudent        = "student"
	RoleLecturer       = "lecturer"
	RoleAdvisor        = "advisor"
	RoleFacultyManager = "faculty_manager"
	RoleSchoolManager  = "school_manager"
	RolePartner        = "partner"
	RoleAdmin          = "admin"
)

// Todo views returned by /me/todo.
const (
	TodoManager  = "manager"
	TodoLecturer = "lecturer"
	TodoStudent  = "student"
)

// HasRole reports whether roles contains any of want.
func HasRole(roles []string, want ...string) bool {
	for _, w := range want {
		if slices.Contains(roles, w) {
			return true
		}
	}
	return false
}

// TodoRole picks the /me/todo view by priority:
// manager (admin, school_manager, faculty_manager) > lecturer (lecturer, advisor) > student.
// It returns "" when no applicable role is present.
func TodoRole(roles []string) string {
	switch {
	case HasRole(roles, RoleAdmin, RoleSchoolManager, RoleFacultyManager):
		return TodoManager
	case HasRole(roles, RoleLecturer, RoleAdvisor):
		return TodoLecturer
	case HasRole(roles, RoleStudent):
		return TodoStudent
	default:
		return ""
	}
}
