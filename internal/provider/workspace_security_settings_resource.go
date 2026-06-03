package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &workspaceSecuritySettingsResource{}
	_ resource.ResourceWithConfigure   = &workspaceSecuritySettingsResource{}
	_ resource.ResourceWithImportState = &workspaceSecuritySettingsResource{}
)

// NewWorkspaceSecuritySettingsResource is a helper function to simplify the provider implementation.
func NewWorkspaceSecuritySettingsResource() resource.Resource {
	return &workspaceSecuritySettingsResource{}
}

// workspaceSecuritySettingsResource manages the per-workspace role-permission
// bag exposed by `GET/PUT /admin/workspaces/{slug}` -> `security_settings`.
//
// The Portkey Admin API requires the FULL 35-field object on every PUT
// (sparse updates return 400 AB01). The resource therefore reads the current
// API values, overlays the user's plan, and PUTs the merged object.
type workspaceSecuritySettingsResource struct {
	client *client.Client
}

// workspaceSecuritySettingsResourceModel maps the resource schema data.
// All 35 boolean fields are Optional+Computed: users may specify any subset,
// and unspecified fields are carried forward via UseStateForUnknown.
type workspaceSecuritySettingsResourceModel struct {
	ID          types.String `tfsdk:"id"`
	WorkspaceID types.String `tfsdk:"workspace_id"`

	MembersViewLogs                   types.Bool `tfsdk:"members_view_logs"`
	ManagersUpdateWs                  types.Bool `tfsdk:"managers_update_ws"`
	ManagersViewLogs                  types.Bool `tfsdk:"managers_view_logs"`
	MembersViewAllData                types.Bool `tfsdk:"members_view_all_data"`
	MembersViewApiKeys                types.Bool `tfsdk:"members_view_api_keys"`
	MembersViewConfigs                types.Bool `tfsdk:"members_view_configs"`
	MembersViewPrompts                types.Bool `tfsdk:"members_view_prompts"`
	ManagersViewAllData               types.Bool `tfsdk:"managers_view_all_data"`
	ManagersViewApiKeys               types.Bool `tfsdk:"managers_view_api_keys"`
	ManagersViewConfigs               types.Bool `tfsdk:"managers_view_configs"`
	ManagersViewPrompts               types.Bool `tfsdk:"managers_view_prompts"`
	MembersWriteApiKeys               types.Bool `tfsdk:"members_write_api_keys"`
	MembersWriteConfigs               types.Bool `tfsdk:"members_write_configs"`
	MembersWritePrompts               types.Bool `tfsdk:"members_write_prompts"`
	ManagersWriteApiKeys              types.Bool `tfsdk:"managers_write_api_keys"`
	ManagersWriteConfigs              types.Bool `tfsdk:"managers_write_configs"`
	ManagersWritePrompts              types.Bool `tfsdk:"managers_write_prompts"`
	ManagersWriteWsUsers              types.Bool `tfsdk:"managers_write_ws_users"`
	MembersViewAnalytics              types.Bool `tfsdk:"members_view_analytics"`
	ManagersViewAnalytics             types.Bool `tfsdk:"managers_view_analytics"`
	MembersViewGuardrails             types.Bool `tfsdk:"members_view_guardrails"`
	ManagersViewGuardrails            types.Bool `tfsdk:"managers_view_guardrails"`
	MembersViewLogMetadata            types.Bool `tfsdk:"members_view_log_metadata"`
	MembersViewVirtualKeys            types.Bool `tfsdk:"members_view_virtual_keys"`
	MembersWriteGuardrails            types.Bool `tfsdk:"members_write_guardrails"`
	ManagersViewLogMetadata           types.Bool `tfsdk:"managers_view_log_metadata"`
	ManagersViewVirtualKeys           types.Bool `tfsdk:"managers_view_virtual_keys"`
	ManagersWriteGuardrails           types.Bool `tfsdk:"managers_write_guardrails"`
	ManagersWriteMcpServers           types.Bool `tfsdk:"managers_write_mcp_servers"`
	MembersWriteVirtualKeys           types.Bool `tfsdk:"members_write_virtual_keys"`
	ManagersWriteVirtualKeys          types.Bool `tfsdk:"managers_write_virtual_keys"`
	OrganisationAdminsViewLogs        types.Bool `tfsdk:"organisation_admins_view_logs"`
	ManagersWriteWsIntegrations       types.Bool `tfsdk:"managers_write_ws_integrations"`
	ManagersWriteWsMcpIntegrations    types.Bool `tfsdk:"managers_write_ws_mcp_integrations"`
	OrganisationAdminsViewLogMetadata types.Bool `tfsdk:"organisation_admins_view_log_metadata"`
}

