package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

func TestPayAsYouGoPricingCacheTokenJSON(t *testing.T) {
	readPrice := 0.01
	writePrice := 0.02
	payload := ModelPricingConfig{
		Type: "static",
		PayAsYouGo: &PayAsYouGoPricing{
			CacheReadInputToken:  &TokenPrice{Price: readPrice},
			CacheWriteInputToken: &TokenPrice{Price: writePrice},
		},
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal pricing: %v", err)
	}

	const expected = `{"type":"static","pay_as_you_go":{"cache_read_input_token":{"price":0.01},"cache_write_input_token":{"price":0.02}}}`
	if string(encoded) != expected {
		t.Fatalf("unexpected pricing JSON: %s", encoded)
	}

	var decoded ModelPricingConfig
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("failed to unmarshal pricing: %v", err)
	}
	if decoded.PayAsYouGo.CacheReadInputToken.Price != readPrice || decoded.PayAsYouGo.CacheWriteInputToken.Price != writePrice {
		t.Fatalf("cache token prices did not round-trip: %+v", decoded.PayAsYouGo)
	}
}

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

func TestDoRequest_ReturnsAPIErrorOn4xx(t *testing.T) {
	// Non-retryable 4xx responses should return a *APIError carrying the
	// status code and raw body so callers can apply per-status handling
	// (e.g. treating 403 on Read as missing-resource for state
	// reconciliation). The error message must preserve the legacy string
	// format so existing callers that match against
	// "API request failed with status N" keep working.
	srv, _ := newSequencedServer(t,
		response{http.StatusNotFound, `{"errorCode":"AB08","message":"Resource not found"}`},
	)

	c := newTestClient(t, srv.URL)
	_, err := c.doRequest(context.Background(), http.MethodGet, "/admin/workspaces/missing", nil)
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected StatusCode %d, got %d", http.StatusNotFound, apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Body, "AB08") {
		t.Errorf("expected Body to contain raw API response, got %q", apiErr.Body)
	}
	if !strings.Contains(err.Error(), "API request failed with status 404") {
		t.Errorf("expected legacy error string, got %q", err.Error())
	}
}

