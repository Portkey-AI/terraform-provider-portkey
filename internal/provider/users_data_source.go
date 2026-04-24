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
	_ datasource.DataSource              = &usersDataSource{}
	_ datasource.DataSourceWithConfigure = &usersDataSource{}
)

// NewUsersDataSource is a helper function to simplify the provider implementation.
func NewUsersDataSource() datasource.DataSource {
	return &usersDataSource{}
}

// usersDataSource is the data source implementation.
type usersDataSource struct {
	client *client.Client
}

// usersDataSourceModel maps the data source schema data.
type usersDataSourceModel struct {
	Email types.String `tfsdk:"email"`
	Role  types.String `tfsdk:"role"`
	Users []userModel  `tfsdk:"users"`
}

// userModel maps user data
type userModel struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	Role      types.String `tfsdk:"role"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// Metadata returns the data source type name.
func (d *usersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

// Schema defines the schema for the data source.
func (d *usersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all Portkey users in the organization. Automatically paginates through all pages. Supports optional server-side filtering by email or role.",
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				Description: "Filter users by email address. When set, only users matching this email are returned.",
				Optional:    true,
			},
			"role": schema.StringAttribute{
				Description: "Filter users by organization role. Valid values: admin, member, owner.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("admin", "member", "owner"),
				},
			},
			"users": schema.ListNestedAttribute{
				Description: "List of users.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "User identifier.",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "Email address of the user.",
							Computed:    true,
						},
						"role": schema.StringAttribute{
							Description: "Organization role of the user.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Status of the user account.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Timestamp when the user was created.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "Timestamp when the user was last updated.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *usersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state usersDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &client.ListUsersParams{}
	if !state.Email.IsNull() && !state.Email.IsUnknown() {
		params.Email = state.Email.ValueString()
	}
	if !state.Role.IsNull() && !state.Role.IsUnknown() {
		params.Role = state.Role.ValueString()
	}

	users, err := d.client.ListUsers(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Portkey Users",
			err.Error(),
		)
		return
	}

	state.Users = []userModel{}
	for _, user := range users {
		userState := userModel{
			ID:        types.StringValue(user.ID),
			Email:     types.StringValue(user.Email),
			Role:      types.StringValue(user.Role),
			Status:    types.StringValue(user.Status),
			CreatedAt: types.StringValue(user.CreatedAt.Format("2006-01-02T15:04:05Z07:00")),
			UpdatedAt: types.StringValue(user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")),
		}
		state.Users = append(state.Users, userState)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
