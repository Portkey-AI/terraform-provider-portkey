package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// These tests drive the resource's CRUD methods directly against a mock
// Portkey API, so they run without TF_ACC or a Terraform CLI. The mock mirrors
// the real API: GET /mcp-integrations/{id}/workspaces identifies every
// workspace by slug, never by UUID.

const (
	unitMcpIntegrationID = "22222222-2222-4222-8222-222222222222"
	unitWorkspaceUUID    = "11111111-1111-4111-8111-111111111111"
	unitWorkspaceSlug    = "ws-exampl-abc123"
)

type mcpWorkspaceAccessMock struct {
	mu sync.Mutex
	// listed is the slug -> enabled map returned by the workspace list.
	listed map[string]bool
	// puts records the workspace ids sent in each PUT body.
	puts []client.McpIntegrationWorkspaceUpdate
	// putStatus, when non-zero, is returned for every PUT instead of applying it.
	putStatus int
}

func newMcpWorkspaceAccessMock(t *testing.T, listed map[string]bool, putStatus int) (*mcpWorkspaceAccessMock, *client.Client) {
	t.Helper()
	m := &mcpWorkspaceAccessMock{listed: listed, putStatus: putStatus}
	listPath := "/mcp-integrations/" + unitMcpIntegrationID + "/workspaces"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/admin/workspaces/"+unitWorkspaceUUID:
			_, _ = w.Write([]byte(`{"id":"` + unitWorkspaceUUID + `","slug":"` + unitWorkspaceSlug + `","name":"Example"}`))
		case r.Method == http.MethodGet && r.URL.Path == listPath:
			var resp client.McpIntegrationWorkspacesResponse
			for slug, enabled := range m.listed {
				resp.Workspaces = append(resp.Workspaces, client.McpIntegrationWorkspace{WorkspaceID: slug, Enabled: enabled})
			}
			_ = json.NewEncoder(w).Encode(resp)
		case r.Method == http.MethodPut && r.URL.Path == listPath:
			body, _ := io.ReadAll(r.Body)
			var req client.McpIntegrationWorkspacesUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("bad PUT body %s: %v", body, err)
			}
			if m.putStatus != 0 {
				m.puts = append(m.puts, req.Workspaces...)
				w.WriteHeader(m.putStatus)
				_, _ = w.Write([]byte(`{"success":false}`))
				return
			}
			for _, u := range req.Workspaces {
				m.puts = append(m.puts, u)
				// Like the real API, only slugs are honoured.
				if _, ok := m.listed[u.WorkspaceID]; ok {
					m.listed[u.WorkspaceID] = u.Enabled
				}
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	zero := 0
	c, err := client.NewClientWithConfig(client.ClientConfig{BaseURL: srv.URL, APIKey: "test-key", MaxRetries: &zero})
	if err != nil {
		t.Fatalf("creating client: %v", err)
	}
	return m, c
}

func mcpWorkspaceAccessSchema(t *testing.T, r resource.Resource) resource.SchemaResponse {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
	return resp
}

func TestMcpIntegrationWorkspaceAccess_CreateWithUUIDWritesSlugAndKeepsUUID(t *testing.T) {
	ctx := context.Background()
	mock, c := newMcpWorkspaceAccessMock(t, map[string]bool{unitWorkspaceSlug: false}, 0)
	r := &mcpIntegrationWorkspaceAccessResource{client: c}
	s := mcpWorkspaceAccessSchema(t, r).Schema

	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, &mcpIntegrationWorkspaceAccessResourceModel{
		ID:               types.StringUnknown(),
		McpIntegrationID: types.StringValue(unitMcpIntegrationID),
		WorkspaceID:      types.StringValue(unitWorkspaceUUID),
		Enabled:          types.BoolValue(true),
	}); diags.HasError() {
		t.Fatalf("plan: %v", diags)
	}

	resp := resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	if len(mock.puts) != 1 || mock.puts[0].WorkspaceID != unitWorkspaceSlug || !mock.puts[0].Enabled {
		t.Fatalf("expected one PUT enabling slug %q, got %+v", unitWorkspaceSlug, mock.puts)
	}

	var got mcpIntegrationWorkspaceAccessResourceModel
	resp.State.Get(ctx, &got)
	if got.WorkspaceID.ValueString() != unitWorkspaceUUID {
		t.Errorf("workspace_id in state = %q, want the configured UUID %q", got.WorkspaceID.ValueString(), unitWorkspaceUUID)
	}
	if want := unitMcpIntegrationID + "/" + unitWorkspaceUUID; got.ID.ValueString() != want {
		t.Errorf("id = %q, want %q", got.ID.ValueString(), want)
	}
	if !got.Enabled.ValueBool() {
		t.Error("enabled = false, want true")
	}
}

