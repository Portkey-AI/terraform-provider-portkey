package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &workspaceResource{}
	_ resource.ResourceWithConfigure   = &workspaceResource{}
	_ resource.ResourceWithImportState = &workspaceResource{}
)

// NewWorkspaceResource is a helper function to simplify the provider implementation.
func NewWorkspaceResource() resource.Resource {
	return &workspaceResource{}
}

// workspaceResource is the resource implementation.
type workspaceResource struct {
	client *client.Client
}

// workspaceResourceModel maps the resource schema data.
type workspaceResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Metadata    types.Map    `tfsdk:"metadata"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *workspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

// Schema defines the schema for the resource.
func (r *workspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Portkey workspace. Workspaces are sub-organizational units that enable granular project and team management.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Workspace identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the workspace.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the workspace.",
				Optional:    true,
			},
			"metadata": schema.MapAttribute{
				Description: "Custom metadata to attach to the workspace. This metadata can be used for tracking, observability, and identifying workspaces. All API keys created in this workspace will inherit this metadata by default.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the workspace was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the workspace was last updated.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *workspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *workspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan workspaceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new workspace
	createReq := client.CreateWorkspaceRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	// Handle metadata
	if !plan.Metadata.IsNull() && !plan.Metadata.IsUnknown() {
		var metadata map[string]string
		diags = plan.Metadata.ElementsAs(ctx, &metadata, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Defaults = &client.WorkspaceDefaults{
			Metadata: metadata,
		}
	}

	workspace, err := r.client.CreateWorkspace(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating workspace",
			"Could not create workspace, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(workspace.ID)
	plan.CreatedAt = types.StringValue(workspace.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	plan.UpdatedAt = types.StringValue(workspace.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	// Handle metadata from API response
	if workspace.Defaults != nil && len(workspace.Defaults.Metadata) > 0 {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, workspace.Defaults.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Metadata = metadataMap
	} else if plan.Metadata.IsNull() {
		plan.Metadata = types.MapNull(types.StringType)
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *workspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state workspaceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed workspace value from Portkey
	workspace, err := r.client.GetWorkspace(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Portkey Workspace",
			"Could not read Portkey workspace ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.Name = types.StringValue(workspace.Name)
	// Only set description if it's not empty - preserve null vs empty distinction
	if workspace.Description != "" {
		state.Description = types.StringValue(workspace.Description)
	} else if state.Description.IsUnknown() {
		state.Description = types.StringNull()
	}
	// Keep state.Description as-is if it was null and API returns empty

	// Handle metadata from API
	if workspace.Defaults != nil && len(workspace.Defaults.Metadata) > 0 {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, workspace.Defaults.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Metadata = metadataMap
	} else {
		state.Metadata = types.MapNull(types.StringType)
	}

	state.CreatedAt = types.StringValue(workspace.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	state.UpdatedAt = types.StringValue(workspace.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *workspaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan workspaceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update existing workspace
	updateReq := client.UpdateWorkspaceRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	// Handle metadata
	if !plan.Metadata.IsNull() && !plan.Metadata.IsUnknown() {
		var metadata map[string]string
		diags = plan.Metadata.ElementsAs(ctx, &metadata, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Defaults = &client.WorkspaceDefaults{
			Metadata: metadata,
		}
	}

	_, err := r.client.UpdateWorkspace(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Portkey Workspace",
			"Could not update workspace, unexpected error: "+err.Error(),
		)
		return
	}

	// Note: Don't update timestamps here - the plan already has the expected values
	// from UseStateForUnknown. Let Read update them on next refresh.

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *workspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state workspaceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing workspace (API requires name in body as confirmation)
	err := r.client.DeleteWorkspace(ctx, state.ID.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Portkey Workspace",
			"Could not delete workspace: "+err.Error()+
				"\n\nIf the workspace name was changed outside of Terraform, run 'terraform refresh' to sync the state first.",
		)
		return
	}
}

// ImportState imports the resource state.
func (r *workspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
