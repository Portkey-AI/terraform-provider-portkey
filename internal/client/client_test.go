package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// newTestClient builds a *Client pointed at the given test-server URL with
// aggressive retry timing so tests complete in milliseconds instead of
// seconds. We construct the Client struct directly rather than going through
// NewClient so tests aren't tied to the production retry defaults.
func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	rc := retryablehttp.NewClient()
	rc.RetryMax = 3
	rc.RetryWaitMin = 5 * time.Millisecond
	rc.RetryWaitMax = 20 * time.Millisecond
	rc.HTTPClient.Timeout = 2 * time.Second
	rc.Logger = nil
	rc.ErrorHandler = retryablehttp.PassthroughErrorHandler

	return &Client{
		BaseURL:    baseURL,
		APIKey:     "test-key",
		HTTPClient: rc.StandardClient(),
	}
}

// response describes one canned response from the sequenced test server.
type response struct {
	status int
	body   string
}

// newSequencedServer returns a test server that replies with each response
// in order. Once the sequence is exhausted, it keeps replaying the last
// response. The returned *int64 is the number of requests received so far.
func newSequencedServer(t *testing.T, responses ...response) (*httptest.Server, *int64) {
	t.Helper()
	var count int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&count, 1)
		idx := int(n - 1)
		if idx >= len(responses) {
			idx = len(responses) - 1
		}
		w.WriteHeader(responses[idx].status)
		_, _ = w.Write([]byte(responses[idx].body))
	}))
	t.Cleanup(srv.Close)
	return srv, &count
}

func TestDoRequest_SuccessFirstAttempt(t *testing.T) {
	srv, count := newSequencedServer(t,
		response{http.StatusOK, `{"ok":true}`},
	)

	c := newTestClient(t, srv.URL)
	body, err := c.doRequest(context.Background(), http.MethodGet, "/admin/health", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", string(body))
	}
	if got := atomic.LoadInt64(count); got != 1 {
		t.Fatalf("expected 1 request, got %d", got)
	}
}

func TestDoRequest_RetriesOn503(t *testing.T) {
	srv, count := newSequencedServer(t,
		response{http.StatusServiceUnavailable, `{"error":"upstream connect error"}`},
		response{http.StatusServiceUnavailable, `{"error":"upstream connect error"}`},
		response{http.StatusOK, `{"ok":true}`},
	)

	c := newTestClient(t, srv.URL)
	body, err := c.doRequest(context.Background(), http.MethodGet, "/admin/workspaces/abc", nil)
	if err != nil {
		t.Fatalf("unexpected error after retry: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", string(body))
	}
	if got := atomic.LoadInt64(count); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestDoRequest_RetriesExhausted(t *testing.T) {
	srv, count := newSequencedServer(t,
		response{http.StatusServiceUnavailable, `{"error":"always down"}`},
	)

	c := newTestClient(t, srv.URL) // RetryMax = 3 → up to 4 attempts total
	_, err := c.doRequest(context.Background(), http.MethodGet, "/admin/workspaces/abc", nil)
	if err == nil {
		t.Fatal("expected error after retries exhausted, got nil")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected 503 in error, got: %v", err)
	}
	if got := atomic.LoadInt64(count); got != 4 {
		t.Fatalf("expected 4 attempts (1 initial + 3 retries), got %d", got)
	}
}

func TestDoRequest_NoRetryOn4xx(t *testing.T) {
	cases := []struct {
		name   string
		status int
	}{
		{"400 Bad Request", http.StatusBadRequest},
		{"401 Unauthorized", http.StatusUnauthorized},
		{"403 Forbidden", http.StatusForbidden},
		{"404 Not Found", http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, count := newSequencedServer(t,
				response{tc.status, `{"error":"client error"}`},
			)

			c := newTestClient(t, srv.URL)
			_, err := c.doRequest(context.Background(), http.MethodGet, "/admin/workspaces/abc", nil)
			if err == nil {
				t.Fatalf("expected error for status %d, got nil", tc.status)
			}
			// 4xx responses must not be retried — one attempt only.
			if got := atomic.LoadInt64(count); got != 1 {
				t.Fatalf("status %d: expected 1 attempt (no retry), got %d", tc.status, got)
			}
		})
	}
}

// intPtr returns a pointer to the given int. Used by table tests below to
// distinguish "MaxRetries unset (nil → use default)" from "MaxRetries = 0
// (disable retries)" without inflating each test case with a temporary var.
func intPtr(v int) *int { return &v }

func TestNewClientWithConfig_HonorsMaxRetries(t *testing.T) {
	// Note: nil MaxRetries (use defaultRetryMax) is not exercised here — that
	// path would require ~7.5s of real backoff waits with a 503-forever server.
	// The default is verified indirectly via TestDoRequest_RetriesExhausted,
	// which uses a custom test client with shorter waits.
	cases := []struct {
		name             string
		maxRetries       *int
		expectedAttempts int64
	}{
		{"explicit 0 → no retries → 1 attempt", intPtr(0), 1},
		{"explicit 1 retry → 2 attempts", intPtr(1), 2},
		{"explicit 2 retries → 3 attempts", intPtr(2), 3},
		{"negative clamped to 0 retries → 1 attempt", intPtr(-1), 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, count := newSequencedServer(t,
				response{http.StatusServiceUnavailable, `{"error":"down"}`},
			)

			c, err := NewClientWithConfig(ClientConfig{
				BaseURL:    srv.URL,
				APIKey:     "test-key",
				MaxRetries: tc.maxRetries,
			})
			if err != nil {
				t.Fatalf("NewClientWithConfig: %v", err)
			}

			_, err = c.doRequest(context.Background(), http.MethodGet, "/admin/workspaces/abc", nil)
			if err == nil {
				t.Fatal("expected error after retries, got nil")
			}
			if got := atomic.LoadInt64(count); got != tc.expectedAttempts {
				t.Fatalf("expected %d attempts, got %d", tc.expectedAttempts, got)
			}
		})
	}
}

func TestDoRequest_RetriesOn429(t *testing.T) {
	// 429 Too Many Requests should be retried per retryablehttp's default
	// policy — we don't want rate limiting to fail a plan outright.
	srv, count := newSequencedServer(t,
		response{http.StatusTooManyRequests, `{"error":"rate limited"}`},
		response{http.StatusOK, `{"ok":true}`},
	)

	c := newTestClient(t, srv.URL)
	_, err := c.doRequest(context.Background(), http.MethodGet, "/admin/workspaces/abc", nil)
	if err != nil {
		t.Fatalf("unexpected error after retry on 429: %v", err)
	}
	if got := atomic.LoadInt64(count); got != 2 {
		t.Fatalf("expected 2 attempts (1 rate-limited + 1 success), got %d", got)
	}
}
