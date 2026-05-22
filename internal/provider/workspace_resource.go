package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	Icon        types.String `tfsdk:"icon"`
	Description types.String `tfsdk:"description"`
	UsageLimits types.List   `tfsdk:"usage_limits"`
	RateLimits  types.List   `tfsdk:"rate_limits"`
	Metadata    types.Map    `tfsdk:"metadata"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// stripIconPrefix removes the icon emoji prefix from a workspace name.
// The Portkey API prepends the icon to the name in GET responses (e.g.
// icon="🚀", name="🚀 Production"). This function strips it so state stores
// the clean name the user configured.
func stripIconPrefix(name, icon string) string {
	if icon == "" {
		return name
	}
	prefix := icon + " "
	if strings.HasPrefix(name, prefix) {
		return strings.TrimPrefix(name, prefix)
	}
	return name
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
				Description: "Name of the workspace. When icon is set, the Portkey API prepends the icon emoji to the name in responses; the provider strips it automatically so this attribute always reflects the clean name you configured.",
				Required:    true,
			},
			"icon": schema.StringAttribute{
				Description: "Emoji icon for the workspace. When set, the Portkey UI displays this icon alongside the workspace name. The API prepends the icon to the name in responses; the provider strips the prefix automatically so the name attribute always reflects the clean name you configured. Set to an empty string to clear the icon. When omitted, the provider preserves backwards-compatible behavior (emoji stays in name if present).",
				Optional:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the workspace.",
				Optional:    true,
			},
			"usage_limits": schema.ListNestedAttribute{
				Description: "Usage limits for this workspace.",
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Type of usage limit: 'cost' or 'tokens'.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("cost", "tokens"),
							},
						},
						"credit_limit": schema.Int64Attribute{
							Description: "The credit limit value.",
							Optional:    true,
						},
						"alert_threshold": schema.Int64Attribute{
							Description: "Alert threshold in dollars. Triggers email notification when usage reaches this amount.",
							Optional:    true,
						},
						"periodic_reset": schema.StringAttribute{
							Description: "When to reset the usage: 'monthly' or 'weekly'.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("monthly", "weekly"),
							},
						},
					},
				},
			},
			"rate_limits": schema.ListNestedAttribute{
				Description: "Rate limits for this workspace.",
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Type of rate limit: 'requests' or 'tokens'.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("requests", "tokens"),
							},
						},
						"unit": schema.StringAttribute{
							Description: "Rate limit unit: 'rpm' (per minute), 'rph' (per hour), or 'rpd' (per day).",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("rpm", "rph", "rpd"),
							},
						},
						"value": schema.Int64Attribute{
							Description: "The rate limit value.",
							Required:    true,
						},
					},
				},
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

	// Include icon if set in config
	if !plan.Icon.IsNull() && !plan.Icon.IsUnknown() {
		createReq.Icon = plan.Icon.ValueString()
	}

	// Build limits from plan
	usageLimits, rateLimits, limitDiags := buildWorkspaceLimitsFromPlan(ctx, &plan)
	resp.Diagnostics.Append(limitDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	createReq.UsageLimits = usageLimits
	createReq.RateLimits = rateLimits

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

	// Handle icon and name from API response after create.
	// Only strip the icon prefix when the user explicitly set icon in config.
	// If icon was not set, preserve today's behavior: store the full API name
	// as-is and keep icon null — even if the API auto-extracted one from an
	// emoji in the name.
	if !plan.Icon.IsNull() && !plan.Icon.IsUnknown() {
		// User explicitly set icon — strip prefix from returned name
		plan.Name = types.StringValue(stripIconPrefix(workspace.Name, plan.Icon.ValueString()))
	} else {
		// User did NOT set icon — backwards compatible. Do NOT store the
		// API's auto-extracted icon in state; that would cause Read to
		// misinterpret it as user-managed and start stripping on refresh.
		plan.Icon = types.StringNull()
	}

	// Handle usage_limits: trust plan values when user specified them, since
	// the API has eventual consistency and may not return limits immediately.
	if !plan.UsageLimits.IsNull() && !plan.UsageLimits.IsUnknown() && len(plan.UsageLimits.Elements()) > 0 {
		// Keep plan values — API response may be stale
	} else {
		ulList, ulDiags := workspaceUsageLimitsToTerraformList(workspace.UsageLimits)
		resp.Diagnostics.Append(ulDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.UsageLimits = ulList
	}

	// Handle rate_limits — same approach
	if !plan.RateLimits.IsNull() && !plan.RateLimits.IsUnknown() && len(plan.RateLimits.Elements()) > 0 {
		// Keep plan values
	} else {
		rlList, rlDiags := workspaceRateLimitsToTerraformList(workspace.RateLimits)
		resp.Diagnostics.Append(rlDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.RateLimits = rlList
	}

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

	// Handle icon and name from API response.
	// The API prepends the icon to the name (e.g. icon="🚀", name="🚀 Production").
	//
	// Backwards compatibility contract:
	// - If the user has NOT opted into icon management (state.Icon is null),
	//   preserve today's behavior: store the full API name as-is, keep icon null.
	// - If the user HAS opted in (state.Icon is non-null — set via Create or
	//   Update with an explicit icon value), strip the prefix and track the icon.
	//
	// We detect opt-in by checking if state.Icon is non-null. This works because:
	// - Create only sets icon to non-null when the user's plan has icon set
	// - Update only sets icon to non-null when the user's config has icon set
	// - ImportState uses passthrough (icon starts null), so Read preserves
	//   backwards-compat on import
	// - The key invariant: icon transitions from null to non-null ONLY in
	//   Create/Update when the user explicitly configures it.
	userManagesIcon := !state.Icon.IsNull()
	if userManagesIcon {
		// User opted in — track icon and strip prefix from name.
		if workspace.Icon != "" {
			state.Icon = types.StringValue(workspace.Icon)
			state.Name = types.StringValue(stripIconPrefix(workspace.Name, workspace.Icon))
		} else {
			state.Icon = types.StringValue("")
			state.Name = types.StringValue(workspace.Name)
		}
	} else {
		// User has NOT opted in — backwards compatible behavior.
		// Store full API name (with emoji prefix) as-is. Keep icon null.
		state.Name = types.StringValue(workspace.Name)
		// Do NOT set state.Icon — keep it null so we don't accidentally
		// trigger icon management on the next Read cycle.
	}

	// Only set description if it's not empty - preserve null vs empty distinction
	if workspace.Description != "" {
		state.Description = types.StringValue(workspace.Description)
	} else if state.Description.IsUnknown() {
		state.Description = types.StringNull()
	}
	// Keep state.Description as-is if it was null and API returns empty

	// Handle usage_limits from API
	ulList, ulDiags := workspaceUsageLimitsToTerraformList(workspace.UsageLimits)
	resp.Diagnostics.Append(ulDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.UsageLimits = ulList

	// Handle rate_limits from API
	rlList, rlDiags := workspaceRateLimitsToTerraformList(workspace.RateLimits)
	resp.Diagnostics.Append(rlDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.RateLimits = rlList

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

	// Read raw config to detect when user removed Optional+Computed attributes.
	// Plan values for these are Unknown (not Null), so config is the reliable signal.
	var config workspaceResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update existing workspace
	updateReq := client.UpdateWorkspaceRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	// Handle icon: use config (not plan) to detect user intent.
	// Serialize as json.RawMessage so clearing (icon="") actually sends
	// "icon":"" in the JSON body rather than being omitted by omitempty.
	if !config.Icon.IsNull() {
		iconJSON, err := json.Marshal(config.Icon.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error serializing icon", err.Error())
			return
		}
		updateReq.Icon = iconJSON
	}

	// Build limits as json.RawMessage for three-state semantics (omit/null/value).
	// Use config (not plan) so IsNull() correctly detects user clearing intent.
	usageRaw, rateRaw, limitDiags := marshalWorkspaceLimitsForUpdate(ctx, &config)
	resp.Diagnostics.Append(limitDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateReq.UsageLimits = usageRaw
	updateReq.RateLimits = rateRaw

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

	workspace, err := r.client.UpdateWorkspace(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Portkey Workspace",
			"Could not update workspace, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.CreatedAt = types.StringValue(workspace.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	plan.UpdatedAt = types.StringValue(workspace.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	// Handle icon and name from API response after update.
	// Same logic as Create: only strip when user explicitly set icon.
	if !config.Icon.IsNull() {
		iconVal := config.Icon.ValueString()
		plan.Icon = types.StringValue(iconVal)
		plan.Name = types.StringValue(stripIconPrefix(workspace.Name, iconVal))
	} else {
		// User did NOT set icon — backwards compatible. Keep icon null,
		// preserve full API name (may include emoji prefix).
		plan.Icon = types.StringNull()
	}

	// Handle usage_limits after Update — three cases:
	//   1. config.UsageLimits is null → user removed the block. We already
	//      sent null on the wire to clear; mirror that as an empty list in
	//      state. (Use config — not plan — because Optional+Computed plan
	//      values become Unknown, not Null, when the user removes the block,
	//      so the API-read branch below would otherwise take effect and
	//      could write stale data.)
	//   2. plan has a known value (empty or non-empty) → trust the plan.
	//      The Portkey API has eventual consistency and the PUT response
	//      may not yet echo the newly-set values, which previously surfaced
	//      as "Provider produced inconsistent result after apply" when a
	//      user changed credit_limit/alert_threshold OR explicitly set
	//      usage_limits = [] (e.g. via variable indirection) to clear.
	//      This mirrors the Create handler above. The next Read reconciles
	//      real API state.
	//   3. otherwise (plan Unknown) → read back from the API.
	//
	// The Create/Update branches are kept parallel deliberately rather than
	// factored into a helper, because Update has the extra null-clearing
	// case (#1) that Create does not.
	if config.UsageLimits.IsNull() {
		plan.UsageLimits = types.ListValueMust(workspaceUsageLimitsObjectType, []attr.Value{})
	} else if !plan.UsageLimits.IsUnknown() {
		// Keep plan values — API response may be stale.
	} else {
		ulList, ulDiags := workspaceUsageLimitsToTerraformList(workspace.UsageLimits)
		resp.Diagnostics.Append(ulDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.UsageLimits = ulList
	}

	// Handle rate_limits — same three cases as usage_limits above.
	if config.RateLimits.IsNull() {
		plan.RateLimits = types.ListValueMust(workspaceRateLimitsObjectType, []attr.Value{})
	} else if !plan.RateLimits.IsUnknown() {
		// Keep plan values — API response may be stale.
	} else {
		rlList, rlDiags := workspaceRateLimitsToTerraformList(workspace.RateLimits)
		resp.Diagnostics.Append(rlDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.RateLimits = rlList
	}

	// Handle metadata from API response
	if workspace.Defaults != nil && len(workspace.Defaults.Metadata) > 0 {
		metadataMap, mDiags := types.MapValueFrom(ctx, types.StringType, workspace.Defaults.Metadata)
		resp.Diagnostics.Append(mDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Metadata = metadataMap
	} else if plan.Metadata.IsNull() {
		plan.Metadata = types.MapNull(types.StringType)
	}

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

	// Delete existing workspace (API requires name in body as confirmation).
	// State stores the clean name (without icon prefix), which is what the API expects.
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
// We use simple passthrough — Read will populate all fields. The icon field
// starts as null in state after import, which triggers backwards-compatible
// behavior in Read (no name stripping). Users who want icon management add
// the icon attribute to their config after import.
func (r *workspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
