package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPromptCollectionResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPromptCollectionResourceConfig(rName, workspaceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_prompt_collection.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_prompt_collection.test", "slug"),
					resource.TestCheckResourceAttr("portkey_prompt_collection.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_prompt_collection.test", "workspace_id", workspaceID),
					resource.TestCheckResourceAttr("portkey_prompt_collection.test", "status", "active"),
					resource.TestCheckResourceAttrSet("portkey_prompt_collection.test", "created_at"),
					resource.TestCheckResourceAttrSet("portkey_prompt_collection.test", "last_updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_prompt_collection.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "last_updated_at", "workspace_id", "parent_collection_id"},
			},
			// Update testing - change name
			{
				Config: testAccPromptCollectionResourceConfig(rName+"-updated", workspaceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_prompt_collection.test", "name", rName+"-updated"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccPromptCollectionResource_nested(t *testing.T) {
	rNameParent := acctest.RandomWithPrefix("tf-acc-parent")
	rNameChild := acctest.RandomWithPrefix("tf-acc-child")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create parent and child collections
			{
				Config: testAccPromptCollectionResourceConfigNested(rNameParent, rNameChild, workspaceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_prompt_collection.parent", "id"),
					resource.TestCheckResourceAttr("portkey_prompt_collection.parent", "name", rNameParent),
					resource.TestCheckResourceAttrSet("portkey_prompt_collection.child", "id"),
					resource.TestCheckResourceAttr("portkey_prompt_collection.child", "name", rNameChild),
					resource.TestCheckResourceAttrPair(
						"portkey_prompt_collection.child", "parent_collection_id",
						"portkey_prompt_collection.parent", "id",
					),
				),
			},
		},
	})
}

func testAccPromptCollectionResourceConfig(name, workspaceID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_prompt_collection" "test" {
  name         = %[1]q
  workspace_id = %[2]q
}
`, name, workspaceID)
}

func testAccPromptCollectionResourceConfigNested(parentName, childName, workspaceID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_prompt_collection" "parent" {
  name         = %[1]q
  workspace_id = %[3]q
}

resource "portkey_prompt_collection" "child" {
  name                 = %[2]q
  workspace_id         = %[3]q
  parent_collection_id = portkey_prompt_collection.parent.id
}
`, parentName, childName, workspaceID)
}
