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
	_ datasource.DataSource              = &workspacesDataSource{}
	_ datasource.DataSourceWithConfigure = &workspacesDataSource{}
)

// NewWorkspacesDataSource is a helper function to simplify the provider implementation.
func NewWorkspacesDataSource() datasource.DataSource {
	return &workspacesDataSource{}
}

// workspacesDataSource is the data source implementation.
type workspacesDataSource struct {
	client *client.Client
}

// workspacesDataSourceModel maps the data source schema data.
type workspacesDataSourceModel struct {
	Workspaces []workspaceModel `tfsdk:"workspaces"`
}

// workspaceModel maps workspace data
type workspaceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Icon             types.String `tfsdk:"icon"`
	Description      types.String `tfsdk:"description"`
	UsageLimits      types.List   `tfsdk:"usage_limits"`
	RateLimits       types.List   `tfsdk:"rate_limits"`
	Metadata         types.Map    `tfsdk:"metadata"`
	InputGuardrails  types.List   `tfsdk:"input_guardrails"`
	OutputGuardrails types.List   `tfsdk:"output_guardrails"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

// Metadata returns the data source type name.
func (d *workspacesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspaces"
}

// Schema defines the schema for the data source.
func (d *workspacesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all Portkey workspaces in the organization.",
		Attributes: map[string]schema.Attribute{
			"workspaces": schema.ListNestedAttribute{
				Description: "List of workspaces.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Workspace identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the workspace. Includes the icon emoji prefix if an icon is set.",
							Computed:    true,
						},
						"icon": schema.StringAttribute{
							Description: "Emoji icon for the workspace, if set.",
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
										Description: "Alert threshold in dollars.",
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
						"input_guardrails": schema.ListAttribute{
							Description: "Default input guardrail IDs or slugs applied to this workspace.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"output_guardrails": schema.ListAttribute{
							Description: "Default output guardrail IDs or slugs applied to this workspace.",
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *workspacesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *workspacesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state workspacesDataSourceModel

	// Get workspaces from Portkey API
	workspaces, err := d.client.ListWorkspaces(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey Workspaces",
			err.Error(),
		)
		return
	}

	// Map response to state
	for _, workspace := range workspaces {
		// Handle metadata
		var metadataMap types.Map
		if workspace.Defaults != nil && len(workspace.Defaults.Metadata) > 0 {
			mm, diags := types.MapValueFrom(ctx, types.StringType, workspace.Defaults.Metadata)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			metadataMap = mm
		} else {
			metadataMap = types.MapNull(types.StringType)
		}

		// Handle usage_limits
		ulList, ulDiags := workspaceUsageLimitsToTerraformList(workspace.UsageLimits)
		resp.Diagnostics.Append(ulDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Handle rate_limits
		rlList, rlDiags := workspaceRateLimitsToTerraformList(workspace.RateLimits)
		resp.Diagnostics.Append(rlDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Handle icon
		var iconVal types.String
		if workspace.Icon != "" {
			iconVal = types.StringValue(workspace.Icon)
		} else {
			iconVal = types.StringNull()
		}

		// Handle input_guardrails
		var inputGuardrails types.List
		if workspace.Defaults != nil && len(workspace.Defaults.InputGuardrails) > 0 {
			list, gDiags := types.ListValueFrom(ctx, types.StringType, workspace.Defaults.InputGuardrails)
			resp.Diagnostics.Append(gDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			inputGuardrails = list
		} else {
			inputGuardrails = types.ListNull(types.StringType)
		}

		// Handle output_guardrails
		var outputGuardrails types.List
		if workspace.Defaults != nil && len(workspace.Defaults.OutputGuardrails) > 0 {
			list, gDiags := types.ListValueFrom(ctx, types.StringType, workspace.Defaults.OutputGuardrails)
			resp.Diagnostics.Append(gDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			outputGuardrails = list
		} else {
			outputGuardrails = types.ListNull(types.StringType)
		}

		workspaceState := workspaceModel{
			ID:               types.StringValue(workspace.ID),
			Name:             types.StringValue(workspace.Name),
			Icon:             iconVal,
			Description:      types.StringValue(workspace.Description),
			UsageLimits:      ulList,
			RateLimits:       rlList,
			Metadata:         metadataMap,
			InputGuardrails:  inputGuardrails,
			OutputGuardrails: outputGuardrails,
			CreatedAt:        types.StringValue(workspace.CreatedAt.Format("2006-01-02T15:04:05Z07:00")),
			UpdatedAt:        types.StringValue(workspace.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")),
		}
		state.Workspaces = append(state.Workspaces, workspaceState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