// Metadata returns the resource type name.
func (r *workspaceSecuritySettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_security_settings"
}

// boolPermAttr returns a uniformly-configured BoolAttribute for every
// permission flag: Optional+Computed with UseStateForUnknown so users may
// specify any subset and the rest are carried forward from API/state.
func boolPermAttr(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Description: desc,
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
}

// Schema defines the schema for the resource.
func (r *workspaceSecuritySettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the role-permission flags (security_settings) on a Portkey workspace. " +
			"Every flag is Optional+Computed: omit a flag to keep its current API value. " +
			"Because the Portkey API requires the full 35-field object on every PUT, the provider " +
			"reads the current settings and overlays any user-specified values before writing. " +
			"Destroying this resource removes it from Terraform state ONLY; the underlying API " +
			"settings are left untouched (Portkey has no endpoint to reset them).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier (equals workspace_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schema.StringAttribute{
				Description: "ID (slug) of the workspace whose security settings are being managed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"members_view_logs":          boolPermAttr("Whether workspace members can view request logs."),
			"managers_update_ws":         boolPermAttr("Whether workspace managers can update workspace-level settings."),
			"managers_view_logs":         boolPermAttr("Whether workspace managers can view request logs."),
			"members_view_all_data":      boolPermAttr("Whether workspace members can view all data in the workspace (not just their own)."),
			"members_view_api_keys":      boolPermAttr("Whether workspace members can view API keys."),
			"members_view_configs":       boolPermAttr("Whether workspace members can view configs."),
			"members_view_prompts":       boolPermAttr("Whether workspace members can view prompts."),
			"managers_view_all_data":     boolPermAttr("Whether workspace managers can view all data in the workspace."),
			"managers_view_api_keys":     boolPermAttr("Whether workspace managers can view API keys."),
			"managers_view_configs":      boolPermAttr("Whether workspace managers can view configs."),
			"managers_view_prompts":      boolPermAttr("Whether workspace managers can view prompts."),
			"members_write_api_keys":     boolPermAttr("Whether workspace members can create or update API keys."),
			"members_write_configs":      boolPermAttr("Whether workspace members can create or update configs."),
			"members_write_prompts":      boolPermAttr("Whether workspace members can create or update prompts."),
			"managers_write_api_keys":    boolPermAttr("Whether workspace managers can create or update API keys."),
			"managers_write_configs":     boolPermAttr("Whether workspace managers can create or update configs."),
			"managers_write_prompts":     boolPermAttr("Whether workspace managers can create or update prompts."),
			"managers_write_ws_users":    boolPermAttr("Whether workspace managers can add, remove, or update workspace members."),
			"members_view_analytics":     boolPermAttr("Whether workspace members can view workspace analytics."),
			"managers_view_analytics":    boolPermAttr("Whether workspace managers can view workspace analytics."),
			"members_view_guardrails":    boolPermAttr("Whether workspace members can view guardrails."),
			"managers_view_guardrails":   boolPermAttr("Whether workspace managers can view guardrails."),
			"members_view_log_metadata":  boolPermAttr("Whether workspace members can view request log metadata."),
			"members_view_virtual_keys":  boolPermAttr("Whether workspace members can view virtual keys (providers)."),
			"members_write_guardrails":   boolPermAttr("Whether workspace members can create or update guardrails."),
			"managers_view_log_metadata": boolPermAttr("Whether workspace managers can view request log metadata."),
			"managers_view_virtual_keys": boolPermAttr("Whether workspace managers can view virtual keys (providers)."),
			"managers_write_guardrails":  boolPermAttr("Whether workspace managers can create or update guardrails."),
			"managers_write_mcp_servers": boolPermAttr("Whether workspace managers can create or update MCP server integrations."),
			"members_write_virtual_keys": boolPermAttr("Whether workspace members can create or update virtual keys (providers)."),
			"managers_write_virtual_keys": boolPermAttr(
				"Whether workspace managers can create or update virtual keys (providers).",
			),
			"organisation_admins_view_logs":         boolPermAttr("Whether organisation admins can view request logs in this workspace."),
			"managers_write_ws_integrations":        boolPermAttr("Whether workspace managers can manage workspace-level integrations."),
			"managers_write_ws_mcp_integrations":    boolPermAttr("Whether workspace managers can manage workspace-level MCP integrations."),
			"organisation_admins_view_log_metadata": boolPermAttr("Whether organisation admins can view request log metadata in this workspace."),
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *workspaceSecuritySettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// mergedSecuritySettings overlays any plan values that the user explicitly set
// (i.e. known and non-null) on top of the current API values. Unknown/null
// plan fields keep the current API value, which preserves untouched settings
// even though the API requires the full 35-field object on PUT.
func mergedSecuritySettings(plan workspaceSecuritySettingsResourceModel, current client.WorkspaceSecuritySettings) client.WorkspaceSecuritySettings {
	pick := func(p types.Bool, cur bool) bool {
		if !p.IsNull() && !p.IsUnknown() {
			return p.ValueBool()
		}
		return cur
	}
	return client.WorkspaceSecuritySettings{
		MembersViewLogs:                   pick(plan.MembersViewLogs, current.MembersViewLogs),
		ManagersUpdateWs:                  pick(plan.ManagersUpdateWs, current.ManagersUpdateWs),
		ManagersViewLogs:                  pick(plan.ManagersViewLogs, current.ManagersViewLogs),
		MembersViewAllData:                pick(plan.MembersViewAllData, current.MembersViewAllData),
		MembersViewApiKeys:                pick(plan.MembersViewApiKeys, current.MembersViewApiKeys),
		MembersViewConfigs:                pick(plan.MembersViewConfigs, current.MembersViewConfigs),
		MembersViewPrompts:                pick(plan.MembersViewPrompts, current.MembersViewPrompts),
		ManagersViewAllData:               pick(plan.ManagersViewAllData, current.ManagersViewAllData),
		ManagersViewApiKeys:               pick(plan.ManagersViewApiKeys, current.ManagersViewApiKeys),
		ManagersViewConfigs:               pick(plan.ManagersViewConfigs, current.ManagersViewConfigs),
		ManagersViewPrompts:               pick(plan.ManagersViewPrompts, current.ManagersViewPrompts),
		MembersWriteApiKeys:               pick(plan.MembersWriteApiKeys, current.MembersWriteApiKeys),
		MembersWriteConfigs:               pick(plan.MembersWriteConfigs, current.MembersWriteConfigs),
		MembersWritePrompts:               pick(plan.MembersWritePrompts, current.MembersWritePrompts),
		ManagersWriteApiKeys:              pick(plan.ManagersWriteApiKeys, current.ManagersWriteApiKeys),
		ManagersWriteConfigs:              pick(plan.ManagersWriteConfigs, current.ManagersWriteConfigs),
		ManagersWritePrompts:              pick(plan.ManagersWritePrompts, current.ManagersWritePrompts),
		ManagersWriteWsUsers:              pick(plan.ManagersWriteWsUsers, current.ManagersWriteWsUsers),
		MembersViewAnalytics:              pick(plan.MembersViewAnalytics, current.MembersViewAnalytics),
		ManagersViewAnalytics:             pick(plan.ManagersViewAnalytics, current.ManagersViewAnalytics),
		MembersViewGuardrails:             pick(plan.MembersViewGuardrails, current.MembersViewGuardrails),
		ManagersViewGuardrails:            pick(plan.ManagersViewGuardrails, current.ManagersViewGuardrails),
		MembersViewLogMetadata:            pick(plan.MembersViewLogMetadata, current.MembersViewLogMetadata),
		MembersViewVirtualKeys:            pick(plan.MembersViewVirtualKeys, current.MembersViewVirtualKeys),
		MembersWriteGuardrails:            pick(plan.MembersWriteGuardrails, current.MembersWriteGuardrails),
		ManagersViewLogMetadata:           pick(plan.ManagersViewLogMetadata, current.ManagersViewLogMetadata),
		ManagersViewVirtualKeys:           pick(plan.ManagersViewVirtualKeys, current.ManagersViewVirtualKeys),
		ManagersWriteGuardrails:           pick(plan.ManagersWriteGuardrails, current.ManagersWriteGuardrails),
		ManagersWriteMcpServers:           pick(plan.ManagersWriteMcpServers, current.ManagersWriteMcpServers),
		MembersWriteVirtualKeys:           pick(plan.MembersWriteVirtualKeys, current.MembersWriteVirtualKeys),
		ManagersWriteVirtualKeys:          pick(plan.ManagersWriteVirtualKeys, current.ManagersWriteVirtualKeys),
		OrganisationAdminsViewLogs:        pick(plan.OrganisationAdminsViewLogs, current.OrganisationAdminsViewLogs),
		ManagersWriteWsIntegrations:       pick(plan.ManagersWriteWsIntegrations, current.ManagersWriteWsIntegrations),
		ManagersWriteWsMcpIntegrations:    pick(plan.ManagersWriteWsMcpIntegrations, current.ManagersWriteWsMcpIntegrations),
		OrganisationAdminsViewLogMetadata: pick(plan.OrganisationAdminsViewLogMetadata, current.OrganisationAdminsViewLogMetadata),
	}
}

// populateModelFromSecuritySettings copies every wire field into the
// corresponding types.Bool on the resource model. The workspace_id / id are
// expected to be set by the caller.
func populateModelFromSecuritySettings(model *workspaceSecuritySettingsResourceModel, s client.WorkspaceSecuritySettings) {
	model.MembersViewLogs = types.BoolValue(s.MembersViewLogs)
	model.ManagersUpdateWs = types.BoolValue(s.ManagersUpdateWs)
	model.ManagersViewLogs = types.BoolValue(s.ManagersViewLogs)
	model.MembersViewAllData = types.BoolValue(s.MembersViewAllData)
	model.MembersViewApiKeys = types.BoolValue(s.MembersViewApiKeys)
	model.MembersViewConfigs = types.BoolValue(s.MembersViewConfigs)
	model.MembersViewPrompts = types.BoolValue(s.MembersViewPrompts)
	model.ManagersViewAllData = types.BoolValue(s.ManagersViewAllData)
	model.ManagersViewApiKeys = types.BoolValue(s.ManagersViewApiKeys)
	model.ManagersViewConfigs = types.BoolValue(s.ManagersViewConfigs)
	model.ManagersViewPrompts = types.BoolValue(s.ManagersViewPrompts)
	model.MembersWriteApiKeys = types.BoolValue(s.MembersWriteApiKeys)
	model.MembersWriteConfigs = types.BoolValue(s.MembersWriteConfigs)
	model.MembersWritePrompts = types.BoolValue(s.MembersWritePrompts)
	model.ManagersWriteApiKeys = types.BoolValue(s.ManagersWriteApiKeys)
	model.ManagersWriteConfigs = types.BoolValue(s.ManagersWriteConfigs)
	model.ManagersWritePrompts = types.BoolValue(s.ManagersWritePrompts)
	model.ManagersWriteWsUsers = types.BoolValue(s.ManagersWriteWsUsers)
	model.MembersViewAnalytics = types.BoolValue(s.MembersViewAnalytics)
	model.ManagersViewAnalytics = types.BoolValue(s.ManagersViewAnalytics)
	model.MembersViewGuardrails = types.BoolValue(s.MembersViewGuardrails)
	model.ManagersViewGuardrails = types.BoolValue(s.ManagersViewGuardrails)
	model.MembersViewLogMetadata = types.BoolValue(s.MembersViewLogMetadata)
	model.MembersViewVirtualKeys = types.BoolValue(s.MembersViewVirtualKeys)
	model.MembersWriteGuardrails = types.BoolValue(s.MembersWriteGuardrails)
	model.ManagersViewLogMetadata = types.BoolValue(s.ManagersViewLogMetadata)
	model.ManagersViewVirtualKeys = types.BoolValue(s.ManagersViewVirtualKeys)
	model.ManagersWriteGuardrails = types.BoolValue(s.ManagersWriteGuardrails)
	model.ManagersWriteMcpServers = types.BoolValue(s.ManagersWriteMcpServers)
	model.MembersWriteVirtualKeys = types.BoolValue(s.MembersWriteVirtualKeys)
	model.ManagersWriteVirtualKeys = types.BoolValue(s.ManagersWriteVirtualKeys)
	model.OrganisationAdminsViewLogs = types.BoolValue(s.OrganisationAdminsViewLogs)
	model.ManagersWriteWsIntegrations = types.BoolValue(s.ManagersWriteWsIntegrations)
	model.ManagersWriteWsMcpIntegrations = types.BoolValue(s.ManagersWriteWsMcpIntegrations)
	model.OrganisationAdminsViewLogMetadata = types.BoolValue(s.OrganisationAdminsViewLogMetadata)
}

// applySecuritySettings fetches the current workspace, merges the user-
// supplied plan values over the API state, and PUTs the full 35-field object.
// It returns the merged settings that were sent so the caller can populate
// state without having to round-trip through the API again (which would be
// vulnerable to eventual-consistency drift).
func (r *workspaceSecuritySettingsResource) applySecuritySettings(
	ctx context.Context,
	workspaceID string,
	plan workspaceSecuritySettingsResourceModel,
) (client.WorkspaceSecuritySettings, error) {
	currentWs, err := r.client.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return client.WorkspaceSecuritySettings{}, fmt.Errorf("fetching current workspace: %w", err)
	}

	var current client.WorkspaceSecuritySettings
	if currentWs.SecuritySettings != nil {
		current = *currentWs.SecuritySettings
	}

	merged := mergedSecuritySettings(plan, current)

	updateReq := client.UpdateWorkspaceRequest{
		SecuritySettings: &merged,
	}
	if _, err := r.client.UpdateWorkspace(ctx, workspaceID, updateReq); err != nil {
		return client.WorkspaceSecuritySettings{}, fmt.Errorf("updating security_settings: %w", err)
	}

	return merged, nil
}

