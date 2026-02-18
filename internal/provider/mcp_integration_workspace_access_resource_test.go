package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccMcpIntegrationWorkspaceAccessResource_basic tests the basic workspace
// access lifecycle: create, import, update (disable), destroy.
func TestAccMcpIntegrationWorkspaceAccessResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-wa")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMcpIntegrationWorkspaceAccessResourceConfigSelfContained(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_integration_workspace_access.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_mcp_integration_workspace_access.test", "mcp_integration_id"),
					resource.TestCheckResourceAttrSet("portkey_mcp_integration_workspace_access.test", "workspace_id"),
					resource.TestCheckResourceAttr("portkey_mcp_integration_workspace_access.test", "enabled", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "portkey_mcp_integration_workspace_access.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: mcpIntegrationWorkspaceAccessImportStateIdFunc("portkey_mcp_integration_workspace_access.test"),
			},
			// Update testing - disable access
			{
				Config: testAccMcpIntegrationWorkspaceAccessResourceConfigSelfContained(rName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_integration_workspace_access.test", "enabled", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccMcpIntegrationWorkspaceAccessResource_update tests toggling workspace
// access from enabled to disabled and back.
func TestAccMcpIntegrationWorkspaceAccessResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-wa-upd")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create enabled
			{
				Config: testAccMcpIntegrationWorkspaceAccessResourceConfigSelfContained(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_integration_workspace_access.test", "enabled", "true"),
				),
			},
			// Disable
			{
				Config: testAccMcpIntegrationWorkspaceAccessResourceConfigSelfContained(rName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_integration_workspace_access.test", "enabled", "false"),
				),
			},
			// Re-enable
			{
				Config: testAccMcpIntegrationWorkspaceAccessResourceConfigSelfContained(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_integration_workspace_access.test", "enabled", "true"),
				),
			},
		},
	})
}

// mcpIntegrationWorkspaceAccessImportStateIdFunc returns a function that generates
// the composite import ID in format: mcp_integration_id/workspace_id
func mcpIntegrationWorkspaceAccessImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}

		integrationID := rs.Primary.Attributes["mcp_integration_id"]
		workspaceID := rs.Primary.Attributes["workspace_id"]
		return fmt.Sprintf("%s/%s", integrationID, workspaceID), nil
	}
}

// testAccMcpIntegrationWorkspaceAccessResourceConfigSelfContained creates a
// self-contained test config that provisions its own workspace (matching the
// integration_workspace_access gold standard from PR #5).
func testAccMcpIntegrationWorkspaceAccessResourceConfigSelfContained(name string, enabled bool) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for MCP integration access"
}

resource "portkey_mcp_integration" "test" {
  name      = "%[1]s-integration"
  url       = "https://example.com/mcp"
  auth_type = "none"
  transport = "sse"
}

resource "portkey_mcp_integration_workspace_access" "test" {
  mcp_integration_id = portkey_mcp_integration.test.id
  workspace_id       = portkey_workspace.test.id
  enabled            = %[2]t
}
`, name, enabled)
}
