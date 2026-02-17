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
	_ datasource.DataSource              = &apiKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &apiKeyDataSource{}
)

// NewAPIKeyDataSource is a helper function to simplify the provider implementation.
func NewAPIKeyDataSource() datasource.DataSource {
	return &apiKeyDataSource{}
}

// apiKeyDataSource is the data source implementation.
type apiKeyDataSource struct {
	client *client.Client
}

// apiKeyDataSourceModel maps the data source schema data.
type apiKeyDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Type           types.String `tfsdk:"type"`
	SubType        types.String `tfsdk:"sub_type"`
	OrganisationID types.String `tfsdk:"organisation_id"`
	WorkspaceID    types.String `tfsdk:"workspace_id"`
	UserID         types.String `tfsdk:"user_id"`
	Status         types.String `tfsdk:"status"`
	Scopes         types.List   `tfsdk:"scopes"`
	RateLimits     types.List   `tfsdk:"rate_limits"`
	UsageLimits    types.Object `tfsdk:"usage_limits"`
	Metadata       types.Map    `tfsdk:"metadata"`
	AlertEmails    types.List   `tfsdk:"alert_emails"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

// Metadata returns the data source type name.
func (d *apiKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

// Schema defines the schema for the data source.
func (d *apiKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a Portkey API Key by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "API Key identifier (UUID).",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name for the API key.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the API key.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type of API key: 'organisation' or 'workspace'.",
				Computed:    true,
			},
			"sub_type": schema.StringAttribute{
				Description: "Sub-type of API key: 'service' or 'user'.",
				Computed:    true,
			},
			"organisation_id": schema.StringAttribute{
				Description: "Organisation ID this key belongs to.",
				Computed:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID for workspace-level keys.",
				Computed:    true,
			},
			"user_id": schema.StringAttribute{
				Description: "User ID for user-type keys.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Status of the API key (active, exhausted).",
				Computed:    true,
			},
			"scopes": schema.ListAttribute{
				Description: "List of permission scopes for this API key.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"usage_limits": schema.SingleNestedAttribute{
				Description: "Usage limits for this API key.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
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
			"rate_limits": schema.ListNestedAttribute{
				Description: "Rate limits for this API key.",
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
				Description: "Custom metadata attached to the API key.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"alert_emails": schema.ListAttribute{
				Description: "List of email addresses that receive alerts for this API key.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the API key was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the API key was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *apiKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *apiKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state apiKeyDataSourceModel

	// Get config
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get API key from Portkey
	apiKey, err := d.client.GetAPIKey(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey API Key",
			err.Error(),
		)
		return
	}

	// Map response to state
	state.Name = types.StringValue(apiKey.Name)
	state.OrganisationID = types.StringValue(apiKey.OrganisationID)
	state.Status = types.StringValue(apiKey.Status)

	// Parse and set type/subtype from combined type field
	parsedType, parsedSubType := parseAPIKeyType(apiKey.Type)
	state.Type = types.StringValue(parsedType)
	state.SubType = types.StringValue(parsedSubType)

	if apiKey.Description != "" {
		state.Description = types.StringValue(apiKey.Description)
	} else {
		state.Description = types.StringNull()
	}

	if apiKey.WorkspaceID != "" {
		state.WorkspaceID = types.StringValue(apiKey.WorkspaceID)
	} else {
		state.WorkspaceID = types.StringNull()
	}

	if apiKey.UserID != "" {
		state.UserID = types.StringValue(apiKey.UserID)
	} else {
		state.UserID = types.StringNull()
	}

	// Handle scopes
	if len(apiKey.Scopes) > 0 {
		scopesList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.Scopes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Scopes = scopesList
	} else {
		state.Scopes = types.ListNull(types.StringType)
	}

	// Handle usage_limits
	ulObj, ulDiags := apiKeyUsageLimitsToTerraform(apiKey.UsageLimits)
	resp.Diagnostics.Append(ulDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.UsageLimits = ulObj

	// Handle rate_limits
	rlList, rlDiags := apiKeyRateLimitsToTerraformList(apiKey.RateLimits)
	resp.Diagnostics.Append(rlDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.RateLimits = rlList

	// Handle metadata
	if apiKey.Defaults != nil && len(apiKey.Defaults.Metadata) > 0 {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, apiKey.Defaults.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Metadata = metadataMap
	} else {
		state.Metadata = types.MapNull(types.StringType)
	}

	// Handle alert_emails
	if len(apiKey.AlertEmails) > 0 {
		alertEmailsList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.AlertEmails)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.AlertEmails = alertEmailsList
	} else {
		state.AlertEmails = types.ListNull(types.StringType)
	}

	state.CreatedAt = types.StringValue(apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !apiKey.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(apiKey.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		state.UpdatedAt = types.StringNull()
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
