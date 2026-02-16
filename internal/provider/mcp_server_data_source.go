package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &mcpServerDataSource{}
	_ datasource.DataSourceWithConfigure = &mcpServerDataSource{}
)

// NewMcpServerDataSource is a helper function to simplify the provider implementation.
func NewMcpServerDataSource() datasource.DataSource {
	return &mcpServerDataSource{}
}

// mcpServerDataSource is the data source implementation.
type mcpServerDataSource struct {
	client *client.Client
}

// mcpServerDataSourceModel maps the data source schema data.
type mcpServerDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Slug             types.String `tfsdk:"slug"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	McpIntegrationID types.String `tfsdk:"mcp_integration_id"`
	WorkspaceID      types.String `tfsdk:"workspace_id"`
	Status           types.String `tfsdk:"status"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

// Metadata returns the data source type name.
func (d *mcpServerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_server"
}

// Schema defines the schema for the data source.
func (d *mcpServerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a specific Portkey MCP server by ID or slug.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "MCP server identifier (UUID or slug).",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "URL-friendly identifier.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the MCP server.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the MCP server.",
				Computed:    true,
			},
			"mcp_integration_id": schema.StringAttribute{
				Description: "ID of the MCP integration this server provisions.",
				Computed:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID where this server is provisioned.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Server status.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when created.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *mcpServerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

// Read refreshes the Terraform state with the latest data.
func (d *mcpServerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state mcpServerDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := d.client.GetMcpServer(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey MCP Server",
			err.Error(),
		)
		return
	}

	state.ID = types.StringValue(server.ID)
	state.Slug = types.StringValue(server.Slug)
	state.Name = types.StringValue(server.Name)
	state.McpIntegrationID = types.StringValue(server.McpIntegrationID)
	state.Description = stringOrNull(server.Description)
	state.WorkspaceID = stringOrNull(server.WorkspaceID)
	state.Status = stringOrNull(server.Status)
	state.CreatedAt = stringOrNull(server.CreatedAt)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
