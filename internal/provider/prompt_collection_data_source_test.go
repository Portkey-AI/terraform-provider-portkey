package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPromptCollectionDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-ds")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create a collection first, then look it up by ID
				Config: testAccPromptCollectionDataSourceConfig(rName, workspaceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.portkey_prompt_collection.test", "id",
						"portkey_prompt_collection.test", "id",
					),
					resource.TestCheckResourceAttr("data.portkey_prompt_collection.test", "name", rName),
					// Data source returns the canonical UUID from API, not the user-provided slug
					resource.TestCheckResourceAttrSet("data.portkey_prompt_collection.test", "workspace_id"),
					resource.TestCheckResourceAttr("data.portkey_prompt_collection.test", "status", "active"),
					resource.TestCheckResourceAttrSet("data.portkey_prompt_collection.test", "slug"),
					resource.TestCheckResourceAttrSet("data.portkey_prompt_collection.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.portkey_prompt_collection.test", "last_updated_at"),
				),
			},
		},
	})
}

func testAccPromptCollectionDataSourceConfig(name, workspaceID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_prompt_collection" "test" {
  name         = %[1]q
  workspace_id = %[2]q
}

data "portkey_prompt_collection" "test" {
  id = portkey_prompt_collection.test.id
}
`, name, workspaceID)
}
