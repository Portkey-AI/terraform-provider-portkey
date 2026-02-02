package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &promptCollectionResource{}
	_ resource.ResourceWithConfigure   = &promptCollectionResource{}
	_ resource.ResourceWithImportState = &promptCollectionResource{}
)

// NewPromptCollectionResource is a helper function to simplify the provider implementation.
func NewPromptCollectionResource() resource.Resource {
	return &promptCollectionResource{}
}

// promptCollectionResource is the resource implementation.
type promptCollectionResource struct {
	client *client.Client
}

// promptCollectionResourceModel maps the resource schema data.
type promptCollectionResourceModel struct {
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

// Metadata returns the resource type name.
func (r *promptCollectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt_collection"
}

// Schema defines the schema for the resource.
func (r *promptCollectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages a Portkey prompt collection.

Collections are used to organize prompts within a workspace. They can be nested using parent_collection_id
to create hierarchical organization structures.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Collection identifier (UUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the collection.",
				Required:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID (UUID) where this collection belongs.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"slug": schema.StringAttribute{
				Description: "URL-friendly identifier for the collection. Auto-generated from name.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_collection_id": schema.StringAttribute{
				Description: "Parent collection ID for nested collections. Leave empty for top-level collections.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_default": schema.BoolAttribute{
				Description: "Whether this is the default collection for the workspace.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Collection status (active, archived).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the collection was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated_at": schema.StringAttribute{
				Description: "Timestamp when the collection was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *promptCollectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *promptCollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan promptCollectionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new collection
	createReq := client.CreatePromptCollectionRequest{
		Name:        plan.Name.ValueString(),
		WorkspaceID: plan.WorkspaceID.ValueString(),
	}

	if !plan.ParentCollectionID.IsNull() && !plan.ParentCollectionID.IsUnknown() {
		createReq.ParentCollectionID = plan.ParentCollectionID.ValueString()
	}

	createResp, err := r.client.CreatePromptCollection(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating prompt collection",
			"Could not create prompt collection, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch full collection details
	collection, err := r.client.GetPromptCollection(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading prompt collection",
			"Collection created but could not read details: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(collection.ID)
	plan.Slug = types.StringValue(collection.Slug)
	// Preserve workspace_id from plan (API returns UUID but user may have provided slug)
	// Don't overwrite - keep the user-provided value
	plan.IsDefault = types.BoolValue(collection.IsDefault == 1)
	plan.Status = types.StringValue(collection.Status)
	plan.CreatedAt = types.StringValue(collection.CreatedAt)
	plan.LastUpdatedAt = types.StringValue(collection.LastUpdatedAt)

	// Preserve parent_collection_id from plan if set (API may return different format)
	if plan.ParentCollectionID.IsUnknown() {
		if collection.ParentCollectionID != "" {
			plan.ParentCollectionID = types.StringValue(collection.ParentCollectionID)
		} else {
			plan.ParentCollectionID = types.StringNull()
		}
	}

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *promptCollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state promptCollectionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed collection from Portkey
	collection, err := r.client.GetPromptCollection(ctx, state.ID.ValueString())
	if err != nil {
		// Check if it's a 404 (not found) - resource was deleted outside Terraform
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Portkey Prompt Collection",
			"Could not read prompt collection ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state with refreshed values
	state.Name = types.StringValue(collection.Name)
	// Preserve workspace_id from state (API returns UUID but user may have provided slug)
	// Don't overwrite - keep the user-provided value
	state.Slug = types.StringValue(collection.Slug)
	state.IsDefault = types.BoolValue(collection.IsDefault == 1)
	state.Status = types.StringValue(collection.Status)
	state.CreatedAt = types.StringValue(collection.CreatedAt)
	state.LastUpdatedAt = types.StringValue(collection.LastUpdatedAt)

	// Preserve parent_collection_id from state if set (API may return different format)
	if state.ParentCollectionID.IsNull() || state.ParentCollectionID.IsUnknown() {
		if collection.ParentCollectionID != "" {
			state.ParentCollectionID = types.StringValue(collection.ParentCollectionID)
		} else {
			state.ParentCollectionID = types.StringNull()
		}
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *promptCollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan promptCollectionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update collection (only name can be updated)
	updateReq := client.UpdatePromptCollectionRequest{
		Name: plan.Name.ValueString(),
	}

	collection, err := r.client.UpdatePromptCollection(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Portkey Prompt Collection",
			"Could not update prompt collection, unexpected error: "+err.Error(),
		)
		return
	}

	// Update state with response
	plan.LastUpdatedAt = types.StringValue(collection.LastUpdatedAt)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *promptCollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state promptCollectionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete collection
	err := r.client.DeletePromptCollection(ctx, state.ID.ValueString())
	if err != nil {
		// If already deleted externally, consider it success
		if strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Portkey Prompt Collection",
			"Could not delete prompt collection: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *promptCollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
