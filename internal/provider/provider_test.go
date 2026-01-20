package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/joho/godotenv"
)

func init() {
	// Load .env file from project root for local development
	// This allows running acceptance tests without manually setting env vars
	loadEnvFile()
}

// loadEnvFile attempts to load environment variables from .env file
func loadEnvFile() {
	// Try to find .env file in current directory or parent directories
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Load(envPath)
			return
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"portkey": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck validates the necessary test environment variables exist.
func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("PORTKEY_API_KEY"); v == "" {
		t.Skip("PORTKEY_API_KEY must be set for acceptance tests. Create a .env file with PORTKEY_API_KEY=your-key")
	}
}

// getTestWorkspaceID returns a workspace ID for testing
// It first checks for TEST_WORKSPACE_ID env var, then falls back to a default
func getTestWorkspaceID() string {
	if v := os.Getenv("TEST_WORKSPACE_ID"); v != "" {
		return v
	}
	// Default test workspace - override in .env for your environment
	return "9da48f29-e564-4bcd-8480-757803acf5ae"
}

// getTestWorkspaceSlug returns a workspace slug for testing slug-to-UUID scenarios
func getTestWorkspaceSlug() string {
	if v := os.Getenv("TEST_WORKSPACE_SLUG"); v != "" {
		return v
	}
	// Default - override in .env for your environment
	return ""
}

// getTestIntegrationID returns an integration ID for testing
func getTestIntegrationID() string {
	if v := os.Getenv("TEST_INTEGRATION_ID"); v != "" {
		return v
	}
	return ""
}

// getTestCollectionID returns a collection ID for prompt testing
func getTestCollectionID() string {
	if v := os.Getenv("TEST_COLLECTION_ID"); v != "" {
		return v
	}
	return ""
}

// getTestVirtualKey returns a virtual key (provider) ID for prompt testing
func getTestVirtualKey() string {
	if v := os.Getenv("TEST_VIRTUAL_KEY"); v != "" {
		return v
	}
	return ""
}

// providerConfig is a shared configuration for all acceptance tests.
const providerConfig = `
provider "portkey" {}
`

// Ensure terraform.State is imported for use in import state functions
var _ = terraform.State{}

// TestProvider_HasChildResources verifies the provider has resources
func TestProvider_HasChildResources(t *testing.T) {
	expectedResources := []string{
		"portkey_workspace",
		"portkey_workspace_member",
		"portkey_user_invite",
		"portkey_integration",
		"portkey_integration_workspace_access",
		"portkey_api_key",
		"portkey_provider",
		"portkey_config",
		"portkey_prompt",
		"portkey_guardrail",
		"portkey_usage_limits_policy",
		"portkey_rate_limits_policy",
	}

	resources := New("test")().Resources(context.Background())

	if len(resources) != len(expectedResources) {
		t.Errorf("Expected %d resources, got %d", len(expectedResources), len(resources))
	}
}

// TestProvider_HasChildDataSources verifies the provider has data sources
func TestProvider_HasChildDataSources(t *testing.T) {
	expectedDataSources := []string{
		"portkey_workspace",
		"portkey_workspaces",
		"portkey_user",
		"portkey_users",
		"portkey_integration",
		"portkey_integrations",
		"portkey_integration_workspaces",
		"portkey_api_key",
		"portkey_api_keys",
		"portkey_provider",
		"portkey_providers",
		"portkey_config",
		"portkey_configs",
		"portkey_prompt",
		"portkey_prompts",
		"portkey_guardrail",
		"portkey_guardrails",
		"portkey_usage_limits_policy",
		"portkey_usage_limits_policies",
		"portkey_rate_limits_policy",
		"portkey_rate_limits_policies",
	}

	dataSources := New("test")().DataSources(context.Background())

	if len(dataSources) != len(expectedDataSources) {
		t.Errorf("Expected %d data sources, got %d", len(expectedDataSources), len(dataSources))
	}
}

// TestAccProvider_Configure validates the provider can be configured
func TestAccProvider_Configure(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
data "portkey_workspaces" "test" {}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_workspaces.test", "workspaces.#"),
				),
			},
		},
	})
}
