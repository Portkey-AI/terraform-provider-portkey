package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &mcpServerCapabilitiesResource{}
	_ resource.ResourceWithConfigure   = &mcpServerCapabilitiesResource{}
	_ resource.ResourceWithImportState = &mcpServerCapabilitiesResource{}
)

// NewMcpServerCapabilitiesResource is a helper function to simplify the provider implementation.
func NewMcpServerCapabilitiesResource() resource.Resource {
	return &mcpServerCapabilitiesResource{}
}

// mcpServerCapabilitiesResource is the resource implementation.
type mcpServerCapabilitiesResource struct {
	client *client.Client
}

// mcpServerCapabilitiesResourceModel maps the resource schema data.
type mcpServerCapabilitiesResourceModel struct {
	ID           types.String         `tfsdk:"id"`
	McpServerID  types.String         `tfsdk:"mcp_server_id"`
	Capabilities []mcpCapabilityModel `tfsdk:"capabilities"`
}

// Metadata returns the resource type name.
func (r *mcpServerCapabilitiesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_server_capabilities"
}

// Schema defines the schema for the resource.
func (r *mcpServerCapabilitiesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages capability overrides for a Portkey MCP server.

Capabilities represent the tools, resources, and prompts exposed by an MCP server. This resource manages which capabilities are enabled or disabled at the server level within a workspace.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier (same as mcp_server_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mcp_server_id": schema.StringAttribute{
				Description: "The MCP server ID or slug to manage capabilities for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"capabilities": schema.ListNestedAttribute{
				Description: "List of capability overrides. Only capabilities listed here will be managed; others retain their defaults.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Name of the capability.",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "Type of capability: 'tool', 'resource', or 'prompt'.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("tool", "resource", "prompt"),
							},
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether this capability is enabled.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *mcpServerCapabilitiesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// Create creates the resource and sets the initial Terraform state.
func (r *mcpServerCapabilitiesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mcpServerCapabilitiesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updates := capabilityModelsToUpdates(plan.Capabilities)

	err := r.client.UpdateMcpServerCapabilities(ctx, plan.McpServerID.ValueString(), updates)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MCP server capabilities",
			"Could not update capabilities: "+err.Error(),
		)
		return
	}

	plan.ID = plan.McpServerID

	removed, readDiags := r.readCapabilities(ctx, &plan)
	resp.Diagnostics.Append(readDiags...)
	if resp.Diagnostics.HasError() || removed {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *mcpServerCapabilitiesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mcpServerCapabilitiesResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	removed, readDiags := r.readCapabilities(ctx, &state)
	resp.Diagnostics.Append(readDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if removed {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mcpServerCapabilitiesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mcpServerCapabilitiesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updates := capabilityModelsToUpdates(plan.Capabilities)

	err := r.client.UpdateMcpServerCapabilities(ctx, plan.McpServerID.ValueString(), updates)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating MCP server capabilities",
			"Could not update capabilities: "+err.Error(),
		)
		return
	}

	removed, readDiags := r.readCapabilities(ctx, &plan)
	resp.Diagnostics.Append(readDiags...)
	if resp.Diagnostics.HasError() || removed {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mcpServerCapabilitiesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mcpServerCapabilitiesResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reset all managed capabilities to enabled (default state)
	var resets []client.McpCapabilityUpdate
	for _, cap := range state.Capabilities {
		resets = append(resets, client.McpCapabilityUpdate{
			Name:    cap.Name.ValueString(),
			Type:    cap.Type.ValueString(),
			Enabled: true,
		})
	}

	if len(resets) > 0 {
		err := r.client.UpdateMcpServerCapabilities(ctx, state.McpServerID.ValueString(), resets)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return
			}
			resp.Diagnostics.AddError(
				"Error deleting MCP server capabilities",
				"Could not reset capabilities to defaults: "+err.Error(),
			)
			return
		}
	}
}

// ImportState imports the resource state.
func (r *mcpServerCapabilitiesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mcp_server_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// readCapabilities reads capabilities from the API and filters to only those managed in state.
// Returns (removed bool, diagnostics). removed=true means the parent resource was deleted.
func (r *mcpServerCapabilitiesResource) readCapabilities(ctx context.Context, state *mcpServerCapabilitiesResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	capabilities, err := r.client.GetMcpServerCapabilities(ctx, state.McpServerID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return true, diags
		}
		diags.AddError(
			"Error reading MCP server capabilities",
			"Could not read capabilities: "+err.Error(),
		)
		return false, diags
	}

	// Build a map for quick lookup
	capMap := make(map[string]client.McpCapability)
	for _, cap := range capabilities {
		key := cap.Type + "/" + cap.Name
		capMap[key] = cap
	}

	// Update state with actual values from API, preserving the user's list order
	var updated []mcpCapabilityModel
	for _, planned := range state.Capabilities {
		key := planned.Type.ValueString() + "/" + planned.Name.ValueString()
		if actual, ok := capMap[key]; ok {
			updated = append(updated, mcpCapabilityModel{
				Name:    types.StringValue(actual.Name),
				Type:    types.StringValue(actual.Type),
				Enabled: types.BoolValue(actual.Enabled),
			})
		} else {
			updated = append(updated, planned)
		}
	}

	state.Capabilities = updated
	return false, diags
}
