package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// ============================================================================
// Workspace-format limit types (used by integration_workspace_access + workspace)
// ============================================================================

// Type definitions for workspace-format nested attributes
var (
	workspaceUsageLimitsAttrTypes = map[string]attr.Type{
		"type":            types.StringType,
		"credit_limit":    types.Int64Type,
		"alert_threshold": types.Int64Type,
		"periodic_reset":  types.StringType,
	}

	workspaceRateLimitsAttrTypes = map[string]attr.Type{
		"type":  types.StringType,
		"unit":  types.StringType,
		"value": types.Int64Type,
	}

	workspaceUsageLimitsObjectType = types.ObjectType{AttrTypes: workspaceUsageLimitsAttrTypes}
	workspaceRateLimitsObjectType  = types.ObjectType{AttrTypes: workspaceRateLimitsAttrTypes}
)

// workspaceUsageLimitsModel maps a workspace usage_limits block
type workspaceUsageLimitsModel struct {
	Type           types.String `tfsdk:"type"`
	CreditLimit    types.Int64  `tfsdk:"credit_limit"`
	AlertThreshold types.Int64  `tfsdk:"alert_threshold"`
	PeriodicReset  types.String `tfsdk:"periodic_reset"`
}

// workspaceRateLimitsModel maps a workspace rate_limits block
type workspaceRateLimitsModel struct {
	Type  types.String `tfsdk:"type"`
	Unit  types.String `tfsdk:"unit"`
	Value types.Int64  `tfsdk:"value"`
}

// buildWorkspaceLimitsFromPlan extracts usage_limits and rate_limits from a workspace plan.
// When the plan value is null (user removed the block), returns an empty non-nil slice
// so the API receives [] to clear limits (omitempty won't omit a non-nil empty slice).
// When the plan value is unknown, returns nil (omitted from JSON).
func buildWorkspaceLimitsFromPlan(ctx context.Context, plan *workspaceResourceModel) ([]client.IntegrationWorkspaceUsageLimits, []client.IntegrationWorkspaceRateLimits, diag.Diagnostics) {
	var diags diag.Diagnostics
	var usageLimits []client.IntegrationWorkspaceUsageLimits
	var rateLimits []client.IntegrationWorkspaceRateLimits

	// Handle usage_limits
	if plan.UsageLimits.IsNull() {
		// User removed usage_limits block: send empty array to clear
		usageLimits = []client.IntegrationWorkspaceUsageLimits{}
	} else if !plan.UsageLimits.IsUnknown() {
		var ulModels []workspaceUsageLimitsModel
		diags.Append(plan.UsageLimits.ElementsAs(ctx, &ulModels, false)...)
		if diags.HasError() {
			return nil, nil, diags
		}
		for _, ul := range ulModels {
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
			usageLimits = append(usageLimits, clientUL)
		}
	}

	// Handle rate_limits
	if plan.RateLimits.IsNull() {
		// User removed rate_limits block: send empty array to clear
		rateLimits = []client.IntegrationWorkspaceRateLimits{}
	} else if !plan.RateLimits.IsUnknown() {
		var rlModels []workspaceRateLimitsModel
		diags.Append(plan.RateLimits.ElementsAs(ctx, &rlModels, false)...)
		if diags.HasError() {
			return nil, nil, diags
		}
		for _, rl := range rlModels {
			clientRL := client.IntegrationWorkspaceRateLimits{
				Type: rl.Type.ValueString(),
				Unit: rl.Unit.ValueString(),
			}
			if !rl.Value.IsNull() {
				v := int(rl.Value.ValueInt64())
				clientRL.Value = &v
			}
			rateLimits = append(rateLimits, clientRL)
		}
	}

	return usageLimits, rateLimits, diags
}

// workspaceUsageLimitsToTerraformList converts client workspace usage limits to a Terraform list
func workspaceUsageLimitsToTerraformList(limits []client.IntegrationWorkspaceUsageLimits) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(limits) == 0 {
		return types.ListValueMust(workspaceUsageLimitsObjectType, []attr.Value{}), diags
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

		objVal, d := types.ObjectValue(workspaceUsageLimitsAttrTypes, attrs)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(workspaceUsageLimitsObjectType), diags
		}
		usageLimitsAttrs = append(usageLimitsAttrs, objVal)
	}

	list, d := types.ListValue(workspaceUsageLimitsObjectType, usageLimitsAttrs)
	diags.Append(d...)
	return list, diags
}

