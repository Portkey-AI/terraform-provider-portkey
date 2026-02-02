package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &portkeyProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &portkeyProvider{
			version: version,
		}
	}
}

// portkeyProvider is the provider implementation.
type portkeyProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// portkeyProviderModel maps provider schema data to a Go type.
type portkeyProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

// Metadata returns the provider type name.
func (p *portkeyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "portkey"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *portkeyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Portkey Admin API for managing workspaces, users, and organization resources.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "Admin API key for Portkey. Can also be set via PORTKEY_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"base_url": schema.StringAttribute{
				Description: "Base URL for Portkey API. Defaults to https://api.portkey.ai/v1. Can be set via PORTKEY_BASE_URL for self-hosted deployments.",
				Optional:    true,
			},
		},
	}
}

// Configure prepares a Portkey API client for data sources and resources.
func (p *portkeyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config portkeyProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown Portkey API Key",
			"The provider cannot create the Portkey API client as there is an unknown configuration value for the Portkey API key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the PORTKEY_API_KEY environment variable.",
		)
	}

	if config.BaseURL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Unknown Portkey API Base URL",
			"The provider cannot create the Portkey API client as there is an unknown configuration value for the Portkey API base URL. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the PORTKEY_BASE_URL environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	apiKey := os.Getenv("PORTKEY_API_KEY")
	baseURL := os.Getenv("PORTKEY_BASE_URL")

	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Portkey API Key",
			"The provider cannot create the Portkey API client as there is a missing or empty value for the Portkey API key. "+
				"Set the api_key value in the configuration or use the PORTKEY_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if baseURL == "" {
		baseURL = "https://api.portkey.ai/v1"
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new Portkey client using the configuration values
	client, err := client.NewClient(baseURL, apiKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Portkey API Client",
			"An unexpected error occurred when creating the Portkey API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Portkey Client Error: "+err.Error(),
		)
		return
	}

	// Make the Portkey client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *portkeyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewWorkspaceDataSource,
		NewWorkspacesDataSource,
		NewUserDataSource,
		NewUsersDataSource,
		NewIntegrationDataSource,
		NewIntegrationsDataSource,
		NewIntegrationWorkspacesDataSource,
		NewIntegrationModelsDataSource,
		NewAPIKeyDataSource,
		NewAPIKeysDataSource,
		NewProviderDataSource,
		NewProvidersDataSource,
		NewConfigDataSource,
		NewConfigsDataSource,
		NewPromptDataSource,
		NewPromptsDataSource,
		NewPromptCollectionDataSource,
		NewPromptCollectionsDataSource,
		NewGuardrailDataSource,
		NewGuardrailsDataSource,
		NewUsageLimitsPolicyDataSource,
		NewUsageLimitsPoliciesDataSource,
		NewRateLimitsPolicyDataSource,
		NewRateLimitsPoliciesDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *portkeyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWorkspaceResource,
		NewWorkspaceMemberResource,
		NewUserInviteResource,
		NewIntegrationResource,
		NewIntegrationWorkspaceAccessResource,
		NewIntegrationModelAccessResource,
		NewAPIKeyResource,
		NewProviderResource,
		NewConfigResource,
		NewPromptResource,
		NewPromptCollectionResource,
		NewGuardrailResource,
		NewUsageLimitsPolicyResource,
		NewRateLimitsPolicyResource,
	}
}
