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
	_ datasource.DataSource              = &promptCollectionDataSource{}
	_ datasource.DataSourceWithConfigure = &promptCollectionDataSource{}
)

// NewPromptCollectionDataSource is a helper function to simplify the provider implementation.
func NewPromptCollectionDataSource() datasource.DataSource {
	return &promptCollectionDataSource{}
}

// promptCollectionDataSource is the data source implementation.
type promptCollectionDataSource struct {
	client *client.Client
}

// promptCollectionDataSourceModel maps the data source schema data.
type promptCollectionDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	WorkspaceID        types.String `tfsdk:"workspace_id"`
	Slug               types.String `tfsdk:"slug"`
	ParentCollectionID types.String `tfsdk:"parent_collection_id"`
	IsDefault          types.Bool   `tfsdk:"is_default"`
	Status             types.String `tfsdk:"status"`
	CreatedAt          types.String `tfsdk:"created_at"`
	LastUpdatedAt      types.String `tfsdk:"last_updated_at"`
}

// Metadata returns the data source type name.
func (d *promptCollectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt_collection"
}

// Schema defines the schema for the data source.
func (d *promptCollectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a specific Portkey prompt collection by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Collection identifier (UUID).",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the collection.",
				Computed:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID where this collection belongs.",
				Computed:    true,
			},
			"slug": schema.StringAttribute{
				Description: "URL-friendly identifier for the collection.",
				Computed:    true,
			},
			"parent_collection_id": schema.StringAttribute{
				Description: "Parent collection ID for nested collections.",
				Computed:    true,
			},
			"is_default": schema.BoolAttribute{
				Description: "Whether this is the default collection for the workspace.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Collection status (active, archived).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the collection was created.",
				Computed:    true,
			},
			"last_updated_at": schema.StringAttribute{
				Description: "Timestamp when the collection was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *promptCollectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *promptCollectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state promptCollectionDataSourceModel

	// Get the ID from the configuration
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get collection from Portkey API
	collection, err := d.client.GetPromptCollection(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey Prompt Collection",
			err.Error(),
		)
		return
	}

	// Map response to state
	state.ID = types.StringValue(collection.ID)
	state.Name = types.StringValue(collection.Name)
	state.WorkspaceID = types.StringValue(collection.WorkspaceID)
	state.Slug = types.StringValue(collection.Slug)
	state.IsDefault = types.BoolValue(collection.IsDefault == 1)
	state.Status = types.StringValue(collection.Status)
	state.CreatedAt = types.StringValue(collection.CreatedAt)
	state.LastUpdatedAt = types.StringValue(collection.LastUpdatedAt)

	if collection.ParentCollectionID != "" {
		state.ParentCollectionID = types.StringValue(collection.ParentCollectionID)
	} else {
		state.ParentCollectionID = types.StringNull()
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
