package provider

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAPIKeyDataSource_basic(t *testing.T) {
	rnd := rand.Int63()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an API key first, then read it with data source
			{
				Config: testAccAPIKeyDataSourceConfigRnd(rnd),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("data.portkey_api_key.test", "name"),
					resource.TestCheckResourceAttrSet("data.portkey_api_key.test", "type"),
					resource.TestCheckResourceAttrSet("data.portkey_api_key.test", "sub_type"),
					resource.TestCheckResourceAttr("data.portkey_api_key.test", "status", "active"),
				),
			},
		},
	})
}

func TestAccAPIKeyDataSource_withUsageLimits(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ds-akul")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyDataSourceConfigWithUsageLimits(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("data.portkey_api_key.test", "name", name),
					resource.TestCheckResourceAttr("data.portkey_api_key.test", "status", "active"),
					resource.TestCheckResourceAttr("data.portkey_api_key.test", "usage_limits.credit_limit", "500"),
					resource.TestCheckResourceAttr("data.portkey_api_key.test", "usage_limits.periodic_reset", "monthly"),
				),
			},
		},
	})
}

func TestAccAPIKeyDataSource_withRateLimits(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ds-akrl")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyDataSourceConfigWithRateLimits(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("data.portkey_api_key.test", "name", name),
					resource.TestCheckResourceAttr("data.portkey_api_key.test", "status", "active"),
					resource.TestCheckResourceAttr("data.portkey_api_key.test", "rate_limits.#", "1"),
					resource.TestCheckResourceAttr("data.portkey_api_key.test", "rate_limits.0.type", "requests"),
					resource.TestCheckResourceAttr("data.portkey_api_key.test", "rate_limits.0.unit", "rpm"),
					resource.TestCheckResourceAttr("data.portkey_api_key.test", "rate_limits.0.value", "100"),
				),
			},
		},
	})
}

func testAccAPIKeyDataSourceConfigRnd(rnd int64) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = "tf-acc-test-api-key-ds-%d"
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]
}

data "portkey_api_key" "test" {
  id = portkey_api_key.test.id
}
`, rnd)
}

func testAccAPIKeyDataSourceConfigWithUsageLimits(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  usage_limits = {
    credit_limit      = 500
    periodic_reset = "monthly"
  }
}

data "portkey_api_key" "test" {
  id = portkey_api_key.test.id
}
`, name)
}

func testAccAPIKeyDataSourceConfigWithRateLimits(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  rate_limits {
    type  = "requests"
    unit  = "rpm"
    value = 100
  }
}

data "portkey_api_key" "test" {
  id = portkey_api_key.test.id
}
`, name)
}
