package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetOrganisationDefaults_OK verifies the {id, slug} read shape is parsed.
func TestGetOrganisationDefaults_OK(t *testing.T) {
	var capturedMethod, capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		_, _ = w.Write([]byte(`{
			"object":"organisation_defaults",
			"input_guardrails":[{"id":"11111111-1111-1111-1111-111111111111","slug":"gr_input"}],
			"output_guardrails":[{"id":"22222222-2222-2222-2222-222222222222","slug":"gr_output"}]
		}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL+"/v1", "test-key")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	defaults, err := c.GetOrganisationDefaults(context.Background())
	if err != nil {
		t.Fatalf("GetOrganisationDefaults failed: %v", err)
	}

	if capturedMethod != http.MethodGet {
		t.Errorf("method = %q; want GET", capturedMethod)
	}
	if capturedPath != "/v1/admin/organisation/defaults" {
		t.Errorf("path = %q; want /v1/admin/organisation/defaults", capturedPath)
	}
	if defaults.Object != "organisation_defaults" {
		t.Errorf("object = %q; want organisation_defaults", defaults.Object)
	}
	if len(defaults.InputGuardrails) != 1 || defaults.InputGuardrails[0].Slug != "gr_input" {
		t.Errorf("unexpected input_guardrails: %+v", defaults.InputGuardrails)
	}
	if len(defaults.OutputGuardrails) != 1 || defaults.OutputGuardrails[0].ID != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("unexpected output_guardrails: %+v", defaults.OutputGuardrails)
	}
}

// TestGetOrganisationDefaults_SelfHostedBaseURL verifies that the relative path
// composition works for self-hosted base_url shapes (with path prefix, without
// trailing /v1, etc.) — coverage that the deleted organisationDefaultsURL helper
// used to provide for these variants.
func TestGetOrganisationDefaults_SelfHostedBaseURL(t *testing.T) {
	cases := []struct {
		name       string
		baseURL    func(srvURL string) string
		wantPath   string
	}{
		{
			name:     "self-hosted with path prefix",
			baseURL:  func(u string) string { return u + "/api/v1" },
			wantPath: "/api/v1/admin/organisation/defaults",
		},
		{
			name:     "default v1 base url",
			baseURL:  func(u string) string { return u + "/v1" },
			wantPath: "/v1/admin/organisation/defaults",
		},
		{
			name:     "base url without version suffix",
			baseURL:  func(u string) string { return u },
			wantPath: "/admin/organisation/defaults",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				_, _ = w.Write([]byte(`{"object":"organisation_defaults","input_guardrails":[],"output_guardrails":[]}`))
			}))
			defer srv.Close()

			c, err := NewClient(tc.baseURL(srv.URL), "test-key")
			if err != nil {
				t.Fatalf("NewClient failed: %v", err)
			}

			if _, err := c.GetOrganisationDefaults(context.Background()); err != nil {
				t.Fatalf("GetOrganisationDefaults failed: %v", err)
			}

			if capturedPath != tc.wantPath {
				t.Errorf("path = %q; want %q", capturedPath, tc.wantPath)
			}
		})
	}
}

// TestUpdateOrganisationDefaults_OmitsNilFields verifies the tri-state write
// behaviour: a nil field is omitted from the body (preserve), while a marshaled
// `[]` is sent (clear). It also verifies the PUT is followed by a re-fetch,
// since the API returns an empty object.
func TestUpdateOrganisationDefaults_OmitsNilFields(t *testing.T) {
	var methods []string
	var putBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodPut {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &putBody)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(`{"object":"organisation_defaults","input_guardrails":[],"output_guardrails":[]}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL+"/v1", "test-key")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if _, err := c.UpdateOrganisationDefaults(context.Background(), UpdateOrganisationDefaultsRequest{
		InputGuardrails: json.RawMessage("[]"),
	}); err != nil {
		t.Fatalf("UpdateOrganisationDefaults failed: %v", err)
	}

	if len(methods) != 2 || methods[0] != http.MethodPut || methods[1] != http.MethodGet {
		t.Errorf("requests = %v; want [PUT GET]", methods)
	}
	if _, ok := putBody["input_guardrails"]; !ok {
		t.Errorf("input_guardrails missing from body: %+v", putBody)
	}
	if _, ok := putBody["output_guardrails"]; ok {
		t.Errorf("output_guardrails should be omitted when nil, body: %+v", putBody)
	}
}
