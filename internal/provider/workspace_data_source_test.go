package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWorkspaceDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkspaceDataSourceConfig(rName, "Test description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check the data source returns the correct workspace
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "description", "Test description"),
					resource.TestCheckResourceAttrSet("data.portkey_workspace.test", "id"),
					resource.TestCheckResourceAttrSet("data.portkey_workspace.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.portkey_workspace.test", "updated_at"),
				),
			},
		},
	})
}

func TestAccWorkspaceDataSource_withUsageLimits(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-ds-ul")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkspaceDataSourceConfigWithUsageLimits(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttrSet("data.portkey_workspace.test", "id"),
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "usage_limits.#", "1"),
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "usage_limits.0.type", "cost"),
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "usage_limits.0.credit_limit", "500"),
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "usage_limits.0.alert_threshold", "400"),
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "usage_limits.0.periodic_reset", "monthly"),
				),
			},
		},
	})
}

func TestAccWorkspaceDataSource_withRateLimits(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-ds-rl")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkspaceDataSourceConfigWithRateLimits(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttrSet("data.portkey_workspace.test", "id"),
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "rate_limits.#", "1"),
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "rate_limits.0.type", "requests"),
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "rate_limits.0.unit", "rpm"),
					resource.TestCheckResourceAttr("data.portkey_workspace.test", "rate_limits.0.value", "100"),
				),
			},
		},
	})
}

func testAccWorkspaceDataSourceConfig(name, description string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = %[2]q
}

data "portkey_workspace" "test" {
  id = portkey_workspace.test.id
}
`, name, description)
}

func testAccWorkspaceDataSourceConfigWithUsageLimits(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Workspace with usage limits for data source test"

  usage_limits {
    type            = "cost"
    credit_limit    = 500
    alert_threshold = 400
    periodic_reset  = "monthly"
  }
}

data "portkey_workspace" "test" {
  id = portkey_workspace.test.id
}
`, name)
}

func testAccWorkspaceDataSourceConfigWithRateLimits(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Workspace with rate limits for data source test"

  rate_limits {
    type  = "requests"
    unit  = "rpm"
    value = 100
  }
}

data "portkey_workspace" "test" {
  id = portkey_workspace.test.id
}
`, name)
}
