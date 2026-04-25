package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &secretReferencesDataSource{}
	_ datasource.DataSourceWithConfigure = &secretReferencesDataSource{}
)

// NewSecretReferencesDataSource is a helper function to simplify the provider implementation.
func NewSecretReferencesDataSource() datasource.DataSource {
	return &secretReferencesDataSource{}
}

type secretReferencesDataSource struct {
	client *client.Client
}

// Filter-exposing model. Only name/slug/id/manager_type/status/timestamps are
// in the list response (secret_path, tags, auth_config etc. require the
// singular data source).
type secretReferencesDataSourceModel struct {
	ManagerType      types.String          `tfsdk:"manager_type"`
	Search           types.String          `tfsdk:"search"`
	SecretReferences []secretReferenceItem `tfsdk:"secret_references"`
}

type secretReferenceItem struct {
	ID          types.String `tfsdk:"id"`
	Slug        types.String `tfsdk:"slug"`
	Name        types.String `tfsdk:"name"`
	ManagerType types.String `tfsdk:"manager_type"`
	Status      types.String `tfsdk:"status"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// Metadata returns the data source type name.
func (d *secretReferencesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_references"
}

// Schema defines the schema for the data source.
func (d *secretReferencesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists Portkey secret references, optionally filtered by manager_type or name search. " +
			"Returns a lightweight item per secret (no secret_path, tags, or credentials); use portkey_secret_reference to fetch details.",
		Attributes: map[string]schema.Attribute{
			"manager_type": schema.StringAttribute{
				Description: "Optional filter by `manager_type`. Available options: `aws_sm`, `azure_kv`, `hashicorp_vault`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						secretManagerAWSSecretsManager,
						secretManagerAzureKeyVault,
						secretManagerHashicorpVault,
					),
				},
			},
			"search": schema.StringAttribute{
				Description: "Optional case-insensitive name substring search.",
				Optional:    true,
			},
			"secret_references": schema.ListNestedAttribute{
				Description: "List of matching secret references.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Secret reference UUID.",
							Computed:    true,
						},
						"slug": schema.StringAttribute{
							Description: "Slug.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Human-readable name.",
							Computed:    true,
						},
						"manager_type": schema.StringAttribute{
							Description: "Secret manager type. Available options: `aws_sm`, `azure_kv`, `hashicorp_vault`.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Status.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Created timestamp.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "Last updated timestamp.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *secretReferencesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read refreshes the Terraform state with the latest data. Pages through the
// API until all matching secret references are returned (default page_size=20).
func (d *secretReferencesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state secretReferencesDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := client.ListSecretReferencesOptions{
		ManagerType: state.ManagerType.ValueString(),
		Search:      state.Search.ValueString(),
		PageSize:    100,
	}

	// Initialize as an empty (non-nil) slice so the attribute is always a
	// concrete list — users can safely call length() on it even when there
	// are zero matches.
	state.SecretReferences = []secretReferenceItem{}

	for page := 0; page < 100; page++ {
		opts.CurrentPage = page
		apiResp, err := d.client.ListSecretReferences(ctx, opts)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to List Portkey Secret References",
				err.Error(),
			)
			return
		}
		for _, sr := range apiResp.Data {
			item := secretReferenceItem{
				ID:          types.StringValue(sr.ID),
				Slug:        types.StringValue(sr.Slug),
				Name:        types.StringValue(sr.Name),
				ManagerType: types.StringValue(sr.ManagerType),
				Status:      types.StringValue(sr.Status),
				CreatedAt:   types.StringValue(sr.CreatedAt.Format("2006-01-02T15:04:05Z07:00")),
			}
			if !sr.UpdatedAt.IsZero() {
				item.UpdatedAt = types.StringValue(sr.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
			} else {
				item.UpdatedAt = types.StringNull()
			}
			state.SecretReferences = append(state.SecretReferences, item)
		}

		if len(apiResp.Data) < opts.PageSize {
			break
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
