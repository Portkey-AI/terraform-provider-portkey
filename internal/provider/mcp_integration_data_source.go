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
	_ datasource.DataSource              = &mcpIntegrationDataSource{}
	_ datasource.DataSourceWithConfigure = &mcpIntegrationDataSource{}
)

// NewMcpIntegrationDataSource is a helper function to simplify the provider implementation.
func NewMcpIntegrationDataSource() datasource.DataSource {
	return &mcpIntegrationDataSource{}
}

// mcpIntegrationDataSource is the data source implementation.
type mcpIntegrationDataSource struct {
	client *client.Client
}

// mcpIntegrationDataSourceModel maps the data source schema data.
type mcpIntegrationDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Slug          types.String `tfsdk:"slug"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	URL           types.String `tfsdk:"url"`
	AuthType      types.String `tfsdk:"auth_type"`
	Transport     types.String `tfsdk:"transport"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	Type          types.String `tfsdk:"type"`
	Status        types.String `tfsdk:"status"`
	OwnerID       types.String `tfsdk:"owner_id"`
	CreatedAt     types.String `tfsdk:"created_at"`
	LastUpdatedAt types.String `tfsdk:"last_updated_at"`
}

// Metadata returns the data source type name.
func (d *mcpIntegrationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_integration"
}

// Schema defines the schema for the data source.
func (d *mcpIntegrationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a specific Portkey MCP integration by ID or slug.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "MCP integration identifier (UUID or slug).",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "URL-friendly identifier.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the MCP integration.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the MCP integration.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "URL of the MCP server.",
				Computed:    true,
			},
			"auth_type": schema.StringAttribute{
				Description: "Authentication type.",
				Computed:    true,
			},
			"transport": schema.StringAttribute{
				Description: "Transport protocol.",
				Computed:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID if scoped to a workspace.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Integration type.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Integration status.",
				Computed:    true,
			},
			"owner_id": schema.StringAttribute{
				Description: "Owner user ID.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when created.",
				Computed:    true,
			},
			"last_updated_at": schema.StringAttribute{
				Description: "Timestamp when last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *mcpIntegrationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *mcpIntegrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state mcpIntegrationDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	integration, err := d.client.GetMcpIntegration(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey MCP Integration",
			err.Error(),
		)
		return
	}

	state.ID = types.StringValue(integration.ID)
	state.Slug = types.StringValue(integration.Slug)
	state.Name = types.StringValue(integration.Name)
	state.URL = types.StringValue(integration.URL)
	state.AuthType = types.StringValue(integration.AuthType)
	state.Transport = types.StringValue(integration.Transport)

	state.Description = stringOrNull(integration.Description)
	state.WorkspaceID = stringOrNull(integration.WorkspaceID)
	state.Type = stringOrNull(integration.Type)
	state.Status = stringOrNull(integration.Status)
	state.OwnerID = stringOrNull(integration.OwnerID)
	state.CreatedAt = stringOrNull(integration.CreatedAt)
	state.LastUpdatedAt = stringOrNull(integration.LastUpdatedAt)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// stringOrNull returns a types.String value or null if the string is empty
func stringOrNull(s string) types.String {
	if s != "" {
		return types.StringValue(s)
	}
	return types.StringNull()
}
