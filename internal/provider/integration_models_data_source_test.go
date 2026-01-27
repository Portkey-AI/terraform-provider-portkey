package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccIntegrationModelsDataSource_basic tests the basic data source functionality.
func TestAccIntegrationModelsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationModelsDataSourceConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_integration_models.test", "id"),
					resource.TestCheckResourceAttrSet("data.portkey_integration_models.test", "integration_id"),
					resource.TestCheckResourceAttrSet("data.portkey_integration_models.test", "allow_all_models"),
					// Models list should exist
					resource.TestCheckResourceAttrSet("data.portkey_integration_models.test", "models.#"),
				),
			},
		},
	})
}

// testAccIntegrationModelsDataSourceConfigBasic creates a basic data source configuration
func testAccIntegrationModelsDataSourceConfigBasic() string {
	return `
provider "portkey" {}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

data "portkey_integration_models" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
}
`
}
