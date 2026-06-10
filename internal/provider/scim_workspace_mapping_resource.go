package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                     = &scimWorkspaceMappingResource{}
	_ resource.ResourceWithConfigure        = &scimWorkspaceMappingResource{}
	_ resource.ResourceWithImportState      = &scimWorkspaceMappingResource{}
	_ resource.ResourceWithConfigValidators = &scimWorkspaceMappingResource{}
)

// NewScimWorkspaceMappingResource is a helper function to simplify the provider implementation.
func NewScimWorkspaceMappingResource() resource.Resource {
	return &scimWorkspaceMappingResource{}
}

// scimWorkspaceMappingResource is the resource implementation.
type scimWorkspaceMappingResource struct {
	client *client.Client
}

// scimWorkspaceMappingResourceModel maps the resource schema data.
type scimWorkspaceMappingResourceModel struct {
	ID            types.String `tfsdk:"id"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	Role          types.String `tfsdk:"role"`
	ScimGroupID   types.String `tfsdk:"scim_group_id"`
	ScimGroupName types.String `tfsdk:"scim_group_name"`
	ScimGroup     types.String `tfsdk:"scim_group"`
}

// Metadata returns the resource type name.
func (r *scimWorkspaceMappingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scim_workspace_mapping"
}

// Schema defines the schema for the resource.
func (r *scimWorkspaceMappingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Portkey SCIM workspace mapping. Binds a SCIM-provisioned group to a workspace at a specific role (admin, member, or manager). " +
			"The Portkey API has no update endpoint for mappings; changing any field destroys and recreates the mapping.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the SCIM workspace mapping.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schema.StringAttribute{
				Description: "ID or slug of the workspace to map the SCIM group to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "Role assigned to group members in the workspace. One of: admin, member, manager.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("admin", "member", "manager"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scim_group_id": schema.StringAttribute{
				Description: "ID of an existing SCIM group. Exactly one of scim_group_id or scim_group_name must be set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"scim_group_name": schema.StringAttribute{
				Description: "Display name of the SCIM group. Used to pre-create the mapping before the identity provider pushes the group. " +
					"Exactly one of scim_group_id or scim_group_name must be set. " +
					"Must not match Portkey's auto-provisioning pattern (e.g. \"ws-<name>-role-admin\"). " +
					"This field is not echoed by the Portkey API; see scim_group for the API-reported display name.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scim_group": schema.StringAttribute{
				Description: "Display name of the mapped SCIM group as returned by the Portkey API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ConfigValidators enforces that exactly one of scim_group_id or scim_group_name is set.
func (r *scimWorkspaceMappingResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("scim_group_id"),
			path.MatchRoot("scim_group_name"),
		),
	}
}

// Configure adds the provider configured client to the resource.
func (r *scimWorkspaceMappingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *scimWorkspaceMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scimWorkspaceMappingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateScimWorkspaceMappingRequest{
		WorkspaceID:   plan.WorkspaceID.ValueString(),
		Role:          plan.Role.ValueString(),
		ScimGroupID:   plan.ScimGroupID.ValueString(),
		ScimGroupName: plan.ScimGroupName.ValueString(),
	}

	mapping, err := r.client.CreateScimWorkspaceMapping(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating SCIM workspace mapping",
			"Could not create SCIM workspace mapping, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(mapping.ID)
	// workspace_id is preserved as the user authored it. The Portkey API
	// normalizes the value on the way back (e.g. slug "ws-foo-abcd12" comes
	// back as the workspace's UUID), which trips the framework's post-apply
	// consistency check because workspace_id is Required (not Computed).
	// Keeping the planned value sidesteps that without forcing the user to
	// write the UUID by hand.
	plan.Role = types.StringValue(mapping.Role)
	plan.ScimGroupID = types.StringValue(mapping.ScimGroupID)
	plan.ScimGroup = types.StringValue(mapping.ScimGroup)
	// scim_group_name is preserved as-authored (Optional, not Computed) —
	// it is not echoed by the API and stays whatever the user wrote, or
	// null if they used scim_group_id instead.

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *scimWorkspaceMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scimWorkspaceMappingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The Portkey API has no GET-by-id for SCIM mappings. We can't filter
	// the list by workspace_id either: the API's filter only matches the
	// workspace's UUID form, while state may hold the slug the user
	// authored. Fetching the unfiltered list is fine — the SCIM mappings
	// list is small (a few hundred entries at most across the org).
	mappings, err := r.client.ListScimWorkspaceMappings(ctx, client.ListScimWorkspaceMappingsOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading SCIM workspace mapping",
			"Could not list SCIM workspace mappings: "+err.Error(),
		)
		return
	}

	stateID := state.ID.ValueString()
	for i := range mappings {
		m := mappings[i]
		if m.ID != stateID {
			continue
		}
		// workspace_id is preserved verbatim from state (the API
		// returns the UUID form regardless of which form the user
		// authored; rewriting it here would surface as drift).
		state.Role = types.StringValue(m.Role)
		state.ScimGroupID = types.StringValue(m.ScimGroupID)
		state.ScimGroup = types.StringValue(m.ScimGroup)
		// scim_group_name is preserved verbatim from state (not echoed by the API).
		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
		return
	}

	// Not found upstream — drop from state so Terraform plans a re-create.
	resp.State.RemoveResource(ctx)
}

// Update is a no-op. Every editable attribute on the resource has
// RequiresReplace, so the framework will never invoke Update with real
// diffs — but the interface still requires the method to exist.
func (r *scimWorkspaceMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scimWorkspaceMappingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *scimWorkspaceMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scimWorkspaceMappingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteScimWorkspaceMapping(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting SCIM workspace mapping",
			"Could not delete SCIM workspace mapping "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}
}

// ImportState imports an existing SCIM workspace mapping into Terraform state.
// Import ID format: workspace_id/mapping_id. The mapping_id is what Read uses
// to find the mapping; workspace_id is supplied here because it's a Required
// attribute and the API doesn't echo it back in a slug-stable form, so it
// can't be derived from the mapping alone.
func (r *scimWorkspaceMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in the format: workspace_id/mapping_id (e.g. ws-example-abcd12/a1b2c3d4-...).",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
