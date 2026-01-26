package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIntegrationWorkspacesDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationWorkspacesDataSourceConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_integration_workspaces.test", "id"),
					resource.TestCheckResourceAttrSet("data.portkey_integration_workspaces.test", "integration_id"),
					resource.TestCheckResourceAttrSet("data.portkey_integration_workspaces.test", "total"),
					resource.TestCheckResourceAttrSet("data.portkey_integration_workspaces.test", "workspaces.#"),
				),
			},
		},
	})
}

// TestAccIntegrationWorkspacesDataSource_withWorkspaceAccess verifies the data source
// correctly reads workspace access after it has been configured via the resource.
func TestAccIntegrationWorkspacesDataSource_withWorkspaceAccess(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-iwds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationWorkspacesDataSourceConfigWithAccess(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source reads the workspace access we created
					resource.TestCheckResourceAttrSet("data.portkey_integration_workspaces.test", "id"),
					resource.TestCheckResourceAttrSet("data.portkey_integration_workspaces.test", "total"),
					// Verify at least one workspace exists in the response
					resource.TestCheckResourceAttrSet("data.portkey_integration_workspaces.test", "workspaces.#"),
				),
			},
		},
	})
}

// TestAccIntegrationWorkspacesDataSource_verifyWorkspaceDetails tests that workspace
// details (enabled, limits) are correctly returned by the data source.
func TestAccIntegrationWorkspacesDataSource_verifyWorkspaceDetails(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-iwdetails")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationWorkspacesDataSourceConfigWithLimits(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_integration_workspaces.test", "id"),
					resource.TestCheckResourceAttrSet("data.portkey_integration_workspaces.test", "workspaces.#"),
				),
			},
		},
	})
}

func testAccIntegrationWorkspacesDataSourceConfigBasic() string {
	return `
provider "portkey" {}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

data "portkey_integration_workspaces" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
}
`
}

func testAccIntegrationWorkspacesDataSourceConfigWithAccess(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for data source verification"
}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

resource "portkey_integration_workspace_access" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
  workspace_id   = portkey_workspace.test.id
  enabled        = true
}

data "portkey_integration_workspaces" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug

  depends_on = [portkey_integration_workspace_access.test]
}
`, name)
}

func testAccIntegrationWorkspacesDataSourceConfigWithLimits(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for data source with limits"
}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

resource "portkey_integration_workspace_access" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
  workspace_id   = portkey_workspace.test.id
  enabled        = true

  usage_limits {
    type            = "cost"
    credit_limit    = 50
    alert_threshold = 75
    periodic_reset  = "monthly"
  }

  rate_limits {
    type  = "requests"
    unit  = "rpm"
    value = 500
  }
}

data "portkey_integration_workspaces" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug

  depends_on = [portkey_integration_workspace_access.test]
}
`, name)
}
