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

// TestAccPromptCollectionDataSource_slugPreserved tests that the data source
// returns "id" exactly as configured when looked up by slug. GET /collections/{id}
// accepts a slug but always echoes the UUID, so assigning the API value to state.ID
// made the attribute change between plan and apply.
func TestAccPromptCollectionDataSource_slugPreserved(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-pc-ds-slug")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPromptCollectionDataSourceSlugConfig(rName, workspaceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// The configured slug must survive the read unchanged...
					resource.TestCheckResourceAttrPair(
						"data.portkey_prompt_collection.by_slug", "id",
						"portkey_prompt_collection.test", "slug",
					),
					// ...and must NOT have been replaced by the API's UUID.
					resource.TestCheckResourceAttrPair(
						"data.portkey_prompt_collection.by_slug", "slug",
						"portkey_prompt_collection.test", "slug",
					),
					resource.TestCheckResourceAttr("data.portkey_prompt_collection.by_slug", "name", rName),
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

func testAccPromptCollectionDataSourceSlugConfig(name, workspaceID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_prompt_collection" "test" {
  name         = %[1]q
  workspace_id = %[2]q
}

data "portkey_prompt_collection" "by_slug" {
  id = portkey_prompt_collection.test.slug
}
`, name, workspaceID)
}
