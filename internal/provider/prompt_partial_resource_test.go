package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckPromptPartialDestroy verifies the prompt partial has been destroyed.
func testAccCheckPromptPartialDestroy(s *terraform.State) error {
	c, err := newTestClient()
	if err != nil {
		return fmt.Errorf("error creating test client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "portkey_prompt_partial" {
			continue
		}

		slug := rs.Primary.Attributes["slug"]
		if slug == "" {
			continue
		}

		_, err := c.GetPromptPartial(context.Background(), slug, "")
		if err == nil {
			return fmt.Errorf("prompt partial %s still exists", slug)
		}

		// If the error contains a 404-style message, the resource is gone
		if !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("unexpected error checking prompt partial %s: %s", slug, err)
		}
	}

	return nil
}

func TestAccPromptPartialResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPromptPartialDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPromptPartialResourceConfig(rName, "Hello, this is a reusable partial."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_prompt_partial.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_prompt_partial.test", "slug"),
					resource.TestCheckResourceAttr("portkey_prompt_partial.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_prompt_partial.test", "content", "Hello, this is a reusable partial."),
					resource.TestCheckResourceAttr("portkey_prompt_partial.test", "status", "active"),
					resource.TestCheckResourceAttr("portkey_prompt_partial.test", "version", "1"),
					resource.TestCheckResourceAttrSet("portkey_prompt_partial.test", "created_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_prompt_partial.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at", "version_description"},
			},
			// Update name testing (should not bump version)
			{
				Config: testAccPromptPartialResourceConfig(rName+"-renamed", "Hello, this is a reusable partial."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_prompt_partial.test", "name", rName+"-renamed"),
					resource.TestCheckResourceAttr("portkey_prompt_partial.test", "version", "1"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccPromptPartialResource_updateContent(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-content")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPromptPartialDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccPromptPartialResourceConfig(rName, "Version 1 content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_prompt_partial.test", "content", "Version 1 content"),
					resource.TestCheckResourceAttr("portkey_prompt_partial.test", "version", "1"),
				),
			},
			// Update content (should bump version)
			{
				Config: testAccPromptPartialResourceConfig(rName, "Version 2 content - updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_prompt_partial.test", "content", "Version 2 content - updated"),
					resource.TestCheckResourceAttr("portkey_prompt_partial.test", "version", "2"),
				),
			},
		},
	})
}

func testAccPromptPartialResourceConfig(name, content string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_prompt_partial" "test" {
  name    = %[1]q
  content = %[2]q
}
`, name, content)
}
