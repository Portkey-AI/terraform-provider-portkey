package provider

import (
	"context"
	"fmt"
	"strings"

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
	_ resource.Resource                = &apiKeyResource{}
	_ resource.ResourceWithConfigure   = &apiKeyResource{}
	_ resource.ResourceWithImportState = &apiKeyResource{}
)

// NewAPIKeyResource is a helper function to simplify the provider implementation.
func NewAPIKeyResource() resource.Resource {
	return &apiKeyResource{}
}

// apiKeyResource is the resource implementation.
type apiKeyResource struct {
	client *client.Client
}

// apiKeyResourceModel maps the resource schema data.
type apiKeyResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Key            types.String `tfsdk:"key"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Type           types.String `tfsdk:"type"`
	SubType        types.String `tfsdk:"sub_type"`
	OrganisationID types.String `tfsdk:"organisation_id"`
	WorkspaceID    types.String `tfsdk:"workspace_id"`
	UserID         types.String `tfsdk:"user_id"`
	Status         types.String `tfsdk:"status"`
	Scopes         types.List   `tfsdk:"scopes"`
	RateLimits     types.List   `tfsdk:"rate_limits"`
	UsageLimits    types.Object `tfsdk:"usage_limits"`
	Metadata       types.Map    `tfsdk:"metadata"`
	AlertEmails    types.List   `tfsdk:"alert_emails"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *apiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

