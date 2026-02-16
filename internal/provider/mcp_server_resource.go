package provider

import (
	"context"
	"fmt"
	"strings"

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
	_ resource.Resource                = &mcpServerResource{}
	_ resource.ResourceWithConfigure   = &mcpServerResource{}
	_ resource.ResourceWithImportState = &mcpServerResource{}
)

// NewMcpServerResource is a helper function to simplify the provider implementation.
func NewMcpServerResource() resource.Resource {
	return &mcpServerResource{}
}

// mcpServerResource is the resource implementation.
type mcpServerResource struct {
	client *client.Client
}

// mcpServerResourceModel maps the resource schema data.
type mcpServerResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Slug             types.String `tfsdk:"slug"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	McpIntegrationID types.String `tfsdk:"mcp_integration_id"`
	WorkspaceID      types.String `tfsdk:"workspace_id"`
	Status           types.String `tfsdk:"status"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

// Metadata returns the resource type name.
func (r *mcpServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_server"
}

// Schema defines the schema for the resource.
func (r *mcpServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages a Portkey MCP server.

MCP servers provision an MCP integration to a specific workspace, making the MCP server's capabilities available to that workspace's users.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "MCP server identifier (UUID).",
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
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the MCP server.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the MCP server.",
				Optional:    true,
			},
			"mcp_integration_id": schema.StringAttribute{
				Description: "ID of the MCP integration this server provisions.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID where this server is provisioned.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Server status.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the server was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *mcpServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *mcpServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mcpServerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateMcpServerRequest{
		Name:             plan.Name.ValueString(),
		McpIntegrationID: plan.McpIntegrationID.ValueString(),
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

	createResp, err := r.client.CreateMcpServer(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MCP server",
			"Could not create MCP server, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch full details
	server, err := r.client.GetMcpServer(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCP server",
			"Server created but could not read details: "+err.Error(),
		)
		return
	}

	mapMcpServerToState(server, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *mcpServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mcpServerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.client.GetMcpServer(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading MCP Server",
			"Could not read MCP server ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	mapMcpServerToState(server, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mcpServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mcpServerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateMcpServerRequest{
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		updateReq.Description = plan.Description.ValueString()
	}

	server, err := r.client.UpdateMcpServer(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating MCP Server",
			"Could not update MCP server, unexpected error: "+err.Error(),
		)
		return
	}

	mapMcpServerToState(server, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mcpServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mcpServerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMcpServer(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting MCP Server",
			"Could not delete MCP server: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *mcpServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapMcpServerToState maps a client McpServer to the resource model
func mapMcpServerToState(server *client.McpServer, state *mcpServerResourceModel) {
	state.ID = types.StringValue(server.ID)
	state.Slug = types.StringValue(server.Slug)
	state.Name = types.StringValue(server.Name)
	state.McpIntegrationID = types.StringValue(server.McpIntegrationID)

	if server.Description != "" {
		state.Description = types.StringValue(server.Description)
	} else {
		state.Description = types.StringNull()
	}

	if server.WorkspaceID != "" {
		state.WorkspaceID = types.StringValue(server.WorkspaceID)
	} else {
		state.WorkspaceID = types.StringNull()
	}

	if server.Status != "" {
		state.Status = types.StringValue(server.Status)
	} else {
		state.Status = types.StringNull()
	}

	if server.CreatedAt != "" {
		state.CreatedAt = types.StringValue(server.CreatedAt)
	}
}
