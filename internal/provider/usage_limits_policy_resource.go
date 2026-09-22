package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	_ resource.Resource                   = &usageLimitsPolicyResource{}
	_ resource.ResourceWithConfigure      = &usageLimitsPolicyResource{}
	_ resource.ResourceWithImportState    = &usageLimitsPolicyResource{}
	_ resource.ResourceWithValidateConfig = &usageLimitsPolicyResource{}
)

// NewUsageLimitsPolicyResource is a helper function to simplify the provider implementation.
func NewUsageLimitsPolicyResource() resource.Resource {
	return &usageLimitsPolicyResource{}
}

// usageLimitsPolicyResource is the resource implementation.
type usageLimitsPolicyResource struct {
	client *client.Client
}

// usageLimitsPolicyResourceModel maps the resource schema data.
type usageLimitsPolicyResourceModel struct {
	ID                types.String         `tfsdk:"id"`
	Name              types.String         `tfsdk:"name"`
	WorkspaceID       types.String         `tfsdk:"workspace_id"`
	Conditions        jsontypes.Normalized `tfsdk:"conditions"`
	GroupBy           jsontypes.Normalized `tfsdk:"group_by"`
	Type              types.String         `tfsdk:"type"`
	CreditLimit       types.Float64        `tfsdk:"credit_limit"`
	AlertThreshold    types.Float64        `tfsdk:"alert_threshold"`
	PeriodicReset     types.String         `tfsdk:"periodic_reset"`
	PeriodicResetDays types.Int64          `tfsdk:"periodic_reset_days"`
	NextUsageResetAt  types.String         `tfsdk:"next_usage_reset_at"`
	Status            types.String         `tfsdk:"status"`
	CreatedAt         types.String         `tfsdk:"created_at"`
	UpdatedAt         types.String         `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *usageLimitsPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_usage_limits_policy"
}

// Schema defines the schema for the resource.
func (r *usageLimitsPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Portkey usage limits policy. Controls total usage (cost or tokens) over a period.",
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
				Description: "JSON array of conditions that define which requests the policy applies to. Each condition has 'key', 'value' (string or array of strings), and an optional 'excludes' (string or array of strings). Updated in place; changing it does not reset the policy's accumulated usage.",
				Required:    true,
			},
			"group_by": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: "JSON array of group by fields that define how usage is aggregated. Each item has 'key'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Policy type: 'cost' or 'tokens'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"credit_limit": schema.Float64Attribute{
				Description: "Maximum usage allowed.",
				Required:    true,
			},
			"alert_threshold": schema.Float64Attribute{
				Description: "Threshold at which to send alerts. Must be less than credit_limit.",
				Optional:    true,
			},
			"periodic_reset": schema.StringAttribute{
				Description: "Reset period: 'monthly' or 'weekly'. Mutually exclusive with periodic_reset_days. If neither is provided, the limit is cumulative.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("monthly", "weekly"),
				},
			},
			"periodic_reset_days": schema.Int64Attribute{
				Description: "Custom reset interval in days (1–365). Mutually exclusive with periodic_reset.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 365),
				},
			},
			"next_usage_reset_at": schema.StringAttribute{
				Description: "ISO8601 datetime for the next scheduled usage reset. Computed by the API when periodic_reset or periodic_reset_days is set.",
				Computed:    true,
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
func (r *usageLimitsPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig validates the resource configuration.
func (r *usageLimitsPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config usageLimitsPolicyResourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	periodicResetSet := !config.PeriodicReset.IsNull() && !config.PeriodicReset.IsUnknown()
	periodicResetDaysSet := !config.PeriodicResetDays.IsNull() && !config.PeriodicResetDays.IsUnknown()
	if periodicResetSet && periodicResetDaysSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("periodic_reset_days"),
			"Conflicting Attributes",
			"periodic_reset and periodic_reset_days are mutually exclusive. Set one or the other, not both.",
		)
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *usageLimitsPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan usageLimitsPolicyResourceModel
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
	createReq := client.CreateUsageLimitsPolicyRequest{
		Name:        plan.Name.ValueString(),
		WorkspaceID: plan.WorkspaceID.ValueString(),
		Conditions:  conditions,
		GroupBy:     groupBy,
		Type:        plan.Type.ValueString(),
		CreditLimit: plan.CreditLimit.ValueFloat64(),
	}

	if !plan.AlertThreshold.IsNull() && !plan.AlertThreshold.IsUnknown() {
		alertThreshold := plan.AlertThreshold.ValueFloat64()
		createReq.AlertThreshold = &alertThreshold
	}

	if !plan.PeriodicReset.IsNull() && !plan.PeriodicReset.IsUnknown() {
		createReq.PeriodicReset = plan.PeriodicReset.ValueString()
	}

	if !plan.PeriodicResetDays.IsNull() && !plan.PeriodicResetDays.IsUnknown() {
		v := int(plan.PeriodicResetDays.ValueInt64())
		createReq.PeriodicResetDays = &v
	}

	createResp, err := r.client.CreateUsageLimitsPolicy(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating usage limits policy",
			"Could not create policy, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch the full policy details
	policy, err := r.client.GetUsageLimitsPolicy(ctx, createResp.ID)
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
func (r *usageLimitsPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state usageLimitsPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed policy value from Portkey
	policy, err := r.client.GetUsageLimitsPolicy(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Portkey Usage Limits Policy",
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
func (r *usageLimitsPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan usageLimitsPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state usageLimitsPolicyResourceModel
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

	// Build update request. Terraform is declarative, so the mutable attributes are
	// sent on every update rather than diffed: that way removing an optional attribute
	// from the configuration actually clears it server-side. The reset cadence is the
	// exception — see below.
	updateReq := client.UpdateUsageLimitsPolicyRequest{
		Conditions: conditions,
	}

	// The API rejects a null name, so an unset name is omitted rather than cleared.
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		updateReq.Name = plan.Name.ValueString()
	}

	creditLimit := plan.CreditLimit.ValueFloat64()
	updateReq.CreditLimit = &creditLimit

	// A nil AlertThreshold marshals to an explicit null, which clears it.
	if !plan.AlertThreshold.IsNull() && !plan.AlertThreshold.IsUnknown() {
		alertThreshold := plan.AlertThreshold.ValueFloat64()
		updateReq.AlertThreshold = &alertThreshold
	}

	// periodic_reset and periodic_reset_days are sent only when they change. The API
	// recomputes next_usage_reset_at whenever either field is present in the body, and
	// for periodic_reset_days that recomputation is "now + N days" — re-sending an
	// unchanged value on every apply would keep pushing the reset date into the future
	// and the policy would never reset.
	if !plan.PeriodicReset.Equal(state.PeriodicReset) {
		updateReq.PeriodicReset = nullableJSON(plan.PeriodicReset.IsNull() || plan.PeriodicReset.IsUnknown(), plan.PeriodicReset.ValueString())
	}
	if !plan.PeriodicResetDays.Equal(state.PeriodicResetDays) {
		updateReq.PeriodicResetDays = nullableJSON(plan.PeriodicResetDays.IsNull() || plan.PeriodicResetDays.IsUnknown(), plan.PeriodicResetDays.ValueInt64())
	}

	policy, err := r.client.UpdateUsageLimitsPolicy(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Portkey Usage Limits Policy",
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
func (r *usageLimitsPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state usageLimitsPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing policy
	err := r.client.DeleteUsageLimitsPolicy(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Portkey Usage Limits Policy",
			"Could not delete policy, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *usageLimitsPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapPolicyToState maps a UsageLimitsPolicy API response to the Terraform state model
// preserveRequiresReplace, when true (the Read path), keeps the user-authored value
// from state for attributes the API may echo back in a different shape, instead of
// overwriting state from the response. Note that this also means changes made outside
// Terraform to those attributes are not detected as drift.
func (r *usageLimitsPolicyResource) mapPolicyToState(state *usageLimitsPolicyResourceModel, policy *client.UsageLimitsPolicy, preserveRequiresReplace bool) {
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
	state.CreditLimit = types.Float64Value(policy.CreditLimit)
	state.Status = types.StringValue(policy.Status)

	if policy.AlertThreshold != nil {
		state.AlertThreshold = types.Float64Value(*policy.AlertThreshold)
	} else {
		state.AlertThreshold = types.Float64Null()
	}

	// Keep the configured periodic_reset rather than the API echo
	if !preserveRequiresReplace || state.PeriodicReset.IsNull() || state.PeriodicReset.IsUnknown() {
		if policy.PeriodicReset != "" {
			state.PeriodicReset = types.StringValue(policy.PeriodicReset)
		} else {
			state.PeriodicReset = types.StringNull()
		}
	}

	// Keep the configured periodic_reset_days rather than the API echo
	if !preserveRequiresReplace || state.PeriodicResetDays.IsNull() || state.PeriodicResetDays.IsUnknown() {
		if policy.PeriodicResetDays != nil {
			state.PeriodicResetDays = types.Int64Value(int64(*policy.PeriodicResetDays))
		} else {
			state.PeriodicResetDays = types.Int64Null()
		}
	}

	if policy.NextUsageResetAt != "" {
		state.NextUsageResetAt = types.StringValue(policy.NextUsageResetAt)
	} else {
		state.NextUsageResetAt = types.StringNull()
	}

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

// nullableJSON renders v for an API field where an explicit JSON null means "clear this
// field" and an omitted field means "leave unchanged". Returning nil omits the field,
// which is the safe fallback: it leaves the server value untouched rather than clearing
// it. Only string and int64 are instantiated and encoding/json cannot fail for either,
// so the error branch is defensive.
func nullableJSON[T string | int64](isNull bool, v T) json.RawMessage {
	if isNull {
		return json.RawMessage("null")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

// canonicalJSON re-encodes v through interface{} so map keys are alphabetically sorted.
func canonicalJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	var normalized any
	if err := json.Unmarshal(b, &normalized); err != nil {
		return "", err
	}
	out, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
