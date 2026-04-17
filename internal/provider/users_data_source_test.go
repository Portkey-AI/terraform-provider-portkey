package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUsersDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_users.all", "users.#"),
				),
			},
		},
	})
}

func TestAccUsersDataSource_filterByEmail(t *testing.T) {
	var firstUserEmail string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					extractAttr("data.portkey_users.all", "users.0.email", &firstUserEmail),
				),
			},
			{
				Config: testAccUsersDataSourceFilterByEmailConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.portkey_users.filtered", "users.#", "1"),
					resource.TestCheckResourceAttrSet("data.portkey_users.filtered", "users.0.id"),
					resource.TestCheckResourceAttrSet("data.portkey_users.filtered", "users.0.email"),
				),
			},
		},
	})
}

func TestAccUsersDataSource_filterByRole(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersDataSourceFilterByRoleConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_users.by_role", "users.#"),
					resource.TestCheckResourceAttr("data.portkey_users.by_role", "users.0.role", "admin"),
				),
			},
		},
	})
}

func testAccUsersDataSourceConfig() string {
	return `
provider "portkey" {}

data "portkey_users" "all" {}
`
}

func testAccUsersDataSourceFilterByEmailConfig() string {
	return `
provider "portkey" {}

data "portkey_users" "all" {}

data "portkey_users" "filtered" {
  email = data.portkey_users.all.users[0].email
}
`
}

func testAccUsersDataSourceFilterByRoleConfig() string {
	return `
provider "portkey" {}

data "portkey_users" "by_role" {
  role = "admin"
}
`
}

// extractAttr saves a Terraform state attribute value into the provided string pointer
// for use across test steps.
func extractAttr(resourceName, attr string, dest *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}
		val, ok := rs.Primary.Attributes[attr]
		if !ok {
			return fmt.Errorf("attribute %s not found on %s", attr, resourceName)
		}
		*dest = val
		return nil
	}
}
