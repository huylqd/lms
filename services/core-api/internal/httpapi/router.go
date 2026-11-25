// Package httpapi wires the REST API under /api/v1.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/huylqd/lms/services/core-api/internal/aiclient"
	"github.com/huylqd/lms/services/core-api/internal/auth"
	"github.com/huylqd/lms/services/core-api/internal/respond"
	"github.com/huylqd/lms/services/core-api/internal/store"
)

// Store is the data access the handlers need (implemented by *store.Store).
type Store interface {
	Ping(ctx context.Context) error
	ListOfferings(ctx context.Context, p *auth.Principal) ([]store.Offering, error)
	OfferingFacts(ctx context.Context, offeringID string, user auth.User) (auth.OfferingFacts, error)
	OfferingDetail(ctx context.Context, offeringID, userID string, privileged bool) (store.OfferingDetail, error)
	StudentTodo(ctx context.Context, studentID string) ([]store.StudentTodoItem, error)
	LecturerTodo(ctx context.Context, userID string) ([]store.LecturerTodoItem, error)
	ManagerTodo(ctx context.Context, all bool, facultyID *string) ([]store.ManagerTodoItem, error)
	TutorTarget(ctx context.Context, studentID, taskID string) (store.TutorTarget, error)
	RecordTutor(ctx context.Context, rec store.TutorRecord) error
}

// Tutor calls the AI tutor (implemented by *aiclient.Client).
type Tutor interface {
	Tutor(ctx context.Context, in aiclient.TutorRequest) (aiclient.TutorResponse, error)
}

// Deps are the router dependencies.
type Deps struct {
	Store  Store
	AI     Tutor
	Auth   func(http.Handler) http.Handler // usually auth.Middleware(...)
	Logger *slog.Logger
}

type api struct {
	store  Store
	ai     Tutor
	logger *slog.Logger
}

// NewRouter builds the HTTP handler.
func NewRouter(d Deps) http.Handler {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	a := &api{store: d.Store, ai: d.AI, logger: d.Logger}

	r := chi.NewRouter()
	r.Use(requestID, accessLog(d.Logger), recoverer(d.Logger), maxBody)
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) { respond.NotFound(w) })
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) { respond.MethodNotAllowed(w) })

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", a.healthz)
		r.Get("/readyz", a.readyz)

		r.Group(func(r chi.Router) {
			r.Use(d.Auth)
			r.Get("/me", a.me)
			r.Get("/me/todo", a.todo)
			r.Get("/offerings", a.listOfferings)
			r.Get("/offerings/{id}", a.getOffering)
			r.Post("/ai/tutor", a.tutor)
		})
	})
	return r
}