// Schema defines the schema for the resource.
func (r *apiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages a Portkey API Key.

API Key Types:
- Admin API Key (type="organisation", sub_type="service"): Access to Admin APIs for organization management
- Workspace Service Key (type="workspace", sub_type="service"): Workspace-scoped service access
- Workspace User Key (type="workspace", sub_type="user"): User-specific workspace access (requires user_id)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "API Key identifier (UUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Description: "The actual API key value. Only returned on creation and stored in state.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name for the API key.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description of the API key.",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type of API key: 'organisation' (for Admin API keys) or 'workspace' (for workspace-scoped keys).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sub_type": schema.StringAttribute{
				Description: "Sub-type of API key: 'service' (machine-to-machine) or 'user' (user-specific, requires user_id).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organisation_id": schema.StringAttribute{
				Description: "Organisation ID this key belongs to.",
				Computed:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID. Required for workspace API keys (type='workspace'). Not used for Admin API keys.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "User ID for user-type keys. Required when sub_type is 'user'.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Status of the API key (active, exhausted).",
				Computed:    true,
			},
			"scopes": schema.ListAttribute{
				Description: "List of permission scopes for this API key.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"usage_limits": schema.SingleNestedAttribute{
				Description: "Usage limits for this API key.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"credits_limit": schema.Float64Attribute{
						Description: "The credit limit value (e.g. 500.0 for $500).",
						Optional:    true,
					},
					"credits_limit_type": schema.StringAttribute{
						Description: "Period for the credit limit: 'per_day', 'monthly', or 'total'.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("per_day", "monthly", "total"),
						},
					},
				},
			},
			"rate_limits": schema.ListNestedAttribute{
				Description: "Rate limits for this API key.",
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
				Description: "Custom metadata to attach to the API key. This metadata will be included with every request made using this key. Useful for tracking, observability, and identifying services. Example: {\"_user\": \"service-name\", \"service_uuid\": \"abc123\"}",
				Optional:    true,
				ElementType: types.StringType,
			},
			"alert_emails": schema.ListAttribute{
				Description: "List of email addresses to receive alerts related to this API key's usage.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the API key was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the API key was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *apiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan apiKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate required fields based on type/subtype
	keyType := plan.Type.ValueString()
	subType := plan.SubType.ValueString()

	if keyType == "workspace" && plan.WorkspaceID.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Field",
			"workspace_id is required when type is 'workspace'",
		)
		return
	}

	if subType == "user" && plan.UserID.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Field",
			"user_id is required when sub_type is 'user'",
		)
		return
	}

	// Build create request
	createReq := client.CreateAPIKeyRequest{
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}

	if !plan.WorkspaceID.IsNull() && !plan.WorkspaceID.IsUnknown() {
		createReq.WorkspaceID = plan.WorkspaceID.ValueString()
	}

	if !plan.UserID.IsNull() && !plan.UserID.IsUnknown() {
		createReq.UserID = plan.UserID.ValueString()
	}

	// Handle scopes
	if !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		var scopes []string
		diags = plan.Scopes.ElementsAs(ctx, &scopes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Scopes = scopes
	}

	// Handle metadata
	if !plan.Metadata.IsNull() && !plan.Metadata.IsUnknown() {
		var metadata map[string]string
		diags = plan.Metadata.ElementsAs(ctx, &metadata, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if createReq.Defaults == nil {
			createReq.Defaults = &client.APIKeyDefaults{}
		}
		createReq.Defaults.Metadata = metadata
	}

	// Handle usage_limits
	if !plan.UsageLimits.IsNull() && !plan.UsageLimits.IsUnknown() {
		createReq.UsageLimits = terraformToAPIKeyUsageLimits(plan.UsageLimits)
	}

	// Handle rate_limits
	if !plan.RateLimits.IsNull() && !plan.RateLimits.IsUnknown() {
		createReq.RateLimits = terraformToAPIKeyRateLimits(plan.RateLimits)
	}

	// Handle alert_emails
	if !plan.AlertEmails.IsNull() && !plan.AlertEmails.IsUnknown() {
		var alertEmails []string
		diags = plan.AlertEmails.ElementsAs(ctx, &alertEmails, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.AlertEmails = alertEmails
	}

	// Create API key
	createResp, err := r.client.CreateAPIKey(ctx, keyType, subType, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating API key",
			"Could not create API key, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch the full API key details
	apiKey, err := r.client.GetAPIKey(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading API key after creation",
			"Could not read API key, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(apiKey.ID)
	plan.Key = types.StringValue(createResp.Key) // Store the full key from creation
	plan.OrganisationID = types.StringValue(apiKey.OrganisationID)
	plan.Status = types.StringValue(apiKey.Status)
	plan.CreatedAt = types.StringValue(apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !apiKey.UpdatedAt.IsZero() {
		plan.UpdatedAt = types.StringValue(apiKey.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		plan.UpdatedAt = types.StringValue(apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Preserve workspace_id from plan if set (API returns UUID but user may have provided slug)
	if plan.WorkspaceID.IsNull() || plan.WorkspaceID.IsUnknown() {
		if apiKey.WorkspaceID != "" {
			plan.WorkspaceID = types.StringValue(apiKey.WorkspaceID)
		}
	}

	// Handle user_id from API
	if apiKey.UserID != "" {
		plan.UserID = types.StringValue(apiKey.UserID)
	}

	// Handle scopes from API
	if len(apiKey.Scopes) > 0 {
		scopesList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.Scopes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Scopes = scopesList
	}

	// Handle usage_limits from API
	ulObj, ulDiags := apiKeyUsageLimitsToTerraform(apiKey.UsageLimits)
	resp.Diagnostics.Append(ulDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.UsageLimits = ulObj

	// Handle rate_limits from API
	rlList, rlDiags := apiKeyRateLimitsToTerraformList(apiKey.RateLimits)
	resp.Diagnostics.Append(rlDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.RateLimits = rlList

	// Handle metadata from API
	if apiKey.Defaults != nil && len(apiKey.Defaults.Metadata) > 0 {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, apiKey.Defaults.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Metadata = metadataMap
	} else if plan.Metadata.IsNull() {
		plan.Metadata = types.MapNull(types.StringType)
	}

	// Handle alert_emails from API
	if len(apiKey.AlertEmails) > 0 {
		alertEmailsList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.AlertEmails)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.AlertEmails = alertEmailsList
	} else if plan.AlertEmails.IsNull() {
		plan.AlertEmails = types.ListNull(types.StringType)
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state apiKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed API key value from Portkey
	apiKey, err := r.client.GetAPIKey(ctx, state.ID.ValueString())
	if err != nil {
		// Check if it's a 404 (not found)
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Portkey API Key",
			"Could not read Portkey API key ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state (keep key from state as it's only returned on creation)
	state.Name = types.StringValue(apiKey.Name)
	state.OrganisationID = types.StringValue(apiKey.OrganisationID)
	state.Status = types.StringValue(apiKey.Status)

	// Preserve type/sub_type from state to avoid triggering RequiresReplace unnecessarily
	if state.Type.IsNull() || state.Type.IsUnknown() {
		parsedType, _ := parseAPIKeyType(apiKey.Type)
		state.Type = types.StringValue(parsedType)
	}
	if state.SubType.IsNull() || state.SubType.IsUnknown() {
		_, parsedSubType := parseAPIKeyType(apiKey.Type)
		state.SubType = types.StringValue(parsedSubType)
	}

	if apiKey.Description != "" {
		state.Description = types.StringValue(apiKey.Description)
	}

	// Preserve workspace_id from state to avoid triggering RequiresReplace unnecessarily
	if state.WorkspaceID.IsNull() || state.WorkspaceID.IsUnknown() {
		if apiKey.WorkspaceID != "" {
			state.WorkspaceID = types.StringValue(apiKey.WorkspaceID)
		}
	}

	// Preserve user_id from state to avoid triggering RequiresReplace unnecessarily
	if state.UserID.IsNull() || state.UserID.IsUnknown() {
		if apiKey.UserID != "" {
			state.UserID = types.StringValue(apiKey.UserID)
		}
	}

	// Handle scopes
	if len(apiKey.Scopes) > 0 {
		scopesList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.Scopes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Scopes = scopesList
	}

	// Handle usage_limits from API
	ulObj, ulDiags := apiKeyUsageLimitsToTerraform(apiKey.UsageLimits)
	resp.Diagnostics.Append(ulDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.UsageLimits = ulObj

	// Handle rate_limits from API
	rlList, rlDiags := apiKeyRateLimitsToTerraformList(apiKey.RateLimits)
	resp.Diagnostics.Append(rlDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.RateLimits = rlList

	// Handle metadata from API
	if apiKey.Defaults != nil && len(apiKey.Defaults.Metadata) > 0 {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, apiKey.Defaults.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Metadata = metadataMap
	} else {
		state.Metadata = types.MapNull(types.StringType)
	}

	// Handle alert_emails from API
	if len(apiKey.AlertEmails) > 0 {
		alertEmailsList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.AlertEmails)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.AlertEmails = alertEmailsList
	} else {
		state.AlertEmails = types.ListNull(types.StringType)
	}

	state.CreatedAt = types.StringValue(apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !apiKey.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(apiKey.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan apiKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state apiKeyResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request
	updateReq := client.UpdateAPIKeyRequest{
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() {
		updateReq.Description = plan.Description.ValueString()
	}

	// Handle scopes
	if !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		var scopes []string
		diags = plan.Scopes.ElementsAs(ctx, &scopes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Scopes = scopes
	}

	// Handle metadata
	if !plan.Metadata.IsNull() && !plan.Metadata.IsUnknown() {
		var metadata map[string]string
		diags = plan.Metadata.ElementsAs(ctx, &metadata, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if updateReq.Defaults == nil {
			updateReq.Defaults = &client.APIKeyDefaults{}
		}
		updateReq.Defaults.Metadata = metadata
	}

	// Handle usage_limits
	if !plan.UsageLimits.IsNull() && !plan.UsageLimits.IsUnknown() {
		updateReq.UsageLimits = terraformToAPIKeyUsageLimits(plan.UsageLimits)
	}

	// Handle rate_limits
	if !plan.RateLimits.IsNull() && !plan.RateLimits.IsUnknown() {
		updateReq.RateLimits = terraformToAPIKeyRateLimits(plan.RateLimits)
	}

	// Handle alert_emails
	if !plan.AlertEmails.IsNull() && !plan.AlertEmails.IsUnknown() {
		var alertEmails []string
		diags = plan.AlertEmails.ElementsAs(ctx, &alertEmails, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.AlertEmails = alertEmails
	}

	apiKey, err := r.client.UpdateAPIKey(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Portkey API Key",
			"Could not update API key, unexpected error: "+err.Error(),
		)
		return
	}

	// Update plan with refreshed values, keeping key from state
	plan.ID = types.StringValue(apiKey.ID)
	plan.Key = state.Key // Keep the key from state
	plan.OrganisationID = types.StringValue(apiKey.OrganisationID)
	plan.Status = types.StringValue(apiKey.Status)
	plan.CreatedAt = types.StringValue(apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !apiKey.UpdatedAt.IsZero() {
		plan.UpdatedAt = types.StringValue(apiKey.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Handle scopes from API
	if len(apiKey.Scopes) > 0 {
		scopesList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.Scopes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Scopes = scopesList
	}

	// Handle usage_limits from API
	ulObj, ulDiags := apiKeyUsageLimitsToTerraform(apiKey.UsageLimits)
	resp.Diagnostics.Append(ulDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.UsageLimits = ulObj

	// Handle rate_limits from API
	rlList, rlDiags := apiKeyRateLimitsToTerraformList(apiKey.RateLimits)
	resp.Diagnostics.Append(rlDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.RateLimits = rlList

	// Handle metadata from API
	if apiKey.Defaults != nil && len(apiKey.Defaults.Metadata) > 0 {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, apiKey.Defaults.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Metadata = metadataMap
	} else if plan.Metadata.IsNull() {
		plan.Metadata = types.MapNull(types.StringType)
	}

	// Handle alert_emails from API
	if len(apiKey.AlertEmails) > 0 {
		alertEmailsList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.AlertEmails)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.AlertEmails = alertEmailsList
	} else if plan.AlertEmails.IsNull() {
		plan.AlertEmails = types.ListNull(types.StringType)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state apiKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing API key
	err := r.client.DeleteAPIKey(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Portkey API Key",
			"Could not delete API key, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// parseAPIKeyType parses the combined type field (e.g., "organisation-service") into type and sub_type
func parseAPIKeyType(combinedType string) (keyType, subType string) {
	parts := strings.SplitN(combinedType, "-", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return combinedType, ""
}
