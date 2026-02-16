package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccMcpE2E_integrationToWorkspaceAccess tests the full MCP workflow:
// integration -> workspace -> workspace_access in a single apply.
// This validates the primary use case: IaC for MCP without manual UI work.
func TestAccMcpE2E_integrationToWorkspaceAccess(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-e2e")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMcpE2EIntegrationToWorkspaceAccess(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify integration was created
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "slug"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "auth_type", "oauth_auto"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "transport", "http"),
					// Verify workspace was created
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					// Verify workspace access was created
					resource.TestCheckResourceAttrSet("portkey_mcp_integration_workspace_access.test", "id"),
					resource.TestCheckResourceAttr("portkey_mcp_integration_workspace_access.test", "enabled", "true"),
				),
			},
		},
	})
}

// TestAccMcpE2E_fullStack tests the complete MCP resource stack:
// integration -> workspace -> workspace_access -> server -> data sources.
// This exercises the full dependency chain in a single test.
func TestAccMcpE2E_fullStack(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-full")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMcpE2EFullStack(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Integration
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "id"),
					// Workspace
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					// Workspace access
					resource.TestCheckResourceAttr("portkey_mcp_integration_workspace_access.test", "enabled", "true"),
					// Server
					resource.TestCheckResourceAttrSet("portkey_mcp_server.test", "id"),
					resource.TestCheckResourceAttr("portkey_mcp_server.test", "name", rName+"-server"),
					// Data sources read back correctly
					resource.TestCheckResourceAttrSet("data.portkey_mcp_integration.test", "id"),
					resource.TestCheckResourceAttr("data.portkey_mcp_integration.test", "name", rName+"-integration"),
					resource.TestCheckResourceAttrSet("data.portkey_mcp_server.test", "id"),
					resource.TestCheckResourceAttr("data.portkey_mcp_server.test", "name", rName+"-server"),
				),
			},
		},
	})
}

func testAccMcpE2EIntegrationToWorkspaceAccess(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name      = "%[1]s-integration"
  url       = "https://mcp.linear.app/mcp"
  auth_type = "oauth_auto"
  transport = "http"
}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "E2E test workspace for MCP"
}

resource "portkey_mcp_integration_workspace_access" "test" {
  mcp_integration_id = portkey_mcp_integration.test.id
  workspace_id       = portkey_workspace.test.id
  enabled            = true
}
`, name)
}

func testAccMcpE2EFullStack(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name      = "%[1]s-integration"
  url       = "https://example.com/mcp"
  auth_type = "none"
  transport = "sse"
}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Full stack E2E test workspace"
}

resource "portkey_mcp_integration_workspace_access" "test" {
  mcp_integration_id = portkey_mcp_integration.test.id
  workspace_id       = portkey_workspace.test.id
  enabled            = true
}

resource "portkey_mcp_server" "test" {
  name               = "%[1]s-server"
  mcp_integration_id = portkey_mcp_integration.test.id
  workspace_id       = portkey_workspace.test.id

  depends_on = [portkey_mcp_integration_workspace_access.test]
}

data "portkey_mcp_integration" "test" {
  id = portkey_mcp_integration.test.id
}

data "portkey_mcp_server" "test" {
  id = portkey_mcp_server.test.id
}
`, name)
}
