package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMcpIntegrationWorkspaceAccessResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMcpIntegrationWorkspaceAccessResourceConfig(rName, workspaceID, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_integration_workspace_access.test", "id"),
					resource.TestCheckResourceAttr("portkey_mcp_integration_workspace_access.test", "workspace_id", workspaceID),
					resource.TestCheckResourceAttr("portkey_mcp_integration_workspace_access.test", "enabled", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "portkey_mcp_integration_workspace_access.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing - disable access
			{
				Config: testAccMcpIntegrationWorkspaceAccessResourceConfig(rName, workspaceID, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_integration_workspace_access.test", "enabled", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccMcpIntegrationWorkspaceAccessResourceConfig(name, workspaceID string, enabled bool) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name      = %[1]q
  url       = "https://example.com/mcp"
  auth_type = "none"
  transport = "sse"
}

resource "portkey_mcp_integration_workspace_access" "test" {
  mcp_integration_id = portkey_mcp_integration.test.id
  workspace_id       = %[2]q
  enabled            = %[3]t
}
`, name, workspaceID, enabled)
}