func TestIsNotFound(t *testing.T) {
	// IsNotFound encapsulates Portkey's quirk: out-of-band-deleted
	// resources return 403 (errorCode AB03) for some endpoints and 404
	// for others. Both should be treated as missing-resource by Read
	// implementations so Terraform can reconcile state.
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error is not not-found", nil, false},
		{"plain error is not not-found", errors.New("network down"), false},
		{"404 is not-found", &APIError{StatusCode: http.StatusNotFound, Body: ""}, true},
		{"403 is not-found", &APIError{StatusCode: http.StatusForbidden, Body: `{"errorCode":"AB03"}`}, true},
		{"500 is not not-found", &APIError{StatusCode: http.StatusInternalServerError, Body: ""}, false},
		{"400 is not not-found", &APIError{StatusCode: http.StatusBadRequest, Body: `{"errorCode":"AB01"}`}, false},
		{"wrapped 404 unwraps via errors.As", fmt.Errorf("read failed: %w", &APIError{StatusCode: http.StatusNotFound}), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestDeleteIntegrationModels_QueryParamShape verifies the request matches
// the Portkey Admin API contract: DELETE /integrations/{slug}/models with
// the model slugs passed as a comma-separated `slugs` query parameter,
// and NO JSON body. See
// https://docs.portkey.ai/docs/api-reference/admin-api/control-plane/integrations/models/delete-custom-model
//
// The previous body-based shape triggered `400 AB01 Validation failed:
// Invalid value` on the `slugs` query param, and cascaded into TFC apply
// wedges every time a custom model was removed from state.
func TestDeleteIntegrationModels_QueryParamShape(t *testing.T) {
	cases := []struct {
		name        string
		integration string
		slugs       []string
		wantSlugs   string // decoded slugs query param
		wantPath    string
	}{
		{
			name:        "single bare-id AIP slug",
			integration: "bedrock-example",
			slugs:       []string{"aip1234567890"},
			wantSlugs:   "aip1234567890",
			wantPath:    "/integrations/bedrock-example/models",
		},
		{
			name:        "single ARN slug (colons and slashes URL-escaped)",
			integration: "bedrock-example",
			slugs:       []string{"arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/aip1234567890"},
			wantSlugs:   "arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/aip1234567890",
			wantPath:    "/integrations/bedrock-example/models",
		},
		{
			name:        "multiple bare-id slugs joined with comma",
			integration: "bedrock-example",
			slugs:       []string{"aip1234567890", "another-slug"},
			wantSlugs:   "aip1234567890,another-slug",
			wantPath:    "/integrations/bedrock-example/models",
		},
		{
			// Anthropic-on-Vertex publisher-model shape.
			// Per RFC 3986 `@` is allowed in the query component (it's in
			// pchar), so escaping is not strictly required. Go's
			// url.QueryEscape encodes it to %40 anyway; net/url and the
			// server-side query parser both round-trip it. Empirically
			// verified against a live Portkey staging endpoint: server-side
			// echo returned the unescaped `anthropic.claude-sonnet-4-5@20250929`.
			name:        "Vertex Anthropic publisher slug with @ version",
			integration: "vertex-ai-example",
			slugs:       []string{"anthropic.claude-sonnet-4-5@20250929"},
			wantSlugs:   "anthropic.claude-sonnet-4-5@20250929",
			wantPath:    "/integrations/vertex-ai-example/models",
		},
		{
			// Vertex fine-tuned endpoint slug shape (dot + numeric).
			// All-safe characters, no escaping needed. Included so the
			// test matrix covers Vertex's three custom-model families
			// (endpoint IDs, Anthropic publishers, Gemini variants).
			name:        "Vertex endpoint ID slug",
			integration: "vertex-ai-example",
			slugs:       []string{"endpoints.5895219608809373696"},
			wantSlugs:   "endpoints.5895219608809373696",
			wantPath:    "/integrations/vertex-ai-example/models",
		},
		{
			// Vertex Gemini variant slug (bare, dot-separated, no colons).
			name:        "Vertex Gemini variant slug",
			integration: "vertex-ai-example",
			slugs:       []string{"gemini-3.1-pro-preview"},
			wantSlugs:   "gemini-3.1-pro-preview",
			wantPath:    "/integrations/vertex-ai-example/models",
		},
		{
			// Multi-slug ARN case: two AIP ARNs comma-joined + URL-escaped
			// as a single value. Server-side split-on-comma is what the
			// Portkey API contract requires. Empirically verified against
			// the live staging endpoint.
			name:        "multi-slug two ARNs comma-joined",
			integration: "bedrock-example",
			slugs: []string{
				"arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/aip1",
				"arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/aip2",
			},
			wantSlugs: "arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/aip1,arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/aip2",
			wantPath:  "/integrations/bedrock-example/models",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				gotMethod string
				gotPath   string
				gotSlugs  string
				gotBody   []byte
			)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotSlugs = r.URL.Query().Get("slugs")
				gotBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			}))
			t.Cleanup(srv.Close)

			c := newTestClient(t, srv.URL)
			err := c.DeleteIntegrationModels(context.Background(), tc.integration, tc.slugs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotMethod != http.MethodDelete {
				t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
			if gotSlugs != tc.wantSlugs {
				t.Errorf("slugs query param = %q, want %q", gotSlugs, tc.wantSlugs)
			}
			if len(gotBody) != 0 {
				t.Errorf("expected empty request body, got %q", string(gotBody))
			}
		})
	}
}

