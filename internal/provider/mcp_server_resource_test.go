package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccMcpServerResource_basic tests the basic MCP server lifecycle:
// create, import, update, destroy. Uses a self-contained workspace.
func TestAccMcpServerResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-srv")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMcpServerResourceConfigSelfContained(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_server.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_mcp_server.test", "slug"),
					resource.TestCheckResourceAttr("portkey_mcp_server.test", "name", rName+"-server"),
					resource.TestCheckResourceAttrSet("portkey_mcp_server.test", "mcp_integration_id"),
					resource.TestCheckResourceAttrSet("portkey_mcp_server.test", "workspace_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "portkey_mcp_server.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing - change name and add description
			{
				Config: testAccMcpServerResourceConfigSelfContainedUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_server.test", "name", rName+"-server-updated"),
					resource.TestCheckResourceAttr("portkey_mcp_server.test", "description", "Updated server description"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccMcpServerResource_withDescription tests creating a server with an
// initial description set at creation time.
func TestAccMcpServerResource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-srv-desc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMcpServerResourceConfigWithDescription(rName, "Initial description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_server.test", "id"),
					resource.TestCheckResourceAttr("portkey_mcp_server.test", "description", "Initial description"),
				),
			},
		},
	})
}

// testAccMcpServerResourceConfigSelfContained creates a self-contained test config
// that provisions its own workspace and integration.
func testAccMcpServerResourceConfigSelfContained(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for MCP server"
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
`, name)
}

func testAccMcpServerResourceConfigSelfContainedUpdated(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for MCP server"
}

resource "portkey_mcp_integration" "test" {
  name      = "%[1]s-integration"
  url       = "https://example.com/mcp"
  auth_type = "none"
  transport = "sse"
}

resource "portkey_mcp_server" "test" {
  name               = "%[1]s-server-updated"
  description        = "Updated server description"
  mcp_integration_id = portkey_mcp_integration.test.id
  workspace_id       = portkey_workspace.test.id
}
`, name)
}

func testAccMcpServerResourceConfigWithDescription(name, description string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for MCP server"
}

resource "portkey_mcp_integration" "test" {
  name      = "%[1]s-integration"
  url       = "https://example.com/mcp"
  auth_type = "none"
  transport = "sse"
}

resource "portkey_mcp_server" "test" {
  name               = "%[1]s-server"
  description        = %[2]q
  mcp_integration_id = portkey_mcp_integration.test.id
  workspace_id       = portkey_workspace.test.id
}
`, name, description)
}
