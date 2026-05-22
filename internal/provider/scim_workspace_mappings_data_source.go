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
	_ datasource.DataSource              = &scimWorkspaceMappingsDataSource{}
	_ datasource.DataSourceWithConfigure = &scimWorkspaceMappingsDataSource{}
)

// NewScimWorkspaceMappingsDataSource is a helper function to simplify the provider implementation.
func NewScimWorkspaceMappingsDataSource() datasource.DataSource {
	return &scimWorkspaceMappingsDataSource{}
}

// scimWorkspaceMappingsDataSource is the data source implementation.
type scimWorkspaceMappingsDataSource struct {
	client *client.Client
}

// scimWorkspaceMappingsDataSourceModel maps the data source schema data.
type scimWorkspaceMappingsDataSourceModel struct {
	WorkspaceID types.String                  `tfsdk:"workspace_id"`
	ScimGroupID types.String                  `tfsdk:"scim_group_id"`
	Role        types.String                  `tfsdk:"role"`
	Mappings    []scimWorkspaceMappingDSModel `tfsdk:"mappings"`
}

// scimWorkspaceMappingDSModel mirrors a single mapping in the data source response.
type scimWorkspaceMappingDSModel struct {
	ID          types.String `tfsdk:"id"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
	ScimGroup   types.String `tfsdk:"scim_group"`
	ScimGroupID types.String `tfsdk:"scim_group_id"`
	Role        types.String `tfsdk:"role"`
}

// Metadata returns the data source type name.
func (d *scimWorkspaceMappingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scim_workspace_mappings"
}

// Schema defines the schema for the data source.
func (d *scimWorkspaceMappingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches Portkey SCIM workspace mappings, optionally filtered by workspace, SCIM group, or role.",
		Attributes: map[string]schema.Attribute{
			"workspace_id": schema.StringAttribute{
				Description: "Filter mappings by workspace ID or slug.",
				Optional:    true,
			},
			"scim_group_id": schema.StringAttribute{
				Description: "Filter mappings by SCIM group ID.",
				Optional:    true,
			},
			"role": schema.StringAttribute{
				Description: "Filter mappings by role (admin, member, or manager).",
				Optional:    true,
			},
			"mappings": schema.ListNestedAttribute{
				Description: "List of SCIM workspace mappings matching the filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Unique identifier of the mapping.",
							Computed:    true,
						},
						"workspace_id": schema.StringAttribute{
							Description: "ID of the mapped workspace.",
							Computed:    true,
						},
						"scim_group": schema.StringAttribute{
							Description: "Display name of the SCIM group.",
							Computed:    true,
						},
						"scim_group_id": schema.StringAttribute{
							Description: "ID of the SCIM group.",
							Computed:    true,
						},
						"role": schema.StringAttribute{
							Description: "Role assigned to group members in the workspace.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *scimWorkspaceMappingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches the mappings list from the Portkey API.
func (d *scimWorkspaceMappingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config scimWorkspaceMappingsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The API's workspace_id filter only matches the workspace's UUID form,
	// not the slug. Users typically pass `portkey_workspace.foo.id` (slug)
	// here, so omit that filter from the API call and post-filter on
	// workspace_id in Go. ScimGroupID and Role are UUIDs / fixed strings and
	// are safe to push to the API.
	mappings, err := d.client.ListScimWorkspaceMappings(ctx, client.ListScimWorkspaceMappingsOptions{
		ScimGroupID: config.ScimGroupID.ValueString(),
		Role:        config.Role.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading SCIM workspace mappings",
			"Could not list SCIM workspace mappings: "+err.Error(),
		)
		return
	}

	if wsFilter := config.WorkspaceID.ValueString(); wsFilter != "" {
		// Try a couple of approaches to match slug vs UUID. First the
		// raw equality (covers UUID-form callers); then look up the
		// workspace via the admin API to resolve slug → UUID and match
		// on that. Falling back to the lookup avoids surfacing an empty
		// list when the user passes the slug that portkey_workspace.id
		// produces.
		filtered := make([]client.ScimWorkspaceMapping, 0, len(mappings))
		for _, m := range mappings {
			if m.WorkspaceID == wsFilter {
				filtered = append(filtered, m)
			}
		}
		if len(filtered) == 0 {
			if ws, lookupErr := d.client.GetWorkspace(ctx, wsFilter); lookupErr == nil && ws != nil {
				for _, m := range mappings {
					if m.WorkspaceID == ws.ID || m.WorkspaceID == ws.Slug {
						filtered = append(filtered, m)
					}
				}
			}
		}
		mappings = filtered
	}

	state := scimWorkspaceMappingsDataSourceModel{
		WorkspaceID: config.WorkspaceID,
		ScimGroupID: config.ScimGroupID,
		Role:        config.Role,
		Mappings:    make([]scimWorkspaceMappingDSModel, 0, len(mappings)),
	}
	for _, m := range mappings {
		state.Mappings = append(state.Mappings, scimWorkspaceMappingDSModel{
			ID:          types.StringValue(m.ID),
			WorkspaceID: types.StringValue(m.WorkspaceID),
			ScimGroup:   types.StringValue(m.ScimGroup),
			ScimGroupID: types.StringValue(m.ScimGroupID),
			Role:        types.StringValue(m.Role),
		})
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
