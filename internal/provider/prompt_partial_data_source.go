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
	_ datasource.DataSource              = &promptPartialDataSource{}
	_ datasource.DataSourceWithConfigure = &promptPartialDataSource{}
)

// NewPromptPartialDataSource is a helper function to simplify the provider implementation.
func NewPromptPartialDataSource() datasource.DataSource {
	return &promptPartialDataSource{}
}

// promptPartialDataSource is the data source implementation.
type promptPartialDataSource struct {
	client *client.Client
}

// promptPartialDataSourceModel maps the data source schema data.
type promptPartialDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Slug                   types.String `tfsdk:"slug"`
	Name                   types.String `tfsdk:"name"`
	Content                types.String `tfsdk:"content"`
	Version                types.String `tfsdk:"version"`
	PartialVersion         types.Int64  `tfsdk:"partial_version"`
	PromptPartialVersionID types.String `tfsdk:"prompt_partial_version_id"`
	Status                 types.String `tfsdk:"status"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
}

// Metadata returns the data source type name.
func (d *promptPartialDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt_partial"
}

// Schema defines the schema for the data source.
func (d *promptPartialDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a Portkey prompt partial.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "The slug of the prompt partial to look up.",
				Required:    true,
			},
			"version": schema.StringAttribute{
				Description: "Version to retrieve: 'latest', 'default', or a specific version number. Defaults to 'default'.",
				Optional:    true,
			},
			"id": schema.StringAttribute{
				Description: "Prompt partial identifier (UUID).",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name for the prompt partial.",
				Computed:    true,
			},
			"content": schema.StringAttribute{
				Description: "The partial template content.",
				Computed:    true,
			},
			"partial_version": schema.Int64Attribute{
				Description: "Version number of the prompt partial.",
				Computed:    true,
			},
			"prompt_partial_version_id": schema.StringAttribute{
				Description: "Version ID of the prompt partial.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Status of the prompt partial (active, archived).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the prompt partial was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the prompt partial was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *promptPartialDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *promptPartialDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state promptPartialDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	version := ""
	if !state.Version.IsNull() {
		version = state.Version.ValueString()
	}

	partial, err := d.client.GetPromptPartial(ctx, state.Slug.ValueString(), version)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey Prompt Partial",
			err.Error(),
		)
		return
	}

	// Map response body to model
	state.ID = types.StringValue(partial.ID)
	state.Slug = types.StringValue(partial.Slug)
	state.Name = types.StringValue(partial.Name)
	state.Content = types.StringValue(partial.String)
	state.PartialVersion = types.Int64Value(int64(partial.Version))
	state.PromptPartialVersionID = types.StringValue(partial.PromptPartialVersionID)
	state.Status = types.StringValue(partial.Status)

	state.CreatedAt = types.StringValue(partial.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !partial.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(partial.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
