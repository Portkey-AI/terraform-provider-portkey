package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIntegrationResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccIntegrationResourceConfig(rName, "openai", "Initial description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_integration.test", "slug"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "openai"),
					resource.TestCheckResourceAttr("portkey_integration.test", "description", "Initial description"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
					resource.TestCheckResourceAttrSet("portkey_integration.test", "created_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "portkey_integration.test",
				ImportState:       true,
				ImportStateVerify: true,
				// key, key_wo: write-only, never returned by API
				// key_version: not stored on server, local trigger only
				// configurations: not returned by API
				// slug: BUG - API returns UUID instead of original slug on GET (needs investigation)
				// updated_at: timestamp may change between operations
				ImportStateVerifyIgnore: []string{"key", "key_wo", "key_version", "configurations", "slug", "updated_at"},
			},
			// Update testing
			{
				Config: testAccIntegrationResourceConfig(rName+"-updated", "openai", "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("portkey_integration.test", "description", "Updated description"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccIntegrationResource_withSlug(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	slug := acctest.RandomWithPrefix("tf-slug")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithSlug(rName, slug, "openai"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "slug", slug),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "openai"),
				),
			},
		},
	})
}

func TestAccIntegrationResource_updateName(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-rename")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfig(rName, "openai", "Initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
				),
			},
			{
				Config: testAccIntegrationResourceConfig(rName+"-renamed", "openai", "Initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName+"-renamed"),
				),
			},
		},
	})
}

func TestAccIntegrationResource_withConfigurations(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-config")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithConfigurations(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "aws-bedrock"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfig(name, aiProviderID, description string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = %[2]q
  description    = %[3]q
  key            = "sk-test-fake-key-12345"
}
`, name, aiProviderID, description)
}

func testAccIntegrationResourceConfigWithSlug(name, slug, aiProviderID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  slug           = %[2]q
  ai_provider_id = %[3]q
  key            = "sk-test-fake-key-12345"
}
`, name, slug, aiProviderID)
}

func testAccIntegrationResourceConfigWithConfigurations(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "aws-bedrock"

  configurations = jsonencode({
    aws_role_arn = "arn:aws:iam::123456789012:role/TestRole"
    aws_region   = "us-east-1"
  })
}
`, name)
}

// Tests for write-only key (key_wo) and key_version trigger

func TestAccIntegrationResource_withWriteOnlyKey(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-wo-key")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with key_wo and key_version
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKey(rName, "sk-test-key-1", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "key_version", "1"),
				),
			},
			// Update key_version to trigger key update
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKey(rName, "sk-test-key-2", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "key_version", "2"),
				),
			},
		},
	})
}

func TestAccIntegrationResource_keyVersionNoChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-no-key-change")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKey(rName, "sk-test-key", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "key_version", "1"),
				),
			},
			// Update name but not key_version - key should NOT be sent
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKey(rName+"-updated", "sk-test-key", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("portkey_integration.test", "key_version", "1"),
				),
			},
		},
	})
}

func TestAccIntegrationResource_withOpenAIConfigurations(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-openai-config")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithOpenAI(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "openai"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithWriteOnlyKey(name, key string, keyVersion int) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key_wo         = %[2]q
  key_version    = %[3]d
}
`, name, key, keyVersion)
}

func testAccIntegrationResourceConfigWithOpenAI(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key_wo         = "sk-test-fake-key-12345"
  key_version    = 1

  configurations = jsonencode({
    openai_organization = "org-test123"
    openai_project      = "proj-test456"
  })
}
`, name)
}

func TestAccIntegrationResource_conflictKeyAndKeyWO(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-conflict")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccIntegrationResourceConfigConflict(rName),
				ExpectError: regexp.MustCompile(`Conflicting API Key Attributes`),
			},
		},
	})
}

func testAccIntegrationResourceConfigConflict(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key            = "sk-deprecated-key"
  key_wo         = "sk-write-only-key"
  key_version    = 1
}
`, name)
}
