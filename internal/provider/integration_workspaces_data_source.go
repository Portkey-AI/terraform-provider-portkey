package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &integrationWorkspacesDataSource{}
	_ datasource.DataSourceWithConfigure = &integrationWorkspacesDataSource{}
)

// NewIntegrationWorkspacesDataSource is a helper function to simplify the provider implementation.
func NewIntegrationWorkspacesDataSource() datasource.DataSource {
	return &integrationWorkspacesDataSource{}
}

// integrationWorkspacesDataSource is the data source implementation.
type integrationWorkspacesDataSource struct {
	client *client.Client
}

// integrationWorkspacesDataSourceModel maps the data source schema data.
type integrationWorkspacesDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	IntegrationID types.String `tfsdk:"integration_id"`
	Total         types.Int64  `tfsdk:"total"`
	Workspaces    types.List   `tfsdk:"workspaces"`
}

// Metadata returns the data source type name.
func (d *integrationWorkspacesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_workspaces"
}

// Schema defines the schema for the data source.
func (d *integrationWorkspacesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches workspace access configuration for a Portkey integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Data source identifier (same as integration_id).",
				Computed:    true,
			},
			"integration_id": schema.StringAttribute{
				Description: "The integration slug or ID to query workspace access for.",
				Required:    true,
			},
			"total": schema.Int64Attribute{
				Description: "Total number of workspaces with access configuration.",
				Computed:    true,
			},
			"workspaces": schema.ListNestedAttribute{
				Description: "List of workspace access configurations.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Workspace identifier.",
							Computed:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether the integration is enabled for this workspace.",
							Computed:    true,
						},
						"usage_limits": schema.ListNestedAttribute{
							Description: "Usage limits for this workspace.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Description: "Type of usage limit: 'cost' or 'tokens'.",
										Computed:    true,
									},
									"credit_limit": schema.Int64Attribute{
										Description: "The credit limit value.",
										Computed:    true,
									},
									"alert_threshold": schema.Int64Attribute{
										Description: "Alert threshold percentage (0-100).",
										Computed:    true,
									},
									"periodic_reset": schema.StringAttribute{
										Description: "When to reset the usage: 'monthly' or 'weekly'.",
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
										Description: "Type of rate limit: 'requests' or 'tokens'.",
										Computed:    true,
									},
									"unit": schema.StringAttribute{
										Description: "Rate limit unit: 'rpm', 'rph', or 'rpd'.",
										Computed:    true,
									},
									"value": schema.Int64Attribute{
										Description: "The rate limit value.",
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *integrationWorkspacesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *integrationWorkspacesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state integrationWorkspacesDataSourceModel

	// Get integration_id from config
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get workspaces from Portkey API
	workspacesResp, err := d.client.GetIntegrationWorkspaces(ctx, state.IntegrationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read integration workspaces",
			err.Error(),
		)
		return
	}

	// Set ID to integration_id for consistency
	state.ID = state.IntegrationID
	state.Total = types.Int64Value(int64(workspacesResp.Total))

	// Define the types for nested objects (reusing the shared types from resource)
	workspaceObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":           types.StringType,
			"enabled":      types.BoolType,
			"usage_limits": types.ListType{ElemType: workspaceUsageLimitsObjectType},
			"rate_limits":  types.ListType{ElemType: workspaceRateLimitsObjectType},
		},
	}

	// Map response to state
	workspaceAttrs := make([]attr.Value, 0, len(workspacesResp.Workspaces))
	for _, ws := range workspacesResp.Workspaces {
		// Use shared helper functions for conversion
		usageLimitsList, diags := workspaceUsageLimitsToTerraformList(ws.UsageLimits)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		rateLimitsList, diags := workspaceRateLimitsToTerraformList(ws.RateLimits)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Create workspace object
		wsAttrs := map[string]attr.Value{
			"id":           types.StringValue(ws.ID),
			"enabled":      types.BoolValue(ws.Enabled),
			"usage_limits": usageLimitsList,
			"rate_limits":  rateLimitsList,
		}

		wsObj, diags := types.ObjectValue(workspaceObjType.AttrTypes, wsAttrs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		workspaceAttrs = append(workspaceAttrs, wsObj)
	}

	workspacesList, diags := types.ListValue(workspaceObjType, workspaceAttrs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Workspaces = workspacesList

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
