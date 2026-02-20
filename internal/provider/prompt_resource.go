package provider

import (
	"context"
	"encoding/json"
	"fmt"
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
	_ resource.Resource                = &promptResource{}
	_ resource.ResourceWithConfigure   = &promptResource{}
	_ resource.ResourceWithImportState = &promptResource{}
)

// NewPromptResource is a helper function to simplify the provider implementation.
func NewPromptResource() resource.Resource {
	return &promptResource{}
}

// promptResource is the resource implementation.
type promptResource struct {
	client *client.Client
}

// promptResourceModel maps the resource schema data.
type promptResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Slug                types.String `tfsdk:"slug"`
	Name                types.String `tfsdk:"name"`
	CollectionID        types.String `tfsdk:"collection_id"`
	Template            types.String `tfsdk:"template"`
	Parameters          types.String `tfsdk:"parameters"`
	Model               types.String `tfsdk:"model"`
	VirtualKey          types.String `tfsdk:"virtual_key"`
	VersionDescription  types.String `tfsdk:"version_description"`
	PromptVersion       types.Int64  `tfsdk:"prompt_version"`
	PromptVersionID     types.String `tfsdk:"prompt_version_id"`
	PromptVersionStatus types.String `tfsdk:"prompt_version_status"`
	Status              types.String `tfsdk:"status"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *promptResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt"
}

// Schema defines the schema for the resource.
func (r *promptResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Portkey prompt. Prompts are reusable templates for AI model interactions with versioning support.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Prompt identifier (UUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Description: "URL-friendly identifier for the prompt. Auto-generated based on name.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name for the prompt.",
				Required:    true,
			},
			"collection_id": schema.StringAttribute{
				Description: "Collection ID (UUID) to organize the prompt.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template": schema.StringAttribute{
				Description: "Prompt template string. Supports {{variable}} syntax for dynamic content.",
				Required:    true,
			},
			"parameters": schema.StringAttribute{
				Description: "JSON string of model parameters (e.g., temperature, max_tokens).",
				Optional:    true,
				Computed:    true,
			},
			"model": schema.StringAttribute{
				Description: "Model to use for this prompt (e.g., 'gpt-4o', 'claude-3-opus').",
				Required:    true,
			},
			"virtual_key": schema.StringAttribute{
				Description: "Virtual key (provider) ID or slug to use for this prompt.",
				Required:    true,
			},
			"version_description": schema.StringAttribute{
				Description: "Description for the prompt version.",
				Optional:    true,
			},
			"prompt_version": schema.Int64Attribute{
				Description: "Current version number of the prompt.",
				Computed:    true,
			},
			"prompt_version_id": schema.StringAttribute{
				Description: "Current version ID of the prompt.",
				Computed:    true,
			},
			"prompt_version_status": schema.StringAttribute{
				Description: "Status of the current version (active, archived).",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Status of the prompt (active, archived).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the prompt was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the prompt was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *promptResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *promptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan promptResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse parameters JSON
	parameters := map[string]interface{}{}
	if !plan.Parameters.IsNull() && !plan.Parameters.IsUnknown() && plan.Parameters.ValueString() != "" {
		if err := json.Unmarshal([]byte(plan.Parameters.ValueString()), &parameters); err != nil {
			resp.Diagnostics.AddError(
				"Invalid Parameters JSON",
				"The parameters attribute must be valid JSON: "+err.Error(),
			)
			return
		}
	}

	// Create new prompt
	createReq := client.CreatePromptRequest{
		Name:         plan.Name.ValueString(),
		CollectionID: plan.CollectionID.ValueString(),
		String:       plan.Template.ValueString(),
		Parameters:   parameters,
		Model:        plan.Model.ValueString(),
		VirtualKey:   plan.VirtualKey.ValueString(),
	}

	if !plan.VersionDescription.IsNull() && !plan.VersionDescription.IsUnknown() {
		createReq.VersionDescription = plan.VersionDescription.ValueString()
	}

	createResp, err := r.client.CreatePrompt(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating prompt",
			"Could not create prompt, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch the full prompt details
	prompt, err := r.client.GetPrompt(ctx, createResp.Slug, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading prompt after creation",
			"Could not read prompt, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema
	r.mapPromptToState(&plan, prompt)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *promptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state promptResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch the prompt from the API. mapPromptToState detects external
	// changes by comparing versions and refreshes content if needed.
	prompt, err := r.client.GetPrompt(ctx, state.Slug.ValueString(), "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Portkey Prompt",
			"Could not read Portkey prompt slug "+state.Slug.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to state, preserving user's parameter formatting
	oldParams := state.Parameters
	r.mapPromptToState(&state, prompt)

	// Keep user's parameter format if semantically equal
	if !oldParams.IsNull() && !oldParams.IsUnknown() {
		var oldParamsMap, newParamsMap map[string]interface{}
		oldErr := json.Unmarshal([]byte(oldParams.ValueString()), &oldParamsMap)
		newErr := json.Unmarshal([]byte(state.Parameters.ValueString()), &newParamsMap)
		if oldErr == nil && newErr == nil {
			oldBytes, _ := json.Marshal(oldParamsMap)
			newBytes, _ := json.Marshal(newParamsMap)
			if string(oldBytes) == string(newBytes) {
				state.Parameters = oldParams
			}
		}
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *promptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan promptResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state for the slug
	var state promptResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check what changed
	nameChanged := plan.Name.ValueString() != state.Name.ValueString()
	templateChanged := plan.Template.ValueString() != state.Template.ValueString()
	modelChanged := plan.Model.ValueString() != state.Model.ValueString()
	paramsChanged := plan.Parameters.ValueString() != state.Parameters.ValueString()
	virtualKeyChanged := plan.VirtualKey.ValueString() != state.VirtualKey.ValueString()
	versionDescChanged := !plan.VersionDescription.Equal(state.VersionDescription)

	versionUpdateRequired := templateChanged || modelChanged || paramsChanged

	// Warn if version_description changed without a version-triggering field change — it only takes effect with new versions
	if versionDescChanged && !versionUpdateRequired {
		resp.Diagnostics.AddWarning(
			"Version Description Change Ignored",
			"The version_description was changed but no version-triggering fields (template, model, or parameters) were modified. Version descriptions are only applied when a new version is created. The version_description change will be stored in state but not sent to the API.",
		)
	}

	// Build update request
	updateReq := client.UpdatePromptRequest{}

	if nameChanged {
		updateReq.Name = plan.Name.ValueString()
	}

	if virtualKeyChanged {
		updateReq.VirtualKey = plan.VirtualKey.ValueString()
	}

	if versionUpdateRequired {
		// For version updates, we need all required fields
		updateReq.String = plan.Template.ValueString()
		updateReq.Model = plan.Model.ValueString()

		// Parse parameters JSON
		parameters := map[string]interface{}{}
		if !plan.Parameters.IsNull() && !plan.Parameters.IsUnknown() && plan.Parameters.ValueString() != "" {
			if err := json.Unmarshal([]byte(plan.Parameters.ValueString()), &parameters); err != nil {
				resp.Diagnostics.AddError(
					"Invalid Parameters JSON",
					"The parameters attribute must be valid JSON: "+err.Error(),
				)
				return
			}
		}
		updateReq.Parameters = parameters

		// is_raw_template is required for version updates
		isRawTemplate := 0
		updateReq.IsRawTemplate = &isRawTemplate

		if !plan.VersionDescription.IsNull() && !plan.VersionDescription.IsUnknown() {
			updateReq.VersionDescription = plan.VersionDescription.ValueString()
		}

		// virtual_key is required for all version-creating updates
		updateReq.VirtualKey = plan.VirtualKey.ValueString()
	}

	// Only call update if there are changes
	var updateResp *client.UpdatePromptResponse
	var newVersion int
	if nameChanged || versionUpdateRequired || virtualKeyChanged {
		var err error
		updateResp, err = r.client.UpdatePrompt(ctx, state.Slug.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Portkey Prompt",
				"Could not update prompt, unexpected error: "+err.Error(),
			)
			return
		}

		// If a new version was created, make it the default.
		// Look up the real version number from the versions list by matching
		// the version ID returned by Update, since the MakeDefault endpoint
		// requires a version number (not a UUID).
		if versionUpdateRequired && updateResp.PromptVersionID != "" {
			versions, err := r.client.ListPromptVersions(ctx, state.Slug.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error listing prompt versions",
					"Could not list versions to find new version number: "+err.Error(),
				)
				return
			}

			newVersion = -1
			for _, v := range versions {
				if v.ID == updateResp.PromptVersionID {
					newVersion = v.PromptVersion
					break
				}
			}

			if newVersion == -1 {
				resp.Diagnostics.AddError(
					"Error finding new prompt version",
					"Could not find version number for version ID "+updateResp.PromptVersionID+" in versions list",
				)
				return
			}

			err = r.client.MakePromptVersionDefault(ctx, state.Slug.ValueString(), newVersion)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error making prompt version default",
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
	plan.CollectionID = state.CollectionID

	// Parameters is Optional+Computed — if user didn't set it, plan value is unknown.
	// Ensure it's always a known value after apply.
	if plan.Parameters.IsNull() || plan.Parameters.IsUnknown() {
		plan.Parameters = state.Parameters
	}

	if versionUpdateRequired && newVersion > 0 {
		plan.PromptVersion = types.Int64Value(int64(newVersion))
		plan.PromptVersionID = types.StringValue(updateResp.PromptVersionID)
	} else {
		plan.PromptVersion = state.PromptVersion
		plan.PromptVersionID = state.PromptVersionID
	}
	plan.PromptVersionStatus = types.StringValue("active")
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
func (r *promptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state promptResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing prompt
	err := r.client.DeletePrompt(ctx, state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Portkey Prompt",
			"Could not delete prompt, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *promptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by slug
	resource.ImportStatePassthroughID(ctx, path.Root("slug"), req, resp)
}

// mapPromptToState maps a Prompt API response to the Terraform state model.
// Detects external changes by comparing API version to state version. If the API
// version differs, someone edited outside Terraform and we refresh from the API
// so Terraform can detect the drift and overwrite back to config values.
func (r *promptResource) mapPromptToState(state *promptResourceModel, prompt *client.Prompt) {
	state.ID = types.StringValue(prompt.ID)
	state.Slug = types.StringValue(prompt.Slug)
	state.Name = types.StringValue(prompt.Name)
	// Preserve collection_id from state to avoid triggering RequiresReplace unnecessarily
	if state.CollectionID.IsNull() || state.CollectionID.IsUnknown() {
		state.CollectionID = types.StringValue(prompt.CollectionID)
	}
	// Detect external changes: if the API version differs from state, someone
	// edited outside Terraform (new version or rollback). Refresh from API so
	// Terraform sees the drift and overwrites back to config values on next apply.
	externalChange := !state.PromptVersion.IsNull() && !state.PromptVersion.IsUnknown() &&
		int64(prompt.PromptVersion) != state.PromptVersion.ValueInt64()

	if externalChange || state.Template.IsNull() || state.Template.IsUnknown() {
		state.Template = types.StringValue(prompt.String)
	}
	if externalChange || state.Model.IsNull() || state.Model.IsUnknown() {
		state.Model = types.StringValue(prompt.Model)
	}
	// Keep the user-provided virtual_key value (API returns slug but user may have provided ID)
	if state.VirtualKey.IsNull() || state.VirtualKey.IsUnknown() {
		state.VirtualKey = types.StringValue(prompt.VirtualKey)
	}

	// Always update version and version ID from API — these are computed fields
	// that should reflect reality.
	state.PromptVersion = types.Int64Value(int64(prompt.PromptVersion))
	state.PromptVersionID = types.StringValue(prompt.PromptVersionID)
	state.PromptVersionStatus = types.StringValue(prompt.PromptVersionStatus)
	state.Status = types.StringValue(prompt.Status)

	// Refresh parameters on external change or first population.
	// Preserved from state otherwise since API may add extra fields like "model".
	if externalChange || state.Parameters.IsNull() || state.Parameters.IsUnknown() {
		if prompt.Parameters != nil {
			paramsBytes, err := json.Marshal(prompt.Parameters)
			if err == nil {
				state.Parameters = types.StringValue(string(paramsBytes))
			}
		} else {
			state.Parameters = types.StringValue("{}")
		}
	}

	// Only preserve version_description from state — never import it from the API.
	// The API may return a version_description set via console edits, but if the
	// Terraform config doesn't set it, importing it would cause perpetual drift
	// (state has value, config doesn't, plan always wants to null it out).

	state.CreatedAt = types.StringValue(prompt.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !prompt.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(prompt.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}
}
