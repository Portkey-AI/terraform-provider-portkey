package provider

import (
	"testing"

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
