package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/huylqd/lms/services/core-api/internal/aiclient"
	"github.com/huylqd/lms/services/core-api/internal/auth"
	"github.com/huylqd/lms/services/core-api/internal/respond"
	"github.com/huylqd/lms/services/core-api/internal/store"
)

const (
	taskID     = "11111111-1111-1111-1111-111111111111"
	offeringID = "22222222-2222-2222-2222-222222222222"
)

type fakeStore struct {
	pingErr    error
	facts      auth.OfferingFacts
	factsErr   error
	target     store.TutorTarget
	targetErr  error
	recorded   []store.TutorRecord
	todoCalled string
	managerAll bool
}

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }
func (f *fakeStore) ListOfferings(context.Context, *auth.Principal) ([]store.Offering, error) {
	return []store.Offering{}, nil
}
func (f *fakeStore) OfferingFacts(context.Context, string, auth.User) (auth.OfferingFacts, error) {
	return f.facts, f.factsErr
}
func (f *fakeStore) OfferingDetail(_ context.Context, id, _ string, _ bool) (store.OfferingDetail, error) {
	return sampleDetail(id), nil
}
func (f *fakeStore) StudentTodo(context.Context, string) ([]store.StudentTodoItem, error) {
	f.todoCalled = "student"
	return []store.StudentTodoItem{}, nil
}
func (f *fakeStore) LecturerTodo(context.Context, string) ([]store.LecturerTodoItem, error) {
	f.todoCalled = "lecturer"
	return []store.LecturerTodoItem{}, nil
}
func (f *fakeStore) ManagerTodo(_ context.Context, all bool, _ *string) ([]store.ManagerTodoItem, error) {
	f.todoCalled, f.managerAll = "manager", all
	return []store.ManagerTodoItem{}, nil
}
func (f *fakeStore) TutorTarget(context.Context, string, string) (store.TutorTarget, error) {
	return f.target, f.targetErr
}
func (f *fakeStore) RecordTutor(_ context.Context, rec store.TutorRecord) error {
	f.recorded = append(f.recorded, rec)
	return nil
}

type fakeAI struct {
	err error
	got aiclient.TutorRequest
}

func (f *fakeAI) Tutor(_ context.Context, in aiclient.TutorRequest) (aiclient.TutorResponse, error) {
	f.got = in
	if f.err != nil {
		return aiclient.TutorResponse{}, f.err
	}
	return aiclient.TutorResponse{Answer: "Giải thích", Provider: "mock", Model: "mock-1"}, nil
}

// fakeAuth authenticates any request carrying "Authorization: Test <roles,comma>".
func fakeAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Test ") {
			respond.Unauthorized(w, "unauthorized", "missing bearer token")
			return
		}
		roles := strings.Split(strings.TrimPrefix(h, "Test "), ",")
		p := &auth.Principal{User: auth.User{ID: "u1", Username: "sv01", FullName: "SV"}, Roles: roles, Scope: auth.ScopeFor(roles)}
		next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
	})
}

func newTestRouter(s *fakeStore, ai *fakeAI) http.Handler {
	return NewRouter(Deps{Store: s, AI: ai, Auth: fakeAuth, Logger: slog.New(slog.NewJSONHandler(io.Discard, nil))})
}

func do(t *testing.T, h http.Handler, method, path, roles, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if roles != "" {
		req.Header.Set("Authorization", "Test "+roles)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("not JSON: %q", rec.Body.String())
	}
	return body.Error.Code
}

