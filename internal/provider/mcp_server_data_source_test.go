package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccMcpServerDataSource_basic tests reading a single MCP server by ID
// and verifying all expected attributes are returned.
func TestAccMcpServerDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-srv-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMcpServerDataSourceConfigSelfContained(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_mcp_server.test", "id"),
					resource.TestCheckResourceAttr("data.portkey_mcp_server.test", "name", rName+"-server"),
					resource.TestCheckResourceAttrSet("data.portkey_mcp_server.test", "mcp_integration_id"),
					resource.TestCheckResourceAttrSet("data.portkey_mcp_server.test", "workspace_id"),
				),
			},
		},
	})
}

// TestAccMcpServersDataSource_basic tests listing MCP servers filtered by
// workspace and verifying results are returned.
func TestAccMcpServersDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-srvs-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMcpServersDataSourceConfigSelfContained(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_mcp_servers.test", "id"),
				),
			},
		},
	})
}

func testAccMcpServerDataSourceConfigSelfContained(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for MCP server data source"
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

data "portkey_mcp_server" "test" {
  id = portkey_mcp_server.test.id
}
`, name)
}

func testAccMcpServersDataSourceConfigSelfContained(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for MCP servers data source"
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

data "portkey_mcp_servers" "test" {
  workspace_id = portkey_workspace.test.id
  depends_on   = [portkey_mcp_server.test]
}
`, name)
}
