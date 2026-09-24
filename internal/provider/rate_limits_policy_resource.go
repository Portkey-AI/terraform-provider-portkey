package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
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
	_ resource.Resource                = &rateLimitsPolicyResource{}
	_ resource.ResourceWithConfigure   = &rateLimitsPolicyResource{}
	_ resource.ResourceWithImportState = &rateLimitsPolicyResource{}
)

// NewRateLimitsPolicyResource is a helper function to simplify the provider implementation.
func NewRateLimitsPolicyResource() resource.Resource {
	return &rateLimitsPolicyResource{}
}

// rateLimitsPolicyResource is the resource implementation.
type rateLimitsPolicyResource struct {
	client *client.Client
}

// rateLimitsPolicyResourceModel maps the resource schema data.
type rateLimitsPolicyResourceModel struct {
	ID          types.String         `tfsdk:"id"`
	Name        types.String         `tfsdk:"name"`
	WorkspaceID types.String         `tfsdk:"workspace_id"`
	Conditions  jsontypes.Normalized `tfsdk:"conditions"`
	GroupBy     jsontypes.Normalized `tfsdk:"group_by"`
	Type        types.String         `tfsdk:"type"`
	Unit        types.String         `tfsdk:"unit"`
	Value       types.Float64        `tfsdk:"value"`
	Status      types.String         `tfsdk:"status"`
	CreatedAt   types.String         `tfsdk:"created_at"`
	UpdatedAt   types.String         `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *rateLimitsPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rate_limits_policy"
}

// Schema defines the schema for the resource.
func (r *rateLimitsPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Portkey rate limits policy. Controls the rate of requests or tokens consumed per minute, hour, or day.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Policy identifier (UUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name for the policy.",
				Optional:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID to create the policy in.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"conditions": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: "JSON array of conditions that define which requests the policy applies to. Each condition has 'key', 'value' (string or array of strings), and an optional 'excludes' (string or array of strings). Updated in place; changing it does not recreate the policy.",
				Required:    true,
			},
			"group_by": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: "JSON array of group by fields that define how rate limiting is applied. Each item has 'key'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Policy type: 'requests' or 'tokens'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"unit": schema.StringAttribute{
				Description: "Rate unit: 'rpm' (per minute), 'rph' (per hour), or 'rpd' (per day).",
				Required:    true,
			},
			"value": schema.Float64Attribute{
				Description: "Rate limit value.",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "Status of the policy (active, archived).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the policy was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the policy was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *rateLimitsPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *rateLimitsPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan rateLimitsPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse conditions JSON
	var conditions []client.PolicyCondition
	if err := json.Unmarshal([]byte(plan.Conditions.ValueString()), &conditions); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Conditions JSON",
			"The conditions attribute must be a valid JSON array: "+err.Error(),
		)
		return
	}

	// Parse group_by JSON
	var groupBy []client.PolicyGroupBy
	if err := json.Unmarshal([]byte(plan.GroupBy.ValueString()), &groupBy); err != nil {
		resp.Diagnostics.AddError(
			"Invalid GroupBy JSON",
			"The group_by attribute must be a valid JSON array: "+err.Error(),
		)
		return
	}

	// Create new policy
	createReq := client.CreateRateLimitsPolicyRequest{
		Name:        plan.Name.ValueString(),
		WorkspaceID: plan.WorkspaceID.ValueString(),
		Conditions:  conditions,
		GroupBy:     groupBy,
		Type:        plan.Type.ValueString(),
		Unit:        plan.Unit.ValueString(),
		Value:       plan.Value.ValueFloat64(),
	}

	createResp, err := r.client.CreateRateLimitsPolicy(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rate limits policy",
			"Could not create policy, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch the full policy details
	policy, err := r.client.GetRateLimitsPolicy(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading policy after creation",
			"Could not read policy, unexpected error: "+err.Error(),
		)
		return
	}

	// Preserve plan values for RequiresReplace JSON attributes so Terraform's
	// post-apply consistency check doesn't fail due to key ordering differences.
	planConditions := plan.Conditions
	planGroupBy := plan.GroupBy

	// Map response body to schema
	r.mapPolicyToState(&plan, policy, false)

	plan.Conditions = planConditions
	plan.GroupBy = planGroupBy

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *rateLimitsPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state rateLimitsPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed policy value from Portkey
	policy, err := r.client.GetRateLimitsPolicy(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Portkey Rate Limits Policy",
			"Could not read policy ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	r.mapPolicyToState(&state, policy, true)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *rateLimitsPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan rateLimitsPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state rateLimitsPolicyResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse conditions JSON
	var conditions []client.PolicyCondition
	if err := json.Unmarshal([]byte(plan.Conditions.ValueString()), &conditions); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Conditions JSON",
			"The conditions attribute must be a valid JSON array: "+err.Error(),
		)
		return
	}

	// Build update request. The mutable attributes are sent on every update rather than
	// diffed, so the request always describes the configured desired state.
	updateReq := client.UpdateRateLimitsPolicyRequest{
		Conditions: conditions,
		Unit:       plan.Unit.ValueString(),
	}

	// The API rejects a null name, so an unset name is omitted rather than cleared.
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		updateReq.Name = plan.Name.ValueString()
	}

	value := plan.Value.ValueFloat64()
	updateReq.Value = &value

	policy, err := r.client.UpdateRateLimitsPolicy(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Portkey Rate Limits Policy",
			"Could not update policy, unexpected error: "+err.Error(),
		)
		return
	}

	// Preserve plan values for the JSON attributes so Terraform's post-apply
	// consistency check doesn't fail if the API echoes a different key ordering.
	planConditions := plan.Conditions
	planGroupBy := plan.GroupBy

	// Map response to plan
	r.mapPolicyToState(&plan, policy, false)

	plan.Conditions = planConditions
	plan.GroupBy = planGroupBy

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *rateLimitsPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state rateLimitsPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing policy
	err := r.client.DeleteRateLimitsPolicy(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Portkey Rate Limits Policy",
			"Could not delete policy, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *rateLimitsPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapPolicyToState maps a RateLimitsPolicy API response to the Terraform state model
// preserveRequiresReplace, when true (the Read path), keeps the user-authored value
// from state for attributes the API may echo back in a different shape, instead of
// overwriting state from the response. Note that this also means changes made outside
// Terraform to those attributes are not detected as drift.
func (r *rateLimitsPolicyResource) mapPolicyToState(state *rateLimitsPolicyResourceModel, policy *client.RateLimitsPolicy, preserveRequiresReplace bool) {
	state.ID = types.StringValue(policy.ID)
	state.Name = types.StringValue(policy.Name)
	// Always preserve workspace_id from state if set (API returns UUID but user may have provided slug)
	if state.WorkspaceID.IsNull() || state.WorkspaceID.IsUnknown() {
		state.WorkspaceID = types.StringValue(policy.WorkspaceID)
	}
	// type still forces replacement: the API's update endpoint does not accept it
	if !preserveRequiresReplace || state.Type.IsNull() || state.Type.IsUnknown() {
		state.Type = types.StringValue(policy.Type)
	}
	state.Unit = types.StringValue(policy.Unit)
	state.Value = types.Float64Value(policy.Value)
	state.Status = types.StringValue(policy.Status)

	if !preserveRequiresReplace || state.Conditions.IsNull() || state.Conditions.IsUnknown() {
		if policy.Conditions != nil {
			if s, err := canonicalJSON(policy.Conditions); err == nil {
				state.Conditions = jsontypes.NewNormalizedValue(s)
			}
		}
	}

	if !preserveRequiresReplace || state.GroupBy.IsNull() || state.GroupBy.IsUnknown() {
		if policy.GroupBy != nil {
			if s, err := canonicalJSON(policy.GroupBy); err == nil {
				state.GroupBy = jsontypes.NewNormalizedValue(s)
			}
		}
	}

	state.CreatedAt = types.StringValue(policy.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !policy.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(policy.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}
}
