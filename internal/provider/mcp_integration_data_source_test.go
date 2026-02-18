package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccMcpIntegrationDataSource_basic tests reading a single MCP integration
// by ID and verifying all expected attributes are returned.
func TestAccMcpIntegrationDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMcpIntegrationDataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_mcp_integration.test", "id"),
					resource.TestCheckResourceAttr("data.portkey_mcp_integration.test", "name", rName),
					resource.TestCheckResourceAttr("data.portkey_mcp_integration.test", "url", "https://example.com/mcp"),
					resource.TestCheckResourceAttr("data.portkey_mcp_integration.test", "auth_type", "none"),
					resource.TestCheckResourceAttr("data.portkey_mcp_integration.test", "transport", "sse"),
					resource.TestCheckResourceAttrSet("data.portkey_mcp_integration.test", "slug"),
					resource.TestCheckResourceAttrSet("data.portkey_mcp_integration.test", "status"),
					resource.TestCheckResourceAttrSet("data.portkey_mcp_integration.test", "created_at"),
				),
			},
		},
	})
}

// TestAccMcpIntegrationsDataSource_basic tests listing MCP integrations
// and verifying at least one integration is returned.
func TestAccMcpIntegrationsDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMcpIntegrationsDataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_mcp_integrations.test", "id"),
				),
			},
		},
	})
}

// TestAccMcpIntegrationDataSource_httpTransport tests reading an integration
// created with HTTP transport to verify enum values survive the API round-trip.
func TestAccMcpIntegrationDataSource_httpTransport(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-ds-http")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMcpIntegrationDataSourceConfigWithEnums(rName, "oauth_auto", "http"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.portkey_mcp_integration.test", "auth_type", "oauth_auto"),
					resource.TestCheckResourceAttr("data.portkey_mcp_integration.test", "transport", "http"),
				),
			},
		},
	})
}

func testAccMcpIntegrationDataSourceConfig(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name      = %[1]q
  url       = "https://example.com/mcp"
  auth_type = "none"
  transport = "sse"
}

data "portkey_mcp_integration" "test" {
  id = portkey_mcp_integration.test.id
}
`, name)
}

func testAccMcpIntegrationsDataSourceConfig(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name      = %[1]q
  url       = "https://example.com/mcp"
  auth_type = "none"
  transport = "sse"
}

data "portkey_mcp_integrations" "test" {
  depends_on = [portkey_mcp_integration.test]
}
`, name)
}

func testAccMcpIntegrationDataSourceConfigWithEnums(name, authType, transport string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name      = %[1]q
  url       = "https://example.com/mcp"
  auth_type = %[2]q
  transport = %[3]q
}

data "portkey_mcp_integration" "test" {
  id = portkey_mcp_integration.test.id
}
`, name, authType, transport)
}
