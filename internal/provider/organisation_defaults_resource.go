package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// organisationDefaultsSingletonID is the fixed state ID for
// portkey_organisation_defaults. GET/PUT /admin/organisation/defaults are
// scoped implicitly to the organisation owning the configured Admin API key,
// so there is no organisation identifier to key the resource on.
const organisationDefaultsSingletonID = "organisation_defaults"

var (
	_ resource.Resource                     = &organisationDefaultsResource{}
	_ resource.ResourceWithConfigure        = &organisationDefaultsResource{}
	_ resource.ResourceWithImportState      = &organisationDefaultsResource{}
	_ resource.ResourceWithConfigValidators = &organisationDefaultsResource{}
)

// NewOrganisationDefaultsResource returns a factory for portkey_organisation_defaults.
func NewOrganisationDefaultsResource() resource.Resource {
	return &organisationDefaultsResource{}
}

type organisationDefaultsResource struct {
	client *client.Client
}

// organisationDefaultsResourceModel deliberately has no organisation_id: the
// Admin API derives the organisation from the API key, so exactly one
// portkey_organisation_defaults may exist per provider configuration.
// Declaring two blocks is a config bug — the last apply wins.
type organisationDefaultsResourceModel struct {
	ID               types.String `tfsdk:"id"`
	InputGuardrails  types.List   `tfsdk:"input_guardrails"`
	OutputGuardrails types.List   `tfsdk:"output_guardrails"`
}

func (r *organisationDefaultsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organisation_defaults"
}

func (r *organisationDefaultsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages organisation-level default input/output guardrails for Portkey. " +
			"Every request routed through the organisation's API keys passes through the configured " +
			"guardrails. The Admin API scopes these defaults to the organisation owning the configured " +
			"API key, so exactly one portkey_organisation_defaults may exist per provider configuration. " +
			"Only organisation-scoped guardrails may be used — the API rejects workspace-scoped ones.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier. Always `organisation_defaults`, since the organisation " +
					"is derived from the configured Admin API key.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Optional+Computed: omitting the attribute means "Terraform does
			// not manage this list", so state adopts whatever the API holds.
			// Without Computed, the preserve-on-omit behaviour below would
			// write an API value against a planned null and Terraform would
			// fail with "Provider produced inconsistent result after apply".
			"input_guardrails": schema.ListAttribute{
				Description: "Guardrails applied to inbound requests across the organisation. The API " +
					"accepts guardrail IDs or slugs but returns both on read, and the provider stores " +
					"slugs, so prefer `portkey_guardrail.foo.slug` in HCL to avoid a permanent plan diff. " +
					"Only organisation-scoped guardrails (created without `workspace_id`) are accepted. " +
					"Omitting this attribute leaves any existing input guardrails untouched; " +
					"set it to `[]` to clear them.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"output_guardrails": schema.ListAttribute{
				Description: "Guardrails applied to model responses across the organisation. The API " +
					"accepts guardrail IDs or slugs but returns both on read, and the provider stores " +
					"slugs, so prefer `portkey_guardrail.foo.slug` in HCL to avoid a permanent plan diff. " +
					"Only organisation-scoped guardrails (created without `workspace_id`) are accepted. " +
					"Omitting this attribute leaves any existing output guardrails untouched; " +
					"set it to `[]` to clear them.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ConfigValidators mirrors the API's requirement that PUT
// /admin/organisation/defaults carry at least one of input_guardrails or
// output_guardrails, surfacing it at plan time instead of as a 400 on apply.
func (r *organisationDefaultsResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("input_guardrails"),
			path.MatchRoot("output_guardrails"),
		),
	}
}

