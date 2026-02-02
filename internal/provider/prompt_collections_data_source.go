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
	_ datasource.DataSource              = &promptCollectionsDataSource{}
	_ datasource.DataSourceWithConfigure = &promptCollectionsDataSource{}
)

// NewPromptCollectionsDataSource is a helper function to simplify the provider implementation.
func NewPromptCollectionsDataSource() datasource.DataSource {
	return &promptCollectionsDataSource{}
}

// promptCollectionsDataSource is the data source implementation.
type promptCollectionsDataSource struct {
	client *client.Client
}

// promptCollectionsDataSourceModel maps the data source schema data.
type promptCollectionsDataSourceModel struct {
	ID          types.String            `tfsdk:"id"`
	WorkspaceID types.String            `tfsdk:"workspace_id"`
	Collections []promptCollectionModel `tfsdk:"collections"`
}

// promptCollectionModel maps collection data
type promptCollectionModel struct {
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
func (d *promptCollectionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt_collections"
}

// Schema defines the schema for the data source.
func (d *promptCollectionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all Portkey prompt collections, optionally filtered by workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this data source.",
				Computed:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Filter collections by workspace ID. If not provided, returns all collections.",
				Optional:    true,
			},
			"collections": schema.ListNestedAttribute{
				Description: "List of prompt collections.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Collection identifier (UUID).",
							Computed:    true,
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *promptCollectionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *promptCollectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state promptCollectionsDataSourceModel

	// Get workspace_id filter from config
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get collections from Portkey API
	workspaceID := ""
	if !state.WorkspaceID.IsNull() && !state.WorkspaceID.IsUnknown() {
		workspaceID = state.WorkspaceID.ValueString()
	}

	collections, err := d.client.ListPromptCollections(ctx, workspaceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey Prompt Collections",
			err.Error(),
		)
		return
	}

	// Map response to state
	for _, collection := range collections {
		var parentCollectionID types.String
		if collection.ParentCollectionID != "" {
			parentCollectionID = types.StringValue(collection.ParentCollectionID)
		} else {
			parentCollectionID = types.StringNull()
		}

		collectionState := promptCollectionModel{
			ID:                 types.StringValue(collection.ID),
			Name:               types.StringValue(collection.Name),
			WorkspaceID:        types.StringValue(collection.WorkspaceID),
			Slug:               types.StringValue(collection.Slug),
			ParentCollectionID: parentCollectionID,
			IsDefault:          types.BoolValue(collection.IsDefault == 1),
			Status:             types.StringValue(collection.Status),
			CreatedAt:          types.StringValue(collection.CreatedAt),
			LastUpdatedAt:      types.StringValue(collection.LastUpdatedAt),
		}
		state.Collections = append(state.Collections, collectionState)
	}

	// Set ID for the data source
	if workspaceID != "" {
		state.ID = types.StringValue(workspaceID)
	} else {
		state.ID = types.StringValue("all")
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
