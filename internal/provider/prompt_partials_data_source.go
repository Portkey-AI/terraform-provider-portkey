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
	_ datasource.DataSource              = &promptPartialsDataSource{}
	_ datasource.DataSourceWithConfigure = &promptPartialsDataSource{}
)

// NewPromptPartialsDataSource is a helper function to simplify the provider implementation.
func NewPromptPartialsDataSource() datasource.DataSource {
	return &promptPartialsDataSource{}
}

// promptPartialsDataSource is the data source implementation.
type promptPartialsDataSource struct {
	client *client.Client
}

// promptPartialsDataSourceModel maps the data source schema data.
type promptPartialsDataSourceModel struct {
	WorkspaceID    types.String                `tfsdk:"workspace_id"`
	PromptPartials []promptPartialSummaryModel `tfsdk:"prompt_partials"`
}

// promptPartialSummaryModel maps prompt partial summary data.
type promptPartialSummaryModel struct {
	ID        types.String `tfsdk:"id"`
	Slug      types.String `tfsdk:"slug"`
	Name      types.String `tfsdk:"name"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// Metadata returns the data source type name.
func (d *promptPartialsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt_partials"
}

// Schema defines the schema for the data source.
func (d *promptPartialsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get a list of Portkey prompt partials.",
		Attributes: map[string]schema.Attribute{
			"workspace_id": schema.StringAttribute{
				Description: "Optional workspace ID to filter prompt partials.",
				Optional:    true,
			},
			"prompt_partials": schema.ListNestedAttribute{
				Description: "List of prompt partials.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Prompt partial identifier (UUID).",
							Computed:    true,
						},
						"slug": schema.StringAttribute{
							Description: "URL-friendly identifier for the prompt partial.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Human-readable name for the prompt partial.",
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *promptPartialsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *promptPartialsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state promptPartialsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := ""
	if !state.WorkspaceID.IsNull() {
		workspaceID = state.WorkspaceID.ValueString()
	}

	partials, err := d.client.ListPromptPartials(ctx, workspaceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey Prompt Partials",
			err.Error(),
		)
		return
	}

	// Map response body to model
	for _, partial := range partials {
		partialState := promptPartialSummaryModel{
			ID:        types.StringValue(partial.ID),
			Slug:      types.StringValue(partial.Slug),
			Name:      types.StringValue(partial.Name),
			Status:    types.StringValue(partial.Status),
			CreatedAt: types.StringValue(partial.CreatedAt.Format("2006-01-02T15:04:05Z07:00")),
		}

		if !partial.UpdatedAt.IsZero() {
			partialState.UpdatedAt = types.StringValue(partial.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
		}

		state.PromptPartials = append(state.PromptPartials, partialState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
