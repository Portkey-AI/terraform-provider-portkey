package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMcpIntegrationResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMcpIntegrationResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "slug"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "url", "https://example.com/mcp"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "auth_type", "none"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "transport", "sse"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_mcp_integration.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"configurations", "created_at", "last_updated_at"},
			},
			// Update testing - change name and description
			{
				Config: testAccMcpIntegrationResourceConfigUpdated(rName + "-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "description", "Updated description"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccMcpIntegrationResourceConfig(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name      = %[1]q
  url       = "https://example.com/mcp"
  auth_type = "none"
  transport = "sse"
}
`, name)
}

func testAccMcpIntegrationResourceConfigUpdated(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name        = %[1]q
  description = "Updated description"
  url         = "https://example.com/mcp"
  auth_type   = "none"
  transport   = "sse"
}
`, name)
}
