package template

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Identity struct {
	SessionID  string   `json:"session_id"`
	UserID     string   `json:"user_id"`
	IdentityID string   `json:"identity_id"`
	Login      string   `json:"login"`
	Groups     []string `json:"groups"`
}

type Resolver interface {
	Resolve(ctx context.Context, sessionID string) (*Identity, error)
}

type ErrorKind string

const (
	ErrKindUnauthorized ErrorKind = "unauthorized"
	ErrKindNotFound     ErrorKind = "not_found"
	ErrKindBadRequest   ErrorKind = "bad_request"
	ErrKindUpstream     ErrorKind = "upstream"
)

type ResolveError struct {
	Kind       ErrorKind
	StatusCode int
	Message    string
}

func (e *ResolveError) Error() string {
	return fmt.Sprintf("resolve identity failed: kind=%s status=%d message=%s", e.Kind, e.StatusCode, e.Message)
}

type resolveRequest struct {
	SessionID string `json:"session_id"`
}

type resolveResponse struct {
	Success  bool     `json:"success"`
	Identity Identity `json:"identity"`
}

type IDMClient struct {
	baseURL       string
	internalToken string
	httpClient    *http.Client
	maxRetries    int
	retryBackoff  time.Duration
}

func NewIDMClient(cfg Config) *IDMClient {
	cfg.Normalize()

	return &IDMClient{
		baseURL:       strings.TrimRight(cfg.BaseURL, "/"),
		internalToken: cfg.InternalToken,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		maxRetries:   cfg.MaxRetries,
		retryBackoff: cfg.RetryBackoff,
	}
}

func (c *IDMClient) Resolve(ctx context.Context, sessionID string) (*Identity, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, &ResolveError{
			Kind:    ErrKindBadRequest,
			Message: "session id is required",
		}
	}

	endpoint := c.baseURL + "/v1/internal/identity/resolve"

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		id, err := c.resolveOnce(ctx, endpoint, sessionID)
		if err == nil {
			return id, nil
		}
		lastErr = err

		if !isRetryable(err) || attempt == c.maxRetries {
			break
		}

		backoff := c.retryBackoff * time.Duration(attempt+1)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, lastErr
}

func (c *IDMClient) resolveOnce(ctx context.Context, endpoint, sessionID string) (*Identity, error) {
	reqBody, err := json.Marshal(resolveRequest{SessionID: sessionID})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.internalToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-ID", sessionID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var payload resolveResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, &ResolveError{
				Kind:       ErrKindUpstream,
				StatusCode: resp.StatusCode,
				Message:    "invalid response body",
			}
		}
		if !payload.Success {
			return nil, &ResolveError{
				Kind:       ErrKindUpstream,
				StatusCode: resp.StatusCode,
				Message:    "unsuccessful upstream payload",
			}
		}
		return &payload.Identity, nil

	case http.StatusUnauthorized:
		return nil, &ResolveError{
			Kind:       ErrKindUnauthorized,
			StatusCode: resp.StatusCode,
			Message:    "idm rejected token or session",
		}
	case http.StatusNotFound:
		return nil, &ResolveError{
			Kind:       ErrKindNotFound,
			StatusCode: resp.StatusCode,
			Message:    "session was not found",
		}
	case http.StatusBadRequest:
		return nil, &ResolveError{
			Kind:       ErrKindBadRequest,
			StatusCode: resp.StatusCode,
			Message:    "request was rejected by idm",
		}
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, &ResolveError{
			Kind:       ErrKindUpstream,
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}
}

func isRetryable(err error) bool {
	var resolveErr *ResolveError
	if errors.As(err, &resolveErr) {
		if resolveErr.Kind == ErrKindUnauthorized || resolveErr.Kind == ErrKindNotFound || resolveErr.Kind == ErrKindBadRequest {
			return false
		}
		// Retry only 5xx-like errors from upstream.
		return resolveErr.StatusCode >= 500
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		var netErr net.Error
		if errors.As(urlErr, &netErr) {
			return netErr.Timeout() || netErr.Temporary()
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	return false
}
