package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
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
	_ resource.ResourceWithModifyPlan  = &apiKeyResource{}
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
	ID                  types.String `tfsdk:"id"`
	Key                 types.String `tfsdk:"key"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Type                types.String `tfsdk:"type"`
	SubType             types.String `tfsdk:"sub_type"`
	OrganisationID      types.String `tfsdk:"organisation_id"`
	WorkspaceID         types.String `tfsdk:"workspace_id"`
	UserID              types.String `tfsdk:"user_id"`
	Status              types.String `tfsdk:"status"`
	Scopes              types.List   `tfsdk:"scopes"`
	RateLimits          types.List   `tfsdk:"rate_limits"`
	UsageLimits         types.Object `tfsdk:"usage_limits"`
	Metadata            types.Map    `tfsdk:"metadata"`
	AlertEmails         types.List   `tfsdk:"alert_emails"`
	ConfigID            types.String `tfsdk:"config_id"`
	AllowConfigOverride types.Bool   `tfsdk:"allow_config_override"`
	ExpiresAt           types.String `tfsdk:"expires_at"`
	LastResetAt         types.String `tfsdk:"last_reset_at"`
	ResetUsage          types.Bool   `tfsdk:"reset_usage"`
	RotationPolicy      types.Object `tfsdk:"rotation_policy"`
	// RotateTrigger is a user-controlled string that triggers an on-demand key
	// rotation when its value changes during an Update. Setting it on Create has
	// no effect; the value is stored in state for future bumps.
	RotateTrigger types.String `tfsdk:"rotate_trigger"`
	// RotateTransitionPeriodMs is the optional transition window (in ms) sent
	// with the next on-demand rotation. Only consulted when rotate_trigger fires.
	RotateTransitionPeriodMs types.Int64 `tfsdk:"rotate_transition_period_ms"`
	// KeyTransitionExpiresAt is the API-returned cut-off for the previous key
	// after the most recent on-demand rotation. Null until a rotation occurs.
	KeyTransitionExpiresAt types.String `tfsdk:"key_transition_expires_at"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *apiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

