package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &mcpIntegrationResource{}
	_ resource.ResourceWithConfigure   = &mcpIntegrationResource{}
	_ resource.ResourceWithImportState = &mcpIntegrationResource{}
)

// NewMcpIntegrationResource is a helper function to simplify the provider implementation.
func NewMcpIntegrationResource() resource.Resource {
	return &mcpIntegrationResource{}
}

// mcpIntegrationResource is the resource implementation.
type mcpIntegrationResource struct {
	client *client.Client
}

// mcpIntegrationResourceModel maps the resource schema data.
type mcpIntegrationResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Slug           types.String `tfsdk:"slug"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	URL            types.String `tfsdk:"url"`
	AuthType       types.String `tfsdk:"auth_type"`
	Transport      types.String `tfsdk:"transport"`
	Configurations types.String `tfsdk:"configurations"`
	WorkspaceID    types.String `tfsdk:"workspace_id"`
	Type           types.String `tfsdk:"type"`
	Status         types.String `tfsdk:"status"`
	OwnerID        types.String `tfsdk:"owner_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	LastUpdatedAt  types.String `tfsdk:"last_updated_at"`
}

// Metadata returns the resource type name.
func (r *mcpIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_integration"
}

// Schema defines the schema for the resource.
func (r *mcpIntegrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages a Portkey MCP integration.

MCP integrations define an MCP server in the organization's registry, specifying the URL, authentication type, and transport protocol.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "MCP integration identifier (UUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Description: "URL-friendly identifier. Auto-generated from name if not provided.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the MCP integration.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the MCP integration.",
				Optional:    true,
			},
			"url": schema.StringAttribute{
				Description: "URL of the MCP server.",
				Required:    true,
			},
			"auth_type": schema.StringAttribute{
				Description: "Authentication type for the MCP server.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("none", "api_key", "bearer", "oauth2"),
				},
			},
			"transport": schema.StringAttribute{
				Description: "Transport protocol for the MCP server.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("sse", "streamable_http"),
				},
			},
			"configurations": schema.StringAttribute{
				Description: "JSON string of additional configurations (e.g., auth credentials). Sensitive.",
				Optional:    true,
				Sensitive:   true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID to scope this integration to. Leave empty for org-level.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Integration type.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Integration status.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"owner_id": schema.StringAttribute{
				Description: "Owner user ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the integration was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated_at": schema.StringAttribute{
				Description: "Timestamp when the integration was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *mcpIntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *mcpIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mcpIntegrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateMcpIntegrationRequest{
		Name:      plan.Name.ValueString(),
		URL:       plan.URL.ValueString(),
		AuthType:  plan.AuthType.ValueString(),
		Transport: plan.Transport.ValueString(),
	}

	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() {
		createReq.Slug = plan.Slug.ValueString()
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}
	if !plan.WorkspaceID.IsNull() && !plan.WorkspaceID.IsUnknown() {
		createReq.WorkspaceID = plan.WorkspaceID.ValueString()
	}
	if !plan.Configurations.IsNull() && !plan.Configurations.IsUnknown() {
		createReq.Configurations = plan.Configurations.ValueString()
	}

	createResp, err := r.client.CreateMcpIntegration(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MCP integration",
			"Could not create MCP integration, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch full details
	integration, err := r.client.GetMcpIntegration(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCP integration",
			"Integration created but could not read details: "+err.Error(),
		)
		return
	}

	mapMcpIntegrationToState(integration, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *mcpIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mcpIntegrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	integration, err := r.client.GetMcpIntegration(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading MCP Integration",
			"Could not read MCP integration ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Preserve sensitive configurations from state (API won't return them)
	savedConfigurations := state.Configurations

	mapMcpIntegrationToState(integration, &state)

	// Restore configurations from state since API doesn't return sensitive data
	if !savedConfigurations.IsNull() && !savedConfigurations.IsUnknown() {
		state.Configurations = savedConfigurations
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mcpIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mcpIntegrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateMcpIntegrationRequest{
		Name:      plan.Name.ValueString(),
		URL:       plan.URL.ValueString(),
		AuthType:  plan.AuthType.ValueString(),
		Transport: plan.Transport.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		updateReq.Description = plan.Description.ValueString()
	}
	if !plan.Configurations.IsNull() && !plan.Configurations.IsUnknown() {
		updateReq.Configurations = plan.Configurations.ValueString()
	}

	integration, err := r.client.UpdateMcpIntegration(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating MCP Integration",
			"Could not update MCP integration, unexpected error: "+err.Error(),
		)
		return
	}

	// Preserve configurations from plan since API doesn't return sensitive data
	savedConfigurations := plan.Configurations

	mapMcpIntegrationToState(integration, &plan)

	if !savedConfigurations.IsNull() && !savedConfigurations.IsUnknown() {
		plan.Configurations = savedConfigurations
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mcpIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mcpIntegrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMcpIntegration(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting MCP Integration",
			"Could not delete MCP integration: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *mcpIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapMcpIntegrationToState maps a client McpIntegration to the resource model
func mapMcpIntegrationToState(integration *client.McpIntegration, state *mcpIntegrationResourceModel) {
	state.ID = types.StringValue(integration.ID)
	state.Slug = types.StringValue(integration.Slug)
	state.Name = types.StringValue(integration.Name)
	state.URL = types.StringValue(integration.URL)
	state.AuthType = types.StringValue(integration.AuthType)
	state.Transport = types.StringValue(integration.Transport)

	if integration.Description != "" {
		state.Description = types.StringValue(integration.Description)
	} else {
		state.Description = types.StringNull()
	}

	if integration.WorkspaceID != "" {
		state.WorkspaceID = types.StringValue(integration.WorkspaceID)
	} else {
		state.WorkspaceID = types.StringNull()
	}

	if integration.Type != "" {
		state.Type = types.StringValue(integration.Type)
	} else {
		state.Type = types.StringNull()
	}

	if integration.Status != "" {
		state.Status = types.StringValue(integration.Status)
	} else {
		state.Status = types.StringNull()
	}

	if integration.OwnerID != "" {
		state.OwnerID = types.StringValue(integration.OwnerID)
	} else {
		state.OwnerID = types.StringNull()
	}

	if integration.CreatedAt != "" {
		state.CreatedAt = types.StringValue(integration.CreatedAt)
	}

	if integration.LastUpdatedAt != "" {
		state.LastUpdatedAt = types.StringValue(integration.LastUpdatedAt)
	}
}
