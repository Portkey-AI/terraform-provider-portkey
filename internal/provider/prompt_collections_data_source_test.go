package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPromptCollectionsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPromptCollectionsDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_prompt_collections.all", "id"),
					resource.TestCheckResourceAttrSet("data.portkey_prompt_collections.all", "collections.#"),
				),
			},
		},
	})
}

func TestAccPromptCollectionsDataSource_withWorkspace(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-ds-list")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create a collection first, then list - ensures list has at least 1 item
				Config: testAccPromptCollectionsDataSourceConfigWithWorkspace(rName, workspaceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.portkey_prompt_collections.workspace", "id", workspaceID),
					resource.TestCheckResourceAttr("data.portkey_prompt_collections.workspace", "workspace_id", workspaceID),
					// Verify at least 1 collection exists
					resource.TestCheckResourceAttrSet("data.portkey_prompt_collections.workspace", "collections.#"),
				),
			},
		},
	})
}

func testAccPromptCollectionsDataSourceConfig() string {
	return `
provider "portkey" {}

data "portkey_prompt_collections" "all" {}
`
}

func testAccPromptCollectionsDataSourceConfigWithWorkspace(name, workspaceID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_prompt_collection" "test" {
  name         = %[1]q
  workspace_id = %[2]q
}

data "portkey_prompt_collections" "workspace" {
  workspace_id = %[2]q
  depends_on   = [portkey_prompt_collection.test]
}
`, name, workspaceID)
}
