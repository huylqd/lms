// Package respond writes JSON responses and the uniform error envelope
// {"error":{"code":"...","message":"..."}}.
package respond

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorBody is the error envelope payload.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// JSON writes v with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("write response failed", "err", err)
	}
}

// Error writes the error envelope.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorEnvelope{Error: ErrorBody{Code: code, Message: message}})
}

// Common error helpers.
func Unauthorized(w http.ResponseWriter, code, message string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="lms-api"`)
	Error(w, http.StatusUnauthorized, code, message)
}

func Forbidden(w http.ResponseWriter, code, message string) {
	Error(w, http.StatusForbidden, code, message)
}

func NotFound(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, "not_found", "resource not found")
}

func MethodNotAllowed(w http.ResponseWriter) {
	Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}

func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, "invalid_request", message)
}

func Internal(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "internal", "internal server error")
}
