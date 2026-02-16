package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccMcpServerUserAccessResource_basic tests the basic user access lifecycle:
// create, import, update (disable), destroy. Self-contained with own workspace.
func TestAccMcpServerUserAccessResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-ua")
	userID := getTestUserID()

	if userID == "" {
		t.Skip("TEST_USER_ID must be set for this acceptance test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMcpServerUserAccessResourceConfigSelfContained(rName, userID, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_server_user_access.test", "id"),
					resource.TestCheckResourceAttr("portkey_mcp_server_user_access.test", "user_id", userID),
					resource.TestCheckResourceAttr("portkey_mcp_server_user_access.test", "enabled", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "portkey_mcp_server_user_access.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: mcpServerUserAccessImportStateIdFunc("portkey_mcp_server_user_access.test"),
			},
			// Update testing - disable access
			{
				Config: testAccMcpServerUserAccessResourceConfigSelfContained(rName, userID, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_server_user_access.test", "enabled", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccMcpServerUserAccessResource_update tests toggling user access
// from enabled to disabled and back.
func TestAccMcpServerUserAccessResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-ua-upd")
	userID := getTestUserID()

	if userID == "" {
		t.Skip("TEST_USER_ID must be set for this acceptance test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create enabled
			{
				Config: testAccMcpServerUserAccessResourceConfigSelfContained(rName, userID, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_server_user_access.test", "enabled", "true"),
				),
			},
			// Disable
			{
				Config: testAccMcpServerUserAccessResourceConfigSelfContained(rName, userID, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_server_user_access.test", "enabled", "false"),
				),
			},
			// Re-enable
			{
				Config: testAccMcpServerUserAccessResourceConfigSelfContained(rName, userID, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_server_user_access.test", "enabled", "true"),
				),
			},
		},
	})
}

// mcpServerUserAccessImportStateIdFunc returns a function that generates
// the composite import ID in format: mcp_server_id/user_id
func mcpServerUserAccessImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}

		serverID := rs.Primary.Attributes["mcp_server_id"]
		userID := rs.Primary.Attributes["user_id"]
		return fmt.Sprintf("%s/%s", serverID, userID), nil
	}
}

// testAccMcpServerUserAccessResourceConfigSelfContained creates a self-contained
// test config that provisions its own workspace, integration, and server.
func testAccMcpServerUserAccessResourceConfigSelfContained(name, userID string, enabled bool) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for MCP server user access"
}

resource "portkey_mcp_integration" "test" {
  name      = "%[1]s-integration"
  url       = "https://example.com/mcp"
  auth_type = "none"
  transport = "sse"
}

resource "portkey_mcp_server" "test" {
  name               = "%[1]s-server"
  mcp_integration_id = portkey_mcp_integration.test.id
  workspace_id       = portkey_workspace.test.id
}

resource "portkey_mcp_server_user_access" "test" {
  mcp_server_id = portkey_mcp_server.test.id
  user_id       = %[2]q
  enabled       = %[3]t
}
`, name, userID, enabled)
}
