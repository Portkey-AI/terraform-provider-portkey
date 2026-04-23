package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &integrationResource{}
	_ resource.ResourceWithConfigure      = &integrationResource{}
	_ resource.ResourceWithImportState    = &integrationResource{}
	_ resource.ResourceWithValidateConfig = &integrationResource{}
)

// secretMappingAttrTypes is the shared object-type description for a
// secret_mappings set element. Declared at package level so that the resource,
// the data source, and helper converters agree on the exact shape.
var secretMappingAttrTypes = map[string]attr.Type{
	"target_field":        types.StringType,
	"secret_reference_id": types.StringType,
	"secret_key":          types.StringType,
}

// secretMappingTargetFieldRegex enforces the two legal shapes documented by
// the Portkey API for Integration target fields: the bare "key" field or
// "configurations.<field>" where <field> is an alphanumeric/underscore
// identifier.
var secretMappingTargetFieldRegex = regexp.MustCompile(`^(key|configurations\.[A-Za-z0-9_]+)$`)

// NewIntegrationResource is a helper function to simplify the provider implementation.
func NewIntegrationResource() resource.Resource {
	return &integrationResource{}
}

// integrationResource is the resource implementation.
type integrationResource struct {
	client *client.Client
}

// integrationResourceModel maps the resource schema data.
type integrationResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Slug           types.String `tfsdk:"slug"`
	Name           types.String `tfsdk:"name"`
	AIProviderID   types.String `tfsdk:"ai_provider_id"`
	Key            types.String `tfsdk:"key"`
	KeyWriteOnly   types.String `tfsdk:"key_wo"`
	KeyVersion     types.Int64  `tfsdk:"key_version"`
	Configurations types.String `tfsdk:"configurations"`
	Description    types.String `tfsdk:"description"`
	WorkspaceID    types.String `tfsdk:"workspace_id"`
	AllowAllModels types.Bool   `tfsdk:"allow_all_models"`
	SecretMappings types.Set    `tfsdk:"secret_mappings"`
	Type           types.String `tfsdk:"type"`
	Status         types.String `tfsdk:"status"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *integrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

// Schema defines the schema for the resource.
func (r *integrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Portkey integration. Integrations connect Portkey to AI providers like OpenAI, Anthropic, Azure, etc.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Integration identifier (UUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Description: "URL-friendly identifier for the integration. Auto-generated if not provided.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name for the integration.",
				Required:    true,
			},
			"ai_provider_id": schema.StringAttribute{
				Description: "ID of the AI provider (e.g., 'openai', 'anthropic', 'azure-openai').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Description: "API key for the provider. Stored in Terraform state (marked sensitive). Use this for simpler workflows or when state is already secured. For enhanced security where keys should never be stored in state, use key_wo instead.",
				Optional:    true,
				Sensitive:   true,
			},
			"key_wo": schema.StringAttribute{
				Description: "API key for the provider (write-only). Never stored in Terraform state or shown in plan output. Requires Terraform 1.11+. Use with key_version to control when the key is sent to the API.",
				Optional:    true,
				WriteOnly:   true,
			},
			"key_version": schema.Int64Attribute{
				Description: "Trigger for applying the write-only API key. Only used with key_wo. Increment this value to update the key - the key is only sent to the API when key_version changes.",
				Optional:    true,
			},
			"configurations": schema.StringAttribute{
				Description: "Provider-specific configurations as JSON. For OpenAI: jsonencode({openai_organization = \"org-...\", openai_project = \"proj-...\"}). For AWS Bedrock: jsonencode({aws_role_arn = \"arn:aws:iam::...\", aws_region = \"us-east-1\"}). For Azure OpenAI: jsonencode({azure_auth_mode = \"default\", azure_resource_name = \"...\", azure_deployment_config = [{azure_deployment_name = \"...\", azure_api_version = \"...\", azure_model_slug = \"gpt-4\", is_default = true}]}).",
				Optional:    true,
				Sensitive:   true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description of the integration.",
				Optional:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID to scope this integration to. When set, creates a workspace-level integration that is only accessible within that workspace. When not set (and using an organisation API key), creates an organisation-level integration. If using a workspace-scoped API key, the integration is automatically scoped to that workspace.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allow_all_models": schema.BoolAttribute{
				Description: "Whether all models are enabled by default for this integration. When true (the default), all models for the provider are available. Set to false to restrict access to only models explicitly enabled via portkey_integration_model_access resources.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"secret_mappings": schema.SetNestedAttribute{
				Description: "Dynamically resolve credentials from a `portkey_secret_reference` at request time instead of storing them on the integration. Each mapping populates a single integration field (`key` or `configurations.<field>`) from the referenced secret. When a mapping with `target_field = \"key\"` is supplied, the `key`/`key_wo` body arguments can be omitted. Omitting this attribute preserves the previous (pre-upgrade) behaviour; set it to `[]` to clear all existing mappings.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"target_field": schema.StringAttribute{
							Description: "Integration field to populate. Either `key` or `configurations.<field>` (e.g. `configurations.aws_secret_access_key`, `configurations.azure_entra_client_secret`). Must be unique within the set.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									secretMappingTargetFieldRegex,
									`must be "key" or "configurations.<field>" where <field> is alphanumeric/underscore`,
								),
							},
						},
						"secret_reference_id": schema.StringAttribute{
							Description: "UUID or slug of the `portkey_secret_reference` whose value should populate `target_field` at request time. Must belong to the same organisation and be accessible by the workspace.",
							Required:    true,
						},
						"secret_key": schema.StringAttribute{
							Description: "Optional override for the secret reference's `secret_key`. Use to pick a specific field out of a multi-value (JSON) secret payload. When unset, the `secret_key` configured on the secret reference itself is used.",
							Optional:    true,
						},
					},
				},
			},
			"type": schema.StringAttribute{
				Description: "Type of integration: 'organisation' for org-level integrations or 'workspace' for workspace-scoped integrations.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Status of the integration (active, archived).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the integration was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the integration was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *integrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *integrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan integrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve write-only values from config (not plan)
	var config integrationResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new integration
	createReq := client.CreateIntegrationRequest{
		Name:         plan.Name.ValueString(),
		AIProviderID: plan.AIProviderID.ValueString(),
	}

	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() {
		createReq.Slug = plan.Slug.ValueString()
	}

	// Validate: cannot use both key and key_wo
	hasKey := !plan.Key.IsNull() && !plan.Key.IsUnknown()
	hasKeyWO := !config.KeyWriteOnly.IsNull() && !config.KeyWriteOnly.IsUnknown()

	if hasKey && hasKeyWO {
		resp.Diagnostics.AddError(
			"Conflicting API Key Attributes",
			"Cannot specify both 'key' and 'key_wo'. Choose one: 'key' (stored in state, simpler) or 'key_wo' (write-only, enhanced security).",
		)
		return
	}

	// Warn if key_wo is used without key_version (key cannot be updated after creation)
	if hasKeyWO && plan.KeyVersion.IsNull() {
		resp.Diagnostics.AddWarning(
			"Missing key_version",
			"Using key_wo without key_version means the API key cannot be updated after initial creation. Consider adding key_version to enable key updates.",
		)
	}

	// Use key_wo (write-only) if provided, otherwise fall back to key
	if hasKeyWO {
		tflog.Debug(ctx, "Creating integration with write-only key (key_wo)", map[string]interface{}{
			"integration_name": plan.Name.ValueString(),
		})
		createReq.Key = config.KeyWriteOnly.ValueString()
	} else if hasKey {
		tflog.Debug(ctx, "Creating integration with key", map[string]interface{}{
			"integration_name": plan.Name.ValueString(),
		})
		createReq.Key = plan.Key.ValueString()
	}

	// Parse configurations JSON if provided
	if !plan.Configurations.IsNull() && !plan.Configurations.IsUnknown() {
		var configurations map[string]interface{}
		if err := json.Unmarshal([]byte(plan.Configurations.ValueString()), &configurations); err != nil {
			resp.Diagnostics.AddError(
				"Invalid configurations JSON",
				"Could not parse configurations: "+err.Error(),
			)
			return
		}
		createReq.Configurations = configurations
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}

	if !plan.WorkspaceID.IsNull() && !plan.WorkspaceID.IsUnknown() {
		createReq.WorkspaceID = plan.WorkspaceID.ValueString()
	}

	// Attach secret_mappings if the user supplied the attribute. A nil pointer
	// omits the field from the wire payload entirely, which is the
	// backward-compatible default for users who never opted in.
	mappings, mapDiags := secretMappingsToClient(ctx, plan.SecretMappings)
	resp.Diagnostics.Append(mapDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if mappings != nil {
		createReq.SecretMappings = mappings
	}

	createResp, err := r.client.CreateIntegration(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating integration",
			"Could not create integration, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch the full integration details
	integration, err := r.client.GetIntegration(ctx, createResp.Slug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading integration after creation",
			"Could not read integration, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and set partial state BEFORE the models call.
	// This ensures that if UpdateIntegrationModels fails, Terraform still tracks
	// the created integration and can reconcile on the next plan/apply.
	plan.ID = types.StringValue(integration.ID)
	plan.Slug = types.StringValue(integration.Slug)
	plan.Status = types.StringValue(integration.Status)
	plan.CreatedAt = types.StringValue(integration.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !integration.UpdatedAt.IsZero() {
		plan.UpdatedAt = types.StringValue(integration.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		plan.UpdatedAt = types.StringValue(integration.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Set type from API response
	if integration.Type != "" {
		plan.Type = types.StringValue(integration.Type)
	}

	// Preserve user-provided workspace_id (API may return slug format instead of UUID)
	// Only update from API if user didn't provide workspace_id
	if plan.WorkspaceID.IsNull() || plan.WorkspaceID.IsUnknown() {
		if integration.WorkspaceID != "" {
			plan.WorkspaceID = types.StringValue(integration.WorkspaceID)
		} else {
			plan.WorkspaceID = types.StringNull()
		}
	}

	// Refresh secret_mappings from the server response so state is faithful to
	// the API's view. For users who never set the attribute, the helper keeps
	// it null (avoiding a spurious null -> [] diff on the next plan).
	newMappings, smDiags := secretMappingsFromClient(integration.SecretMappings, plan.SecretMappings)
	resp.Diagnostics.Append(smDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.SecretMappings = newMappings

	// Only call UpdateIntegrationModels when allow_all_models is false,
	// since the API already defaults to true.
	if !plan.AllowAllModels.ValueBool() {
		allowAll := plan.AllowAllModels.ValueBool()
		modelsReq := client.BulkUpdateModelsRequest{
			AllowAllModels: &allowAll,
			Models:         []client.IntegrationModel{},
		}
		err = r.client.UpdateIntegrationModels(ctx, createResp.Slug, modelsReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error setting allow_all_models",
				"Could not update allow_all_models for integration: "+err.Error(),
			)
			// State is set below so Terraform tracks the integration even on failure
		}
	}

	// Read allow_all_models from the models endpoint to get the actual API value
	modelsResp, err := r.client.GetIntegrationModels(ctx, integration.Slug)
	if err != nil {
		// If we can't read models, use the plan value so state is still saved
		resp.Diagnostics.AddWarning(
			"Error reading integration models after creation",
			"Could not read integration models, using plan value: "+err.Error(),
		)
	} else {
		plan.AllowAllModels = types.BoolValue(modelsResp.AllowAllModels)
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *integrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state integrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed integration value from Portkey
	integration, err := r.client.GetIntegration(ctx, state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Portkey Integration",
			"Could not read Portkey integration slug "+state.Slug.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.ID = types.StringValue(integration.ID)
	// Preserve slug from state to avoid triggering RequiresReplace unnecessarily
	if state.Slug.IsNull() || state.Slug.IsUnknown() {
		state.Slug = types.StringValue(integration.Slug)
	}
	state.Name = types.StringValue(integration.Name)
	// Preserve ai_provider_id from state to avoid triggering RequiresReplace unnecessarily
	if state.AIProviderID.IsNull() || state.AIProviderID.IsUnknown() {
		state.AIProviderID = types.StringValue(integration.AIProviderID)
	}
	state.Status = types.StringValue(integration.Status)
	if integration.Description != "" {
		state.Description = types.StringValue(integration.Description)
	}
	state.CreatedAt = types.StringValue(integration.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !integration.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(integration.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Set type from API response
	if integration.Type != "" {
		state.Type = types.StringValue(integration.Type)
	}

	// Preserve user-provided workspace_id from state (API may return slug format instead of UUID)
	// Only update from API if not already set in state
	if state.WorkspaceID.IsNull() || state.WorkspaceID.IsUnknown() {
		if integration.WorkspaceID != "" {
			state.WorkspaceID = types.StringValue(integration.WorkspaceID)
		} else {
			state.WorkspaceID = types.StringNull()
		}
	}

	// Reflect server-side secret_mappings into state so drift shows up on the
	// next plan if an out-of-band change was made via the UI or API.
	newMappings, smDiags := secretMappingsFromClient(integration.SecretMappings, state.SecretMappings)
	resp.Diagnostics.Append(smDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.SecretMappings = newMappings

	// Read allow_all_models from the models endpoint
	modelsResp, err := r.client.GetIntegrationModels(ctx, state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading integration models",
			"Could not read integration models: "+err.Error(),
		)
		return
	}
	state.AllowAllModels = types.BoolValue(modelsResp.AllowAllModels)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *integrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan integrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve write-only values from config (not plan)
	var config integrationResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state for the slug
	var state integrationResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update existing integration
	updateReq := client.UpdateIntegrationRequest{
		Name: plan.Name.ValueString(),
	}

	// Validate: cannot use both key and key_wo
	hasKey := !plan.Key.IsNull() && !plan.Key.IsUnknown()
	hasKeyWO := !config.KeyWriteOnly.IsNull() && !config.KeyWriteOnly.IsUnknown()

	if hasKey && hasKeyWO {
		resp.Diagnostics.AddError(
			"Conflicting API Key Attributes",
			"Cannot specify both 'key' and 'key_wo'. Choose one: 'key' (stored in state, simpler) or 'key_wo' (write-only, enhanced security).",
		)
		return
	}

	// Warn if key_wo is used without key_version (key cannot be updated)
	if hasKeyWO && plan.KeyVersion.IsNull() {
		resp.Diagnostics.AddWarning(
			"Missing key_version",
			"Using key_wo without key_version means the API key cannot be updated after initial creation. Consider adding key_version to enable key updates.",
		)
	}

	// Handle key updates:
	// - For key_wo (write-only): only send if key_version changed (trigger-based)
	// - For key: send if provided
	if hasKeyWO {
		// Using write-only key - check if key_version changed
		keyVersionChanged := !plan.KeyVersion.Equal(state.KeyVersion)
		if keyVersionChanged {
			tflog.Debug(ctx, "Updating integration key (key_version changed)", map[string]interface{}{
				"integration_name": plan.Name.ValueString(),
				"old_key_version":  state.KeyVersion.ValueInt64(),
				"new_key_version":  plan.KeyVersion.ValueInt64(),
			})
			updateReq.Key = config.KeyWriteOnly.ValueString()
		} else {
			tflog.Debug(ctx, "Skipping key update (key_version unchanged)", map[string]interface{}{
				"integration_name": plan.Name.ValueString(),
				"key_version":      plan.KeyVersion.ValueInt64(),
			})
		}
	} else if hasKey {
		// Using key (stored in state) - send if provided
		tflog.Debug(ctx, "Updating integration with key", map[string]interface{}{
			"integration_name": plan.Name.ValueString(),
		})
		updateReq.Key = plan.Key.ValueString()
	}

	// Parse configurations JSON if provided
	if !plan.Configurations.IsNull() && !plan.Configurations.IsUnknown() {
		var configurations map[string]interface{}
		if err := json.Unmarshal([]byte(plan.Configurations.ValueString()), &configurations); err != nil {
			resp.Diagnostics.AddError(
				"Invalid configurations JSON",
				"Could not parse configurations: "+err.Error(),
			)
			return
		}
		updateReq.Configurations = configurations
	}

	if !plan.Description.IsNull() {
		updateReq.Description = plan.Description.ValueString()
	}

	// Propagate secret_mappings. If the plan's value changed from the prior
	// state, we must send the field so that the API reconciles (including
	// clearing, which is why an explicit empty set emits an empty array).
	// If the user never set the attribute, we leave the pointer nil and the
	// field is omitted from the wire payload - preserving pre-upgrade
	// behaviour.
	if !plan.SecretMappings.Equal(state.SecretMappings) {
		mappings, mapDiags := secretMappingsToClient(ctx, plan.SecretMappings)
		resp.Diagnostics.Append(mapDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if mappings == nil {
			// Plan transitioned from a populated set back to null. Send an
			// explicit empty array so the API clears any existing mappings.
			empty := []client.SecretMapping{}
			updateReq.SecretMappings = &empty
		} else {
			updateReq.SecretMappings = mappings
		}
	}

	integration, err := r.client.UpdateIntegration(ctx, state.Slug.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Portkey Integration",
			"Could not update integration, unexpected error: "+err.Error(),
		)
		return
	}

	// Update allow_all_models if it changed
	if !plan.AllowAllModels.Equal(state.AllowAllModels) {
		allowAll := plan.AllowAllModels.ValueBool()
		modelsReq := client.BulkUpdateModelsRequest{
			AllowAllModels: &allowAll,
			Models:         []client.IntegrationModel{},
		}
		err = r.client.UpdateIntegrationModels(ctx, state.Slug.ValueString(), modelsReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating allow_all_models",
				"Could not update allow_all_models for integration: "+err.Error(),
			)
			return
		}
	}

	// Update resource state with updated items and timestamp
	plan.ID = types.StringValue(integration.ID)
	plan.Slug = types.StringValue(integration.Slug)
	plan.Status = types.StringValue(integration.Status)
	plan.CreatedAt = types.StringValue(integration.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !integration.UpdatedAt.IsZero() {
		plan.UpdatedAt = types.StringValue(integration.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Set type from API response
	if integration.Type != "" {
		plan.Type = types.StringValue(integration.Type)
	}

	// Preserve workspace_id from state (workspace_id is immutable, RequiresReplace)
	plan.WorkspaceID = state.WorkspaceID

	// Refresh secret_mappings from the PUT response to surface any
	// server-side normalisation (e.g. slug vs. UUID resolution).
	newMappings, smDiags := secretMappingsFromClient(integration.SecretMappings, plan.SecretMappings)
	resp.Diagnostics.Append(smDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.SecretMappings = newMappings

	// Read allow_all_models from the models endpoint
	modelsResp, err := r.client.GetIntegrationModels(ctx, integration.Slug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading integration models after update",
			"Could not read integration models: "+err.Error(),
		)
		return
	}
	plan.AllowAllModels = types.BoolValue(modelsResp.AllowAllModels)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *integrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state integrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing integration
	err := r.client.DeleteIntegration(ctx, state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Portkey Integration",
			"Could not delete integration, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *integrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by slug
	resource.ImportStatePassthroughID(ctx, path.Root("slug"), req, resp)
}

// ValidateConfig enforces cross-attribute invariants on secret_mappings that
// the per-attribute validators cannot express: uniqueness of target_field
// across the set. (Terraform set semantics only dedupe fully-identical
// objects, so two mappings with the same target_field but different
// secret_reference_id would otherwise slip through and be rejected by the
// API.)
func (r *integrationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config integrationResourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.SecretMappings.IsNull() || config.SecretMappings.IsUnknown() {
		return
	}

	var mappings []secretMappingModel
	diags = config.SecretMappings.ElementsAs(ctx, &mappings, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	seen := make(map[string]int, len(mappings))
	for i, m := range mappings {
		if m.TargetField.IsNull() || m.TargetField.IsUnknown() {
			continue
		}
		tf := m.TargetField.ValueString()
		if prev, ok := seen[tf]; ok {
			resp.Diagnostics.AddAttributeError(
				path.Root("secret_mappings"),
				"Duplicate target_field in secret_mappings",
				fmt.Sprintf("target_field %q appears on secret_mappings[%d] and secret_mappings[%d]. Each target_field may appear at most once.", tf, prev, i),
			)
			return
		}
		seen[tf] = i
	}
}

// secretMappingModel is the tfsdk-facing representation of a single element
// inside the secret_mappings set. It mirrors client.SecretMapping field-for-
// field so we can freely convert between the two with tfsdk decoding.
type secretMappingModel struct {
	TargetField       types.String `tfsdk:"target_field"`
	SecretReferenceID types.String `tfsdk:"secret_reference_id"`
	SecretKey         types.String `tfsdk:"secret_key"`
}

// secretMappingsToClient projects a plan/state Set into the slice shape the
// Portkey client expects.
//
// Return semantics are deliberate and relied upon by the caller when building
// the request body:
//   - (nil, diags)     - user did not supply the attribute (null or unknown);
//     the request should omit the field entirely so existing mappings on the
//     API side are preserved.
//   - (&[]{}, diags)   - user explicitly supplied an empty set; the request
//     should send an explicit empty array so the API clears all mappings.
//   - (&[{...}], diags) - user supplied mappings; send them as-is.
func secretMappingsToClient(ctx context.Context, set types.Set) (*[]client.SecretMapping, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}

	var models []secretMappingModel
	diags := set.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}

	out := make([]client.SecretMapping, 0, len(models))
	for _, m := range models {
		mapping := client.SecretMapping{
			TargetField:       m.TargetField.ValueString(),
			SecretReferenceID: m.SecretReferenceID.ValueString(),
		}
		if !m.SecretKey.IsNull() && !m.SecretKey.IsUnknown() {
			v := m.SecretKey.ValueString()
			mapping.SecretKey = &v
		}
		out = append(out, mapping)
	}
	return &out, diags
}

// secretMappingsFromClient converts the slice returned by the API back into
// a types.Set suitable for state storage.
//
// The `prior` argument is the existing state value for the attribute. When
// the API returns an empty/nil slice we preserve `prior` as-is: that way
// users who never opted into secret_mappings keep a null value (no spurious
// null->[] drift), and users who explicitly set secret_mappings = [] keep
// their empty set.
func secretMappingsFromClient(mappings []client.SecretMapping, prior types.Set) (types.Set, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: secretMappingAttrTypes}
	var diags diag.Diagnostics

	if len(mappings) == 0 {
		if !prior.IsNull() && !prior.IsUnknown() {
			return prior, diags
		}
		return types.SetNull(objType), diags
	}

	elems := make([]attr.Value, 0, len(mappings))
	for _, m := range mappings {
		attrs := map[string]attr.Value{
			"target_field":        types.StringValue(m.TargetField),
			"secret_reference_id": types.StringValue(m.SecretReferenceID),
			"secret_key":          types.StringNull(),
		}
		if m.SecretKey != nil {
			attrs["secret_key"] = types.StringValue(*m.SecretKey)
		}
		obj, d := types.ObjectValue(secretMappingAttrTypes, attrs)
		diags.Append(d...)
		if diags.HasError() {
			return types.SetNull(objType), diags
		}
		elems = append(elems, obj)
	}

	setVal, d := types.SetValue(objType, elems)
	diags.Append(d...)
	if diags.HasError() {
		return types.SetNull(objType), diags
	}
	return setVal, diags
}