// TestUpdateUsageLimitsPolicyRequestJSON pins the tri-state encoding of the usage
// limits update body. The Portkey API treats an omitted field as "leave unchanged" and
// an explicit null as "clear", so these three shapes are not interchangeable and a
// stray omitempty would silently change the meaning of an update.
func TestUpdateUsageLimitsPolicyRequestJSON(t *testing.T) {
	threshold := 800.0
	limit := 1000.0
	conds := []PolicyCondition{
		{Key: "metadata._user", Value: json.RawMessage(`["aqua-agent-bot"]`)},
	}
	condsJSON := `"conditions":[{"key":"metadata._user","value":["aqua-agent-bot"]}]`

	tests := []struct {
		name string
		req  UpdateUsageLimitsPolicyRequest
		want string
	}{
		{
			name: "conditions always sent, alert_threshold cleared, reset fields omitted",
			req: UpdateUsageLimitsPolicyRequest{
				Conditions:  conds,
				CreditLimit: &limit,
			},
			want: `{` + condsJSON + `,"credit_limit":1000,"alert_threshold":null}`,
		},
		{
			name: "alert_threshold set",
			req: UpdateUsageLimitsPolicyRequest{
				Conditions:     conds,
				CreditLimit:    &limit,
				AlertThreshold: &threshold,
			},
			want: `{` + condsJSON + `,"credit_limit":1000,"alert_threshold":800}`,
		},
		{
			name: "periodic_reset set clears periodic_reset_days",
			req: UpdateUsageLimitsPolicyRequest{
				Conditions:        conds,
				AlertThreshold:    &threshold,
				PeriodicReset:     json.RawMessage(`"monthly"`),
				PeriodicResetDays: json.RawMessage(`null`),
			},
			want: `{` + condsJSON + `,"alert_threshold":800,"periodic_reset":"monthly","periodic_reset_days":null}`,
		},
		{
			name: "periodic_reset_days set clears periodic_reset",
			req: UpdateUsageLimitsPolicyRequest{
				Conditions:        conds,
				AlertThreshold:    &threshold,
				PeriodicReset:     json.RawMessage(`null`),
				PeriodicResetDays: json.RawMessage(`30`),
			},
			want: `{` + condsJSON + `,"alert_threshold":800,"periodic_reset":null,"periodic_reset_days":30}`,
		},
		{
			name: "name omitted when empty rather than sent as null",
			req:  UpdateUsageLimitsPolicyRequest{Conditions: conds, AlertThreshold: &threshold},
			want: `{` + condsJSON + `,"alert_threshold":800}`,
		},
		{
			// An empty array must reach the wire so the API can reject it. Dropping it
			// would leave the server's old targeting live while state recorded [].
			name: "empty conditions are sent, not dropped",
			req: UpdateUsageLimitsPolicyRequest{
				Conditions:     []PolicyCondition{},
				AlertThreshold: &threshold,
			},
			want: `{"conditions":[],"alert_threshold":800}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("unexpected error marshaling request: %v", err)
			}
			if string(encoded) != tt.want {
				t.Errorf("unexpected body\n got: %s\nwant: %s", encoded, tt.want)
			}
		})
	}
}

// TestUpdateRateLimitsPolicyRequestJSON verifies conditions reach the rate limits
// update body; before this was added the field did not exist on the struct, so a
// conditions change could never be applied in place.
func TestUpdateRateLimitsPolicyRequestJSON(t *testing.T) {
	value := 100.0
	req := UpdateRateLimitsPolicyRequest{
		Conditions: []PolicyCondition{
			{Key: "metadata._user", Value: json.RawMessage(`"aqua-agent-bot"`)},
		},
		Unit:  "rpm",
		Value: &value,
	}

	encoded, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected error marshaling request: %v", err)
	}

	want := `{"conditions":[{"key":"metadata._user","value":"aqua-agent-bot"}],"unit":"rpm","value":100}`
	if string(encoded) != want {
		t.Errorf("unexpected body\n got: %s\nwant: %s", encoded, want)
	}

	// An empty array must reach the wire so the API can reject it, rather than being
	// dropped and leaving the server's old targeting in place.
	encoded, err = json.Marshal(UpdateRateLimitsPolicyRequest{
		Conditions: []PolicyCondition{},
		Unit:       "rpm",
		Value:      &value,
	})
	if err != nil {
		t.Fatalf("unexpected error marshaling request: %v", err)
	}
	if want := `{"conditions":[],"unit":"rpm","value":100}`; string(encoded) != want {
		t.Errorf("unexpected body for empty conditions\n got: %s\nwant: %s", encoded, want)
	}
}
