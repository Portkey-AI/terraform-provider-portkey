package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &workspaceMemberResource{}
	_ resource.ResourceWithConfigure   = &workspaceMemberResource{}
	_ resource.ResourceWithImportState = &workspaceMemberResource{}
)

// NewWorkspaceMemberResource is a helper function to simplify the provider implementation.
func NewWorkspaceMemberResource() resource.Resource {
	return &workspaceMemberResource{}
}

// workspaceMemberResource is the resource implementation.
type workspaceMemberResource struct {
	client *client.Client
}

// workspaceMemberResourceModel maps the resource schema data.
type workspaceMemberResourceModel struct {
	ID          types.String `tfsdk:"id"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
	UserID      types.String `tfsdk:"user_id"`
	Role        types.String `tfsdk:"role"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

// Metadata returns the resource type name.
func (r *workspaceMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_member"
}

// Schema defines the schema for the resource.
func (r *workspaceMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Portkey workspace member. Assigns users to workspaces with specific roles.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Workspace member identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schema.StringAttribute{
				Description: "ID of the workspace.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "ID of the user to add to the workspace.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "Role of the user in the workspace (e.g., 'admin', 'member', 'manager').",
				Required:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the member was added to the workspace.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *workspaceMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *workspaceMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan workspaceMemberResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Add member to workspace
	addReq := client.AddWorkspaceMemberRequest{
		UserID: plan.UserID.ValueString(),
		Role:   plan.Role.ValueString(),
	}

	member, err := r.client.AddWorkspaceMember(ctx, plan.WorkspaceID.ValueString(), addReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding workspace member",
			"Could not add workspace member, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(member.ID)
	plan.CreatedAt = types.StringValue(member.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *workspaceMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state workspaceMemberResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed member value from Portkey using user_id
	member, err := r.client.GetWorkspaceMember(ctx, state.WorkspaceID.ValueString(), state.UserID.ValueString())
	if err != nil {
		// Portkey returns 404 (AB08) for memberships removed out-of-band,
		// meaning the membership no longer exists.
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Portkey Workspace Member",
			"Could not read Portkey workspace member for user "+state.UserID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state with refreshed values
	state.ID = types.StringValue(member.ID)
	state.Role = types.StringValue(member.Role)
	if !member.CreatedAt.IsZero() {
		state.CreatedAt = types.StringValue(member.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *workspaceMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan workspaceMemberResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update workspace member role using user_id
	updateReq := client.UpdateWorkspaceMemberRequest{
		Role: plan.Role.ValueString(),
	}

	member, err := r.client.UpdateWorkspaceMember(ctx, plan.WorkspaceID.ValueString(), plan.UserID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Portkey Workspace Member",
			"Could not update workspace member, unexpected error: "+err.Error(),
		)
		return
	}

	// Update state with response
	plan.ID = types.StringValue(member.ID)
	if !member.CreatedAt.IsZero() {
		plan.CreatedAt = types.StringValue(member.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *workspaceMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state workspaceMemberResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Remove member from workspace using user_id
	err := r.client.RemoveWorkspaceMember(ctx, state.WorkspaceID.ValueString(), state.UserID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Removing Portkey Workspace Member",
			"Could not remove workspace member, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *workspaceMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID should be in format: workspace_id/user_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: workspace_id/user_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
