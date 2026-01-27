package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &integrationModelAccessResource{}
	_ resource.ResourceWithConfigure   = &integrationModelAccessResource{}
	_ resource.ResourceWithImportState = &integrationModelAccessResource{}
)

// Type definitions for nested attributes
var (
	payAsYouGoAttrTypes = map[string]attr.Type{
		"request_token_price":  types.Float64Type,
		"response_token_price": types.Float64Type,
	}

	pricingConfigAttrTypes = map[string]attr.Type{
		"type": types.StringType,
		"pay_as_you_go": types.ObjectType{
			AttrTypes: payAsYouGoAttrTypes,
		},
	}

	pricingConfigObjectType = types.ObjectType{AttrTypes: pricingConfigAttrTypes}
)

// NewIntegrationModelAccessResource is a helper function to simplify the provider implementation.
func NewIntegrationModelAccessResource() resource.Resource {
	return &integrationModelAccessResource{}
}

// integrationModelAccessResource is the resource implementation.
type integrationModelAccessResource struct {
	client *client.Client
}

// integrationModelAccessResourceModel maps the resource schema data.
type integrationModelAccessResourceModel struct {
	ID            types.String `tfsdk:"id"`
	IntegrationID types.String `tfsdk:"integration_id"`
	ModelSlug     types.String `tfsdk:"model_slug"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	IsCustom      types.Bool   `tfsdk:"is_custom"`
	IsFinetune    types.Bool   `tfsdk:"is_finetune"`
	BaseModelSlug types.String `tfsdk:"base_model_slug"`
	PricingConfig types.Object `tfsdk:"pricing_config"`
}

// pricingConfigModel maps the pricing_config block
type pricingConfigModel struct {
	Type       types.String `tfsdk:"type"`
	PayAsYouGo types.Object `tfsdk:"pay_as_you_go"`
}

// payAsYouGoModel maps the pay_as_you_go block
type payAsYouGoModel struct {
	RequestTokenPrice  types.Float64 `tfsdk:"request_token_price"`
	ResponseTokenPrice types.Float64 `tfsdk:"response_token_price"`
}

// Metadata returns the resource type name.
func (r *integrationModelAccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_model_access"
}

// Schema defines the schema for the resource.
func (r *integrationModelAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages model access for a Portkey integration. Enables or disables specific models for an integration, optionally with custom pricing.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier in format integration_id/model_slug.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"integration_id": schema.StringAttribute{
				Description: "The integration slug or ID to configure model access for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"model_slug": schema.StringAttribute{
				Description: "The model identifier (slug).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the model is enabled for this integration. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"is_custom": schema.BoolAttribute{
				Description: "Whether this is a custom model (not a built-in provider model). Custom models can be deleted; built-in models can only be disabled.",
				Optional:    true,
				Computed:    true,
			},
			"is_finetune": schema.BoolAttribute{
				Description: "Whether this is a fine-tuned model.",
				Optional:    true,
				Computed:    true,
			},
			"base_model_slug": schema.StringAttribute{
				Description: "The base model slug for fine-tuned models.",
				Optional:    true,
				Computed:    true,
			},
			"pricing_config": schema.SingleNestedAttribute{
				Description: "Custom pricing configuration for this model.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Pricing type: 'static' for fixed pricing.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("static"),
						},
					},
					"pay_as_you_go": schema.SingleNestedAttribute{
						Description: "Pay-as-you-go pricing configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"request_token_price": schema.Float64Attribute{
								Description: "Price per request token (input).",
								Optional:    true,
								Validators: []validator.Float64{
									float64validator.AtLeast(0),
								},
							},
							"response_token_price": schema.Float64Attribute{
								Description: "Price per response token (output).",
								Optional:    true,
								Validators: []validator.Float64{
									float64validator.AtLeast(0),
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *integrationModelAccessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *integrationModelAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan integrationModelAccessResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the model update request
	modelReq, diags := buildModelUpdateRequest(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create/update model access
	err := r.client.UpdateIntegrationModel(ctx, plan.IntegrationID.ValueString(), modelReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating integration model access",
			"Could not create integration model access: "+err.Error(),
		)
		return
	}

	// Set the ID
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", plan.IntegrationID.ValueString(), plan.ModelSlug.ValueString()))

	// Fetch the actual state from API to ensure consistency
	model, err := r.client.GetIntegrationModel(ctx, plan.IntegrationID.ValueString(), plan.ModelSlug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading integration model access after creation",
			"Could not read integration model access: "+err.Error(),
		)
		return
	}

	// Update plan with actual values from API
	diags = mapModelToState(ctx, model, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *integrationModelAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state integrationModelAccessResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed model access from Portkey
	model, err := r.client.GetIntegrationModel(ctx, state.IntegrationID.ValueString(), state.ModelSlug.ValueString())
	if err != nil {
		// Check if this is a "not found" error - if so, remove from state
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading integration model access",
			"Could not read integration model access for model "+state.ModelSlug.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state with refreshed values
	diags = mapModelToState(ctx, model, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *integrationModelAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan integrationModelAccessResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the model update request
	modelReq, diags := buildModelUpdateRequest(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update model access
	err := r.client.UpdateIntegrationModel(ctx, plan.IntegrationID.ValueString(), modelReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating integration model access",
			"Could not update integration model access: "+err.Error(),
		)
		return
	}

	// Fetch the actual state from API to ensure consistency
	model, err := r.client.GetIntegrationModel(ctx, plan.IntegrationID.ValueString(), plan.ModelSlug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading integration model access after update",
			"Could not read integration model access: "+err.Error(),
		)
		return
	}

	// Update plan with actual values from API
	diags = mapModelToState(ctx, model, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *integrationModelAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state integrationModelAccessResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if resource still exists before attempting to delete/disable
	model, err := r.client.GetIntegrationModel(ctx, state.IntegrationID.ValueString(), state.ModelSlug.ValueString())
	if err != nil {
		// If not found, resource is already gone - success
		if strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting integration model access",
			"Could not verify integration model access exists: "+err.Error(),
		)
		return
	}

	// For custom models, use DELETE endpoint to fully remove
	if model.IsCustom || state.IsCustom.ValueBool() {
		err = r.client.DeleteIntegrationModels(ctx, state.IntegrationID.ValueString(), []string{state.ModelSlug.ValueString()})
		if err != nil {
			// If not found during delete, resource is already gone - success
			if strings.Contains(err.Error(), "not found") {
				return
			}
			resp.Diagnostics.AddError(
				"Error deleting custom model",
				"Could not delete custom model: "+err.Error(),
			)
			return
		}
	} else {
		// For built-in models, disable via PUT (set enabled=false)
		modelReq := client.IntegrationModel{
			Slug:    state.ModelSlug.ValueString(),
			Enabled: false,
		}

		err = r.client.UpdateIntegrationModel(ctx, state.IntegrationID.ValueString(), modelReq)
		if err != nil {
			// If not found during disable, resource is already gone - success
			if strings.Contains(err.Error(), "not found") {
				return
			}
			resp.Diagnostics.AddError(
				"Error disabling integration model access",
				"Could not disable integration model access: "+err.Error(),
			)
			return
		}
	}
}

// ImportState imports the resource state.
func (r *integrationModelAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID should be in format: integration_id/model_slug
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in format: integration_id/model_slug",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("integration_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("model_slug"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// Helper functions

// buildModelUpdateRequest builds a client.IntegrationModel from the resource model
func buildModelUpdateRequest(ctx context.Context, plan *integrationModelAccessResourceModel) (client.IntegrationModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	modelReq := client.IntegrationModel{
		Slug:    plan.ModelSlug.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
	}

	// Set custom model flags
	if !plan.IsCustom.IsNull() && !plan.IsCustom.IsUnknown() {
		modelReq.IsCustom = plan.IsCustom.ValueBool()
	}
	if !plan.IsFinetune.IsNull() && !plan.IsFinetune.IsUnknown() {
		modelReq.IsFinetune = plan.IsFinetune.ValueBool()
	}
	if !plan.BaseModelSlug.IsNull() && !plan.BaseModelSlug.IsUnknown() {
		modelReq.BaseModelSlug = plan.BaseModelSlug.ValueString()
	}

	// Parse pricing config
	if !plan.PricingConfig.IsNull() && !plan.PricingConfig.IsUnknown() {
		var pricingConfig pricingConfigModel
		diags.Append(plan.PricingConfig.As(ctx, &pricingConfig, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return modelReq, diags
		}

		modelReq.PricingConfig = &client.ModelPricingConfig{
			Type: pricingConfig.Type.ValueString(),
		}

		// Parse pay_as_you_go if present
		if !pricingConfig.PayAsYouGo.IsNull() && !pricingConfig.PayAsYouGo.IsUnknown() {
			var payg payAsYouGoModel
			diags.Append(pricingConfig.PayAsYouGo.As(ctx, &payg, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				return modelReq, diags
			}

			modelReq.PricingConfig.PayAsYouGo = &client.PayAsYouGoPricing{}

			if !payg.RequestTokenPrice.IsNull() && !payg.RequestTokenPrice.IsUnknown() {
				modelReq.PricingConfig.PayAsYouGo.RequestToken = &client.TokenPrice{
					Price: payg.RequestTokenPrice.ValueFloat64(),
				}
			}
			if !payg.ResponseTokenPrice.IsNull() && !payg.ResponseTokenPrice.IsUnknown() {
				modelReq.PricingConfig.PayAsYouGo.ResponseToken = &client.TokenPrice{
					Price: payg.ResponseTokenPrice.ValueFloat64(),
				}
			}
		}
	}

	return modelReq, diags
}

// mapModelToState maps client.IntegrationModel to the Terraform state model
func mapModelToState(ctx context.Context, model *client.IntegrationModel, state *integrationModelAccessResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	state.Enabled = types.BoolValue(model.Enabled)
	state.IsCustom = types.BoolValue(model.IsCustom)
	state.IsFinetune = types.BoolValue(model.IsFinetune)

	if model.BaseModelSlug != "" {
		state.BaseModelSlug = types.StringValue(model.BaseModelSlug)
	} else {
		state.BaseModelSlug = types.StringNull()
	}

	// Map pricing config
	if model.PricingConfig != nil {
		pricingAttrs := map[string]attr.Value{
			"type": types.StringValue(model.PricingConfig.Type),
		}

		// Map pay_as_you_go if present
		if model.PricingConfig.PayAsYouGo != nil {
			paygAttrs := map[string]attr.Value{
				"request_token_price":  types.Float64Null(),
				"response_token_price": types.Float64Null(),
			}

			if model.PricingConfig.PayAsYouGo.RequestToken != nil {
				paygAttrs["request_token_price"] = types.Float64Value(model.PricingConfig.PayAsYouGo.RequestToken.Price)
			}
			if model.PricingConfig.PayAsYouGo.ResponseToken != nil {
				paygAttrs["response_token_price"] = types.Float64Value(model.PricingConfig.PayAsYouGo.ResponseToken.Price)
			}

			paygObj, d := types.ObjectValue(payAsYouGoAttrTypes, paygAttrs)
			diags.Append(d...)
			if diags.HasError() {
				return diags
			}
			pricingAttrs["pay_as_you_go"] = paygObj
		} else {
			pricingAttrs["pay_as_you_go"] = types.ObjectNull(payAsYouGoAttrTypes)
		}

		pricingObj, d := types.ObjectValue(pricingConfigAttrTypes, pricingAttrs)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		state.PricingConfig = pricingObj
	} else {
		state.PricingConfig = types.ObjectNull(pricingConfigAttrTypes)
	}

	return diags
}
