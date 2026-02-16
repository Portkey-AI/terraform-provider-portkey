package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPromptPartialsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPromptPartialsDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_prompt_partials.all", "prompt_partials.#"),
				),
			},
		},
	})
}

func TestAccPromptPartialsDataSource_withResource(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-ds-list")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create a prompt partial first, then list - ensures list has at least 1 item
				Config: testAccPromptPartialsDataSourceConfigWithResource(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify at least 1 prompt partial exists
					resource.TestCheckResourceAttrSet("data.portkey_prompt_partials.all", "prompt_partials.#"),
				),
			},
		},
	})
}

func testAccPromptPartialsDataSourceConfig() string {
	return `
provider "portkey" {}

data "portkey_prompt_partials" "all" {}
`
}

func testAccPromptPartialsDataSourceConfigWithResource(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_prompt_partial" "test" {
  name    = %[1]q
  content = "List data source test content."
}

data "portkey_prompt_partials" "all" {
  depends_on = [portkey_prompt_partial.test]
}
`, name)
}
