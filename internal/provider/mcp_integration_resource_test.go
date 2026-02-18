package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccMcpIntegrationResource_basic tests the basic MCP integration lifecycle
// with SSE transport and no authentication.
func TestAccMcpIntegrationResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMcpIntegrationResourceConfig(rName, "none", "sse"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "slug"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "url", "https://example.com/mcp"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "auth_type", "none"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "transport", "sse"),
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "status"),
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "created_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_mcp_integration.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"configurations"},
			},
			// Update testing - change name and add description
			{
				Config: testAccMcpIntegrationResourceConfigWithDescription(rName+"-updated", "none", "sse", "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "description", "Updated description"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccMcpIntegrationResource_httpTransport tests creating an integration
// with Streamable HTTP transport (the "http" enum value).
func TestAccMcpIntegrationResource_httpTransport(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-http")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMcpIntegrationResourceConfig(rName, "none", "http"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "transport", "http"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "auth_type", "none"),
				),
			},
		},
	})
}

// TestAccMcpIntegrationResource_oauthAuth tests creating an integration
// with OAuth 2.1 authentication (the "oauth_auto" enum value).
func TestAccMcpIntegrationResource_oauthAuth(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-oauth")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMcpIntegrationResourceConfig(rName, "oauth_auto", "http"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "auth_type", "oauth_auto"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "transport", "http"),
				),
			},
		},
	})
}

// TestAccMcpIntegrationResource_headersAuth tests creating an integration
// with header-based authentication.
func TestAccMcpIntegrationResource_headersAuth(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-hdrs")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMcpIntegrationResourceConfig(rName, "headers", "sse"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "auth_type", "headers"),
				),
			},
		},
	})
}

// TestAccMcpIntegrationResource_updateFields tests updating auth_type, transport,
// and URL in successive steps to verify in-place updates work correctly.
func TestAccMcpIntegrationResource_updateFields(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mcp-upd")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with SSE and no auth
			{
				Config: testAccMcpIntegrationResourceConfig(rName, "none", "sse"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "auth_type", "none"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "transport", "sse"),
				),
			},
			// Update auth_type and transport
			{
				Config: testAccMcpIntegrationResourceConfig(rName, "headers", "http"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "auth_type", "headers"),
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "transport", "http"),
				),
			},
			// Update URL
			{
				Config: testAccMcpIntegrationResourceConfigCustomURL(rName, "headers", "http", "https://example.com/mcp/v2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_integration.test", "url", "https://example.com/mcp/v2"),
				),
			},
		},
	})
}

func testAccMcpIntegrationResourceConfig(name, authType, transport string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name      = %[1]q
  url       = "https://example.com/mcp"
  auth_type = %[2]q
  transport = %[3]q
}
`, name, authType, transport)
}

func testAccMcpIntegrationResourceConfigWithDescription(name, authType, transport, description string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name        = %[1]q
  description = %[4]q
  url         = "https://example.com/mcp"
  auth_type   = %[2]q
  transport   = %[3]q
}
`, name, authType, transport, description)
}

func testAccMcpIntegrationResourceConfigCustomURL(name, authType, transport, url string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_mcp_integration" "test" {
  name      = %[1]q
  url       = %[4]q
  auth_type = %[2]q
  transport = %[3]q
}
`, name, authType, transport, url)
}