// ModifyPlan enforces the cross-attribute rule that allow_config_override=true
// requires config_id to be present. Running this in ModifyPlan means the error
// surfaces during `terraform plan`, before any API call is made.
//
// For Create: config_id must be present in the HCL config.
// For Update: config_id can be omitted from HCL when it is already bound on the
// key (Computed preserves it in state); we fall back to the state value in that case.
// Skip when either attribute is Unknown (expression not yet resolved at plan time).
func (r *apiKeyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to validate on destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	var config apiKeyResourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read prior state once (only non-null during Update; null on Create).
	var priorState apiKeyResourceModel
	hasPriorState := !req.State.Raw.IsNull()
	if hasPriorState {
		diags = req.State.Get(ctx, &priorState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// -------------------------------------------------------------------------
	// 1. allow_config_override = true requires config_id.
	//    Runs independently of all other checks below.
	// -------------------------------------------------------------------------
	if !config.AllowConfigOverride.IsNull() && !config.AllowConfigOverride.IsUnknown() &&
		config.AllowConfigOverride.ValueBool() {
		// If config_id is non-null but Unknown (e.g., config_id = portkey_config.test.id
		// in the same plan), the user has provided it — the value is just not resolved yet.
		// Skip; Terraform re-runs ModifyPlan at apply time once the value is known.
		if config.ConfigID.IsNull() || (!config.ConfigID.IsNull() && !config.ConfigID.IsUnknown()) {
			// Determine the effective config_id:
			//   1. Start with the value already bound on the key (prior state, for Update).
			//   2. Override with the HCL value when it is explicitly set and known.
			effectiveConfigID := ""
			if hasPriorState {
				effectiveConfigID = priorState.ConfigID.ValueString()
			}
			if !config.ConfigID.IsNull() && !config.ConfigID.IsUnknown() {
				effectiveConfigID = config.ConfigID.ValueString()
			}
			if effectiveConfigID == "" {
				resp.Diagnostics.AddAttributeError(
					path.Root("allow_config_override"),
					"Invalid Attribute Combination",
					"allow_config_override can only be set to true when config_id is also specified.",
				)
			}
		}
	}

	// -------------------------------------------------------------------------
	// 2. rotation_policy: treat an object where every field is null as "unset".
	//    Prevents sending {} to the API, which would cause a confusing API error.
	// -------------------------------------------------------------------------
	if !config.RotationPolicy.IsNull() && !config.RotationPolicy.IsUnknown() {
		attrs := config.RotationPolicy.Attributes()
		rotationPeriod := attrs["rotation_period"]
		nextRotationAt := attrs["next_rotation_at"]
		keyTransitionMs := attrs["key_transition_period_ms"]

		allNull := (rotationPeriod == nil || rotationPeriod.IsNull()) &&
			(nextRotationAt == nil || nextRotationAt.IsNull()) &&
			(keyTransitionMs == nil || keyTransitionMs.IsNull())

		if allNull {
			resp.Diagnostics.AddAttributeError(
				path.Root("rotation_policy"),
				"Empty rotation_policy Block",
				"rotation_policy was set but all its fields are null. "+
					"Provide at least one of rotation_period, next_rotation_at, or key_transition_period_ms, "+
					"or remove the rotation_policy block entirely.",
			)
		}
	}

	// -------------------------------------------------------------------------
	// 3. usage_limits: credit_limit is required when the block is present;
	//    periodic_reset and periodic_reset_days are mutually exclusive.
	//    Skip individual attribute checks when the value is Unknown (unresolved
	//    expression) — validation will re-run at apply time once values are known.
	// -------------------------------------------------------------------------
	if !config.UsageLimits.IsNull() && !config.UsageLimits.IsUnknown() {
		attrs := config.UsageLimits.Attributes()

		creditLimit, hasCL := attrs["credit_limit"]
		// Only enforce when credit_limit is known-null (deliberately omitted), not Unknown.
		if !hasCL || (creditLimit.IsNull() && !creditLimit.IsUnknown()) {
			resp.Diagnostics.AddAttributeError(
				path.Root("usage_limits").AtName("credit_limit"),
				"Missing Required Attribute",
				"credit_limit is required when usage_limits is configured.",
			)
		}

		periodicReset := attrs["periodic_reset"]
		periodicResetDays := attrs["periodic_reset_days"]
		periodicResetSet := periodicReset != nil && !periodicReset.IsNull() && !periodicReset.IsUnknown()
		periodicResetDaysSet := periodicResetDays != nil && !periodicResetDays.IsNull() && !periodicResetDays.IsUnknown()
		if periodicResetSet && periodicResetDaysSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("usage_limits").AtName("periodic_reset_days"),
				"Conflicting Attributes",
				"periodic_reset and periodic_reset_days are mutually exclusive. Set one or the other, not both.",
			)
		}
	}

	// -------------------------------------------------------------------------
	// 4. expires_at: validate it is a non-empty RFC3339 datetime when explicitly set.
	// -------------------------------------------------------------------------
	if !config.ExpiresAt.IsNull() && !config.ExpiresAt.IsUnknown() {
		val := config.ExpiresAt.ValueString()
		if _, err := time.Parse(time.RFC3339, val); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("expires_at"),
				"Invalid RFC3339 Datetime",
				fmt.Sprintf("expires_at must be a valid RFC3339 datetime (e.g. \"2026-12-31T23:59:59Z\"), got: %q", val),
			)
		}
	}

	// -------------------------------------------------------------------------
	// 5. rotation_policy: rotation_period and next_rotation_at are mutually
	//    exclusive per the API spec. Also validate next_rotation_at is RFC3339.
	// -------------------------------------------------------------------------
	if !config.RotationPolicy.IsNull() && !config.RotationPolicy.IsUnknown() {
		attrs := config.RotationPolicy.Attributes()

		rotationPeriod := attrs["rotation_period"]
		nextRotationAt := attrs["next_rotation_at"]

		rotationPeriodSet := rotationPeriod != nil && !rotationPeriod.IsNull() && !rotationPeriod.IsUnknown()
		nextRotationAtSet := nextRotationAt != nil && !nextRotationAt.IsNull() && !nextRotationAt.IsUnknown()

		if rotationPeriodSet && nextRotationAtSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("rotation_policy").AtName("next_rotation_at"),
				"Conflicting Attributes",
				"rotation_period and next_rotation_at are mutually exclusive. Set one or the other, not both.",
			)
		}

		if nextRotationAtSet {
			if strVal, ok := nextRotationAt.(types.String); ok {
				if _, err := time.Parse(time.RFC3339, strVal.ValueString()); err != nil {
					resp.Diagnostics.AddAttributeError(
						path.Root("rotation_policy").AtName("next_rotation_at"),
						"Invalid RFC3339 Datetime",
						fmt.Sprintf("next_rotation_at must be a valid RFC3339 datetime (e.g. \"2026-06-01T00:00:00Z\"), got: %q", strVal.ValueString()),
					)
				}
			}
		}
	}

	// -------------------------------------------------------------------------
	// 6. usage_limits: alert_threshold must not exceed credit_limit.
	// -------------------------------------------------------------------------
	if !config.UsageLimits.IsNull() && !config.UsageLimits.IsUnknown() {
		attrs := config.UsageLimits.Attributes()

		creditLimitAttr := attrs["credit_limit"]
		alertThresholdAttr := attrs["alert_threshold"]

		creditLimitKnown := creditLimitAttr != nil && !creditLimitAttr.IsNull() && !creditLimitAttr.IsUnknown()
		alertThresholdKnown := alertThresholdAttr != nil && !alertThresholdAttr.IsNull() && !alertThresholdAttr.IsUnknown()

		if creditLimitKnown && alertThresholdKnown {
			if clVal, ok := creditLimitAttr.(types.Int64); ok {
				if atVal, ok := alertThresholdAttr.(types.Int64); ok {
					if atVal.ValueInt64() > clVal.ValueInt64() {
						resp.Diagnostics.AddAttributeError(
							path.Root("usage_limits").AtName("alert_threshold"),
							"Invalid Attribute Value",
							fmt.Sprintf("alert_threshold (%d) must be less than or equal to credit_limit (%d).",
								atVal.ValueInt64(), clVal.ValueInt64()),
						)
					}
				}
			}
		}
	}

	// -------------------------------------------------------------------------
	// 7. reset_usage: write-only trigger (Optional only, not Computed).
	//    Only mark last_reset_at / updated_at Unknown when the reset is truly
	//    new — i.e. config=true but the prior state did NOT already have true.
	//    During a no-op non-refresh plan (config=true, state=true after apply),
	//    we must NOT set Unknown or Terraform reports "non-refresh plan not empty".
	// -------------------------------------------------------------------------
	resetTriggered := !config.ResetUsage.IsNull() && !config.ResetUsage.IsUnknown() && config.ResetUsage.ValueBool()
	if resetTriggered && hasPriorState &&
		!priorState.ResetUsage.IsNull() && priorState.ResetUsage.ValueBool() {
		// Reset was already in state from a previous apply — not a new trigger.
		resetTriggered = false
	}
	if resetTriggered {
		resp.Plan.SetAttribute(ctx, path.Root("last_reset_at"), types.StringUnknown())
	}

	// -------------------------------------------------------------------------
	// 8a. config_id clearing: when the user removes config_id from HCL while
	//     the prior state holds a bound config UUID, explicitly null the plan.
	//     Without this, UseStateForUnknown would copy the old UUID into the plan,
	//     but the provider will return null after clearing → "inconsistent result".
	// -------------------------------------------------------------------------
	if hasPriorState && config.ConfigID.IsNull() &&
		!priorState.ConfigID.IsNull() && priorState.ConfigID.ValueString() != "" {
		resp.Plan.SetAttribute(ctx, path.Root("config_id"), types.StringNull())
	}

	// -------------------------------------------------------------------------
	// 8b. allow_config_override clearing: when the user removes the attribute
	//     from HCL while prior state was non-null, explicitly null the plan so
	//     the Update handler can write null without an "inconsistent result".
	// -------------------------------------------------------------------------
	if hasPriorState && config.AllowConfigOverride.IsNull() &&
		!priorState.AllowConfigOverride.IsNull() {
		resp.Plan.SetAttribute(ctx, path.Root("allow_config_override"), types.BoolNull())
	}

	// -------------------------------------------------------------------------
	// 9. updated_at + rotation_policy.next_rotation_at consistency.
	//
	//    The TF Framework keeps the prior Computed value in the plan for Updates.
	//    If the API's write timestamp differs from that prior value (e.g. the
	//    apply spans a second boundary), Terraform reports "inconsistent result".
	//
	//    Detection strategy: compare user-controlled scalar attributes (name,
	//    scopes, config_id …) and rotation_policy user sub-fields individually
	//    to detect a true change — without false-positives from Computed sub-
	//    fields (type, next_rotation_at, …) that live in state but not in HCL.
	//
	//    When a change is detected:
	//      • Mark updated_at Unknown → provider can return any new timestamp.
	//      • Mark rotation_policy.next_rotation_at Unknown → provider can return
	//        the API-recomputed date without causing an inconsistency.
	// -------------------------------------------------------------------------
	if hasPriorState {
		// allow_config_override: compare "effectively true" rather than raw value
		// to avoid false positives when the API returns nil for an explicit
		// false (config = false, state = null → both effectively not-true).
		configACO := !config.AllowConfigOverride.IsNull() && !config.AllowConfigOverride.IsUnknown() && config.AllowConfigOverride.ValueBool()
		priorACO := !priorState.AllowConfigOverride.IsNull() && !priorState.AllowConfigOverride.IsUnknown() && priorState.AllowConfigOverride.ValueBool()

		// Scalar / simple Optional attribute diff.
		// resetTriggered is already computed above (outside this block).
		scalarChange := !config.Name.Equal(priorState.Name) ||
			!config.Scopes.Equal(priorState.Scopes) ||
			!config.ConfigID.Equal(priorState.ConfigID) ||
			configACO != priorACO ||
			!config.ExpiresAt.Equal(priorState.ExpiresAt) ||
			// Null-vs-non-null transition for collections / nested blocks.
			(config.UsageLimits.IsNull() != priorState.UsageLimits.IsNull()) ||
			(config.RateLimits.IsNull() != priorState.RateLimits.IsNull()) ||
			(config.Metadata.IsNull() != priorState.Metadata.IsNull()) ||
			(config.AlertEmails.IsNull() != priorState.AlertEmails.IsNull()) ||
			resetTriggered

		// rotation_policy user-controlled sub-field diff (avoids false positives
		// from the Computed next_rotation_at in state vs null in config).
		var rotationChange bool
		if config.RotationPolicy.IsNull() != priorState.RotationPolicy.IsNull() {
			rotationChange = true
		} else if !config.RotationPolicy.IsNull() && !priorState.RotationPolicy.IsNull() {
			cAttrs := config.RotationPolicy.Attributes()
			pAttrs := priorState.RotationPolicy.Attributes()
			rotationChange = !cAttrs["rotation_period"].Equal(pAttrs["rotation_period"]) ||
				!cAttrs["key_transition_period_ms"].Equal(pAttrs["key_transition_period_ms"])
		}

		// -------------------------------------------------------------------------
		// 10. rotate_trigger: bumping the trigger value calls the /rotate endpoint
		//     during Update, which returns a new `key` and a fresh
		//     `key_transition_expires_at`. Mark those Computed attributes Unknown
		//     so the post-apply state can carry the API-returned values without
		//     producing an "inconsistent result" error.
		//
		//     Detection rule (Update only — Create never rotates):
		//       • The trigger value changed AND the new value is non-null.
		//       • A null→value transition fires rotation (user explicitly
		//         opting in to rotate). A value→null transition does not.
		// -------------------------------------------------------------------------
		rotateTriggered := !config.RotateTrigger.Equal(priorState.RotateTrigger) &&
			!config.RotateTrigger.IsNull() && !config.RotateTrigger.IsUnknown()

		if scalarChange || rotationChange || rotateTriggered {
			resp.Plan.SetAttribute(ctx, path.Root("updated_at"), types.StringUnknown())
		}
		if rotationChange {
			// next_rotation_at is recomputed by the API on every rotation_policy
			// change. Mark it Unknown so the actual API-returned date is accepted.
			resp.Plan.SetAttribute(ctx, path.Root("rotation_policy").AtName("next_rotation_at"), types.StringUnknown())
		}
		if rotateTriggered {
			// key and key_transition_expires_at are both refreshed by the rotate
			// call. UseStateForUnknown would otherwise carry the stale prior values.
			resp.Plan.SetAttribute(ctx, path.Root("key"), types.StringUnknown())
			resp.Plan.SetAttribute(ctx, path.Root("key_transition_expires_at"), types.StringUnknown())
		}
	}
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
				Description: "Human-readable name for the API key (max 100 characters).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
				},
			},
			"description": schema.StringAttribute{
				Description: "Optional description of the API key (max 200 characters).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
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
					"type": schema.StringAttribute{
						Description: "Limit type: 'tokens' to cap on token consumption, 'cost' to cap on dollar spend.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("tokens", "cost"),
						},
					},
					"credit_limit": schema.Int64Attribute{
						Description: "The credit limit value (required when usage_limits is set). Interpreted as USD when type is 'cost', or token count when type is 'tokens'. Minimum 0.",
						Optional:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
						},
					},
					"alert_threshold": schema.Int64Attribute{
						Description: "Send alert emails when usage reaches this value. Must be ≤ credit_limit. Minimum 0.",
						Optional:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
						},
					},
					"periodic_reset": schema.StringAttribute{
						Description: "When to reset the usage counter: 'monthly' or 'weekly'.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("monthly", "weekly"),
						},
					},
					"periodic_reset_days": schema.Int64Attribute{
						Description: "Custom reset interval in days (1–365). Alternative to periodic_reset for non-standard cadences.",
						Optional:    true,
						Validators: []validator.Int64{
							int64validator.Between(1, 365),
						},
					},
					"next_usage_reset_at": schema.StringAttribute{
						Description: "ISO8601 datetime for the next scheduled usage reset. Optional override; computed by the API when periodic_reset is set.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
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
							Description: "Rate limit unit: 'rpm' (per minute), 'rph' (per hour), 'rpd' (per day), 'rps' (per second), 'rpw' (per week).",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("rpm", "rph", "rpd", "rps", "rpw"),
							},
						},
						"value": schema.Int64Attribute{
							Description: "The rate limit value. Minimum 0.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.AtLeast(0),
							},
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
			"config_id": schema.StringAttribute{
				Description: "ID of the Portkey config to bind as the default config for all requests made using this API key. " +
					"When set, every request using this key will apply this config by default. " +
					"Removing this field from your Terraform config sends null to the API, clearing the binding.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"allow_config_override": schema.BoolAttribute{
				Description: "Controls whether callers using this API key can override the bound config_id at request time. " +
					"Only meaningful when config_id is set. " +
					"When false, the pinned config cannot be overridden at request time. " +
					"When true (API default), per-request config overrides are allowed. " +
					"Optional. Once set, clearing this field in Terraform will not remove it from the API — use the Portkey API directly to unset it.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"expires_at": schema.StringAttribute{
				Description: "RFC3339 datetime when this API key expires (e.g. \"2026-12-31T23:59:59Z\"). " +
					"Can be set on create or updated later. Omit to create a non-expiring key. " +
					"Note: clearing this field in Terraform after it has been set will not remove the expiry from the API — use the Portkey API directly to unset it.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"last_reset_at": schema.StringAttribute{
				Description: "Timestamp when this API key's usage counters were last reset. Managed by the API; read-only.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"reset_usage": schema.BoolAttribute{
				Description: "Set to true to trigger an immediate reset of this key's usage counters. " +
					"The value is stored in state as-is; Terraform will trigger a reset on every apply " +
					"as long as reset_usage = true remains in your configuration. " +
					"Remove or set to false to stop triggering resets.",
				Optional: true,
			},
			"rotation_policy": schema.SingleNestedAttribute{
				Description: "Automatic key rotation policy. When set, Portkey will rotate this key on the configured schedule. " +
					"Removing this block from your Terraform config will not clear the policy from the API — use the Portkey API directly to unset it.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"rotation_period": schema.StringAttribute{
						Description: "How often to rotate the key: 'weekly' or 'monthly'.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("weekly", "monthly"),
						},
					},
					"next_rotation_at": schema.StringAttribute{
						Description: "RFC3339 datetime of the next scheduled rotation (e.g. \"2026-06-01T00:00:00Z\"). " +
							"Mutually exclusive with rotation_period. Computed by the API when rotation_period is set.",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"key_transition_period_ms": schema.Int64Attribute{
						Description: "How long (in milliseconds) both the old and new keys remain valid after rotation. Minimum 1800000 (30 minutes).",
						Optional:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(1800000),
						},
					},
				},
			},
			"rotate_trigger": schema.StringAttribute{
				Description: "User-controlled trigger string for on-demand key rotation (`POST /api-keys/{id}/rotate`). " +
					"Setting this on a new resource has no effect — the value is recorded in state. " +
					"Changing it on an existing resource (any new string, e.g. a timestamp, version tag, or hash) " +
					"calls the rotate endpoint during the next apply: a new key value is issued, the old key " +
					"remains valid for the transition period, and `key` / `key_transition_expires_at` are refreshed. " +
					"WARNING: do NOT bind this to a value that changes every plan (e.g. `timestamp()`); " +
					"that would rotate the key on every apply and may break consumers still holding the previous value.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"rotate_transition_period_ms": schema.Int64Attribute{
				Description: "Optional transition period (in milliseconds) applied the next time `rotate_trigger` fires. " +
					"Determines how long the previous key value keeps working after an on-demand rotation. " +
					"Minimum 1800000 (30 minutes), per the Portkey API spec. " +
					"Independent of `rotation_policy.key_transition_period_ms`, which governs scheduled rotations.",
				Optional: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1800000),
				},
			},
			"key_transition_expires_at": schema.StringAttribute{
				Description: "Timestamp (RFC3339) at which the previous key value stops being accepted after the most recent " +
					"on-demand rotation. Populated only after `rotate_trigger` has fired at least once; null otherwise.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

	// Handle config_id
	if !plan.ConfigID.IsNull() && !plan.ConfigID.IsUnknown() {
		if createReq.Defaults == nil {
			createReq.Defaults = &client.APIKeyDefaults{}
		}
		createReq.Defaults.ConfigID = plan.ConfigID.ValueString()
	}

	// Handle allow_config_override
	if !plan.AllowConfigOverride.IsNull() && !plan.AllowConfigOverride.IsUnknown() {
		if createReq.Defaults == nil {
			createReq.Defaults = &client.APIKeyDefaults{}
		}
		v := plan.AllowConfigOverride.ValueBool()
		createReq.Defaults.AllowConfigOverride = &v
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

	// Handle expires_at
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		createReq.ExpiresAt = plan.ExpiresAt.ValueString()
	}

	// Handle rotation_policy
	if !plan.RotationPolicy.IsNull() && !plan.RotationPolicy.IsUnknown() {
		createReq.RotationPolicy = terraformToAPIKeyRotationPolicy(plan.RotationPolicy)
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

	// Handle scopes from API — always normalize to avoid Unknown in state.
	if len(apiKey.Scopes) > 0 {
		scopesList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.Scopes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Scopes = scopesList
	} else {
		plan.Scopes = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Handle usage_limits from API
	ulObj, ulDiags := apiKeyUsageLimitsToTerraform(apiKey.UsageLimits)
	resp.Diagnostics.Append(ulDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.UsageLimits = ulObj

	// Handle rate_limits from API.
	// Apply the same nil-vs-empty guard as the Read handler: when the API returns
	// [] and the plan had null/unknown (user never configured rate_limits), store
	// null so that subsequent anyChange comparisons don't see a spurious diff
	// (null in config vs [] in state → false positive → updated_at = (known after apply)).
	if apiKey.RateLimits == nil {
		plan.RateLimits = types.ListNull(apiKeyRateLimitsObjectType)
	} else if len(apiKey.RateLimits) == 0 {
		if plan.RateLimits.IsNull() || plan.RateLimits.IsUnknown() {
			plan.RateLimits = types.ListNull(apiKeyRateLimitsObjectType)
		} else {
			plan.RateLimits = types.ListValueMust(apiKeyRateLimitsObjectType, []attr.Value{})
		}
	} else {
		rlList, rlDiags := apiKeyRateLimitsToTerraformList(apiKey.RateLimits)
		resp.Diagnostics.Append(rlDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.RateLimits = rlList
	}

	// Handle metadata from API.
	// For Optional (non-Computed) attributes the post-apply state must match the
	// plan exactly. Only read back from the API when the user actually configured
	// metadata; otherwise keep plan's null to avoid an "inconsistent result" error
	// when the API returns {} for a key whose metadata was never set.
	if !plan.Metadata.IsNull() {
		switch {
		case apiKey.Defaults == nil || apiKey.Defaults.Metadata == nil:
			plan.Metadata = types.MapNull(types.StringType)
		case len(apiKey.Defaults.Metadata) == 0:
			plan.Metadata = types.MapValueMust(types.StringType, map[string]attr.Value{})
		default:
			metadataMap, diags := types.MapValueFrom(ctx, types.StringType, apiKey.Defaults.Metadata)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			plan.Metadata = metadataMap
		}
	}

	// Handle config_id from API
	if apiKey.Defaults != nil && apiKey.Defaults.ConfigID != "" {
		plan.ConfigID = types.StringValue(apiKey.Defaults.ConfigID)
	} else {
		plan.ConfigID = types.StringNull()
	}

	// Handle allow_config_override from API.
	// Because the attribute is Optional+Computed, the plan value is Unknown when
	// the user omitted it (UseStateForUnknown has nothing to copy on Create).
	// Only sync from the API response when the user explicitly provided a value
	// (non-null AND non-unknown). When Unknown or null, store null so the next
	// non-refresh plan sees null==null and does not report a spurious diff.
	if !plan.AllowConfigOverride.IsNull() && !plan.AllowConfigOverride.IsUnknown() {
		if apiKey.AllowConfigOverride != nil {
			plan.AllowConfigOverride = types.BoolValue(*apiKey.AllowConfigOverride != 0)
		} else {
			plan.AllowConfigOverride = types.BoolNull()
		}
	} else {
		// User did not configure it (omitted or Unknown) → store null.
		plan.AllowConfigOverride = types.BoolNull()
	}

	// Handle expires_at from API
	if apiKey.ExpiresAt != nil {
		plan.ExpiresAt = types.StringValue(apiKey.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		plan.ExpiresAt = types.StringNull()
	}

	// Handle last_reset_at from API (read-only)
	if apiKey.LastResetAt != nil {
		plan.LastResetAt = types.StringValue(apiKey.LastResetAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		plan.LastResetAt = types.StringNull()
	}

	// Handle rotation_policy from API
	rpObj, rpDiags := apiKeyRotationPolicyToTerraform(apiKey.RotationPolicy)
	resp.Diagnostics.Append(rpDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.RotationPolicy = rpObj

	// Handle alert_emails from API.
	// Same Optional-field rule as metadata: only read from the API response when
	// the user configured alert_emails; keep null otherwise.
	if !plan.AlertEmails.IsNull() {
		switch {
		case apiKey.AlertEmails == nil:
			plan.AlertEmails = types.ListNull(types.StringType)
		case len(apiKey.AlertEmails) == 0:
			plan.AlertEmails = types.ListValueMust(types.StringType, []attr.Value{})
		default:
			alertEmailsList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.AlertEmails)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			plan.AlertEmails = alertEmailsList
		}
	}

	// reset_usage is a write-only trigger not returned by the API.
	// plan.ResetUsage already holds the config value from req.Plan.Get above —
	// keep it so the post-apply state matches the plan (avoids "inconsistent result").

	// rotate_trigger and rotate_transition_period_ms already hold the config
	// values from req.Plan.Get above. They are user-driven inputs; Create never
	// triggers a rotation (the key is fresh).
	// key_transition_expires_at: Computed with no rotation yet → must be null.
	plan.KeyTransitionExpiresAt = types.StringNull()

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

	// Handle scopes — always normalize to avoid Unknown in state.
	if len(apiKey.Scopes) > 0 {
		scopesList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.Scopes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Scopes = scopesList
	} else {
		state.Scopes = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Handle usage_limits from API.
	// When prior state has usage_limits, merge: preserve user-configured Optional
	// fields from prior state (avoids stale API data overwriting values just
	// applied) and only update the server-computed next_usage_reset_at from the
	// API response. When prior state is null (Create, Import, or just-cleared),
	// use the full API value.
	if apiKey.UsageLimits != nil {
		if !state.UsageLimits.IsNull() && !state.UsageLimits.IsUnknown() {
			priorUL := terraformToAPIKeyUsageLimits(state.UsageLimits)
			mergedUL := &client.UsageLimits{
				Type:              priorUL.Type,
				CreditLimit:       priorUL.CreditLimit,
				AlertThreshold:    priorUL.AlertThreshold,
				PeriodicReset:     priorUL.PeriodicReset,
				PeriodicResetDays: priorUL.PeriodicResetDays,
				NextUsageResetAt:  apiKey.UsageLimits.NextUsageResetAt,
			}
			ulObj, ulDiags := apiKeyUsageLimitsToTerraform(mergedUL)
			resp.Diagnostics.Append(ulDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			state.UsageLimits = ulObj
		} else {
			ulObj, ulDiags := apiKeyUsageLimitsToTerraform(apiKey.UsageLimits)
			resp.Diagnostics.Append(ulDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			state.UsageLimits = ulObj
		}
	} else {
		state.UsageLimits = types.ObjectNull(apiKeyUsageLimitsAttrTypes)
	}

	// Handle rate_limits from API.
	// Distinguish nil (field absent → null) from non-nil empty (API returns [] for
	// every key regardless of whether rate_limits was ever configured).
	// When the API returns [] and the prior state was null (user never configured
	// rate_limits), keep null to prevent a perpetual "empty list vs null" diff
	// and to avoid plan inconsistency in subsequent update steps.
	if apiKey.RateLimits == nil {
		state.RateLimits = types.ListNull(apiKeyRateLimitsObjectType)
	} else if len(apiKey.RateLimits) == 0 {
		// API returned empty rate_limits. If the prior state was null, preserve null.
		if !state.RateLimits.IsNull() {
			state.RateLimits = types.ListValueMust(apiKeyRateLimitsObjectType, []attr.Value{})
		}
		// else: state.RateLimits stays null (prior state was null → keep it null)
	} else {
		rlList, rlDiags := apiKeyRateLimitsToTerraformList(apiKey.RateLimits)
		resp.Diagnostics.Append(rlDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.RateLimits = rlList
	}

	// Handle metadata from API.
	// Distinguish nil (field absent → null) from non-nil empty (API returns {} for
	// every key regardless of whether metadata was ever configured).
	// When the API returns {} and the prior state was null (user never configured
	// metadata), keep null to prevent a perpetual "empty map vs null" diff.
	switch {
	case apiKey.Defaults == nil || apiKey.Defaults.Metadata == nil:
		state.Metadata = types.MapNull(types.StringType)
	case len(apiKey.Defaults.Metadata) == 0:
		// API returned empty metadata. If the prior state was null (user never
		// configured metadata), preserve null to avoid a perpetual diff where the
		// plan sees config=null vs state={}.
		if !state.Metadata.IsNull() {
			state.Metadata = types.MapValueMust(types.StringType, map[string]attr.Value{})
		}
		// else: state.Metadata stays null (prior state was null → keep it null)
	default:
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, apiKey.Defaults.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Metadata = metadataMap
	}

	// Handle config_id from API
	if apiKey.Defaults != nil && apiKey.Defaults.ConfigID != "" {
		state.ConfigID = types.StringValue(apiKey.Defaults.ConfigID)
	} else {
		state.ConfigID = types.StringNull()
	}

	// Handle allow_config_override from API.
	// The API returns this as a top-level integer (null/0/1), not inside defaults.
	// Only sync from the API when the prior state already tracked a value (i.e. the
	// user explicitly configured it via Terraform). When the prior state is null
	// (user never set it), the API may return a server-side default (0 or 1) that
	// we should not silently import — doing so would cause ImportStateVerify
	// mismatches and unexpected state drift for resources the user didn't configure.
	if !state.AllowConfigOverride.IsNull() {
		// User previously configured this attribute → always sync from API.
		if apiKey.AllowConfigOverride != nil {
			state.AllowConfigOverride = types.BoolValue(*apiKey.AllowConfigOverride != 0)
		} else {
			state.AllowConfigOverride = types.BoolNull()
		}
	}
	// else: prior state was null → keep null, regardless of the API value.

	// Handle expires_at from API
	if apiKey.ExpiresAt != nil {
		state.ExpiresAt = types.StringValue(apiKey.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		state.ExpiresAt = types.StringNull()
	}

	// Handle last_reset_at from API (read-only)
	if apiKey.LastResetAt != nil {
		state.LastResetAt = types.StringValue(apiKey.LastResetAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		state.LastResetAt = types.StringNull()
	}

	// Handle rotation_policy from API
	rpObj, rpDiags := apiKeyRotationPolicyToTerraform(apiKey.RotationPolicy)
	resp.Diagnostics.Append(rpDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.RotationPolicy = rpObj

	// Handle alert_emails from API.
	// Distinguish nil (field absent → null) from non-nil empty (API returns [] for
	// every key regardless of whether alert_emails was ever configured).
	// When the API returns [] and the prior state was null (user never configured
	// alert_emails), keep null to prevent a perpetual "empty list vs null" diff.
	switch {
	case apiKey.AlertEmails == nil:
		state.AlertEmails = types.ListNull(types.StringType)
	case len(apiKey.AlertEmails) == 0:
		// API returned empty alert_emails. If the prior state was null (user never
		// configured alert_emails), preserve null to avoid a perpetual diff.
		if !state.AlertEmails.IsNull() {
			state.AlertEmails = types.ListValueMust(types.StringType, []attr.Value{})
		}
		// else: state.AlertEmails stays null (prior state was null → keep it null)
	default:
		alertEmailsList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.AlertEmails)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.AlertEmails = alertEmailsList
	}

	state.CreatedAt = types.StringValue(apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !apiKey.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(apiKey.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// reset_usage is a write-only trigger not returned by the API.
	// Preserve whatever was in prior state (null or true) — the API never
	// returns this field, so Read must not change it. If the user keeps
	// reset_usage = true in HCL and Read zeroes it out, Terraform would see
	// a diff (config=true, state=null) on every refresh → non-empty plan.
	// The value only transitions true→null when the user removes it from HCL.

	// rotate_trigger, rotate_transition_period_ms, and key_transition_expires_at
	// are NOT returned by GET /api-keys/{id}. They are managed locally:
	//   - rotate_trigger / rotate_transition_period_ms: user inputs, preserved as-is.
	//   - key_transition_expires_at: populated only by RotateAPIKey responses.
	// Leaving the existing state values untouched ensures Read does not zero them.

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

	// Read raw config to detect when user removed Optional+Computed attributes.
	// Plan values for these are Unknown (not Null), so config is the reliable signal.
	var config apiKeyResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request
	updateReq := client.UpdateAPIKeyRequest{
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
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
			updateReq.Defaults = &client.UpdateAPIKeyDefaults{}
		}
		updateReq.Defaults.Metadata = metadata
	}

	// Handle config_id and allow_config_override.
	//
	// config_id uses three-state json.RawMessage in UpdateAPIKeyDefaults:
	//   - nil         → field omitted → no change to existing binding
	//   - JSONNull    → "config_id": null → removes the pinned config
	//   - JSON string → sets a new config UUID
	//
	// Because config_id has no UseStateForUnknown, config.ConfigID is null when
	// the user removes the field from HCL, which is an explicit intent to clear it.
	if !config.ConfigID.IsNull() && !config.ConfigID.IsUnknown() {
		// User explicitly set a config UUID.
		if updateReq.Defaults == nil {
			updateReq.Defaults = &client.UpdateAPIKeyDefaults{}
		}
		data, err := json.Marshal(config.ConfigID.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("config_id"),
				"Error Encoding config_id",
				"Could not encode config_id: "+err.Error())
			return
		}
		updateReq.Defaults.ConfigID = data
	} else if !state.ConfigID.IsNull() && state.ConfigID.ValueString() != "" {
		// User removed config_id from HCL while state had a value → send null to clear.
		if updateReq.Defaults == nil {
			updateReq.Defaults = &client.UpdateAPIKeyDefaults{}
		}
		updateReq.Defaults.ConfigID = client.JSONNull
	}
	// else: both config and state are null → no config_id entry in defaults

	// allow_config_override
	if !config.AllowConfigOverride.IsNull() && !config.AllowConfigOverride.IsUnknown() {
		// User explicitly set a new value. Use *bool so explicit false is sent.
		if updateReq.Defaults == nil {
			updateReq.Defaults = &client.UpdateAPIKeyDefaults{}
		}
		v := config.AllowConfigOverride.ValueBool()
		updateReq.Defaults.AllowConfigOverride = &v
	} else if !state.AllowConfigOverride.IsNull() && !state.AllowConfigOverride.IsUnknown() {
		// User omitted it — preserve the existing server-side value.
		if updateReq.Defaults == nil {
			updateReq.Defaults = &client.UpdateAPIKeyDefaults{}
		}
		v := state.AllowConfigOverride.ValueBool()
		updateReq.Defaults.AllowConfigOverride = &v
	}

	// Handle usage_limits: use config (not plan) to detect user intent.
	// Config is null when user removed the block; plan would be Unknown.
	updateReq.UsageLimits = marshalAPIKeyUsageLimitsForUpdate(config.UsageLimits)

	// Handle rate_limits — same approach
	updateReq.RateLimits = marshalAPIKeyRateLimitsForUpdate(config.RateLimits)

	// Handle rotation_policy
	if !config.RotationPolicy.IsNull() && !config.RotationPolicy.IsUnknown() {
		updateReq.RotationPolicy = terraformToAPIKeyRotationPolicy(config.RotationPolicy)
	}

	// Handle expires_at: send an updated value when the user changes it in HCL.
	// When the user omits it (config is null) and there is an existing value in state,
	// we leave updateReq.ExpiresAt nil so the API preserves the server-side value.
	// To clear an expiry, use the Portkey API directly (Terraform cannot distinguish
	// "user set null" from "user omitted the field" for Optional+Computed attributes).
	if !config.ExpiresAt.IsNull() && !config.ExpiresAt.IsUnknown() {
		// User explicitly set a new expiry datetime.
		// ModifyPlan already validates RFC3339 format, but guard here too for safety.
		val := config.ExpiresAt.ValueString()
		if _, err := time.Parse(time.RFC3339, val); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("expires_at"),
				"Invalid RFC3339 Datetime",
				fmt.Sprintf("expires_at must be a valid RFC3339 datetime (e.g. \"2026-12-31T23:59:59Z\"), got: %q", val),
			)
			return
		}
		data, err := json.Marshal(val)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("expires_at"),
				"Error Encoding expires_at",
				fmt.Sprintf("Could not encode expires_at value %q: %s", val, err),
			)
			return
		}
		updateReq.ExpiresAt = data
	}

	// Handle reset_usage: if the user set it to true, trigger a usage counter reset.
	// This is a write-only trigger — the API does not return this field, so it is
	// always stored as null in state after apply.
	if !config.ResetUsage.IsNull() && !config.ResetUsage.IsUnknown() && config.ResetUsage.ValueBool() {
		v := true
		updateReq.ResetUsage = &v
	}

	// Handle alert_emails using three-state json.RawMessage semantics:
	//   - config null, state null   → nil    → no change (omit field)
	//   - config null, state non-null → JSONNull → send null to clear all emails
	//   - config non-null            → marshal → send the new list (including empty [])
	if !config.AlertEmails.IsNull() && !config.AlertEmails.IsUnknown() {
		var alertEmails []string
		diags = config.AlertEmails.ElementsAs(ctx, &alertEmails, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data, err := json.Marshal(alertEmails)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("alert_emails"),
				"Error Encoding alert_emails",
				"Could not encode alert_emails: "+err.Error())
			return
		}
		updateReq.AlertEmails = data
	} else if !state.AlertEmails.IsNull() {
		// User removed alert_emails from HCL while state had values → send null to clear.
		updateReq.AlertEmails = client.JSONNull
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
	} else {
		plan.UpdatedAt = state.UpdatedAt
	}

	// Handle scopes from API — always normalize to avoid Unknown in state.
	if len(apiKey.Scopes) > 0 {
		scopesList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.Scopes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Scopes = scopesList
	} else {
		plan.Scopes = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Handle usage_limits: if we sent null to clear, trust that (API has
	// eventual consistency and may return stale data). Otherwise build the result
	// by merging: config values for all user-set fields (avoids stale API response
	// giving back old values like type), and the computed next_usage_reset_at from
	// the API response.
	if config.UsageLimits.IsNull() {
		plan.UsageLimits = types.ObjectNull(apiKeyUsageLimitsAttrTypes)
	} else {
		configUL := terraformToAPIKeyUsageLimits(config.UsageLimits)
		if configUL != nil {
			// Fill in the server-computed field from the API response when the user
			// did not explicitly supply it in config.
			if apiKey.UsageLimits != nil && configUL.NextUsageResetAt == "" {
				configUL.NextUsageResetAt = apiKey.UsageLimits.NextUsageResetAt
			}
		}
		ulObj, ulDiags := apiKeyUsageLimitsToTerraform(configUL)
		resp.Diagnostics.Append(ulDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.UsageLimits = ulObj
	}

	// Handle rate_limits — same approach
	if config.RateLimits.IsNull() {
		plan.RateLimits = types.ListNull(apiKeyRateLimitsObjectType)
	} else {
		rlList, rlDiags := apiKeyRateLimitsToTerraformList(apiKey.RateLimits)
		resp.Diagnostics.Append(rlDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.RateLimits = rlList
	}

	// Handle metadata from API.
	// metadata is Optional (not Computed): the plan always holds exactly what the
	// user wrote in config (null if omitted). Only read back from the API when the
	// user actually configured metadata; otherwise keep the plan's null to avoid an
	// "inconsistent result after apply" error (UpdateAPIKey can return {} even when
	// metadata was never set).
	if !plan.Metadata.IsNull() {
		switch {
		case apiKey.Defaults == nil || apiKey.Defaults.Metadata == nil:
			plan.Metadata = types.MapNull(types.StringType)
		case len(apiKey.Defaults.Metadata) == 0:
			plan.Metadata = types.MapValueMust(types.StringType, map[string]attr.Value{})
		default:
			metadataMap, diags := types.MapValueFrom(ctx, types.StringType, apiKey.Defaults.Metadata)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			plan.Metadata = metadataMap
		}
	}

	// Handle config_id from API — always reflect the actual API state so that
	// Computed correctly tracks the value when the user has not set it in HCL.
	if apiKey.Defaults != nil && apiKey.Defaults.ConfigID != "" {
		plan.ConfigID = types.StringValue(apiKey.Defaults.ConfigID)
	} else {
		plan.ConfigID = types.StringNull()
	}

	// Handle allow_config_override from API.
	// Only sync from the API response when the user still actively manages the
	// attribute (config is non-null). When the user has dropped it from HCL
	// (config is null), store null so the next plan comparison sees null==null
	// and does not produce a spurious "updated_at = (known after apply)" diff.
	if !config.AllowConfigOverride.IsNull() {
		if apiKey.AllowConfigOverride != nil {
			plan.AllowConfigOverride = types.BoolValue(*apiKey.AllowConfigOverride != 0)
		} else {
			plan.AllowConfigOverride = types.BoolNull()
		}
	} else {
		// User removed allow_config_override from HCL — stop tracking it.
		plan.AllowConfigOverride = types.BoolNull()
	}

	// Handle expires_at from API
	if apiKey.ExpiresAt != nil {
		plan.ExpiresAt = types.StringValue(apiKey.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		plan.ExpiresAt = types.StringNull()
	}

	// Handle last_reset_at from API (read-only; updated when reset_usage was triggered)
	if apiKey.LastResetAt != nil {
		plan.LastResetAt = types.StringValue(apiKey.LastResetAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		plan.LastResetAt = types.StringNull()
	}

	// reset_usage is not returned by the API. Keep whatever the plan had
	// (i.e. the config value) so the post-apply state matches the plan.
	// plan.ResetUsage is already set from req.Plan.Get above.

	// Handle rotation_policy from API.
	// Only read back from the API when the user has rotation_policy in their config.
	// When config is null, UseStateForUnknown already set the plan to the prior state
	// value; overwriting it here with the API response would cause an inconsistent result.
	// When config IS present and rotation_policy changed, ModifyPlan already marked
	// next_rotation_at Unknown, so returning the API-recomputed date is accepted.
	if !config.RotationPolicy.IsNull() {
		rpObj, rpDiags := apiKeyRotationPolicyToTerraform(apiKey.RotationPolicy)
		resp.Diagnostics.Append(rpDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.RotationPolicy = rpObj
	}
	// else: plan.RotationPolicy retains the UseStateForUnknown value (prior state).

	// Handle alert_emails from API.
	// alert_emails is Optional (not Computed): only read back from the API when the
	// user configured it; otherwise preserve the plan's null so the post-apply
	// state matches the plan (UpdateAPIKey can return [] even when never set).
	if !plan.AlertEmails.IsNull() {
		switch {
		case apiKey.AlertEmails == nil:
			plan.AlertEmails = types.ListNull(types.StringType)
		case len(apiKey.AlertEmails) == 0:
			plan.AlertEmails = types.ListValueMust(types.StringType, []attr.Value{})
		default:
			alertEmailsList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.AlertEmails)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			plan.AlertEmails = alertEmailsList
		}
	}

	// -------------------------------------------------------------------------
	// On-demand rotation (POST /api-keys/{id}/rotate).
	//
	// Fires when the user-controlled rotate_trigger string changes between the
	// prior state and the new config (null→value also counts; value→null does
	// not). Mirrors the detection used in ModifyPlan so the Computed attributes
	// it marked Unknown — `key`, `key_transition_expires_at`, `updated_at` —
	// match what we set here.
	//
	// Order: any field-level UpdateAPIKey changes are applied first (above);
	// the rotation runs last so the new key value and transition timestamp
	// returned by /rotate become the final state.
	// -------------------------------------------------------------------------
	rotateTriggered := !config.RotateTrigger.Equal(state.RotateTrigger) &&
		!config.RotateTrigger.IsNull() && !config.RotateTrigger.IsUnknown()

	if rotateTriggered {
		var rotateReq *client.RotateAPIKeyRequest
		if !plan.RotateTransitionPeriodMs.IsNull() && !plan.RotateTransitionPeriodMs.IsUnknown() {
			v := int(plan.RotateTransitionPeriodMs.ValueInt64())
			rotateReq = &client.RotateAPIKeyRequest{KeyTransitionPeriodMs: &v}
		}

		rotateResp, err := r.client.RotateAPIKey(ctx, state.ID.ValueString(), rotateReq)
		if err != nil {
			detail := fmt.Sprintf("Could not rotate API key %s: %s", state.ID.ValueString(), err.Error())
			// 403 from this endpoint almost always means the calling Admin
			// API key is missing the matching `*_api_keys.update` scope.
			// Surface that hint directly so the user doesn't have to dig.
			if strings.Contains(err.Error(), "403") {
				detail += "\n\nHint: rotating a key requires the calling Admin API Key to have " +
					"the matching update scope (organisation_service_api_keys.update, " +
					"workspace_service_api_keys.update, or workspace_user_api_keys.update). " +
					"Update the Admin Key's scopes in the Portkey dashboard and re-apply."
			}
			resp.Diagnostics.AddError("Error Rotating Portkey API Key", detail)
			return
		}

		plan.Key = types.StringValue(rotateResp.Key)
		if rotateResp.KeyTransitionExpiresAt != "" {
			plan.KeyTransitionExpiresAt = types.StringValue(rotateResp.KeyTransitionExpiresAt)
		} else {
			// API may omit the field when no transition period is in effect.
			plan.KeyTransitionExpiresAt = types.StringNull()
		}

		// The /rotate response does not include the updated last_updated_at
		// timestamp. Re-fetch the API key so updated_at reflects the rotation.
		// Without this, Terraform's ModifyPlan marked updated_at Unknown but the
		// post-apply value would carry the pre-rotation timestamp from
		// UpdateAPIKey above, producing user-visible drift on the next refresh.
		refreshed, refreshErr := r.client.GetAPIKey(ctx, state.ID.ValueString())
		if refreshErr != nil {
			resp.Diagnostics.AddError(
				"Error Reading Portkey API Key after rotation",
				fmt.Sprintf("API key %s rotated successfully, but refreshing it failed: %s",
					state.ID.ValueString(), refreshErr.Error()),
			)
			return
		}
		if !refreshed.UpdatedAt.IsZero() {
			plan.UpdatedAt = types.StringValue(refreshed.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
		}
	} else {
		// No rotation: preserve the prior key_transition_expires_at value from
		// state. plan.KeyTransitionExpiresAt currently holds the
		// UseStateForUnknown carry-over, but make the intent explicit so future
		// edits don't accidentally null it out.
		plan.KeyTransitionExpiresAt = state.KeyTransitionExpiresAt
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
