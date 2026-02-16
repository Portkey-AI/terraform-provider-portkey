package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &mcpIntegrationWorkspaceAccessResource{}
	_ resource.ResourceWithConfigure   = &mcpIntegrationWorkspaceAccessResource{}
	_ resource.ResourceWithImportState = &mcpIntegrationWorkspaceAccessResource{}
)

// NewMcpIntegrationWorkspaceAccessResource is a helper function to simplify the provider implementation.
func NewMcpIntegrationWorkspaceAccessResource() resource.Resource {
	return &mcpIntegrationWorkspaceAccessResource{}
}

// mcpIntegrationWorkspaceAccessResource is the resource implementation.
type mcpIntegrationWorkspaceAccessResource struct {
	client *client.Client
}

// mcpIntegrationWorkspaceAccessResourceModel maps the resource schema data.
type mcpIntegrationWorkspaceAccessResourceModel struct {
	ID               types.String `tfsdk:"id"`
	McpIntegrationID types.String `tfsdk:"mcp_integration_id"`
	WorkspaceID      types.String `tfsdk:"workspace_id"`
	Enabled          types.Bool   `tfsdk:"enabled"`
}

// Metadata returns the resource type name.
func (r *mcpIntegrationWorkspaceAccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_integration_workspace_access"
}

// Schema defines the schema for the resource.
func (r *mcpIntegrationWorkspaceAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages workspace access for a Portkey MCP integration. Controls which workspaces can use a specific MCP integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier in format mcp_integration_id/workspace_id.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mcp_integration_id": schema.StringAttribute{
				Description: "The MCP integration ID or slug to grant workspace access to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"workspace_id": schema.StringAttribute{
				Description: "The workspace ID to grant access to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the MCP integration is enabled for this workspace. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *mcpIntegrationWorkspaceAccessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// Create creates the resource and sets the initial Terraform state.
func (r *mcpIntegrationWorkspaceAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mcpIntegrationWorkspaceAccessResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := client.McpIntegrationWorkspaceUpdate{
		WorkspaceID: plan.WorkspaceID.ValueString(),
		Enabled:     plan.Enabled.ValueBool(),
	}

	err := r.client.UpdateMcpIntegrationWorkspace(ctx, plan.McpIntegrationID.ValueString(), update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MCP integration workspace access",
			"Could not create MCP integration workspace access: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", plan.McpIntegrationID.ValueString(), plan.WorkspaceID.ValueString()))

	// Fetch actual state from API
	workspace, err := r.client.GetMcpIntegrationWorkspace(ctx, plan.McpIntegrationID.ValueString(), plan.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCP integration workspace access after creation",
			"Could not read MCP integration workspace access: "+err.Error(),
		)
		return
	}

	plan.Enabled = types.BoolValue(workspace.Enabled)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *mcpIntegrationWorkspaceAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mcpIntegrationWorkspaceAccessResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspace, err := r.client.GetMcpIntegrationWorkspace(ctx, state.McpIntegrationID.ValueString(), state.WorkspaceID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading MCP integration workspace access",
			"Could not read workspace access for workspace "+state.WorkspaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.Enabled = types.BoolValue(workspace.Enabled)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mcpIntegrationWorkspaceAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mcpIntegrationWorkspaceAccessResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := client.McpIntegrationWorkspaceUpdate{
		WorkspaceID: plan.WorkspaceID.ValueString(),
		Enabled:     plan.Enabled.ValueBool(),
	}

	err := r.client.UpdateMcpIntegrationWorkspace(ctx, plan.McpIntegrationID.ValueString(), update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating MCP integration workspace access",
			"Could not update MCP integration workspace access: "+err.Error(),
		)
		return
	}

	// Fetch actual state from API
	workspace, err := r.client.GetMcpIntegrationWorkspace(ctx, plan.McpIntegrationID.ValueString(), plan.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCP integration workspace access after update",
			"Could not read MCP integration workspace access: "+err.Error(),
		)
		return
	}

	plan.Enabled = types.BoolValue(workspace.Enabled)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mcpIntegrationWorkspaceAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mcpIntegrationWorkspaceAccessResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if resource still exists
	_, err := r.client.GetMcpIntegrationWorkspace(ctx, state.McpIntegrationID.ValueString(), state.WorkspaceID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting MCP integration workspace access",
			"Could not verify MCP integration workspace access exists: "+err.Error(),
		)
		return
	}

	// Disable workspace access
	update := client.McpIntegrationWorkspaceUpdate{
		WorkspaceID: state.WorkspaceID.ValueString(),
		Enabled:     false,
	}

	err = r.client.UpdateMcpIntegrationWorkspace(ctx, state.McpIntegrationID.ValueString(), update)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting MCP integration workspace access",
			"Could not disable MCP integration workspace access: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *mcpIntegrationWorkspaceAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in format: mcp_integration_id/workspace_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mcp_integration_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
