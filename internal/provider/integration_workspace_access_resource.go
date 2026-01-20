package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &integrationWorkspaceAccessResource{}
	_ resource.ResourceWithConfigure   = &integrationWorkspaceAccessResource{}
	_ resource.ResourceWithImportState = &integrationWorkspaceAccessResource{}
)

// Type definitions for nested attributes
var (
	usageLimitsAttrTypes = map[string]attr.Type{
		"type":            types.StringType,
		"credit_limit":    types.Int64Type,
		"alert_threshold": types.Int64Type,
		"periodic_reset":  types.StringType,
	}

	rateLimitsAttrTypes = map[string]attr.Type{
		"type":  types.StringType,
		"unit":  types.StringType,
		"value": types.Int64Type,
	}

	usageLimitsObjectType = types.ObjectType{AttrTypes: usageLimitsAttrTypes}
	rateLimitsObjectType  = types.ObjectType{AttrTypes: rateLimitsAttrTypes}
)

// NewIntegrationWorkspaceAccessResource is a helper function to simplify the provider implementation.
func NewIntegrationWorkspaceAccessResource() resource.Resource {
	return &integrationWorkspaceAccessResource{}
}

// integrationWorkspaceAccessResource is the resource implementation.
type integrationWorkspaceAccessResource struct {
	client *client.Client
}

