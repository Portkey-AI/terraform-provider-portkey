package provider

import (
	"context"
	"fmt"

	"github.com/portkey-ai/terraform-provider-portkey/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &workspaceDataSource{}
	_ datasource.DataSourceWithConfigure = &workspaceDataSource{}
)

// NewWorkspaceDataSource is a helper function to simplify the provider implementation.
func NewWorkspaceDataSource() datasource.DataSource {
	return &workspaceDataSource{}
}

// workspaceDataSource is the data source implementation.
type workspaceDataSource struct {
	client *client.Client
}

// workspaceDataSourceModel maps the data source schema data.
type workspaceDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	UsageLimits types.List   `tfsdk:"usage_limits"`
	RateLimits  types.List   `tfsdk:"rate_limits"`
	Metadata    types.Map    `tfsdk:"metadata"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// Metadata returns the data source type name.
func (d *workspaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

// Schema defines the schema for the data source.
func (d *workspaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a specific Portkey workspace by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Workspace identifier.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the workspace.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the workspace.",
				Computed:    true,
			},
			"usage_limits": schema.ListNestedAttribute{
				Description: "Usage limits for this workspace.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Type of usage limit.",
							Computed:    true,
						},
						"credit_limit": schema.Int64Attribute{
							Description: "The credit limit value.",
							Computed:    true,
						},
						"alert_threshold": schema.Int64Attribute{
							Description: "Alert threshold percentage.",
							Computed:    true,
						},
						"periodic_reset": schema.StringAttribute{
							Description: "When to reset the usage.",
							Computed:    true,
						},
					},
				},
			},
			"rate_limits": schema.ListNestedAttribute{
				Description: "Rate limits for this workspace.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Type of rate limit.",
							Computed:    true,
						},
						"unit": schema.StringAttribute{
							Description: "Rate limit unit.",
							Computed:    true,
						},
						"value": schema.Int64Attribute{
							Description: "The rate limit value.",
							Computed:    true,
						},
					},
				},
			},
			"metadata": schema.MapAttribute{
				Description: "Custom metadata attached to the workspace.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the workspace was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the workspace was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *workspaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *workspaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state workspaceDataSourceModel

	// Get the ID from the configuration
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get workspace from Portkey API
	workspace, err := d.client.GetWorkspace(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey Workspace",
			err.Error(),
		)
		return
	}

	// Map response to state
	state.ID = types.StringValue(workspace.ID)
	state.Name = types.StringValue(workspace.Name)
	state.Description = types.StringValue(workspace.Description)

	// Handle usage_limits
	ulList, ulDiags := workspaceUsageLimitsToTerraformList(workspace.UsageLimits)
	resp.Diagnostics.Append(ulDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.UsageLimits = ulList

	// Handle rate_limits
	rlList, rlDiags := workspaceRateLimitsToTerraformList(workspace.RateLimits)
	resp.Diagnostics.Append(rlDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.RateLimits = rlList

	// Handle metadata
	if workspace.Defaults != nil && len(workspace.Defaults.Metadata) > 0 {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, workspace.Defaults.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Metadata = metadataMap
	} else {
		state.Metadata = types.MapNull(types.StringType)
	}

	state.CreatedAt = types.StringValue(workspace.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	state.UpdatedAt = types.StringValue(workspace.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