func (r *organisationDefaultsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *organisationDefaultsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organisationDefaultsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config organisationDefaultsResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read from config, not plan: under Optional+Computed an omitted list is
	// unknown in the plan but null in the config. marshalGuardrailsForUpdate
	// returns nil (omit field) for both, so guardrails already attached
	// out-of-band (e.g. via the Portkey UI) are preserved unless the user
	// explicitly asked for a value.
	inRaw, gDiags := marshalGuardrailsForUpdate(ctx, config.InputGuardrails)
	resp.Diagnostics.Append(gDiags...)
	outRaw, gDiags := marshalGuardrailsForUpdate(ctx, config.OutputGuardrails)
	resp.Diagnostics.Append(gDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaults, err := r.client.UpdateOrganisationDefaults(ctx, client.UpdateOrganisationDefaultsRequest{
		InputGuardrails:  inRaw,
		OutputGuardrails: outRaw,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating organisation defaults",
			"Could not update organisation defaults: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(organisationDefaultsSingletonID)
	resp.Diagnostics.Append(applyOrganisationDefaultsFromAPI(&plan, defaults, config.InputGuardrails, config.OutputGuardrails)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *organisationDefaultsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organisationDefaultsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaults, err := r.client.GetOrganisationDefaults(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Organisation Defaults",
			"Could not read organisation defaults: "+err.Error(),
		)
		return
	}

	inFromAPI, gDiags := organisationGuardrailRefsToList(defaults.InputGuardrails)
	resp.Diagnostics.Append(gDiags...)
	outFromAPI, gDiags := organisationGuardrailRefsToList(defaults.OutputGuardrails)
	resp.Diagnostics.Append(gDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve state when both state and API agree the list is "empty" in any
	// form (null or []), so refresh doesn't flip [] to null (or vice-versa)
	// against the user's config. Real drift still surfaces.
	state.ID = types.StringValue(organisationDefaultsSingletonID)
	state.InputGuardrails = reconcileListAfterRead(state.InputGuardrails, inFromAPI)
	state.OutputGuardrails = reconcileListAfterRead(state.OutputGuardrails, outFromAPI)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *organisationDefaultsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organisationDefaultsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config organisationDefaultsResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	inRaw, gDiags := marshalGuardrailsForUpdate(ctx, config.InputGuardrails)
	resp.Diagnostics.Append(gDiags...)
	outRaw, gDiags := marshalGuardrailsForUpdate(ctx, config.OutputGuardrails)
	resp.Diagnostics.Append(gDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaults, err := r.client.UpdateOrganisationDefaults(ctx, client.UpdateOrganisationDefaultsRequest{
		InputGuardrails:  inRaw,
		OutputGuardrails: outRaw,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating organisation defaults",
			"Could not update organisation defaults: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(organisationDefaultsSingletonID)
	resp.Diagnostics.Append(applyOrganisationDefaultsFromAPI(&plan, defaults, config.InputGuardrails, config.OutputGuardrails)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete clears both guardrail lists on the organisation. There is no delete
// endpoint for organisation defaults, so removing this Terraform resource is
// equivalent to clearing them.
func (r *organisationDefaultsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organisationDefaultsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	empty := []byte("[]")
	_, err := r.client.UpdateOrganisationDefaults(ctx, client.UpdateOrganisationDefaultsRequest{
		InputGuardrails:  empty,
		OutputGuardrails: empty,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting organisation defaults",
			"Could not clear organisation defaults: "+err.Error(),
		)
	}
}

// ImportState ignores the supplied ID: the organisation comes from the Admin
// API key, so there is nothing to look up. Setting the singleton ID is enough
// for the subsequent Read to hydrate both guardrail lists.
func (r *organisationDefaultsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), organisationDefaultsSingletonID)...)
}

// applyOrganisationDefaultsFromAPI writes guardrails from the update response
// into `plan`, trusting the API but falling back to the user-supplied config
// when the API lags under eventual consistency (otherwise Terraform would
// error with "Provider produced inconsistent result after apply"). When config
// is [] we mirror an empty list so state matches the user's HCL even if the
// API happened to return nothing.
func applyOrganisationDefaultsFromAPI(plan *organisationDefaultsResourceModel, defaults *client.OrganisationDefaults, inputCfg, outputCfg types.List) diag.Diagnostics {
	var diags diag.Diagnostics
	// Capture before the assignments below overwrite them: these are the values
	// Terraform planned for the attributes, which the applied state must match.
	plannedIn, plannedOut := plan.InputGuardrails, plan.OutputGuardrails
	if defaults == nil {
		plan.InputGuardrails = coalesceListForConfig(inputCfg, types.ListNull(types.StringType), plannedIn)
		plan.OutputGuardrails = coalesceListForConfig(outputCfg, types.ListNull(types.StringType), plannedOut)
		return diags
	}
	inList, d := organisationGuardrailRefsToList(defaults.InputGuardrails)
	diags.Append(d...)
	outList, d := organisationGuardrailRefsToList(defaults.OutputGuardrails)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	plan.InputGuardrails = coalesceListForConfig(inputCfg, inList, plannedIn)
	plan.OutputGuardrails = coalesceListForConfig(outputCfg, outList, plannedOut)
	return diags
}
