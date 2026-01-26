package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &integrationResource{}
	_ resource.ResourceWithConfigure   = &integrationResource{}
	_ resource.ResourceWithImportState = &integrationResource{}
)

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
				Description: "Provider-specific configurations as JSON. For OpenAI: jsonencode({openai_organization = \"org-...\", openai_project = \"proj-...\"}). For AWS Bedrock: jsonencode({aws_role_arn = \"arn:aws:iam::...\", aws_region = \"us-east-1\"}). For Azure OpenAI: jsonencode({resource_name = \"...\", deployment_id = \"...\", api_version = \"...\"}).",
				Optional:    true,
				Sensitive:   true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description of the integration.",
				Optional:    true,
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

	// Map response body to schema
	plan.ID = types.StringValue(integration.ID)
	plan.Slug = types.StringValue(integration.Slug)
	plan.Status = types.StringValue(integration.Status)
	plan.CreatedAt = types.StringValue(integration.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !integration.UpdatedAt.IsZero() {
		plan.UpdatedAt = types.StringValue(integration.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		plan.UpdatedAt = types.StringValue(integration.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
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

	integration, err := r.client.UpdateIntegration(ctx, state.Slug.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Portkey Integration",
			"Could not update integration, unexpected error: "+err.Error(),
		)
		return
	}

	// Update resource state with updated items and timestamp
	plan.ID = types.StringValue(integration.ID)
	plan.Slug = types.StringValue(integration.Slug)
	plan.Status = types.StringValue(integration.Status)
	plan.CreatedAt = types.StringValue(integration.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !integration.UpdatedAt.IsZero() {
		plan.UpdatedAt = types.StringValue(integration.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

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
