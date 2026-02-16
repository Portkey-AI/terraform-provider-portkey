package provider

import (
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
		"credits_limit":      types.Float64Type,
		"credits_limit_type": types.StringType,
	}

	apiKeyRateLimitsAttrTypes = map[string]attr.Type{
		"type":  types.StringType,
		"unit":  types.StringType,
		"value": types.Int64Type,
	}

	apiKeyUsageLimitsObjectType = types.ObjectType{AttrTypes: apiKeyUsageLimitsAttrTypes}
	apiKeyRateLimitsObjectType  = types.ObjectType{AttrTypes: apiKeyRateLimitsAttrTypes}
)

// apiKeyUsageLimitsToTerraform converts client *UsageLimits to a Terraform Object
func apiKeyUsageLimitsToTerraform(ul *client.UsageLimits) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	if ul == nil {
		return types.ObjectNull(apiKeyUsageLimitsAttrTypes), diags
	}

	attrs := map[string]attr.Value{
		"credits_limit":      types.Float64Null(),
		"credits_limit_type": types.StringNull(),
	}
	if ul.CreditsLimit != nil {
		attrs["credits_limit"] = types.Float64Value(*ul.CreditsLimit)
	}
	if ul.CreditsLimitType != "" {
		attrs["credits_limit_type"] = types.StringValue(ul.CreditsLimitType)
	}

	obj, d := types.ObjectValue(apiKeyUsageLimitsAttrTypes, attrs)
	diags.Append(d...)
	return obj, diags
}

// terraformToAPIKeyUsageLimits converts a Terraform Object to client *UsageLimits
func terraformToAPIKeyUsageLimits(obj types.Object) *client.UsageLimits {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	ul := &client.UsageLimits{}

	if v, ok := obj.Attributes()["credits_limit"]; ok && !v.IsNull() && !v.IsUnknown() {
		f := v.(types.Float64).ValueFloat64()
		ul.CreditsLimit = &f
	}
	if v, ok := obj.Attributes()["credits_limit_type"]; ok && !v.IsNull() && !v.IsUnknown() {
		ul.CreditsLimitType = v.(types.String).ValueString()
	}

	return ul
}

// apiKeyRateLimitsToTerraformList converts client []RateLimit to a Terraform List
// Note: API key RateLimit.Value is int (not *int like workspace rate limits)
func apiKeyRateLimitsToTerraformList(limits []client.RateLimit) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

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

// terraformToAPIKeyRateLimits converts a Terraform List to client []RateLimit
func terraformToAPIKeyRateLimits(list types.List) []client.RateLimit {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var rateLimits []client.RateLimit
	for _, elem := range list.Elements() {
		obj := elem.(types.Object)
		rl := client.RateLimit{}

		if v, ok := obj.Attributes()["type"]; ok && !v.IsNull() && !v.IsUnknown() {
			rl.Type = v.(types.String).ValueString()
		}
		if v, ok := obj.Attributes()["unit"]; ok && !v.IsNull() && !v.IsUnknown() {
			rl.Unit = v.(types.String).ValueString()
		}
		if v, ok := obj.Attributes()["value"]; ok && !v.IsNull() && !v.IsUnknown() {
			rl.Value = int(v.(types.Int64).ValueInt64())
		}

		rateLimits = append(rateLimits, rl)
	}

	return rateLimits
}
