package aiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTutor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tutor" || r.Method != http.MethodPost {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		if r.Header.Get("X-Internal-Token") != "tok" {
			http.Error(w, "forbidden", http.StatusUnauthorized)
			return
		}
		var in TutorRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Policy != "explain_only" || in.Question != "q?" {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(TutorResponse{Answer: "a", Provider: "mock", Model: "m1"})
	}))
	defer srv.Close()

	out, err := New(srv.URL, "tok").Tutor(context.Background(), TutorRequest{Question: "q?", Policy: "explain_only"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Answer != "a" || out.Provider != "mock" || out.Model != "m1" {
		t.Fatalf("got %+v", out)
	}

	if _, err := New(srv.URL, "wrong").Tutor(context.Background(), TutorRequest{Question: "q?", Policy: "explain_only"}); err == nil {
		t.Fatal("expected error on non-200")
	}
}
