package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

var (
	_ resource.Resource                = &workspaceDefaultsResource{}
	_ resource.ResourceWithConfigure   = &workspaceDefaultsResource{}
	_ resource.ResourceWithImportState = &workspaceDefaultsResource{}
)

// NewWorkspaceDefaultsResource returns a factory for portkey_workspace_defaults.
func NewWorkspaceDefaultsResource() resource.Resource {
	return &workspaceDefaultsResource{}
}

type workspaceDefaultsResource struct {
	client *client.Client
}

// workspaceDefaultsResourceModel is keyed by workspace_id. The Portkey Admin
// API models workspace defaults as fields on the workspace itself, so this
// resource is a "manages-a-slice-of-the-parent" pattern: exactly one
// portkey_workspace_defaults may exist per workspace. Writing two blocks
// against the same workspace_id is a config bug — the last apply wins.
type workspaceDefaultsResourceModel struct {
	ID               types.String `tfsdk:"id"`
	WorkspaceID      types.String `tfsdk:"workspace_id"`
	InputGuardrails  types.List   `tfsdk:"input_guardrails"`
	OutputGuardrails types.List   `tfsdk:"output_guardrails"`
}

func (r *workspaceDefaultsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_defaults"
}

func (r *workspaceDefaultsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages default input/output guardrails for a Portkey workspace. " +
			"Kept separate from portkey_workspace because the Portkey Admin API requires guardrails to " +
			"live in the target workspace, which would otherwise create a Terraform DAG cycle. " +
			"Exactly one portkey_workspace_defaults may exist per workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier. Equal to workspace_id.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schema.StringAttribute{
				Description: "ID or slug of the workspace to configure defaults for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			// Optional+Computed: omitting the attribute means "Terraform does
			// not manage this list", so state adopts whatever the API holds.
			// Without Computed, the preserve-on-omit behaviour below would
			// write an API value against a planned null and Terraform would
			// fail with "Provider produced inconsistent result after apply".
			"input_guardrails": schema.ListAttribute{
				Description: "Guardrails applied to inbound requests. The API accepts guardrail IDs " +
					"or slugs but returns slugs on read under admin-API-key auth, so prefer " +
					"`portkey_guardrail.foo.slug` in HCL to avoid a permanent plan diff. " +
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
				Description: "Guardrails applied to model responses. The API accepts guardrail IDs " +
					"or slugs but returns slugs on read under admin-API-key auth, so prefer " +
					"`portkey_guardrail.foo.slug` in HCL to avoid a permanent plan diff. " +
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

func (r *workspaceDefaultsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *workspaceDefaultsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workspaceDefaultsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config workspaceDefaultsResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read from config, not plan: under Optional+Computed an omitted list is
	// unknown in the plan but null in the config, and only the config
	// distinguishes "leave this alone" from "clear it". marshalGuardrailsForUpdate
	// yields nil for an omitted list, which preserveOmittedGuardrails then fills
	// in from the workspace's current value.
	updateReq := client.UpdateWorkspaceRequest{
		Defaults: &client.UpdateWorkspaceDefaults{},
	}
	inRaw, gDiags := marshalGuardrailsForUpdate(ctx, config.InputGuardrails)
	resp.Diagnostics.Append(gDiags...)
	outRaw, gDiags := marshalGuardrailsForUpdate(ctx, config.OutputGuardrails)
	resp.Diagnostics.Append(gDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	inRaw, outRaw, gDiags = r.preserveOmittedGuardrails(ctx, plan.WorkspaceID.ValueString(), inRaw, outRaw)
	resp.Diagnostics.Append(gDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateReq.Defaults.InputGuardrails = inRaw
	updateReq.Defaults.OutputGuardrails = outRaw

	workspace, err := r.client.UpdateWorkspace(ctx, plan.WorkspaceID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating workspace defaults",
			"Could not update workspace defaults: "+err.Error(),
		)
		return
	}

	// Preserve the user-provided workspace_id (may be slug or UUID). The
	// API accepts either, and rewriting to workspace.ID triggers "Provider
	// produced inconsistent result after apply" whenever the user passed
	// a slug. Mirror the same value into `id` so the two attributes stay
	// aligned for imports and Reads.
	plan.ID = plan.WorkspaceID
	resp.Diagnostics.Append(applyDefaultsFromAPI(&plan, workspace, config.InputGuardrails, config.OutputGuardrails)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *workspaceDefaultsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workspaceDefaultsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspace, err := r.client.GetWorkspace(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Parent workspace is gone. Removing this resource from state
			// lets terraform apply reconcile (either re-plan or accept the
			// deletion), mirroring workspace_resource.go Read.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Workspace Defaults",
			"Could not read workspace defaults for workspace "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	var inFromAPI, outFromAPI types.List
	var gDiags diag.Diagnostics
	if workspace.Defaults != nil {
		inFromAPI, gDiags = guardrailsFromAPIToList(workspace.Defaults.InputGuardrails)
		resp.Diagnostics.Append(gDiags...)
		outFromAPI, gDiags = guardrailsFromAPIToList(workspace.Defaults.OutputGuardrails)
		resp.Diagnostics.Append(gDiags...)
	} else {
		inFromAPI = types.ListNull(types.StringType)
		outFromAPI = types.ListNull(types.StringType)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve state when both state and API agree the list is "empty" in
	// any form (null or []). This prevents Read from flipping state.[] to
	// state.null (or vice-versa) on refresh, which would produce a
	// permanent plan diff against the user's config. When there is real
	// drift (state populated / API empty, or state empty / API populated),
	// trust the API so the diff surfaces.
	state.InputGuardrails = reconcileListAfterRead(state.InputGuardrails, inFromAPI)
	state.OutputGuardrails = reconcileListAfterRead(state.OutputGuardrails, outFromAPI)
	// Do NOT rewrite state.WorkspaceID from workspace.ID — the user may
	// have configured a slug and the API accepts either. Overwriting to
	// the UUID would create a permanent plan diff.

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *workspaceDefaultsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan workspaceDefaultsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config workspaceDefaultsResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateWorkspaceRequest{
		Defaults: &client.UpdateWorkspaceDefaults{},
	}
	inRaw, gDiags := marshalGuardrailsForUpdate(ctx, config.InputGuardrails)
	resp.Diagnostics.Append(gDiags...)
	outRaw, gDiags := marshalGuardrailsForUpdate(ctx, config.OutputGuardrails)
	resp.Diagnostics.Append(gDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	inRaw, outRaw, gDiags = r.preserveOmittedGuardrails(ctx, plan.WorkspaceID.ValueString(), inRaw, outRaw)
	resp.Diagnostics.Append(gDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateReq.Defaults.InputGuardrails = inRaw
	updateReq.Defaults.OutputGuardrails = outRaw

	workspace, err := r.client.UpdateWorkspace(ctx, plan.WorkspaceID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating workspace defaults",
			"Could not update workspace defaults: "+err.Error(),
		)
		return
	}

	// Preserve user-provided workspace_id (see Create for rationale).
	plan.ID = plan.WorkspaceID
	resp.Diagnostics.Append(applyDefaultsFromAPI(&plan, workspace, config.InputGuardrails, config.OutputGuardrails)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete clears both guardrail lists on the workspace via the update
// endpoint (there is no delete endpoint for workspace defaults, so
// removing this Terraform resource is equivalent to clearing).
func (r *workspaceDefaultsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workspaceDefaultsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	empty := []byte("[]")
	updateReq := client.UpdateWorkspaceRequest{
		Defaults: &client.UpdateWorkspaceDefaults{
			InputGuardrails:  empty,
			OutputGuardrails: empty,
		},
	}
	if _, err := r.client.UpdateWorkspace(ctx, state.ID.ValueString(), updateReq); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting workspace defaults",
			"Could not clear workspace defaults on workspace "+state.ID.ValueString()+": "+err.Error(),
		)
	}
}

// ImportState imports by workspace ID (or slug). Passthrough to `id` and
// mirror the same value into `workspace_id` so Read has both fields hydrated.
func (r *workspaceDefaultsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), req.ID)...)
}

// applyDefaultsFromAPI writes guardrails from the update response into
// `plan`, trusting the API but falling back to the user-supplied config
// when the API lags under eventual consistency (otherwise Terraform would
// error with "Provider produced inconsistent result after apply"). When
// config is [] we mirror an empty list so state matches the user's HCL
// even if the API happened to echo null.
func applyDefaultsFromAPI(plan *workspaceDefaultsResourceModel, workspace *client.Workspace, inputCfg, outputCfg types.List) diag.Diagnostics {
	var diags diag.Diagnostics
	// Capture before the assignments below overwrite them: these are the values
	// Terraform planned for the attributes, which the applied state must match.
	plannedIn, plannedOut := plan.InputGuardrails, plan.OutputGuardrails
	if workspace.Defaults == nil {
		plan.InputGuardrails = coalesceListForConfig(inputCfg, types.ListNull(types.StringType), plannedIn)
		plan.OutputGuardrails = coalesceListForConfig(outputCfg, types.ListNull(types.StringType), plannedOut)
		return diags
	}
	inList, d := guardrailsFromAPIToList(workspace.Defaults.InputGuardrails)
	diags.Append(d...)
	outList, d := guardrailsFromAPIToList(workspace.Defaults.OutputGuardrails)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	plan.InputGuardrails = coalesceListForConfig(inputCfg, inList, plannedIn)
	plan.OutputGuardrails = coalesceListForConfig(outputCfg, outList, plannedOut)
	return diags
}

// preserveOmittedGuardrails fills in guardrail lists the user did not configure
// with the workspace's current values.
//
// Unlike PUT /admin/organisation/defaults, which preserves fields absent from
// the request body, the workspace endpoint replaces the whole `defaults` object:
// a PUT carrying only input_guardrails clears output_guardrails. Sending the
// current value back unchanged is therefore the only way to make an omitted
// attribute mean "leave this alone". This mirrors how
// portkey_workspace_security_settings overlays a partial config onto the current
// server state before writing.
//
// A nil raw value means "omitted"; only those are filled in.
func (r *workspaceDefaultsResource) preserveOmittedGuardrails(ctx context.Context, workspaceID string, inRaw, outRaw json.RawMessage) (json.RawMessage, json.RawMessage, diag.Diagnostics) {
	var diags diag.Diagnostics
	if inRaw != nil && outRaw != nil {
		return inRaw, outRaw, diags
	}

	current, err := r.client.GetWorkspace(ctx, workspaceID)
	if err != nil {
		diags.AddError(
			"Error reading current workspace defaults",
			"Could not read workspace "+workspaceID+" to preserve unmanaged guardrails: "+err.Error(),
		)
		return inRaw, outRaw, diags
	}
	if current.Defaults == nil {
		return inRaw, outRaw, diags
	}

	if inRaw == nil {
		inRaw, err = json.Marshal(nonEmptyStrings(current.Defaults.InputGuardrails))
		if err != nil {
			diags.AddError("Error marshaling current input guardrails", err.Error())
			return inRaw, outRaw, diags
		}
	}
	if outRaw == nil {
		outRaw, err = json.Marshal(nonEmptyStrings(current.Defaults.OutputGuardrails))
		if err != nil {
			diags.AddError("Error marshaling current output guardrails", err.Error())
			return inRaw, outRaw, diags
		}
	}
	return inRaw, outRaw, diags
}

// nonEmptyStrings drops blank entries and normalizes nil to an empty slice so it
// marshals as [] rather than null.
func nonEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// reconcileListAfterRead reconciles prior state with the just-read API
// value. When both are "empty" (null or []), state is preserved as-is so
// refresh doesn't flip [] ↔ null against the user's config. Otherwise
// state is updated to the API value so drift surfaces in the next plan.
func reconcileListAfterRead(state, apiList types.List) types.List {
	stateEmpty := state.IsNull() || len(state.Elements()) == 0
	apiEmpty := apiList.IsNull() || len(apiList.Elements()) == 0
	if stateEmpty && apiEmpty {
		return state
	}
	return apiList
}

// coalesceListForConfig reconciles the user's config with the API response.
//   - config null → the attribute is unmanaged. Under Optional+Computed with
//     UseStateForUnknown Terraform planned the prior state for it, and the
//     applied value has to match that plan, so preserve the planned value when
//     it and the API agree the list is empty (null and [] mean the same thing
//     here, and the API reports both as null). Otherwise adopt the API list so
//     drift surfaces. `planned` is unknown on create, where there is no prior
//     state to preserve — returning it would leave an unknown value in state.
//   - config [] → force empty list (user asked to clear; state must match HCL).
//   - config populated, API null (lag) → trust config.
//   - otherwise → trust API list.
func coalesceListForConfig(cfg, apiList, planned types.List) types.List {
	if cfg.IsNull() || cfg.IsUnknown() {
		if planned.IsUnknown() {
			return apiList
		}
		return reconcileListAfterRead(planned, apiList)
	}
	if len(cfg.Elements()) == 0 {
		return types.ListValueMust(types.StringType, []attr.Value{})
	}
	if apiList.IsNull() {
		return cfg
	}
	return apiList
}
