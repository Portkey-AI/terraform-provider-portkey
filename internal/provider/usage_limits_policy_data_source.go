package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &usageLimitsPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &usageLimitsPolicyDataSource{}
)

// NewUsageLimitsPolicyDataSource is a helper function to simplify the provider implementation.
func NewUsageLimitsPolicyDataSource() datasource.DataSource {
	return &usageLimitsPolicyDataSource{}
}

// usageLimitsPolicyDataSource is the data source implementation.
type usageLimitsPolicyDataSource struct {
	client *client.Client
}

// usageLimitsPolicyDataSourceModel maps the data source schema data.
type usageLimitsPolicyDataSourceModel struct {
	ID                types.String  `tfsdk:"id"`
	Name              types.String  `tfsdk:"name"`
	WorkspaceID       types.String  `tfsdk:"workspace_id"`
	Conditions        types.String  `tfsdk:"conditions"`
	GroupBy           types.String  `tfsdk:"group_by"`
	Type              types.String  `tfsdk:"type"`
	CreditLimit       types.Float64 `tfsdk:"credit_limit"`
	AlertThreshold    types.Float64 `tfsdk:"alert_threshold"`
	PeriodicReset     types.String  `tfsdk:"periodic_reset"`
	PeriodicResetDays types.Int64   `tfsdk:"periodic_reset_days"`
	NextUsageResetAt  types.String  `tfsdk:"next_usage_reset_at"`
	Status            types.String  `tfsdk:"status"`
	CreatedAt         types.String  `tfsdk:"created_at"`
	UpdatedAt         types.String  `tfsdk:"updated_at"`
}

// Metadata returns the data source type name.
func (d *usageLimitsPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_usage_limits_policy"
}

// Schema defines the schema for the data source.
func (d *usageLimitsPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a Portkey usage limits policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the usage limits policy to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name for the policy.",
				Computed:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID the policy belongs to.",
				Computed:    true,
			},
			"conditions": schema.StringAttribute{
				Description: "JSON array of conditions.",
				Computed:    true,
			},
			"group_by": schema.StringAttribute{
				Description: "JSON array of group by fields.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Policy type: 'cost' or 'tokens'.",
				Computed:    true,
			},
			"credit_limit": schema.Float64Attribute{
				Description: "Maximum usage allowed.",
				Computed:    true,
			},
			"alert_threshold": schema.Float64Attribute{
				Description: "Alert threshold.",
				Computed:    true,
			},
			"periodic_reset": schema.StringAttribute{
				Description: "Reset period: 'monthly' or 'weekly'.",
				Computed:    true,
			},
			"periodic_reset_days": schema.Int64Attribute{
				Description: "Custom reset interval in days (1–365).",
				Computed:    true,
			},
			"next_usage_reset_at": schema.StringAttribute{
				Description: "ISO8601 datetime for the next scheduled usage reset.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Status of the policy (active, archived).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the policy was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the policy was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *usageLimitsPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *usageLimitsPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state usageLimitsPolicyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := d.client.GetUsageLimitsPolicy(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey Usage Limits Policy",
			err.Error(),
		)
		return
	}

	// Map response body to model
	// "id" is a Required argument, so it must be returned exactly as configured.
	// Terraform rejects a data source whose result differs from its config, and
	// when the read is deferred to apply the mismatch also invalidates the plan
	// of every resource that interpolated this value.
	state.Name = types.StringValue(policy.Name)
	state.WorkspaceID = types.StringValue(policy.WorkspaceID)
	state.Type = types.StringValue(policy.Type)
	state.CreditLimit = types.Float64Value(policy.CreditLimit)
	state.Status = types.StringValue(policy.Status)

	if policy.AlertThreshold != nil {
		state.AlertThreshold = types.Float64Value(*policy.AlertThreshold)
	} else {
		state.AlertThreshold = types.Float64Null()
	}

	if policy.PeriodicReset != "" {
		state.PeriodicReset = types.StringValue(policy.PeriodicReset)
	} else {
		state.PeriodicReset = types.StringNull()
	}

	if policy.PeriodicResetDays != nil {
		state.PeriodicResetDays = types.Int64Value(int64(*policy.PeriodicResetDays))
	} else {
		state.PeriodicResetDays = types.Int64Null()
	}

	if policy.NextUsageResetAt != "" {
		state.NextUsageResetAt = types.StringValue(policy.NextUsageResetAt)
	} else {
		state.NextUsageResetAt = types.StringNull()
	}

	// Convert conditions to JSON string
	if policy.Conditions != nil {
		conditionsBytes, err := json.Marshal(policy.Conditions)
		if err == nil {
			state.Conditions = types.StringValue(string(conditionsBytes))
		}
	}

	// Convert group_by to JSON string
	if policy.GroupBy != nil {
		groupByBytes, err := json.Marshal(policy.GroupBy)
		if err == nil {
			state.GroupBy = types.StringValue(string(groupByBytes))
		}
	}

	state.CreatedAt = types.StringValue(policy.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !policy.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(policy.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
