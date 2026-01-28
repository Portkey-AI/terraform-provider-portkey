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

func TestAccIntegrationResource_withAzureOpenAIConfigurations(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-azure")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithAzureOpenAI(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithAzureOpenAI(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "azure-openai"
  key            = "test-azure-api-key-12345"

  configurations = jsonencode({
    azure_auth_mode     = "default"
    azure_resource_name = "test-azure-resource"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      }
    ]
  })
}
`, name)
}

func TestAccIntegrationResource_withAzureOpenAIMultipleDeployments(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-azure-multi")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithAzureOpenAIMultiple(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithAzureOpenAIMultiple(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "azure-openai"
  key            = "test-azure-api-key-12345"

  configurations = jsonencode({
    azure_auth_mode     = "default"
    azure_resource_name = "test-azure-resource"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      },
      {
        alias                 = "gpt35"
        azure_deployment_name = "gpt-35-turbo-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-35-turbo"
      }
    ]
  })
}
`, name)
}

func TestAccIntegrationResource_withAzureOpenAIEntraAuth(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-azure-entra")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithAzureOpenAIEntra(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithAzureOpenAIEntra(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "azure-openai"

  configurations = jsonencode({
    azure_auth_mode           = "entra"
    azure_resource_name       = "test-azure-resource"
    azure_entra_tenant_id     = "test-tenant-id-12345"
    azure_entra_client_id     = "test-client-id-12345"
    azure_entra_client_secret = "test-client-secret-12345"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      }
    ]
  })
}
`, name)
}

func TestAccIntegrationResource_withAzureOpenAIManagedAuth(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-azure-managed")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithAzureOpenAIManaged(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithAzureOpenAIManaged(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "azure-openai"

  configurations = jsonencode({
    azure_auth_mode         = "managed"
    azure_resource_name     = "test-azure-resource"
    azure_managed_client_id = "test-managed-client-id-12345"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      }
    ]
  })
}
`, name)
}

// Test updating Azure OpenAI configuration (add a deployment)
func TestAccIntegrationResource_updateAzureOpenAIConfig(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-azure-update")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with single deployment
			{
				Config: testAccIntegrationResourceConfigAzureOpenAISingle(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
				),
			},
			// Update: add another deployment
			{
				Config: testAccIntegrationResourceConfigAzureOpenAIUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigAzureOpenAISingle(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "azure-openai"
  key            = "test-azure-api-key-12345"

  configurations = jsonencode({
    azure_auth_mode     = "default"
    azure_resource_name = "test-azure-resource"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      }
    ]
  })
}
`, name)
}

func testAccIntegrationResourceConfigAzureOpenAIUpdated(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = "%[1]s-updated"
  ai_provider_id = "azure-openai"
  key            = "test-azure-api-key-12345"

  configurations = jsonencode({
    azure_auth_mode     = "default"
    azure_resource_name = "test-azure-resource-updated"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      },
      {
        alias                 = "gpt35"
        azure_deployment_name = "gpt-35-turbo-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-35-turbo"
      }
    ]
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

// Test key_wo without key_version - should create successfully with warning
func TestAccIntegrationResource_withWriteOnlyKeyNoVersion(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-wo-no-ver")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with key_wo but NO key_version - should work (with warning)
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKeyNoVersion(rName, "sk-test-key-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckNoResourceAttr("portkey_integration.test", "key_version"),
				),
			},
			// Update name only - key should NOT be sent (key_version is null and unchanged)
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKeyNoVersion(rName+"-updated", "sk-test-key-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName+"-updated"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithWriteOnlyKeyNoVersion(name, key string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key_wo         = %[2]q
}
`, name, key)
}
