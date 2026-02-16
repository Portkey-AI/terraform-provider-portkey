package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource              = &mcpIntegrationCapabilitiesResource{}
	_ resource.ResourceWithConfigure = &mcpIntegrationCapabilitiesResource{}
)

// NewMcpIntegrationCapabilitiesResource is a helper function to simplify the provider implementation.
func NewMcpIntegrationCapabilitiesResource() resource.Resource {
	return &mcpIntegrationCapabilitiesResource{}
}

// mcpIntegrationCapabilitiesResource is the resource implementation.
type mcpIntegrationCapabilitiesResource struct {
	client *client.Client
}

// mcpIntegrationCapabilitiesResourceModel maps the resource schema data.
type mcpIntegrationCapabilitiesResourceModel struct {
	ID               types.String         `tfsdk:"id"`
	McpIntegrationID types.String         `tfsdk:"mcp_integration_id"`
	Capabilities     []mcpCapabilityModel `tfsdk:"capabilities"`
}

// mcpCapabilityModel maps a single capability
type mcpCapabilityModel struct {
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

// Metadata returns the resource type name.
func (r *mcpIntegrationCapabilitiesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_integration_capabilities"
}

// Schema defines the schema for the resource.
func (r *mcpIntegrationCapabilitiesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages capability overrides for a Portkey MCP integration.

Capabilities represent the tools, resources, and prompts exposed by an MCP server. This resource manages which capabilities are enabled or disabled at the integration level.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier (same as mcp_integration_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mcp_integration_id": schema.StringAttribute{
				Description: "The MCP integration ID or slug to manage capabilities for.",
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
func (r *mcpIntegrationCapabilitiesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *mcpIntegrationCapabilitiesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mcpIntegrationCapabilitiesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updates := capabilityModelsToUpdates(plan.Capabilities)

	err := r.client.UpdateMcpIntegrationCapabilities(ctx, plan.McpIntegrationID.ValueString(), updates)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MCP integration capabilities",
			"Could not update capabilities: "+err.Error(),
		)
		return
	}

	plan.ID = plan.McpIntegrationID

	// Read back the capabilities to get actual state
	removed, readDiags := r.readCapabilities(ctx, &plan)
	resp.Diagnostics.Append(readDiags...)
	if resp.Diagnostics.HasError() || removed {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *mcpIntegrationCapabilitiesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mcpIntegrationCapabilitiesResourceModel
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
func (r *mcpIntegrationCapabilitiesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mcpIntegrationCapabilitiesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updates := capabilityModelsToUpdates(plan.Capabilities)

	err := r.client.UpdateMcpIntegrationCapabilities(ctx, plan.McpIntegrationID.ValueString(), updates)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating MCP integration capabilities",
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
func (r *mcpIntegrationCapabilitiesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mcpIntegrationCapabilitiesResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reset all managed capabilities to enabled (default state)
	var resets []client.McpCapability
	for _, cap := range state.Capabilities {
		resets = append(resets, client.McpCapability{
			Name:    cap.Name.ValueString(),
			Type:    cap.Type.ValueString(),
			Enabled: true,
		})
	}

	if len(resets) > 0 {
		err := r.client.UpdateMcpIntegrationCapabilities(ctx, state.McpIntegrationID.ValueString(), resets)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return
			}
			resp.Diagnostics.AddError(
				"Error deleting MCP integration capabilities",
				"Could not reset capabilities to defaults: "+err.Error(),
			)
			return
		}
	}
}

// readCapabilities reads capabilities from the API and filters to only those managed in state.
// Returns (removed bool, diagnostics). removed=true means the parent resource was deleted.
func (r *mcpIntegrationCapabilitiesResource) readCapabilities(ctx context.Context, state *mcpIntegrationCapabilitiesResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	capabilities, err := r.client.GetMcpIntegrationCapabilities(ctx, state.McpIntegrationID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return true, diags
		}
		diags.AddError(
			"Error reading MCP integration capabilities",
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
			// Capability not found in API - keep planned value
			updated = append(updated, planned)
		}
	}

	state.Capabilities = updated
	return false, diags
}

// capabilityModelsToUpdates converts capability models to client update requests
func capabilityModelsToUpdates(models []mcpCapabilityModel) []client.McpCapability {
	var updates []client.McpCapability
	for _, m := range models {
		updates = append(updates, client.McpCapability{
			Name:    m.Name.ValueString(),
			Type:    m.Type.ValueString(),
			Enabled: m.Enabled.ValueBool(),
		})
	}
	return updates
}