// Create writes the merged security_settings and persists state.
func (r *workspaceSecuritySettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workspaceSecuritySettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := plan.WorkspaceID.ValueString()
	merged, err := r.applySecuritySettings(ctx, workspaceID, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Portkey Workspace Security Settings",
			"Could not set security_settings for workspace "+workspaceID+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(workspaceID)
	populateModelFromSecuritySettings(&plan, merged)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest API data.
func (r *workspaceSecuritySettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workspaceSecuritySettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := state.WorkspaceID.ValueString()
	ws, err := r.client.GetWorkspace(ctx, workspaceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Portkey Workspace Security Settings",
			"Could not read workspace "+workspaceID+": "+err.Error(),
		)
		return
	}

	if ws.SecuritySettings == nil {
		// API did not return security_settings for this workspace; treat as
		// drift and clear state so Terraform re-creates / re-applies.
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(workspaceID)
	populateModelFromSecuritySettings(&state, *ws.SecuritySettings)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update writes a merged 35-field object using the latest API state for any
// fields the user did not explicitly set in the plan.
func (r *workspaceSecuritySettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan workspaceSecuritySettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := plan.WorkspaceID.ValueString()
	merged, err := r.applySecuritySettings(ctx, workspaceID, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Portkey Workspace Security Settings",
			"Could not update security_settings for workspace "+workspaceID+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(workspaceID)
	populateModelFromSecuritySettings(&plan, merged)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete removes the resource from Terraform state without touching the
// underlying API. Portkey provides no endpoint to reset security_settings to
// any well-defined default, so any "delete" semantic would either be a no-op
// or destructively reset flags the user did not specify. We emit a warning
// so this is visible in plan/apply output.
func (r *workspaceSecuritySettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workspaceSecuritySettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning(
		"portkey_workspace_security_settings removed from state only",
		"The Portkey API has no endpoint to reset workspace security_settings; the "+
			"current API values for workspace "+state.WorkspaceID.ValueString()+
			" remain unchanged. To revert them, re-import this resource and apply the "+
			"desired values, or delete the workspace itself.",
	)
}

// ImportState lets users adopt existing settings by importing on workspace_id.
func (r *workspaceSecuritySettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
