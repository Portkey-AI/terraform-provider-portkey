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
	_ datasource.DataSource              = &integrationModelsDataSource{}
	_ datasource.DataSourceWithConfigure = &integrationModelsDataSource{}
)

// NewIntegrationModelsDataSource is a helper function to simplify the provider implementation.
func NewIntegrationModelsDataSource() datasource.DataSource {
	return &integrationModelsDataSource{}
}

// integrationModelsDataSource is the data source implementation.
type integrationModelsDataSource struct {
	client *client.Client
}

// integrationModelsDataSourceModel maps the data source schema data.
type integrationModelsDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	IntegrationID  types.String `tfsdk:"integration_id"`
	AllowAllModels types.Bool   `tfsdk:"allow_all_models"`
	Models         types.List   `tfsdk:"models"`
}

// Metadata returns the data source type name.
func (d *integrationModelsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_models"
}

// Schema defines the schema for the data source.
func (d *integrationModelsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches model access configuration for a Portkey integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Data source identifier (same as integration_id).",
				Computed:    true,
			},
			"integration_id": schema.StringAttribute{
				Description: "The integration slug or ID to query model access for.",
				Required:    true,
			},
			"allow_all_models": schema.BoolAttribute{
				Description: "Whether all models are allowed by default for this integration.",
				Computed:    true,
			},
			"models": schema.ListNestedAttribute{
				Description: "List of model access configurations.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"slug": schema.StringAttribute{
							Description: "Model identifier (slug).",
							Computed:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether the model is enabled for this integration.",
							Computed:    true,
						},
						"is_custom": schema.BoolAttribute{
							Description: "Whether this is a custom model.",
							Computed:    true,
						},
						"is_finetune": schema.BoolAttribute{
							Description: "Whether this is a fine-tuned model.",
							Computed:    true,
						},
						"base_model_slug": schema.StringAttribute{
							Description: "Base model slug for fine-tuned models.",
							Computed:    true,
						},
						"pricing_config": schema.SingleNestedAttribute{
							Description: "Pricing configuration for this model.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Description: "Pricing type.",
									Computed:    true,
								},
								"pay_as_you_go": schema.SingleNestedAttribute{
									Description: "Pay-as-you-go pricing configuration.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"request_token_price": schema.Float64Attribute{
											Description: "Price per request token.",
											Computed:    true,
										},
										"response_token_price": schema.Float64Attribute{
											Description: "Price per response token.",
											Computed:    true,
										},
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
func (d *integrationModelsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *integrationModelsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state integrationModelsDataSourceModel

	// Get integration_id from config
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get models from Portkey API
	modelsResp, err := d.client.GetIntegrationModels(ctx, state.IntegrationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read integration models",
			err.Error(),
		)
		return
	}

	// Set ID to integration_id for consistency
	state.ID = state.IntegrationID
	state.AllowAllModels = types.BoolValue(modelsResp.AllowAllModels)

	// Define the types for nested objects
	modelObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"slug":            types.StringType,
			"enabled":         types.BoolType,
			"is_custom":       types.BoolType,
			"is_finetune":     types.BoolType,
			"base_model_slug": types.StringType,
			"pricing_config":  pricingConfigObjectType,
		},
	}

	// Map response to state
	modelAttrs := make([]attr.Value, 0, len(modelsResp.Models))
	for _, m := range modelsResp.Models {
		// Build pricing config
		var pricingConfigVal attr.Value
		if m.PricingConfig != nil {
			pricingAttrs := map[string]attr.Value{
				"type": types.StringValue(m.PricingConfig.Type),
			}

			if m.PricingConfig.PayAsYouGo != nil {
				paygAttrs := map[string]attr.Value{
					"request_token_price":  types.Float64Null(),
					"response_token_price": types.Float64Null(),
				}
				if m.PricingConfig.PayAsYouGo.RequestToken != nil {
					paygAttrs["request_token_price"] = types.Float64Value(m.PricingConfig.PayAsYouGo.RequestToken.Price)
				}
				if m.PricingConfig.PayAsYouGo.ResponseToken != nil {
					paygAttrs["response_token_price"] = types.Float64Value(m.PricingConfig.PayAsYouGo.ResponseToken.Price)
				}

				paygObj, d := types.ObjectValue(payAsYouGoAttrTypes, paygAttrs)
				resp.Diagnostics.Append(d...)
				if resp.Diagnostics.HasError() {
					return
				}
				pricingAttrs["pay_as_you_go"] = paygObj
			} else {
				pricingAttrs["pay_as_you_go"] = types.ObjectNull(payAsYouGoAttrTypes)
			}

			pricingObj, d := types.ObjectValue(pricingConfigAttrTypes, pricingAttrs)
			resp.Diagnostics.Append(d...)
			if resp.Diagnostics.HasError() {
				return
			}
			pricingConfigVal = pricingObj
		} else {
			pricingConfigVal = types.ObjectNull(pricingConfigAttrTypes)
		}

		// Build model object
		baseModelSlug := types.StringNull()
		if m.BaseModelSlug != "" {
			baseModelSlug = types.StringValue(m.BaseModelSlug)
		}

		mAttrs := map[string]attr.Value{
			"slug":            types.StringValue(m.Slug),
			"enabled":         types.BoolValue(m.Enabled),
			"is_custom":       types.BoolValue(m.IsCustom),
			"is_finetune":     types.BoolValue(m.IsFinetune),
			"base_model_slug": baseModelSlug,
			"pricing_config":  pricingConfigVal,
		}

		mObj, d := types.ObjectValue(modelObjType.AttrTypes, mAttrs)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		modelAttrs = append(modelAttrs, mObj)
	}

	modelsList, diags := types.ListValue(modelObjType, modelAttrs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Models = modelsList

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
