package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	_ resource.Resource                = &promptPartialResource{}
	_ resource.ResourceWithConfigure   = &promptPartialResource{}
	_ resource.ResourceWithImportState = &promptPartialResource{}
)

// NewPromptPartialResource is a helper function to simplify the provider implementation.
func NewPromptPartialResource() resource.Resource {
	return &promptPartialResource{}
}

// promptPartialResource is the resource implementation.
type promptPartialResource struct {
	client *client.Client
}

// promptPartialResourceModel maps the resource schema data.
type promptPartialResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Slug                   types.String `tfsdk:"slug"`
	Name                   types.String `tfsdk:"name"`
	Content                types.String `tfsdk:"content"`
	WorkspaceID            types.String `tfsdk:"workspace_id"`
	VersionDescription     types.String `tfsdk:"version_description"`
	Version                types.Int64  `tfsdk:"version"`
	PromptPartialVersionID types.String `tfsdk:"prompt_partial_version_id"`
	Status                 types.String `tfsdk:"status"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *promptPartialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt_partial"
}

// Schema defines the schema for the resource.
func (r *promptPartialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Portkey prompt partial. Prompt partials are reusable template fragments referenced in prompts via Mustache syntax ({{>partial-slug}}).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Prompt partial identifier (UUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Description: "URL-friendly identifier for the prompt partial. Auto-generated based on name.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name for the prompt partial.",
				Required:    true,
			},
			"content": schema.StringAttribute{
				Description: "The partial template content. Maps to the API 'string' field.",
				Required:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID to scope the prompt partial to. Required when using an org-level API key.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version_description": schema.StringAttribute{
				Description: "Description for the prompt partial version.",
				Optional:    true,
			},
			"version": schema.Int64Attribute{
				Description: "Current version number of the prompt partial.",
				Computed:    true,
			},
			"prompt_partial_version_id": schema.StringAttribute{
				Description: "Current version ID of the prompt partial.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Status of the prompt partial (active, archived).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the prompt partial was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the prompt partial was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *promptPartialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *promptPartialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan promptPartialResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new prompt partial
	createReq := client.CreatePromptPartialRequest{
		Name:   plan.Name.ValueString(),
		String: plan.Content.ValueString(),
	}

	if !plan.WorkspaceID.IsNull() && !plan.WorkspaceID.IsUnknown() {
		createReq.WorkspaceID = plan.WorkspaceID.ValueString()
	}

	if !plan.VersionDescription.IsNull() && !plan.VersionDescription.IsUnknown() {
		createReq.VersionDescription = plan.VersionDescription.ValueString()
	}

	createResp, err := r.client.CreatePromptPartial(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating prompt partial",
			"Could not create prompt partial, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch the full prompt partial details
	partial, err := r.client.GetPromptPartial(ctx, createResp.Slug, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading prompt partial after creation",
			"Could not read prompt partial, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema
	r.mapPartialToState(&plan, partial)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *promptPartialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state promptPartialResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch the partial from the API. mapPartialToState detects external
	// changes by comparing versions and refreshes content if needed.
	partial, err := r.client.GetPromptPartial(ctx, state.Slug.ValueString(), "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Portkey Prompt Partial",
			"Could not read Portkey prompt partial slug "+state.Slug.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to state
	r.mapPartialToState(&state, partial)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *promptPartialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan promptPartialResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state for the slug
	var state promptPartialResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check what changed
	nameChanged := plan.Name.ValueString() != state.Name.ValueString()
	contentChanged := plan.Content.ValueString() != state.Content.ValueString()
	versionDescChanged := !plan.VersionDescription.Equal(state.VersionDescription)

	// Warn if version_description changed without content — it only takes effect with new versions
	if versionDescChanged && !contentChanged {
		resp.Diagnostics.AddWarning(
			"Version Description Change Ignored",
			"The version_description was changed but content was not. Version descriptions are only applied when a new version is created (i.e., when content changes). The version_description change will be stored in state but not sent to the API.",
		)
	}

	// Build update request
	updateReq := client.UpdatePromptPartialRequest{}

	if nameChanged {
		updateReq.Name = plan.Name.ValueString()
	}

	if contentChanged {
		updateReq.String = plan.Content.ValueString()

		if !plan.VersionDescription.IsNull() && !plan.VersionDescription.IsUnknown() {
			updateReq.VersionDescription = plan.VersionDescription.ValueString()
		}
	}

	// Only call update if there are changes
	var updateResp *client.UpdatePromptPartialResponse
	var newVersion int
	if nameChanged || contentChanged {
		var err error
		updateResp, err = r.client.UpdatePromptPartial(ctx, state.Slug.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Portkey Prompt Partial",
				"Could not update prompt partial, unexpected error: "+err.Error(),
			)
			return
		}

		// If a new version was created (content changed), make it the default.
		// Look up the real version number from the versions list by matching
		// the version ID returned by Update, since the MakeDefault endpoint
		// requires a version number (not a UUID).
		if contentChanged && updateResp.PromptPartialVersionID != "" {
			versions, err := r.client.ListPromptPartialVersions(ctx, state.Slug.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error listing prompt partial versions",
					"Could not list versions to find new version number: "+err.Error(),
				)
				return
			}

			newVersion = -1
			for _, v := range versions {
				if v.PromptPartialVersionID == updateResp.PromptPartialVersionID {
					newVersion = v.Version
					break
				}
			}

			if newVersion == -1 {
				resp.Diagnostics.AddError(
					"Error finding new prompt partial version",
					"Could not find version number for version ID "+updateResp.PromptPartialVersionID+" in versions list",
				)
				return
			}

			err = r.client.MakePromptPartialVersionDefault(ctx, state.Slug.ValueString(), newVersion)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error making prompt partial version default",
					"Could not make latest version default: "+err.Error(),
				)
				return
			}
		}
	}

	// Set state from plan values to avoid stale API reads due to eventual consistency.
	// We trust the plan values for fields we sent, and derive computed fields.
	plan.ID = state.ID
	plan.Slug = state.Slug

	if contentChanged && newVersion > 0 {
		plan.Version = types.Int64Value(int64(newVersion))
		plan.PromptPartialVersionID = types.StringValue(updateResp.PromptPartialVersionID)
	} else {
		plan.Version = state.Version
		plan.PromptPartialVersionID = state.PromptPartialVersionID
	}
	plan.Status = types.StringValue("active")
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = types.StringValue(time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *promptPartialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state promptPartialResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing prompt partial
	err := r.client.DeletePromptPartial(ctx, state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Portkey Prompt Partial",
			"Could not delete prompt partial, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *promptPartialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support both "slug" and "workspace_id/slug" import formats
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), parts[1])...)
	} else {
		resource.ImportStatePassthroughID(ctx, path.Root("slug"), req, resp)
	}
}

// mapPartialToState maps a PromptPartial API response to the Terraform state model.
// Detects external changes by comparing API version to state version. If the API
// version is higher, someone edited outside Terraform and we refresh from the API
// so Terraform can detect the drift and overwrite back to config values.
func (r *promptPartialResource) mapPartialToState(state *promptPartialResourceModel, partial *client.PromptPartial) {
	state.ID = types.StringValue(partial.ID)
	state.Slug = types.StringValue(partial.Slug)
	state.Name = types.StringValue(partial.Name)
	state.Status = types.StringValue(partial.Status)

	// Detect external changes: if the API version differs from state, someone
	// edited outside Terraform (new version or rollback). Refresh from API so
	// Terraform sees the drift and overwrites back to config values on next apply.
	externalChange := !state.Version.IsNull() && !state.Version.IsUnknown() &&
		int64(partial.Version) != state.Version.ValueInt64()

	if externalChange || state.Content.IsNull() || state.Content.IsUnknown() {
		state.Content = types.StringValue(partial.String)
	}

	// Always update version and version ID from API — these are computed fields
	// that should reflect reality.
	state.Version = types.Int64Value(int64(partial.Version))
	state.PromptPartialVersionID = types.StringValue(partial.PromptPartialVersionID)

	// Preserve workspace_id from state — the API does not return it in the
	// PromptPartial response, so we must never overwrite the user-supplied value.

	// Only preserve version_description from state — never import it from the API.
	// The API may return a version_description set via console edits, but if the
	// Terraform config doesn't set it, importing it would cause perpetual drift
	// (state has value, config doesn't, plan always wants to null it out).

	state.CreatedAt = types.StringValue(partial.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !partial.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(partial.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}
}
