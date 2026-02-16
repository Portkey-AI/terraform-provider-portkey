package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMcpServerResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMcpServerResourceConfig(rName, workspaceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_server.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_mcp_server.test", "slug"),
					resource.TestCheckResourceAttr("portkey_mcp_server.test", "name", rName+"-server"),
					resource.TestCheckResourceAttrSet("portkey_mcp_server.test", "mcp_integration_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_mcp_server.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at"},
			},
			// Update testing - change name
			{
				Config: testAccMcpServerResourceConfigUpdated(rName, workspaceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_server.test", "name", rName+"-server-updated"),
					resource.TestCheckResourceAttr("portkey_mcp_server.test", "description", "Updated server"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccMcpServerResourceConfig(name, workspaceID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name      = %[1]q
  url       = "https://example.com/mcp"
  auth_type = "none"
  transport = "sse"
}

resource "portkey_mcp_server" "test" {
  name               = "%[1]s-server"
  mcp_integration_id = portkey_mcp_integration.test.id
  workspace_id       = %[2]q
}
`, name, workspaceID)
}

func testAccMcpServerResourceConfigUpdated(name, workspaceID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name      = %[1]q
  url       = "https://example.com/mcp"
  auth_type = "none"
  transport = "sse"
}

resource "portkey_mcp_server" "test" {
  name               = "%[1]s-server-updated"
  description        = "Updated server"
  mcp_integration_id = portkey_mcp_integration.test.id
  workspace_id       = %[2]q
}
`, name, workspaceID)
}
