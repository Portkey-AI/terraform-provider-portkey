package provider

import (
	"context"
	"fmt"

	"github.com/portkey-ai/terraform-provider-portkey/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &integrationDataSource{}
	_ datasource.DataSourceWithConfigure = &integrationDataSource{}
)

// NewIntegrationDataSource is a helper function to simplify the provider implementation.
func NewIntegrationDataSource() datasource.DataSource {
	return &integrationDataSource{}
}

// integrationDataSource is the data source implementation.
type integrationDataSource struct {
	client *client.Client
}

// integrationDataSourceModel maps the data source schema data.
type integrationDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Slug           types.String `tfsdk:"slug"`
	Name           types.String `tfsdk:"name"`
	AIProviderID   types.String `tfsdk:"ai_provider_id"`
	Description    types.String `tfsdk:"description"`
	Status         types.String `tfsdk:"status"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	SecretMappings types.List   `tfsdk:"secret_mappings"`
}

// Metadata returns the data source type name.
func (d *integrationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

// Schema defines the schema for the data source.
func (d *integrationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a specific Portkey integration by slug.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "Integration slug identifier.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "Integration UUID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name of the integration.",
				Computed:    true,
			},
			"ai_provider_id": schema.StringAttribute{
				Description: "ID of the AI provider (e.g., 'openai', 'anthropic').",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the integration.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Status of the integration (active, archived).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the integration was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the integration was last updated.",
				Computed:    true,
			},
			"secret_mappings": schema.ListNestedAttribute{
				Description: "Secret reference mappings configured on this integration. Each entry resolves a single integration field (`key` or `configurations.<field>`) from a `portkey_secret_reference` at request time.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"target_field": schema.StringAttribute{
							Description: "Integration field populated by the mapping (`key` or `configurations.<field>`).",
							Computed:    true,
						},
						"secret_reference_id": schema.StringAttribute{
							Description: "UUID or slug of the referenced `portkey_secret_reference`.",
							Computed:    true,
						},
						"secret_key": schema.StringAttribute{
							Description: "Optional override for the secret reference's `secret_key`.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *integrationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *integrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state integrationDataSourceModel

	// Get the slug from the configuration
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get integration from Portkey API
	integration, err := d.client.GetIntegration(ctx, state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey Integration",
			err.Error(),
		)
		return
	}

	// Map response to state
	state.ID = types.StringValue(integration.ID)
	state.Slug = types.StringValue(integration.Slug)
	state.Name = types.StringValue(integration.Name)
	state.AIProviderID = types.StringValue(integration.AIProviderID)
	state.Status = types.StringValue(integration.Status)
	if integration.Description != "" {
		state.Description = types.StringValue(integration.Description)
	} else {
		state.Description = types.StringNull()
	}
	state.CreatedAt = types.StringValue(integration.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !integration.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(integration.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		state.UpdatedAt = types.StringNull()
	}

	// Project secret_mappings from the API response. The data-source variant
	// uses a List (order-preserving) since the server's returned order is the
	// only meaningful one for a read-only view.
	smObjType := types.ObjectType{AttrTypes: secretMappingAttrTypes}
	smElems := make([]attr.Value, 0, len(integration.SecretMappings))
	for _, m := range integration.SecretMappings {
		attrs := map[string]attr.Value{
			"target_field":        types.StringValue(m.TargetField),
			"secret_reference_id": types.StringValue(m.SecretReferenceID),
			"secret_key":          types.StringNull(),
		}
		if m.SecretKey != nil {
			attrs["secret_key"] = types.StringValue(*m.SecretKey)
		}
		obj, objDiags := types.ObjectValue(secretMappingAttrTypes, attrs)
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		smElems = append(smElems, obj)
	}
	smList, listDiags := types.ListValue(smObjType, smElems)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.SecretMappings = smList

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