func TestMcpIntegrationWorkspaceAccess_Delete(t *testing.T) {
	cases := []struct {
		name        string
		workspaceID string
		listed      map[string]bool
		putStatus   int
		wantErr     string
		wantPutSlug string
		// wantUnapplied means the PUT was sent but the mock rejected it.
		wantUnapplied bool
	}{
		{
			name:        "UUID disables access by slug",
			workspaceID: unitWorkspaceUUID,
			listed:      map[string]bool{unitWorkspaceSlug: true},
			wantPutSlug: unitWorkspaceSlug,
		},
		{
			name:        "slug disables access",
			workspaceID: unitWorkspaceSlug,
			listed:      map[string]bool{unitWorkspaceSlug: true},
			wantPutSlug: unitWorkspaceSlug,
		},
		{
			// Previously this returned success without disabling anything.
			name:        "workspace not listed fails loudly",
			workspaceID: "ws-missing-zzz999",
			listed:      map[string]bool{unitWorkspaceSlug: true},
			wantErr:     "is not listed for MCP integration",
		},
		{
			// An integration archived between the lookup and the PUT returns 403.
			name:          "integration gone before PUT succeeds",
			workspaceID:   unitWorkspaceUUID,
			listed:        map[string]bool{unitWorkspaceSlug: true},
			putStatus:     http.StatusForbidden,
			wantPutSlug:   unitWorkspaceSlug,
			wantUnapplied: true,
		},
		{
			name:        "PUT server error fails",
			workspaceID: unitWorkspaceUUID,
			listed:      map[string]bool{unitWorkspaceSlug: true},
			putStatus:   http.StatusInternalServerError,
			wantErr:     "Could not disable MCP integration workspace access",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			mock, c := newMcpWorkspaceAccessMock(t, tc.listed, tc.putStatus)
			r := &mcpIntegrationWorkspaceAccessResource{client: c}
			s := mcpWorkspaceAccessSchema(t, r).Schema

			state := tfsdk.State{Schema: s}
			if diags := state.Set(ctx, &mcpIntegrationWorkspaceAccessResourceModel{
				ID:               types.StringValue(unitMcpIntegrationID + "/" + tc.workspaceID),
				McpIntegrationID: types.StringValue(unitMcpIntegrationID),
				WorkspaceID:      types.StringValue(tc.workspaceID),
				Enabled:          types.BoolValue(true),
			}); diags.HasError() {
				t.Fatalf("state: %v", diags)
			}

			resp := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &resp)

			if tc.wantErr != "" {
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected Delete to fail")
				}
				var details []string
				for _, d := range resp.Diagnostics.Errors() {
					details = append(details, d.Detail())
				}
				if joined := strings.Join(details, "\n"); !strings.Contains(joined, tc.wantErr) {
					t.Fatalf("error %q does not contain %q", joined, tc.wantErr)
				}
				if tc.putStatus == 0 && len(mock.puts) != 0 {
					t.Fatalf("expected no PUT, got %+v", mock.puts)
				}
				return
			}

			if resp.Diagnostics.HasError() {
				t.Fatalf("delete: %v", resp.Diagnostics)
			}
			if len(mock.puts) != 1 || mock.puts[0].WorkspaceID != tc.wantPutSlug || mock.puts[0].Enabled {
				t.Fatalf("expected one PUT disabling slug %q, got %+v", tc.wantPutSlug, mock.puts)
			}
			if !tc.wantUnapplied && mock.listed[tc.wantPutSlug] {
				t.Fatal("access is still enabled after Delete")
			}
		})
	}
}
