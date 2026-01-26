package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccIntegrationWorkspaceAccessResource_basic tests the basic workspace access lifecycle.
func TestAccIntegrationWorkspaceAccessResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-iwaccess")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create integration, workspace, and workspace access
			{
				Config: testAccIntegrationWorkspaceAccessResourceConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration_workspace_access.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_integration_workspace_access.test", "integration_id"),
					resource.TestCheckResourceAttrSet("portkey_integration_workspace_access.test", "workspace_id"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "enabled", "true"),
				),
			},
			// Import testing
			{
				ResourceName:      "portkey_integration_workspace_access.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: integrationWorkspaceAccessImportStateIdFunc("portkey_integration_workspace_access.test"),
			},
		},
	})
}

// TestAccIntegrationWorkspaceAccessResource_withLimits tests workspace access with usage and rate limits.
func TestAccIntegrationWorkspaceAccessResource_withLimits(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-iwlimits")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with limits
			{
				Config: testAccIntegrationWorkspaceAccessResourceConfigWithLimits(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration_workspace_access.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "enabled", "true"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "usage_limits.#", "1"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "usage_limits.0.type", "cost"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "usage_limits.0.credit_limit", "100"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "rate_limits.#", "1"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "rate_limits.0.type", "requests"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "rate_limits.0.unit", "rpm"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "rate_limits.0.value", "1000"),
				),
			},
		},
	})
}

// TestAccIntegrationWorkspaceAccessResource_update tests updating workspace access.
func TestAccIntegrationWorkspaceAccessResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-iwupdate")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create basic
			{
				Config: testAccIntegrationWorkspaceAccessResourceConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "enabled", "true"),
				),
			},
			// Update to disabled
			{
				Config: testAccIntegrationWorkspaceAccessResourceConfigDisabled(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "enabled", "false"),
				),
			},
		},
	})
}

// integrationWorkspaceAccessImportStateIdFunc returns a function that generates the import ID
func integrationWorkspaceAccessImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}

		integrationID := rs.Primary.Attributes["integration_id"]
		workspaceID := rs.Primary.Attributes["workspace_id"]
		return fmt.Sprintf("%s/%s", integrationID, workspaceID), nil
	}
}

// testAccIntegrationWorkspaceAccessResourceConfigBasic creates a basic workspace access configuration
func testAccIntegrationWorkspaceAccessResourceConfigBasic(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for integration access"
}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

resource "portkey_integration_workspace_access" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
  workspace_id   = portkey_workspace.test.id
  enabled        = true
}
`, name)
}

// testAccIntegrationWorkspaceAccessResourceConfigWithLimits creates workspace access with limits
func testAccIntegrationWorkspaceAccessResourceConfigWithLimits(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for integration access with limits"
}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

resource "portkey_integration_workspace_access" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
  workspace_id   = portkey_workspace.test.id
  enabled        = true

  usage_limits = [{
    type            = "cost"
    credit_limit    = 100
    alert_threshold = 80
    periodic_reset  = "monthly"
  }]

  rate_limits = [{
    type  = "requests"
    unit  = "rpm"
    value = 1000
  }]
}
`, name)
}

// testAccIntegrationWorkspaceAccessResourceConfigDisabled creates disabled workspace access
func testAccIntegrationWorkspaceAccessResourceConfigDisabled(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for integration access"
}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

resource "portkey_integration_workspace_access" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
  workspace_id   = portkey_workspace.test.id
  enabled        = false
}
`, name)
}

// TestAccIntegrationWorkspaceAccessResource_updateLimits tests updating limits on workspace access.
func TestAccIntegrationWorkspaceAccessResource_updateLimits(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-iwlimupd")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with initial limits
			{
				Config: testAccIntegrationWorkspaceAccessResourceConfigWithLimits(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "usage_limits.0.credit_limit", "100"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "rate_limits.0.value", "1000"),
				),
			},
			// Update limits
			{
				Config: testAccIntegrationWorkspaceAccessResourceConfigUpdatedLimits(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "usage_limits.0.credit_limit", "200"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "usage_limits.0.alert_threshold", "90"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "rate_limits.0.value", "2000"),
				),
			},
		},
	})
}

// testAccIntegrationWorkspaceAccessResourceConfigUpdatedLimits creates workspace access with updated limits
func testAccIntegrationWorkspaceAccessResourceConfigUpdatedLimits(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for integration access with limits"
}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

resource "portkey_integration_workspace_access" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
  workspace_id   = portkey_workspace.test.id
  enabled        = true

  usage_limits = [{
    type            = "cost"
    credit_limit    = 200
    alert_threshold = 90
    periodic_reset  = "monthly"
  }]

  rate_limits = [{
    type  = "requests"
    unit  = "rpm"
    value = 2000
  }]
}
`, name)
}

// TestAccIntegrationWorkspaceAccessResource_e2eWithProvider tests the full E2E workflow:
// integration -> workspace -> workspace_access -> provider in a single apply.
// This validates the primary use case of enabling IaC without manual UI work.
func TestAccIntegrationWorkspaceAccessResource_e2eWithProvider(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-iwe2e")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationWorkspaceAccessResourceConfigE2EWithProvider(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify workspace was created
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					// Verify workspace access was created
					resource.TestCheckResourceAttrSet("portkey_integration_workspace_access.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration_workspace_access.test", "enabled", "true"),
					// Verify provider was created successfully (this is the key validation)
					resource.TestCheckResourceAttrSet("portkey_provider.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_provider.test", "slug"),
				),
			},
		},
	})
}

// testAccIntegrationWorkspaceAccessResourceConfigE2EWithProvider creates the full E2E workflow
func testAccIntegrationWorkspaceAccessResourceConfigE2EWithProvider(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "E2E test workspace"
}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

resource "portkey_integration_workspace_access" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
  workspace_id   = portkey_workspace.test.id
  enabled        = true
}

# Create a provider (virtual key) using the workspace access
# This should succeed without manual UI intervention
resource "portkey_provider" "test" {
  name          = "%[1]s-provider"
  workspace_id  = portkey_workspace.test.id
  integration_id = data.portkey_integrations.all.integrations[0].slug

  depends_on = [portkey_integration_workspace_access.test]
}
`, name)
}

// Note: Multiple limits per workspace are not supported by the API (maxItems: 1).
// The schema uses ListNestedAttribute to match the API structure, but only one item is allowed.
