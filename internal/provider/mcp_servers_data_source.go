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
	_ datasource.DataSource              = &mcpServersDataSource{}
	_ datasource.DataSourceWithConfigure = &mcpServersDataSource{}
)

// NewMcpServersDataSource is a helper function to simplify the provider implementation.
func NewMcpServersDataSource() datasource.DataSource {
	return &mcpServersDataSource{}
}

// mcpServersDataSource is the data source implementation.
type mcpServersDataSource struct {
	client *client.Client
}

// mcpServersDataSourceModel maps the data source schema data.
type mcpServersDataSourceModel struct {
	ID          types.String       `tfsdk:"id"`
	WorkspaceID types.String       `tfsdk:"workspace_id"`
	Servers     []mcpServerListModel `tfsdk:"servers"`
}

// mcpServerListModel maps server data in the list
type mcpServerListModel struct {
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
func (d *mcpServersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_servers"
}

// Schema defines the schema for the data source.
func (d *mcpServersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all Portkey MCP servers, optionally filtered by workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this data source.",
				Computed:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Filter servers by workspace ID.",
				Optional:    true,
			},
			"servers": schema.ListNestedAttribute{
				Description: "List of MCP servers.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                 schema.StringAttribute{Computed: true, Description: "Server ID."},
						"slug":               schema.StringAttribute{Computed: true, Description: "URL-friendly identifier."},
						"name":               schema.StringAttribute{Computed: true, Description: "Server name."},
						"description":        schema.StringAttribute{Computed: true, Description: "Server description."},
						"mcp_integration_id": schema.StringAttribute{Computed: true, Description: "MCP integration ID."},
						"workspace_id":       schema.StringAttribute{Computed: true, Description: "Workspace ID."},
						"status":             schema.StringAttribute{Computed: true, Description: "Status."},
						"created_at":         schema.StringAttribute{Computed: true, Description: "Creation timestamp."},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *mcpServersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *mcpServersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state mcpServersDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := ""
	if !state.WorkspaceID.IsNull() && !state.WorkspaceID.IsUnknown() {
		workspaceID = state.WorkspaceID.ValueString()
	}

	servers, err := d.client.ListMcpServers(ctx, workspaceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey MCP Servers",
			err.Error(),
		)
		return
	}

	for _, s := range servers {
		state.Servers = append(state.Servers, mcpServerListModel{
			ID:               types.StringValue(s.ID),
			Slug:             types.StringValue(s.Slug),
			Name:             types.StringValue(s.Name),
			Description:      stringOrNull(s.Description),
			McpIntegrationID: types.StringValue(s.McpIntegrationID),
			WorkspaceID:      stringOrNull(s.WorkspaceID),
			Status:           stringOrNull(s.Status),
			CreatedAt:        stringOrNull(s.CreatedAt),
		})
	}

	if workspaceID != "" {
		state.ID = types.StringValue(workspaceID)
	} else {
		state.ID = types.StringValue("all")
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
