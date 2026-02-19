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

	// Fetch the partial from the API. Content/Version/VersionID are preserved
	// from state by mapPartialToState to avoid eventual consistency issues.
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

		// If a new version was created (content changed), make it the default
		if contentChanged && updateResp.PromptPartialVersionID != "" {
			// NOTE: This assumes versions increment by 1. If versions are created
			// outside Terraform (UI, API, another workspace), the version in state
			// may be stale, causing this to target the wrong version number.
			// The Portkey API's makeDefault endpoint requires a version number,
			// not a version ID, so we cannot use the returned version_id directly.
			newVersion := int(state.Version.ValueInt64()) + 1

			// Make the new version the default
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
	// plan.Name already has correct value from plan
	// plan.Content already has correct value from plan
	// plan.WorkspaceID already has correct value from plan
	// plan.VersionDescription already has correct value from plan

	if contentChanged {
		plan.Version = types.Int64Value(state.Version.ValueInt64() + 1)
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
// Fields managed by Terraform (Content, Version, PromptPartialVersionID) are preserved
// from state when already set to avoid eventual consistency issues with the Portkey API.
func (r *promptPartialResource) mapPartialToState(state *promptPartialResourceModel, partial *client.PromptPartial) {
	state.ID = types.StringValue(partial.ID)
	state.Slug = types.StringValue(partial.Slug)
	state.Name = types.StringValue(partial.Name)
	state.Status = types.StringValue(partial.Status)

	// Preserve Content, Version, and VersionID from state if already set.
	// The Portkey API has eventual consistency — GET may return stale data
	// after updates. We trust the values set during Create/Update.
	if state.Content.IsNull() || state.Content.IsUnknown() {
		state.Content = types.StringValue(partial.String)
	}
	if state.Version.IsNull() || state.Version.IsUnknown() {
		state.Version = types.Int64Value(int64(partial.Version))
	}
	if state.PromptPartialVersionID.IsNull() || state.PromptPartialVersionID.IsUnknown() {
		state.PromptPartialVersionID = types.StringValue(partial.PromptPartialVersionID)
	}

	// Preserve workspace_id from state — the API does not return it in the
	// PromptPartial response, so we must never overwrite the user-supplied value.

	if partial.VersionDescription != "" {
		state.VersionDescription = types.StringValue(partial.VersionDescription)
	}

	state.CreatedAt = types.StringValue(partial.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !partial.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(partial.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}
}