// workspaceRateLimitsToTerraformList converts client workspace rate limits to a Terraform list
func workspaceRateLimitsToTerraformList(limits []client.IntegrationWorkspaceRateLimits) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(limits) == 0 {
		return types.ListValueMust(workspaceRateLimitsObjectType, []attr.Value{}), diags
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

		objVal, d := types.ObjectValue(workspaceRateLimitsAttrTypes, attrs)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(workspaceRateLimitsObjectType), diags
		}
		rateLimitsAttrs = append(rateLimitsAttrs, objVal)
	}

	list, d := types.ListValue(workspaceRateLimitsObjectType, rateLimitsAttrs)
	diags.Append(d...)
	return list, diags
}

// ============================================================================
// API-key-format limit types (different schema from workspace limits)
// ============================================================================

// Type definitions for API key nested attributes
var (
	apiKeyUsageLimitsAttrTypes = map[string]attr.Type{
		"credit_limit":    types.Int64Type,
		"alert_threshold": types.Int64Type,
		"periodic_reset":  types.StringType,
	}

	apiKeyRateLimitsAttrTypes = map[string]attr.Type{
		"type":  types.StringType,
		"unit":  types.StringType,
		"value": types.Int64Type,
	}

	apiKeyRateLimitsObjectType = types.ObjectType{AttrTypes: apiKeyRateLimitsAttrTypes}
)

// apiKeyUsageLimitsToTerraform converts client *UsageLimits to a Terraform Object.
// Returns null object when the API returns nil (no limits configured).
func apiKeyUsageLimitsToTerraform(ul *client.UsageLimits) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	if ul == nil {
		return types.ObjectNull(apiKeyUsageLimitsAttrTypes), diags
	}

	attrs := map[string]attr.Value{
		"credit_limit":    types.Int64Null(),
		"alert_threshold": types.Int64Null(),
		"periodic_reset":  types.StringNull(),
	}
	if ul.CreditLimit != nil {
		attrs["credit_limit"] = types.Int64Value(int64(*ul.CreditLimit))
	}
	if ul.AlertThreshold != nil {
		attrs["alert_threshold"] = types.Int64Value(int64(*ul.AlertThreshold))
	}
	if ul.PeriodicReset != "" {
		attrs["periodic_reset"] = types.StringValue(ul.PeriodicReset)
	}

	obj, d := types.ObjectValue(apiKeyUsageLimitsAttrTypes, attrs)
	diags.Append(d...)
	return obj, diags
}

// terraformToAPIKeyUsageLimits converts a Terraform Object to client *UsageLimits.
// Uses safe type assertions with comma-ok pattern.
func terraformToAPIKeyUsageLimits(obj types.Object) *client.UsageLimits {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	ul := &client.UsageLimits{}

	if v, ok := obj.Attributes()["credit_limit"]; ok && !v.IsNull() && !v.IsUnknown() {
		if i64Val, ok := v.(types.Int64); ok {
			i := int(i64Val.ValueInt64())
			ul.CreditLimit = &i
		}
	}
	if v, ok := obj.Attributes()["alert_threshold"]; ok && !v.IsNull() && !v.IsUnknown() {
		if i64Val, ok := v.(types.Int64); ok {
			i := int(i64Val.ValueInt64())
			ul.AlertThreshold = &i
		}
	}
	if v, ok := obj.Attributes()["periodic_reset"]; ok && !v.IsNull() && !v.IsUnknown() {
		if strVal, ok := v.(types.String); ok {
			ul.PeriodicReset = strVal.ValueString()
		}
	}

	return ul
}

