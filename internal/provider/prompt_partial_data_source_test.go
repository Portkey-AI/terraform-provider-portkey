package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPromptPartialDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create a prompt partial first, then look it up via the data source
				Config: testAccPromptPartialDataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.portkey_prompt_partial.test", "slug",
						"portkey_prompt_partial.test", "slug",
					),
					resource.TestCheckResourceAttr("data.portkey_prompt_partial.test", "name", rName),
					resource.TestCheckResourceAttr("data.portkey_prompt_partial.test", "content", "Data source test content."),
					resource.TestCheckResourceAttrSet("data.portkey_prompt_partial.test", "id"),
					resource.TestCheckResourceAttrSet("data.portkey_prompt_partial.test", "partial_version"),
					resource.TestCheckResourceAttr("data.portkey_prompt_partial.test", "status", "active"),
					resource.TestCheckResourceAttrSet("data.portkey_prompt_partial.test", "created_at"),
				),
			},
		},
	})
}

func testAccPromptPartialDataSourceConfig(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_prompt_partial" "test" {
  name         = %[1]q
  content      = "Data source test content."
  workspace_id = %[2]q
}

data "portkey_prompt_partial" "test" {
  slug = portkey_prompt_partial.test.slug
}
`, name, getTestWorkspaceID())
}
