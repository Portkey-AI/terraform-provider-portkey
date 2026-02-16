package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &apiKeysDataSource{}
	_ datasource.DataSourceWithConfigure = &apiKeysDataSource{}
)

// NewAPIKeysDataSource is a helper function to simplify the provider implementation.
func NewAPIKeysDataSource() datasource.DataSource {
	return &apiKeysDataSource{}
}

// apiKeysDataSource is the data source implementation.
type apiKeysDataSource struct {
	client *client.Client
}

// apiKeysDataSourceModel maps the data source schema data.
type apiKeysDataSourceModel struct {
	WorkspaceID types.String          `tfsdk:"workspace_id"`
	APIKeys     []apiKeyDataItemModel `tfsdk:"api_keys"`
}

// apiKeyDataItemModel maps individual API key data.
type apiKeyDataItemModel struct {
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
func (d *apiKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_keys"
}

// Schema defines the schema for the data source.
func (d *apiKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all Portkey API Keys. Optionally filter by workspace_id.",
		Attributes: map[string]schema.Attribute{
			"workspace_id": schema.StringAttribute{
				Description: "Optional workspace ID to filter API keys.",
				Optional:    true,
			},
			"api_keys": schema.ListNestedAttribute{
				Description: "List of API keys.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "API Key identifier (UUID).",
							Computed:    true,
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
								"credits_limit": schema.Float64Attribute{
									Description: "The credit limit value.",
									Computed:    true,
								},
								"credits_limit_type": schema.StringAttribute{
									Description: "Period for the credit limit.",
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *apiKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *apiKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state apiKeysDataSourceModel

	// Get config
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get workspace filter
	workspaceID := ""
	if !state.WorkspaceID.IsNull() && !state.WorkspaceID.IsUnknown() {
		workspaceID = state.WorkspaceID.ValueString()
	}

	// Get API keys from Portkey
	apiKeys, err := d.client.ListAPIKeys(ctx, workspaceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey API Keys",
			err.Error(),
		)
		return
	}

	// Map response to state
	state.APIKeys = make([]apiKeyDataItemModel, 0, len(apiKeys))
	for _, apiKey := range apiKeys {
		// Parse type
		parsedType, parsedSubType := parseAPIKeyTypeList(apiKey.Type)

		// Handle scopes
		var scopesList types.List
		if len(apiKey.Scopes) > 0 {
			sl, diags := types.ListValueFrom(ctx, types.StringType, apiKey.Scopes)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			scopesList = sl
		} else {
			scopesList = types.ListNull(types.StringType)
		}

		// Handle metadata
		var metadataMap types.Map
		if apiKey.Defaults != nil && len(apiKey.Defaults.Metadata) > 0 {
			mm, diags := types.MapValueFrom(ctx, types.StringType, apiKey.Defaults.Metadata)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			metadataMap = mm
		} else {
			metadataMap = types.MapNull(types.StringType)
		}

		// Handle alert_emails
		var alertEmailsList types.List
		if len(apiKey.AlertEmails) > 0 {
			ael, diags := types.ListValueFrom(ctx, types.StringType, apiKey.AlertEmails)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			alertEmailsList = ael
		} else {
			alertEmailsList = types.ListNull(types.StringType)
		}

		// Handle usage_limits
		ulObj, ulDiags := apiKeyUsageLimitsToTerraform(apiKey.UsageLimits)
		resp.Diagnostics.Append(ulDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Handle rate_limits
		rlList, rlDiags := apiKeyRateLimitsToTerraformList(apiKey.RateLimits)
		resp.Diagnostics.Append(rlDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		keyItem := apiKeyDataItemModel{
			ID:             types.StringValue(apiKey.ID),
			Name:           types.StringValue(apiKey.Name),
			Type:           types.StringValue(parsedType),
			SubType:        types.StringValue(parsedSubType),
			OrganisationID: types.StringValue(apiKey.OrganisationID),
			Status:         types.StringValue(apiKey.Status),
			Scopes:         scopesList,
			RateLimits:     rlList,
			UsageLimits:    ulObj,
			Metadata:       metadataMap,
			AlertEmails:    alertEmailsList,
			CreatedAt:      types.StringValue(apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00")),
		}

		if apiKey.Description != "" {
			keyItem.Description = types.StringValue(apiKey.Description)
		} else {
			keyItem.Description = types.StringNull()
		}

		if apiKey.WorkspaceID != "" {
			keyItem.WorkspaceID = types.StringValue(apiKey.WorkspaceID)
		} else {
			keyItem.WorkspaceID = types.StringNull()
		}

		if apiKey.UserID != "" {
			keyItem.UserID = types.StringValue(apiKey.UserID)
		} else {
			keyItem.UserID = types.StringNull()
		}

		if !apiKey.UpdatedAt.IsZero() {
			keyItem.UpdatedAt = types.StringValue(apiKey.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
		} else {
			keyItem.UpdatedAt = types.StringNull()
		}

		state.APIKeys = append(state.APIKeys, keyItem)
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// parseAPIKeyTypeList parses the combined type field for list data source
func parseAPIKeyTypeList(combinedType string) (keyType, subType string) {
	parts := strings.SplitN(combinedType, "-", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return combinedType, ""
}
