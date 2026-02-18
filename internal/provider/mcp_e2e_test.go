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
					// Verify data source reads back correctly
					resource.TestCheckResourceAttrSet("data.portkey_mcp_integration.test", "id"),
					resource.TestCheckResourceAttr("data.portkey_mcp_integration.test", "name", rName+"-integration"),
					resource.TestCheckResourceAttr("data.portkey_mcp_integration.test", "auth_type", "oauth_auto"),
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

data "portkey_mcp_integration" "test" {
  id = portkey_mcp_integration.test.id
}
`, name)
}
