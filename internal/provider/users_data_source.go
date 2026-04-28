package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// usersDataSourceMaxPages is the page-traversal safety cap. With a default
// page size of 100 this allows up to 100k users; larger orgs are extremely
// unlikely and we surface a clear error rather than spinning indefinitely.
const usersDataSourceMaxPages = 1000

// usersDataSourceDefaultPageSize is the default page size used when the user
// does not specify page_size. 100 minimises round-trips for typical orgs
// while staying well within Portkey API per-request limits.
const usersDataSourceDefaultPageSize = 100

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
//
// PageSize / Role / Email are optional inputs forwarded to the API; the data
// source auto-paginates regardless of PageSize so callers always receive every
// matching user. Total is the API-reported total count after filters.
type usersDataSourceModel struct {
	PageSize types.Int64  `tfsdk:"page_size"`
	Role     types.String `tfsdk:"role"`
	Email    types.String `tfsdk:"email"`
	Total    types.Int64  `tfsdk:"total"`
	Users    []userModel  `tfsdk:"users"`
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
		Description: "Fetches Portkey users in the organization. Supports server-side filtering by role and email " +
			"and auto-paginates the underlying GET /admin/users endpoint until every matching user is returned.",
		Attributes: map[string]schema.Attribute{
			"page_size": schema.Int64Attribute{
				Description: fmt.Sprintf("Page size for upstream API calls. Defaults to %d. The data source paginates "+
					"transparently — increasing this value reduces the number of round-trips for large organizations. "+
					"Must be at least 1.", usersDataSourceDefaultPageSize),
				Optional: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"role": schema.StringAttribute{
				Description: "Optional server-side filter by role. Available options: `admin`, `member`, `owner`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("admin", "member", "owner"),
				},
			},
			"email": schema.StringAttribute{
				Description: "Optional server-side filter by exact email address.",
				Optional:    true,
			},
			"total": schema.Int64Attribute{
				Description: "Total number of users returned by the API after any role/email filters were applied.",
				Computed:    true,
			},
			"users": schema.ListNestedAttribute{
				Description: "List of users matching the filters.",
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
							Description: "Timestamp when the user was last updated. Null if the user has never been updated.",
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
// API (default page size 100) until all matching users are returned.
func (d *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state usersDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	pageSize := usersDataSourceDefaultPageSize
	if !state.PageSize.IsNull() && !state.PageSize.IsUnknown() && state.PageSize.ValueInt64() > 0 {
		pageSize = int(state.PageSize.ValueInt64())
	}

	opts := client.ListUsersOptions{PageSize: pageSize}
	if !state.Role.IsNull() && !state.Role.IsUnknown() {
		opts.Role = state.Role.ValueString()
	}
	if !state.Email.IsNull() && !state.Email.IsUnknown() {
		opts.Email = state.Email.ValueString()
	}

	// Initialise as empty (non-nil) slice so callers can safely call length()
	// on the attribute even when the result set is empty.
	state.Users = []userModel{}
	totalCaptured := false

	for page := 0; page < usersDataSourceMaxPages; page++ {
		opts.CurrentPage = page
		apiResp, err := d.client.ListUsersPaginated(ctx, opts)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read Portkey Users",
				err.Error(),
			)
			return
		}

		// Capture total from the first response only — the API returns the
		// same total on every page, and we want to record the value as the
		// API saw it at the start of the traversal.
		if !totalCaptured {
			state.Total = types.Int64Value(int64(apiResp.Total))
			totalCaptured = true
		}

		for _, u := range apiResp.Data {
			item := userModel{
				ID:        types.StringValue(u.ID),
				Email:     types.StringValue(u.Email),
				Role:      types.StringValue(u.Role),
				Status:    types.StringValue(u.Status),
				CreatedAt: types.StringValue(u.CreatedAt.Format("2006-01-02T15:04:05Z07:00")),
			}
			// The Portkey API may omit updated_at for users that have never
			// been modified. Mirror the project-wide convention of returning
			// null (not the Go zero time) so the attribute is unambiguous.
			if !u.UpdatedAt.IsZero() {
				item.UpdatedAt = types.StringValue(u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
			} else {
				item.UpdatedAt = types.StringNull()
			}
			state.Users = append(state.Users, item)
		}

		// Last page reached when the API returns fewer than the requested
		// page size (or zero rows on the very first request).
		if len(apiResp.Data) < pageSize {
			diags = resp.State.Set(ctx, &state)
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	// Reached the safety cap without seeing a short page — surface the
	// truncation explicitly instead of silently returning a partial list.
	resp.Diagnostics.AddError(
		"Too Many User Pages",
		fmt.Sprintf(
			"Stopped paginating after %d pages of size %d (collected %d users) — "+
				"the result set is larger than expected. Increase page_size or contact "+
				"the provider maintainers if this is a legitimate scenario.",
			usersDataSourceMaxPages, pageSize, len(state.Users),
		),
	)
}