// integrationWorkspaceAccessResourceModel maps the resource schema data.
type integrationWorkspaceAccessResourceModel struct {
	ID            types.String `tfsdk:"id"`
	IntegrationID types.String `tfsdk:"integration_id"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	UsageLimits   types.List   `tfsdk:"usage_limits"`
	RateLimits    types.List   `tfsdk:"rate_limits"`
}

// usageLimitsModel maps the usage_limits block
type integrationWorkspaceUsageLimitsModel struct {
	Type           types.String `tfsdk:"type"`
	CreditLimit    types.Int64  `tfsdk:"credit_limit"`
	AlertThreshold types.Int64  `tfsdk:"alert_threshold"`
	PeriodicReset  types.String `tfsdk:"periodic_reset"`
}

// rateLimitsModel maps the rate_limits block
type integrationWorkspaceRateLimitsModel struct {
	Type  types.String `tfsdk:"type"`
	Unit  types.String `tfsdk:"unit"`
	Value types.Int64  `tfsdk:"value"`
}

// Metadata returns the resource type name.
func (r *integrationWorkspaceAccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_workspace_access"
}

// Schema defines the schema for the resource.
func (r *integrationWorkspaceAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages workspace access for a Portkey integration. Enables an integration to be used within a specific workspace, optionally with usage and rate limits.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier in format integration_id/workspace_id.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"integration_id": schema.StringAttribute{
				Description: "The integration slug or ID to grant workspace access to.",
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
				Description: "Whether the integration is enabled for this workspace. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
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
							Description: "Alert threshold percentage (0-100).",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.Between(0, 100),
							},
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
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *integrationWorkspaceAccessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *integrationWorkspaceAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan integrationWorkspaceAccessResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the workspace update request
	workspaceReq, diags := buildWorkspaceUpdateRequest(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create/update workspace access
	err := r.client.UpdateIntegrationWorkspace(ctx, plan.IntegrationID.ValueString(), workspaceReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating integration workspace access",
			"Could not create integration workspace access: "+err.Error(),
		)
		return
	}

	// Set the ID
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", plan.IntegrationID.ValueString(), plan.WorkspaceID.ValueString()))

	// Fetch the actual state from API to ensure consistency
	workspace, err := r.client.GetIntegrationWorkspace(ctx, plan.IntegrationID.ValueString(), plan.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading integration workspace access after creation",
			"Could not read integration workspace access: "+err.Error(),
		)
		return
	}

	// Update plan with actual values from API
	plan.Enabled = types.BoolValue(workspace.Enabled)
	plan.UsageLimits, diags = usageLimitsToTerraformList(workspace.UsageLimits)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.RateLimits, diags = rateLimitsToTerraformList(workspace.RateLimits)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *integrationWorkspaceAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state integrationWorkspaceAccessResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed workspace access from Portkey
	workspace, err := r.client.GetIntegrationWorkspace(ctx, state.IntegrationID.ValueString(), state.WorkspaceID.ValueString())
	if err != nil {
		// Check if this is a "not found" error - if so, remove from state
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading integration workspace access",
			"Could not read integration workspace access for workspace "+state.WorkspaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state with refreshed values
	state.Enabled = types.BoolValue(workspace.Enabled)

	// Map usage limits
	state.UsageLimits, diags = usageLimitsToTerraformList(workspace.UsageLimits)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map rate limits
	state.RateLimits, diags = rateLimitsToTerraformList(workspace.RateLimits)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *integrationWorkspaceAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan integrationWorkspaceAccessResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the workspace update request
	workspaceReq, diags := buildWorkspaceUpdateRequest(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update workspace access
	err := r.client.UpdateIntegrationWorkspace(ctx, plan.IntegrationID.ValueString(), workspaceReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating integration workspace access",
			"Could not update integration workspace access: "+err.Error(),
		)
		return
	}

	// Fetch the actual state from API to ensure consistency
	workspace, err := r.client.GetIntegrationWorkspace(ctx, plan.IntegrationID.ValueString(), plan.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading integration workspace access after update",
			"Could not read integration workspace access: "+err.Error(),
		)
		return
	}

	// Update plan with actual values from API
	plan.Enabled = types.BoolValue(workspace.Enabled)
	plan.UsageLimits, diags = usageLimitsToTerraformList(workspace.UsageLimits)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.RateLimits, diags = rateLimitsToTerraformList(workspace.RateLimits)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *integrationWorkspaceAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state integrationWorkspaceAccessResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if resource still exists before attempting to disable
	_, err := r.client.GetIntegrationWorkspace(ctx, state.IntegrationID.ValueString(), state.WorkspaceID.ValueString())
	if err != nil {
		// If not found, resource is already gone - success
		if strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting integration workspace access",
			"Could not verify integration workspace access exists: "+err.Error(),
		)
		return
	}

	// Disable workspace access (set enabled=false)
	workspaceReq := client.WorkspaceUpdateRequest{
		ID:      state.WorkspaceID.ValueString(),
		Enabled: false,
	}

	err = r.client.UpdateIntegrationWorkspace(ctx, state.IntegrationID.ValueString(), workspaceReq)
	if err != nil {
		// If not found during disable, resource is already gone - success
		if strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting integration workspace access",
			"Could not disable integration workspace access: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *integrationWorkspaceAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID should be in format: integration_id/workspace_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in format: integration_id/workspace_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("integration_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// Helper functions

// buildWorkspaceUpdateRequest builds a client.WorkspaceUpdateRequest from the resource model
func buildWorkspaceUpdateRequest(ctx context.Context, plan *integrationWorkspaceAccessResourceModel) (client.WorkspaceUpdateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	workspaceReq := client.WorkspaceUpdateRequest{
		ID:      plan.WorkspaceID.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
	}

	// Parse usage limits
	if !plan.UsageLimits.IsNull() && !plan.UsageLimits.IsUnknown() {
		var usageLimits []integrationWorkspaceUsageLimitsModel
		diags.Append(plan.UsageLimits.ElementsAs(ctx, &usageLimits, false)...)
		if diags.HasError() {
			return workspaceReq, diags
		}

		for _, ul := range usageLimits {
			clientUL := client.IntegrationWorkspaceUsageLimits{
				Type:          ul.Type.ValueString(),
				PeriodicReset: ul.PeriodicReset.ValueString(),
			}
			if !ul.CreditLimit.IsNull() {
				v := int(ul.CreditLimit.ValueInt64())
				clientUL.CreditLimit = &v
			}
			if !ul.AlertThreshold.IsNull() {
				v := int(ul.AlertThreshold.ValueInt64())
				clientUL.AlertThreshold = &v
			}
			workspaceReq.UsageLimits = append(workspaceReq.UsageLimits, clientUL)
		}
	}

	// Parse rate limits
	if !plan.RateLimits.IsNull() && !plan.RateLimits.IsUnknown() {
		var rateLimits []integrationWorkspaceRateLimitsModel
		diags.Append(plan.RateLimits.ElementsAs(ctx, &rateLimits, false)...)
		if diags.HasError() {
			return workspaceReq, diags
		}

		for _, rl := range rateLimits {
			clientRL := client.IntegrationWorkspaceRateLimits{
				Type: rl.Type.ValueString(),
				Unit: rl.Unit.ValueString(),
			}
			if !rl.Value.IsNull() {
				v := int(rl.Value.ValueInt64())
				clientRL.Value = &v
			}
			workspaceReq.RateLimits = append(workspaceReq.RateLimits, clientRL)
		}
	}

	return workspaceReq, diags
}

// usageLimitsToTerraformList converts client usage limits to a Terraform list
func usageLimitsToTerraformList(limits []client.IntegrationWorkspaceUsageLimits) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(limits) == 0 {
		// Return empty list instead of null to avoid perpetual diffs
		return types.ListValueMust(usageLimitsObjectType, []attr.Value{}), diags
	}

	usageLimitsAttrs := make([]attr.Value, 0, len(limits))
	for _, ul := range limits {
		attrs := map[string]attr.Value{
			"type":            types.StringValue(ul.Type),
			"periodic_reset":  types.StringValue(ul.PeriodicReset),
			"credit_limit":    types.Int64Null(),
			"alert_threshold": types.Int64Null(),
		}
		if ul.CreditLimit != nil {
			attrs["credit_limit"] = types.Int64Value(int64(*ul.CreditLimit))
		}
		if ul.AlertThreshold != nil {
			attrs["alert_threshold"] = types.Int64Value(int64(*ul.AlertThreshold))
		}

		objVal, d := types.ObjectValue(usageLimitsAttrTypes, attrs)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(usageLimitsObjectType), diags
		}
		usageLimitsAttrs = append(usageLimitsAttrs, objVal)
	}

	list, d := types.ListValue(usageLimitsObjectType, usageLimitsAttrs)
	diags.Append(d...)
	return list, diags
}

// rateLimitsToTerraformList converts client rate limits to a Terraform list
func rateLimitsToTerraformList(limits []client.IntegrationWorkspaceRateLimits) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(limits) == 0 {
		// Return empty list instead of null to avoid perpetual diffs
		return types.ListValueMust(rateLimitsObjectType, []attr.Value{}), diags
	}

	rateLimitsAttrs := make([]attr.Value, 0, len(limits))
	for _, rl := range limits {
		attrs := map[string]attr.Value{
			"type":  types.StringValue(rl.Type),
			"unit":  types.StringValue(rl.Unit),
			"value": types.Int64Null(),
		}
		if rl.Value != nil {
			attrs["value"] = types.Int64Value(int64(*rl.Value))
		}

		objVal, d := types.ObjectValue(rateLimitsAttrTypes, attrs)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(rateLimitsObjectType), diags
		}
		rateLimitsAttrs = append(rateLimitsAttrs, objVal)
	}

	list, d := types.ListValue(rateLimitsObjectType, rateLimitsAttrs)
	diags.Append(d...)
	return list, diags
}
