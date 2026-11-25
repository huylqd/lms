package respond

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, http.StatusNotFound, "not_found", "nope")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type %q", ct)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"]["code"] != "not_found" || body["error"]["message"] != "nope" {
		t.Fatalf("body %s", rec.Body.String())
	}
	if len(body) != 1 || len(body["error"]) != 2 {
		t.Fatalf("unexpected extra fields: %s", rec.Body.String())
	}
}

func TestUnauthorizedSetsChallenge(t *testing.T) {
	rec := httptest.NewRecorder()
	Unauthorized(rec, "unauthorized", "missing token")
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("got %d %v", rec.Code, rec.Header())
	}
}