// apiKeyRateLimitsToTerraformList converts client []RateLimit to a Terraform List.
// Returns null list when the API returns nil (no limits configured), matching the
// convention for Terraform Optional+Computed list attributes.
// Note: API key RateLimit.Value is int (not *int like workspace rate limits).
func apiKeyRateLimitsToTerraformList(limits []client.RateLimit) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if limits == nil {
		return types.ListNull(apiKeyRateLimitsObjectType), diags
	}

	if len(limits) == 0 {
		return types.ListValueMust(apiKeyRateLimitsObjectType, []attr.Value{}), diags
	}

	rateLimitsAttrs := make([]attr.Value, 0, len(limits))
	for _, rl := range limits {
		attrs := map[string]attr.Value{
			"type":  types.StringValue(rl.Type),
			"unit":  types.StringValue(rl.Unit),
			"value": types.Int64Value(int64(rl.Value)),
		}

		objVal, d := types.ObjectValue(apiKeyRateLimitsAttrTypes, attrs)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(apiKeyRateLimitsObjectType), diags
		}
		rateLimitsAttrs = append(rateLimitsAttrs, objVal)
	}

	list, d := types.ListValue(apiKeyRateLimitsObjectType, rateLimitsAttrs)
	diags.Append(d...)
	return list, diags
}

// terraformToAPIKeyRateLimits converts a Terraform List to client []RateLimit.
// Uses safe type assertions with comma-ok pattern.
func terraformToAPIKeyRateLimits(list types.List) []client.RateLimit {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var rateLimits []client.RateLimit
	for _, elem := range list.Elements() {
		obj, ok := elem.(types.Object)
		if !ok {
			continue
		}
		rl := client.RateLimit{}

		if v, ok := obj.Attributes()["type"]; ok && !v.IsNull() && !v.IsUnknown() {
			if strVal, ok := v.(types.String); ok {
				rl.Type = strVal.ValueString()
			}
		}
		if v, ok := obj.Attributes()["unit"]; ok && !v.IsNull() && !v.IsUnknown() {
			if strVal, ok := v.(types.String); ok {
				rl.Unit = strVal.ValueString()
			}
		}
		if v, ok := obj.Attributes()["value"]; ok && !v.IsNull() && !v.IsUnknown() {
			if i64Val, ok := v.(types.Int64); ok {
				rl.Value = int(i64Val.ValueInt64())
			}
		}

		rateLimits = append(rateLimits, rl)
	}

	return rateLimits
}

// ============================================================================
// Integration workspace access limit helpers
// ============================================================================

// buildIntegrationWorkspaceLimitsFromPlan extracts usage_limits and rate_limits
// from an integration workspace access plan. When the plan value is null (user
// removed the block), returns an empty non-nil slice so the API receives [] to
// clear limits. When the plan value is unknown, returns nil (omitted from JSON).
func buildIntegrationWorkspaceLimitsFromPlan(ctx context.Context, plan *integrationWorkspaceAccessResourceModel) ([]client.IntegrationWorkspaceUsageLimits, []client.IntegrationWorkspaceRateLimits, diag.Diagnostics) {
	var diags diag.Diagnostics
	var usageLimits []client.IntegrationWorkspaceUsageLimits
	var rateLimits []client.IntegrationWorkspaceRateLimits

	// Handle usage_limits
	if plan.UsageLimits.IsNull() {
		// User removed usage_limits block: send empty array to clear
		usageLimits = []client.IntegrationWorkspaceUsageLimits{}
	} else if !plan.UsageLimits.IsUnknown() {
		var ulModels []workspaceUsageLimitsModel
		diags.Append(plan.UsageLimits.ElementsAs(ctx, &ulModels, false)...)
		if diags.HasError() {
			return nil, nil, diags
		}
		for _, ul := range ulModels {
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
			usageLimits = append(usageLimits, clientUL)
		}
	}

	// Handle rate_limits
	if plan.RateLimits.IsNull() {
		// User removed rate_limits block: send empty array to clear
		rateLimits = []client.IntegrationWorkspaceRateLimits{}
	} else if !plan.RateLimits.IsUnknown() {
		var rlModels []workspaceRateLimitsModel
		diags.Append(plan.RateLimits.ElementsAs(ctx, &rlModels, false)...)
		if diags.HasError() {
			return nil, nil, diags
		}
		for _, rl := range rlModels {
			clientRL := client.IntegrationWorkspaceRateLimits{
				Type: rl.Type.ValueString(),
				Unit: rl.Unit.ValueString(),
			}
			if !rl.Value.IsNull() {
				v := int(rl.Value.ValueInt64())
				clientRL.Value = &v
			}
			rateLimits = append(rateLimits, clientRL)
		}
	}

	return usageLimits, rateLimits, diags
}
