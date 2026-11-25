// Package aiclient calls the internal ai-service.
package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TutorRequest is the body of POST /v1/tutor.
type TutorRequest struct {
	CourseName       string `json:"course_name"`
	TaskTitle        string `json:"task_title"`
	TaskInstructions string `json:"task_instructions"`
	Question         string `json:"question"`
	Policy           string `json:"policy"` // explain_only | feedback | full
}

// TutorResponse is the ai-service reply.
type TutorResponse struct {
	Answer   string `json:"answer"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// Client talks to ai-service with the shared internal token.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New returns a client with a 20s timeout.
func New(baseURL, token string) *Client {
	return &Client{baseURL: baseURL, token: token, http: &http.Client{Timeout: 20 * time.Second}}
}

const maxResponseBytes = 1 << 20

// Tutor calls POST {base}/v1/tutor.
func (c *Client) Tutor(ctx context.Context, in TutorRequest) (TutorResponse, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return TutorResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/tutor", bytes.NewReader(body))
	if err != nil {
		return TutorResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Internal-Token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return TutorResponse{}, fmt.Errorf("ai-service request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		return TutorResponse{}, fmt.Errorf("ai-service returned status %d", resp.StatusCode)
	}
	var out TutorResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&out); err != nil {
		return TutorResponse{}, fmt.Errorf("decode ai-service response: %w", err)
	}
	if out.Answer == "" || out.Provider == "" || out.Model == "" {
		return TutorResponse{}, errors.New("ai-service response missing answer/provider/model")
	}
	return out, nil
}
