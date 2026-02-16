package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func getTestUserID() string {
	if v := os.Getenv("TEST_USER_ID"); v != "" {
		return v
	}
	return "test-user-id"
}

func TestAccMcpServerUserAccessResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	workspaceID := getTestWorkspaceID()
	userID := getTestUserID()

	if userID == "test-user-id" {
		t.Skip("TEST_USER_ID must be set for this acceptance test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMcpServerUserAccessResourceConfig(rName, workspaceID, userID, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_mcp_server_user_access.test", "id"),
					resource.TestCheckResourceAttr("portkey_mcp_server_user_access.test", "user_id", userID),
					resource.TestCheckResourceAttr("portkey_mcp_server_user_access.test", "enabled", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "portkey_mcp_server_user_access.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing - disable access
			{
				Config: testAccMcpServerUserAccessResourceConfig(rName, workspaceID, userID, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_mcp_server_user_access.test", "enabled", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccMcpServerUserAccessResourceConfig(name, workspaceID, userID string, enabled bool) string {
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

resource "portkey_mcp_server_user_access" "test" {
  mcp_server_id = portkey_mcp_server.test.id
  user_id       = %[3]q
  enabled       = %[4]t
}
`, name, workspaceID, userID, enabled)
}