func TestPublicAndErrorRoutes(t *testing.T) {
	h := newTestRouter(&fakeStore{}, &fakeAI{})

	if rec := do(t, h, http.MethodGet, "/api/v1/healthz", "", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("healthz %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, h, http.MethodGet, "/api/v1/readyz", "", ""); rec.Code != 200 {
		t.Fatalf("readyz %d", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/api/v1/healthz", "", ""); rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID")
	}

	cases := []struct {
		method, path string
		status       int
		code         string
	}{
		{http.MethodGet, "/nope", 404, "not_found"},
		{http.MethodGet, "/api/v1/nope", 404, "not_found"},
		{http.MethodPost, "/api/v1/healthz", 405, "method_not_allowed"},
		{http.MethodGet, "/api/v1/me", 401, "unauthorized"},
	}
	for _, c := range cases {
		rec := do(t, h, c.method, c.path, "", "")
		if rec.Code != c.status || errCode(t, rec) != c.code {
			t.Errorf("%s %s = %d %s", c.method, c.path, rec.Code, rec.Body.String())
		}
	}

	h = newTestRouter(&fakeStore{pingErr: errors.New("down")}, &fakeAI{})
	if rec := do(t, h, http.MethodGet, "/api/v1/readyz", "", ""); rec.Code != 503 || errCode(t, rec) != "not_ready" {
		t.Fatalf("readyz down %d", rec.Code)
	}
}

func TestTodoRoleDispatch(t *testing.T) {
	for roles, want := range map[string]string{
		"student":                  "student",
		"student,advisor":          "lecturer",
		"lecturer,faculty_manager": "manager",
	} {
		s := &fakeStore{}
		rec := do(t, newTestRouter(s, &fakeAI{}), http.MethodGet, "/api/v1/me/todo", roles, "")
		if rec.Code != 200 || s.todoCalled != want || !strings.Contains(rec.Body.String(), `"role":"`+want+`"`) {
			t.Errorf("roles %s: %d called=%s body=%s", roles, rec.Code, s.todoCalled, rec.Body.String())
		}
	}
}

func TestOfferingOutOfScopeIs404(t *testing.T) {
	s := &fakeStore{facts: auth.OfferingFacts{OfferingFacultyID: "f1"}}
	h := newTestRouter(s, &fakeAI{})
	if rec := do(t, h, http.MethodGet, "/api/v1/offerings/"+offeringID, "student", ""); rec.Code != 404 {
		t.Fatalf("out of scope = %d", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/api/v1/offerings/not-a-uuid", "admin", ""); rec.Code != 404 {
		t.Fatalf("bad id = %d", rec.Code)
	}
	s.facts.IsEnrolled = true
	if rec := do(t, h, http.MethodGet, "/api/v1/offerings/"+offeringID, "student", ""); rec.Code != 200 {
		t.Fatalf("enrolled = %d", rec.Code)
	}
}

func TestTutor(t *testing.T) {
	okTarget := store.TutorTarget{TaskID: taskID, CourseName: "Marketing số", TaskTitle: "T", AIPolicy: "explain_only"}
	body := `{"task_id":"` + taskID + `","question":"Chân dung khách hàng là gì?"}`

	t.Run("success records interaction", func(t *testing.T) {
		s, ai := &fakeStore{target: okTarget}, &fakeAI{}
		rec := do(t, newTestRouter(s, ai), http.MethodPost, "/api/v1/ai/tutor", "student", body)
		if rec.Code != 200 {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
		var out map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		if out["answer"] != "Giải thích" || out["provider"] != "mock" || out["model"] != "mock-1" {
			t.Fatalf("body %v", out)
		}
		if ai.got.Policy != "explain_only" || ai.got.CourseName != "Marketing số" {
			t.Fatalf("ai request %+v", ai.got)
		}
		if len(s.recorded) != 1 || s.recorded[0].PromptChars != utf8.RuneCountInString("Chân dung khách hàng là gì?") || s.recorded[0].Provider != "mock" {
			t.Fatalf("recorded %+v", s.recorded)
		}
	})

	cases := []struct {
		name   string
		roles  string
		body   string
		store  *fakeStore
		ai     *fakeAI
		status int
		code   string
	}{
		{"not a student", "lecturer", body, &fakeStore{target: okTarget}, &fakeAI{}, 403, "forbidden"},
		{"bad json", "student", `{`, &fakeStore{target: okTarget}, &fakeAI{}, 400, "invalid_request"},
		{"bad task id", "student", `{"task_id":"x","question":"q"}`, &fakeStore{target: okTarget}, &fakeAI{}, 400, "invalid_request"},
		{"empty question", "student", `{"task_id":"` + taskID + `","question":"   "}`, &fakeStore{target: okTarget}, &fakeAI{}, 400, "invalid_request"},
		{"too long", "student", `{"task_id":"` + taskID + `","question":"` + strings.Repeat("ă", 2001) + `"}`, &fakeStore{target: okTarget}, &fakeAI{}, 400, "invalid_request"},
		{"not enrolled", "student", body, &fakeStore{targetErr: store.ErrNotFound}, &fakeAI{}, 404, "not_found"},
		{"policy none", "student", body, &fakeStore{target: store.TutorTarget{TaskID: taskID, AIPolicy: "none"}}, &fakeAI{}, 403, "ai_not_allowed"},
		{"ai down", "student", body, &fakeStore{target: okTarget}, &fakeAI{err: errors.New("boom")}, 502, "ai_unavailable"},
		{"too large", "student", `{"task_id":"` + taskID + `","question":"` + strings.Repeat("a", MaxBodyBytes) + `"}`, &fakeStore{target: okTarget}, &fakeAI{}, 413, "payload_too_large"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := do(t, newTestRouter(c.store, c.ai), http.MethodPost, "/api/v1/ai/tutor", c.roles, c.body)
			if rec.Code != c.status || errCode(t, rec) != c.code {
				t.Fatalf("got %d %s", rec.Code, rec.Body.String())
			}
			if len(c.store.recorded) != 0 {
				t.Fatal("must not record on failure")
			}
		})
	}
}

func sampleDetail(id string) store.OfferingDetail {
	url := "https://media.example.test/v.mp4"
	return store.OfferingDetail{
		Offering: store.Offering{ID: id, Code: "MKT201-01", Title: "Marketing số – Nhóm 01", Status: "open",
			Course: store.Course{ID: "c", Code: "MKT201", Name: "Marketing số", Credits: 3},
			Term:   store.Term{ID: "t", Code: "2026-HK1", Name: "HK1", StartsOn: "2026-08-17", EndsOn: "2027-01-15"}},
		Staff:     []store.StaffMember{{UserID: "u", FullName: "Trần Minh Tuấn", Role: "lecturer"}},
		Tasks:     []store.Task{{ID: taskID, Kind: "assignment", Title: "T", Instructions: "I", AIPolicy: "explain_only", AssignmentScope: "all", Status: "published"}},
		Materials: []store.Material{{ID: "m1", Title: "Doc", Kind: "document"}, {ID: "m2", Title: "Vid", Kind: "video", URL: &url}},
	}
}

// The web app is built against the exact shapes in docs/ARCHITECTURE.md.
func TestOfferingDetailContractShape(t *testing.T) {
	s := &fakeStore{facts: auth.OfferingFacts{IsEnrolled: true}}
	rec := do(t, newTestRouter(s, &fakeAI{}), http.MethodGet, "/api/v1/offerings/"+offeringID, "student", "")
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	want := `{"id":"` + offeringID + `","code":"MKT201-01","title":"Marketing số – Nhóm 01","status":"open",` +
		`"course":{"code":"MKT201","name":"Marketing số","credits":3},"term":{"code":"2026-HK1","name":"HK1"},` +
		`"staff":[{"full_name":"Trần Minh Tuấn","role":"lecturer"}],` +
		`"tasks":[{"id":"` + taskID + `","kind":"assignment","title":"T","instructions":"I","ai_policy":"explain_only","due_at":null}],` +
		`"materials":[{"id":"m1","title":"Doc","kind":"document","url":null},{"id":"m2","title":"Vid","kind":"video","url":"https://media.example.test/v.mp4"}]}`
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Fatalf("shape mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestOfferingListContractShape(t *testing.T) {
	b, err := json.Marshal(map[string]any{"items": []store.Offering{sampleDetail(offeringID).Offering}})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"items":[{"id":"` + offeringID + `","code":"MKT201-01","title":"Marketing số – Nhóm 01","status":"open",` +
		`"course":{"code":"MKT201","name":"Marketing số","credits":3},"term":{"code":"2026-HK1","name":"HK1"}}]}`
	if string(b) != want {
		t.Fatalf("got %s", b)
	}
	rec := do(t, newTestRouter(&fakeStore{}, &fakeAI{}), http.MethodGet, "/api/v1/offerings", "student", "")
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"items":[]}` {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
