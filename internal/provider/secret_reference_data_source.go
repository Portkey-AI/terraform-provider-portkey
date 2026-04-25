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
	_ datasource.DataSource              = &secretReferenceDataSource{}
	_ datasource.DataSourceWithConfigure = &secretReferenceDataSource{}
)

// NewSecretReferenceDataSource is a helper function to simplify the provider implementation.
func NewSecretReferenceDataSource() datasource.DataSource {
	return &secretReferenceDataSource{}
}

type secretReferenceDataSource struct {
	client *client.Client
}

// Data source model. Mirrors the resource model but omits the 9 typed auth
// blocks, since the API returns auth_config with secrets masked and we don't
// want to surface those to a data source consumer.
type secretReferenceDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Slug               types.String `tfsdk:"slug"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	ManagerType        types.String `tfsdk:"manager_type"`
	SecretPath         types.String `tfsdk:"secret_path"`
	SecretKey          types.String `tfsdk:"secret_key"`
	AllowAllWorkspaces types.Bool   `tfsdk:"allow_all_workspaces"`
	AllowedWorkspaces  types.Set    `tfsdk:"allowed_workspaces"`
	Tags               types.Map    `tfsdk:"tags"`
	Status             types.String `tfsdk:"status"`
	CreatedBy          types.String `tfsdk:"created_by"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

// Metadata returns the data source type name.
func (d *secretReferenceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_reference"
}

// Schema defines the schema for the data source.
func (d *secretReferenceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a specific Portkey secret reference by UUID or slug. " +
			"Credential values (auth_config) are intentionally not exposed by this data source.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "Slug or UUID of the secret reference. Either works.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "Secret reference UUID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description.",
				Computed:    true,
			},
			"manager_type": schema.StringAttribute{
				Description: "Secret manager type. Available options: `aws_sm`, `azure_kv`, `hashicorp_vault`.",
				Computed:    true,
			},
			"secret_path": schema.StringAttribute{
				Description: "Path to the secret in the external manager.",
				Computed:    true,
			},
			"secret_key": schema.StringAttribute{
				Description: "Optional key within the secret payload.",
				Computed:    true,
			},
			"allow_all_workspaces": schema.BoolAttribute{
				Description: "Whether all workspaces can use this secret reference.",
				Computed:    true,
			},
			"allowed_workspaces": schema.SetAttribute{
				Description: "Set of workspace UUIDs or slugs allowed to use this secret reference.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"tags": schema.MapAttribute{
				Description: "Custom metadata tags.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"status": schema.StringAttribute{
				Description: "Status of the secret reference.",
				Computed:    true,
			},
			"created_by": schema.StringAttribute{
				Description: "Identity that created the secret reference.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the secret reference was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the secret reference was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *secretReferenceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read refreshes the Terraform state with the latest data.
func (d *secretReferenceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state secretReferenceDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretRef, err := d.client.GetSecretReference(ctx, state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey Secret Reference",
			err.Error(),
		)
		return
	}

	state.ID = types.StringValue(secretRef.ID)
	// Preserve whatever the user configured (slug or UUID) — do not overwrite
	// a Required input with the canonical slug. The canonical value is available
	// via `id` (UUID) and, if the user configured a slug, it already matches.
	state.Name = types.StringValue(secretRef.Name)
	state.ManagerType = types.StringValue(secretRef.ManagerType)
	state.SecretPath = types.StringValue(secretRef.SecretPath)
	state.AllowAllWorkspaces = types.BoolValue(secretRef.AllowAllWorkspaces)

	if secretRef.Description != "" {
		state.Description = types.StringValue(secretRef.Description)
	} else {
		state.Description = types.StringNull()
	}
	if secretRef.SecretKey != "" {
		state.SecretKey = types.StringValue(secretRef.SecretKey)
	} else {
		state.SecretKey = types.StringNull()
	}

	if len(secretRef.AllowedWorkspaces) > 0 {
		ws, convDiags := types.SetValueFrom(ctx, types.StringType, secretRef.AllowedWorkspaces)
		resp.Diagnostics.Append(convDiags...)
		state.AllowedWorkspaces = ws
	} else {
		state.AllowedWorkspaces = types.SetNull(types.StringType)
	}

	if len(secretRef.Tags) > 0 {
		tags, convDiags := types.MapValueFrom(ctx, types.StringType, secretRef.Tags)
		resp.Diagnostics.Append(convDiags...)
		state.Tags = tags
	} else {
		state.Tags = types.MapNull(types.StringType)
	}

	state.Status = types.StringValue(secretRef.Status)
	state.CreatedBy = types.StringValue(secretRef.CreatedBy)
	state.CreatedAt = types.StringValue(secretRef.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !secretRef.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(secretRef.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		state.UpdatedAt = types.StringNull()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
