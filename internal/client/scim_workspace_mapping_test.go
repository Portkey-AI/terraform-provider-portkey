package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestScimWorkspacesURL verifies that SCIM endpoint URLs swap the configured
// /v1 BaseURL suffix for /v2, which is a temporary workaround documented in
// the client while Portkey fixes the /v1 routing for SCIM endpoints.
func TestScimWorkspacesURL(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		suffix  string
		want    string
	}{
		{
			name:    "default v1 base url",
			baseURL: "https://api.portkey.ai/v1",
			suffix:  "",
			want:    "https://api.portkey.ai/v2/scim/workspaces",
		},
		{
			name:    "default v1 base url with id suffix",
			baseURL: "https://api.portkey.ai/v1",
			suffix:  "/abc-123",
			want:    "https://api.portkey.ai/v2/scim/workspaces/abc-123",
		},
		{
			name:    "default v1 base url with query suffix",
			baseURL: "https://api.portkey.ai/v1",
			suffix:  "?workspace_id=ws-foo",
			want:    "https://api.portkey.ai/v2/scim/workspaces?workspace_id=ws-foo",
		},
		{
			name:    "self-hosted base url that does not end in /v1",
			baseURL: "https://portkey.internal.example.com/api",
			suffix:  "",
			want:    "https://portkey.internal.example.com/api/v2/scim/workspaces",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{BaseURL: tc.baseURL}
			got := c.scimWorkspacesURL(tc.suffix)
			if got != tc.want {
				t.Errorf("scimWorkspacesURL(%q) = %q; want %q", tc.suffix, got, tc.want)
			}
		})
	}
}

// TestCreateScimWorkspaceMapping_OK verifies that Create POSTs the right body
// to /v2/scim/workspaces (BaseURL /v1 swapped for /v2) and parses the
// response correctly.
func TestCreateScimWorkspaceMapping_OK(t *testing.T) {
	var capturedMethod, capturedPath string
	var capturedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &capturedBody)
		_, _ = w.Write([]byte(`{
			"id":"map-1",
			"workspace_id":"ws-example-abcd12",
			"scim_group":"Engineering Team",
			"scim_group_id":"d290f1ee-6c54-4b01-90e6-d701748f0851",
			"role":"member"
		}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL+"/v1")

	got, err := c.CreateScimWorkspaceMapping(context.Background(), CreateScimWorkspaceMappingRequest{
		WorkspaceID:   "ws-example-abcd12",
		Role:          "member",
		ScimGroupName: "Engineering Team",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("method: got %q, want POST", capturedMethod)
	}
	if capturedPath != "/v2/scim/workspaces" {
		t.Errorf("path: got %q, want /v2/scim/workspaces (the v1->v2 swap)", capturedPath)
	}
	if capturedBody["workspace_id"] != "ws-example-abcd12" ||
		capturedBody["role"] != "member" ||
		capturedBody["scim_group_name"] != "Engineering Team" {
		t.Errorf("body: got %+v", capturedBody)
	}
	if _, hasID := capturedBody["scim_group_id"]; hasID {
		t.Errorf("body unexpectedly carried scim_group_id when only scim_group_name was set: %+v", capturedBody)
	}

	if got.ID != "map-1" || got.WorkspaceID != "ws-example-abcd12" ||
		got.ScimGroup != "Engineering Team" ||
		got.ScimGroupID != "d290f1ee-6c54-4b01-90e6-d701748f0851" ||
		got.Role != "member" {
		t.Errorf("parsed mapping: got %+v", got)
	}
}

// TestCreateScimWorkspaceMapping_4xx propagates a 4xx error verbatim.
func TestCreateScimWorkspaceMapping_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"validation failed"}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL+"/v1")
	_, err := c.CreateScimWorkspaceMapping(context.Background(), CreateScimWorkspaceMappingRequest{
		WorkspaceID: "ws-1",
		Role:        "admin",
		ScimGroupID: "abc",
	})
	if err == nil {
		t.Fatalf("expected error on 400, got nil")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("error did not surface the response body: %v", err)
	}
}

// TestListScimWorkspaceMappings_WithFilters verifies that filters are encoded
// in the query string and the data array is unmarshalled correctly.
func TestListScimWorkspaceMappings_WithFilters(t *testing.T) {
	var capturedRawQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRawQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{
			"total_count": 2,
			"mappings": [
				{"id":"map-1","workspace_id":"ws-example-abcd12","scim_group":"A","scim_group_id":"sg-1","role":"admin"},
				{"id":"map-2","workspace_id":"ws-example-abcd12","scim_group":"B","scim_group_id":"sg-2","role":"member"}
			]
		}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL+"/v1")
	got, err := c.ListScimWorkspaceMappings(context.Background(), ListScimWorkspaceMappingsOptions{
		WorkspaceID: "ws-example-abcd12",
		Role:        "admin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(got))
	}

	// Filters must be present; tolerate either order in the query string.
	if !strings.Contains(capturedRawQuery, "workspace_id=ws-example-abcd12") {
		t.Errorf("missing workspace_id filter; raw query was %q", capturedRawQuery)
	}
	if !strings.Contains(capturedRawQuery, "role=admin") {
		t.Errorf("missing role filter; raw query was %q", capturedRawQuery)
	}

	if got[0].ID != "map-1" || got[1].Role != "member" {
		t.Errorf("unexpected mappings parsed: %+v", got)
	}
}

// TestListScimWorkspaceMappings_NoFilters omits the query string entirely
// when no filter is set.
func TestListScimWorkspaceMappings_NoFilters(t *testing.T) {
	var capturedRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRawQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"total_count":0,"mappings":[]}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL+"/v1")
	_, err := c.ListScimWorkspaceMappings(context.Background(), ListScimWorkspaceMappingsOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedRawQuery != "" {
		t.Errorf("expected no query string with empty options, got %q", capturedRawQuery)
	}
}

// TestDeleteScimWorkspaceMapping_OK verifies the DELETE path uses the
// /v2/scim/workspaces/{id} form.
func TestDeleteScimWorkspaceMapping_OK(t *testing.T) {
	var capturedMethod, capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL+"/v1")
	if err := c.DeleteScimWorkspaceMapping(context.Background(), "map-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedMethod != http.MethodDelete {
		t.Errorf("method: got %q, want DELETE", capturedMethod)
	}
	if capturedPath != "/v2/scim/workspaces/map-1" {
		t.Errorf("path: got %q, want /v2/scim/workspaces/map-1", capturedPath)
	}
}
