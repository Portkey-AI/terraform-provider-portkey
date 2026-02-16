package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

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
				),
			},
		},
	})
}

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
