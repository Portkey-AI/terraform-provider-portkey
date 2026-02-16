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
	_ resource.Resource                = &mcpServerUserAccessResource{}
	_ resource.ResourceWithConfigure   = &mcpServerUserAccessResource{}
	_ resource.ResourceWithImportState = &mcpServerUserAccessResource{}
)

// NewMcpServerUserAccessResource is a helper function to simplify the provider implementation.
func NewMcpServerUserAccessResource() resource.Resource {
	return &mcpServerUserAccessResource{}
}

// mcpServerUserAccessResource is the resource implementation.
type mcpServerUserAccessResource struct {
	client *client.Client
}

// mcpServerUserAccessResourceModel maps the resource schema data.
type mcpServerUserAccessResourceModel struct {
	ID          types.String `tfsdk:"id"`
	McpServerID types.String `tfsdk:"mcp_server_id"`
	UserID      types.String `tfsdk:"user_id"`
	Enabled     types.Bool   `tfsdk:"enabled"`
}

// Metadata returns the resource type name.
func (r *mcpServerUserAccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_server_user_access"
}

// Schema defines the schema for the resource.
func (r *mcpServerUserAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages user access for a Portkey MCP server. Controls which users can use a specific MCP server within a workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier in format mcp_server_id/user_id.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mcp_server_id": schema.StringAttribute{
				Description: "The MCP server ID or slug to grant user access to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "The user ID to grant access to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the MCP server is enabled for this user. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *mcpServerUserAccessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *mcpServerUserAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mcpServerUserAccessResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := client.McpServerUserAccessUpdate{
		UserID:  plan.UserID.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
	}

	err := r.client.UpdateMcpServerUserAccess(ctx, plan.McpServerID.ValueString(), update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MCP server user access",
			"Could not create MCP server user access: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", plan.McpServerID.ValueString(), plan.UserID.ValueString()))

	// Fetch actual state from API
	userAccess, err := r.client.GetMcpServerUserAccessItem(ctx, plan.McpServerID.ValueString(), plan.UserID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCP server user access after creation",
			"Could not read MCP server user access: "+err.Error(),
		)
		return
	}

	plan.Enabled = types.BoolValue(userAccess.Enabled)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *mcpServerUserAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mcpServerUserAccessResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	userAccess, err := r.client.GetMcpServerUserAccessItem(ctx, state.McpServerID.ValueString(), state.UserID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading MCP server user access",
			"Could not read user access for user "+state.UserID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.Enabled = types.BoolValue(userAccess.Enabled)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mcpServerUserAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mcpServerUserAccessResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := client.McpServerUserAccessUpdate{
		UserID:  plan.UserID.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
	}

	err := r.client.UpdateMcpServerUserAccess(ctx, plan.McpServerID.ValueString(), update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating MCP server user access",
			"Could not update MCP server user access: "+err.Error(),
		)
		return
	}

	// Fetch actual state from API
	userAccess, err := r.client.GetMcpServerUserAccessItem(ctx, plan.McpServerID.ValueString(), plan.UserID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCP server user access after update",
			"Could not read MCP server user access: "+err.Error(),
		)
		return
	}

	plan.Enabled = types.BoolValue(userAccess.Enabled)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mcpServerUserAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mcpServerUserAccessResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if resource still exists
	_, err := r.client.GetMcpServerUserAccessItem(ctx, state.McpServerID.ValueString(), state.UserID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting MCP server user access",
			"Could not verify MCP server user access exists: "+err.Error(),
		)
		return
	}

	// Disable user access
	update := client.McpServerUserAccessUpdate{
		UserID:  state.UserID.ValueString(),
		Enabled: false,
	}

	err = r.client.UpdateMcpServerUserAccess(ctx, state.McpServerID.ValueString(), update)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting MCP server user access",
			"Could not disable MCP server user access: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *mcpServerUserAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in format: mcp_server_id/user_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mcp_server_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
