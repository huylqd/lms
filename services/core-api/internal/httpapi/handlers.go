package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/huylqd/lms/services/core-api/internal/aiclient"
	"github.com/huylqd/lms/services/core-api/internal/audit"
	"github.com/huylqd/lms/services/core-api/internal/auth"
	"github.com/huylqd/lms/services/core-api/internal/respond"
	"github.com/huylqd/lms/services/core-api/internal/store"
)

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// MaxQuestionChars is the maximum tutor question length (in characters).
const MaxQuestionChars = 2000

func (a *api) fail(w http.ResponseWriter, r *http.Request, msg string, err error) {
	a.logger.ErrorContext(r.Context(), msg, "request_id", audit.RequestID(r.Context()), "err", err)
	respond.Internal(w)
}

func (a *api) healthz(w http.ResponseWriter, _ *http.Request) {
	respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *api) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.store.Ping(ctx); err != nil {
		a.logger.WarnContext(r.Context(), "readiness check failed", "err", err)
		respond.Error(w, http.StatusServiceUnavailable, "not_ready", "database unavailable")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type facultyJSON struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type userJSON struct {
	ID       string       `json:"id"`
	Username string       `json:"username"`
	FullName string       `json:"full_name"`
	Email    *string      `json:"email"`
	Faculty  *facultyJSON `json:"faculty"`
}

func (a *api) me(w http.ResponseWriter, r *http.Request) {
	p := auth.PrincipalFrom(r.Context())
	u := userJSON{ID: p.User.ID, Username: p.User.Username, FullName: p.User.FullName, Email: p.User.Email}
	if p.User.FacultyID != nil {
		u.Faculty = &facultyJSON{ID: *p.User.FacultyID, Code: deref(p.User.FacultyCode), Name: deref(p.User.FacultyName)}
	}
	respond.JSON(w, http.StatusOK, map[string]any{"user": u, "roles": p.Roles})
}

func (a *api) todo(w http.ResponseWriter, r *http.Request) {
	p := auth.PrincipalFrom(r.Context())
	var (
		items any
		err   error
	)
	role := auth.TodoRole(p.Roles)
	switch role {
	case auth.TodoManager:
		var facultyID *string
		if p.Scope.Faculty {
			facultyID = p.User.FacultyID
		}
		items, err = a.store.ManagerTodo(r.Context(), p.Scope.All, facultyID)
	case auth.TodoLecturer:
		items, err = a.store.LecturerTodo(r.Context(), p.User.ID)
	case auth.TodoStudent:
		items, err = a.store.StudentTodo(r.Context(), p.User.ID)
	default:
		role, items = "none", []struct{}{}
	}
	if err != nil {
		a.fail(w, r, "load todo failed", err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{"role": role, "items": items})
}

func (a *api) listOfferings(w http.ResponseWriter, r *http.Request) {
	p := auth.PrincipalFrom(r.Context())
	items, err := a.store.ListOfferings(r.Context(), p)
	if err != nil {
		a.fail(w, r, "list offerings failed", err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *api) getOffering(w http.ResponseWriter, r *http.Request) {
	p := auth.PrincipalFrom(r.Context())
	id := chi.URLParam(r, "id")
	if !uuidRE.MatchString(id) {
		respond.NotFound(w)
		return
	}
	facts, err := a.store.OfferingFacts(r.Context(), id, p.User)
	if errors.Is(err, store.ErrNotFound) {
		respond.NotFound(w)
		return
	}
	if err != nil {
		a.fail(w, r, "offering facts failed", err)
		return
	}
	// Out-of-scope offerings are reported as 404 so their existence is not revealed.
	if !p.Scope.CanView(facts) {
		respond.NotFound(w)
		return
	}
	d, err := a.store.OfferingDetail(r.Context(), id, p.User.ID, p.Scope.Privileged(facts))
	if errors.Is(err, store.ErrNotFound) {
		respond.NotFound(w)
		return
	}
	if err != nil {
		a.fail(w, r, "offering detail failed", err)
		return
	}
	respond.JSON(w, http.StatusOK, d)
}

type tutorRequest struct {
	TaskID   string `json:"task_id"`
	Question string `json:"question"`
}

func (a *api) tutor(w http.ResponseWriter, r *http.Request) {
	p := auth.PrincipalFrom(r.Context())
	if !auth.HasRole(p.Roles, auth.RoleStudent) {
		respond.Forbidden(w, "forbidden", "only students can use the AI tutor")
		return
	}

	var in tutorRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			respond.Error(w, http.StatusRequestEntityTooLarge, "payload_too_large", "request body too large")
			return
		}
		respond.BadRequest(w, "request body must be JSON {task_id, question}")
		return
	}
	if !uuidRE.MatchString(in.TaskID) {
		respond.BadRequest(w, "task_id must be a UUID")
		return
	}
	question := strings.TrimSpace(in.Question)
	n := utf8.RuneCountInString(question)
	if !utf8.ValidString(question) || n < 1 || n > MaxQuestionChars {
		respond.BadRequest(w, "question must be 1..2000 characters")
		return
	}

	target, err := a.store.TutorTarget(r.Context(), p.User.ID, in.TaskID)
	if errors.Is(err, store.ErrNotFound) {
		respond.NotFound(w)
		return
	}
	if err != nil {
		a.fail(w, r, "load tutor target failed", err)
		return
	}
	switch target.AIPolicy {
	case "explain_only", "feedback", "full":
	default: // "none" (or anything unexpected): blocked
		respond.Forbidden(w, "ai_not_allowed", "AI is not allowed for this task")
		return
	}

	resp, err := a.ai.Tutor(r.Context(), aiclient.TutorRequest{
		CourseName:       target.CourseName,
		TaskTitle:        target.TaskTitle,
		TaskInstructions: target.Instructions,
		Question:         question,
		Policy:           target.AIPolicy,
	})
	if err != nil {
		a.logger.WarnContext(r.Context(), "ai-service call failed", "request_id", audit.RequestID(r.Context()), "err", err)
		respond.Error(w, http.StatusBadGateway, "ai_unavailable", "AI service is unavailable")
		return
	}

	if err := a.store.RecordTutor(r.Context(), store.TutorRecord{
		UserID:        p.User.ID,
		Username:      p.User.Username,
		TaskID:        target.TaskID,
		Policy:        target.AIPolicy,
		Provider:      resp.Provider,
		Model:         resp.Model,
		PromptChars:   n,
		ResponseChars: utf8.RuneCountInString(resp.Answer),
		Meta:          audit.Meta{RequestID: audit.RequestID(r.Context()), IP: audit.ClientIP(r)},
	}); err != nil {
		a.fail(w, r, "record tutor interaction failed", err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"answer": resp.Answer, "provider": resp.Provider, "model": resp.Model})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
