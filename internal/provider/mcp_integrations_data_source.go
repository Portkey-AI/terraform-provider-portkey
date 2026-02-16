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
	_ datasource.DataSource              = &mcpIntegrationsDataSource{}
	_ datasource.DataSourceWithConfigure = &mcpIntegrationsDataSource{}
)

// NewMcpIntegrationsDataSource is a helper function to simplify the provider implementation.
func NewMcpIntegrationsDataSource() datasource.DataSource {
	return &mcpIntegrationsDataSource{}
}

// mcpIntegrationsDataSource is the data source implementation.
type mcpIntegrationsDataSource struct {
	client *client.Client
}

// mcpIntegrationsDataSourceModel maps the data source schema data.
type mcpIntegrationsDataSourceModel struct {
	ID           types.String              `tfsdk:"id"`
	WorkspaceID  types.String              `tfsdk:"workspace_id"`
	Integrations []mcpIntegrationListModel `tfsdk:"integrations"`
}

// mcpIntegrationListModel maps integration data in the list
type mcpIntegrationListModel struct {
	ID            types.String `tfsdk:"id"`
	Slug          types.String `tfsdk:"slug"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	URL           types.String `tfsdk:"url"`
	AuthType      types.String `tfsdk:"auth_type"`
	Transport     types.String `tfsdk:"transport"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	Status        types.String `tfsdk:"status"`
	CreatedAt     types.String `tfsdk:"created_at"`
	LastUpdatedAt types.String `tfsdk:"last_updated_at"`
}

// Metadata returns the data source type name.
func (d *mcpIntegrationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_integrations"
}

// Schema defines the schema for the data source.
func (d *mcpIntegrationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all Portkey MCP integrations, optionally filtered by workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this data source.",
				Computed:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Filter integrations by workspace ID.",
				Optional:    true,
			},
			"integrations": schema.ListNestedAttribute{
				Description: "List of MCP integrations.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":              schema.StringAttribute{Computed: true, Description: "Integration ID."},
						"slug":            schema.StringAttribute{Computed: true, Description: "URL-friendly identifier."},
						"name":            schema.StringAttribute{Computed: true, Description: "Integration name."},
						"description":     schema.StringAttribute{Computed: true, Description: "Integration description."},
						"url":             schema.StringAttribute{Computed: true, Description: "MCP server URL."},
						"auth_type":       schema.StringAttribute{Computed: true, Description: "Authentication type."},
						"transport":       schema.StringAttribute{Computed: true, Description: "Transport protocol."},
						"workspace_id":    schema.StringAttribute{Computed: true, Description: "Workspace ID."},
						"status":          schema.StringAttribute{Computed: true, Description: "Status."},
						"created_at":      schema.StringAttribute{Computed: true, Description: "Creation timestamp."},
						"last_updated_at": schema.StringAttribute{Computed: true, Description: "Last update timestamp."},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *mcpIntegrationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *mcpIntegrationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state mcpIntegrationsDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := ""
	if !state.WorkspaceID.IsNull() && !state.WorkspaceID.IsUnknown() {
		workspaceID = state.WorkspaceID.ValueString()
	}

	integrations, err := d.client.ListMcpIntegrations(ctx, workspaceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey MCP Integrations",
			err.Error(),
		)
		return
	}

	for _, i := range integrations {
		state.Integrations = append(state.Integrations, mcpIntegrationListModel{
			ID:            types.StringValue(i.ID),
			Slug:          types.StringValue(i.Slug),
			Name:          types.StringValue(i.Name),
			Description:   stringOrNull(i.Description),
			URL:           types.StringValue(i.URL),
			AuthType:      types.StringValue(i.AuthType),
			Transport:     types.StringValue(i.Transport),
			WorkspaceID:   stringOrNull(i.WorkspaceID),
			Status:        stringOrNull(i.Status),
			CreatedAt:     stringOrNull(i.CreatedAt),
			LastUpdatedAt: stringOrNull(i.LastUpdatedAt),
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
