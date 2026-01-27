package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccIntegrationModelAccessResource_basic tests the basic model access lifecycle.
func TestAccIntegrationModelAccessResource_basic(t *testing.T) {
	// This test requires an existing integration with models available
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create model access (enable a model)
			{
				Config: testAccIntegrationModelAccessResourceConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration_model_access.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_integration_model_access.test", "integration_id"),
					resource.TestCheckResourceAttrSet("portkey_integration_model_access.test", "model_slug"),
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "enabled", "true"),
				),
			},
			// Import testing
			{
				ResourceName:      "portkey_integration_model_access.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: integrationModelAccessImportStateIdFunc("portkey_integration_model_access.test"),
			},
		},
	})
}

// TestAccIntegrationModelAccessResource_disable tests disabling a model.
func TestAccIntegrationModelAccessResource_disable(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create enabled
			{
				Config: testAccIntegrationModelAccessResourceConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "enabled", "true"),
				),
			},
			// Update to disabled
			{
				Config: testAccIntegrationModelAccessResourceConfigDisabled(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "enabled", "false"),
				),
			},
		},
	})
}

// TestAccIntegrationModelAccessResource_withPricing tests model access with custom pricing.
func TestAccIntegrationModelAccessResource_withPricing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with pricing
			{
				Config: testAccIntegrationModelAccessResourceConfigWithPricing(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration_model_access.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "enabled", "true"),
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "pricing_config.type", "static"),
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "pricing_config.pay_as_you_go.request_token_price", "0.03"),
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "pricing_config.pay_as_you_go.response_token_price", "0.06"),
				),
			},
		},
	})
}

// TestAccIntegrationModelAccessResource_updatePricing tests updating pricing configuration.
func TestAccIntegrationModelAccessResource_updatePricing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with initial pricing
			{
				Config: testAccIntegrationModelAccessResourceConfigWithPricing(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "pricing_config.pay_as_you_go.request_token_price", "0.03"),
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "pricing_config.pay_as_you_go.response_token_price", "0.06"),
				),
			},
			// Update pricing values
			{
				Config: testAccIntegrationModelAccessResourceConfigWithUpdatedPricing(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "pricing_config.pay_as_you_go.request_token_price", "0.05"),
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "pricing_config.pay_as_you_go.response_token_price", "0.10"),
				),
			},
		},
	})
}

// TestAccIntegrationModelAccessResource_customModel tests custom/fine-tuned model creation.
func TestAccIntegrationModelAccessResource_customModel(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-custom")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create custom model
			{
				Config: testAccIntegrationModelAccessResourceConfigCustomModel(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration_model_access.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "enabled", "true"),
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "is_custom", "true"),
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "is_finetune", "true"),
					resource.TestCheckResourceAttr("portkey_integration_model_access.test", "base_model_slug", "gpt-3.5-turbo"),
				),
			},
		},
	})
}

// integrationModelAccessImportStateIdFunc returns a function that generates the import ID
func integrationModelAccessImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}

		integrationID := rs.Primary.Attributes["integration_id"]
		modelSlug := rs.Primary.Attributes["model_slug"]
		return fmt.Sprintf("%s/%s", integrationID, modelSlug), nil
	}
}

// testAccIntegrationModelAccessResourceConfigBasic creates a basic model access configuration
func testAccIntegrationModelAccessResourceConfigBasic() string {
	return `
provider "portkey" {}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

# Get models available for the first integration
data "portkey_integration_models" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
}

resource "portkey_integration_model_access" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
  model_slug     = data.portkey_integration_models.test.models[0].slug
  enabled        = true
}
`
}

// testAccIntegrationModelAccessResourceConfigDisabled creates disabled model access
func testAccIntegrationModelAccessResourceConfigDisabled() string {
	return `
provider "portkey" {}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

# Get models available for the first integration
data "portkey_integration_models" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
}

resource "portkey_integration_model_access" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
  model_slug     = data.portkey_integration_models.test.models[0].slug
  enabled        = false
}
`
}

// testAccIntegrationModelAccessResourceConfigWithPricing creates model access with custom pricing
func testAccIntegrationModelAccessResourceConfigWithPricing() string {
	return `
provider "portkey" {}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

# Get models available for the first integration
data "portkey_integration_models" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
}

resource "portkey_integration_model_access" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
  model_slug     = data.portkey_integration_models.test.models[0].slug
  enabled        = true

  pricing_config = {
    type = "static"
    pay_as_you_go = {
      request_token_price  = 0.03
      response_token_price = 0.06
    }
  }
}
`
}

// testAccIntegrationModelAccessResourceConfigWithUpdatedPricing creates model access with updated pricing
func testAccIntegrationModelAccessResourceConfigWithUpdatedPricing() string {
	return `
provider "portkey" {}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

# Get models available for the first integration
data "portkey_integration_models" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
}

resource "portkey_integration_model_access" "test" {
  integration_id = data.portkey_integrations.all.integrations[0].slug
  model_slug     = data.portkey_integration_models.test.models[0].slug
  enabled        = true

  pricing_config = {
    type = "static"
    pay_as_you_go = {
      request_token_price  = 0.05
      response_token_price = 0.10
    }
  }
}
`
}

// testAccIntegrationModelAccessResourceConfigCustomModel creates a custom/fine-tuned model
func testAccIntegrationModelAccessResourceConfigCustomModel(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

# Get existing integrations from the organization
data "portkey_integrations" "all" {}

resource "portkey_integration_model_access" "test" {
  integration_id  = data.portkey_integrations.all.integrations[0].slug
  model_slug      = "ft:gpt-3.5-turbo:my-org:%s"
  enabled         = true
  is_custom       = true
  is_finetune     = true
  base_model_slug = "gpt-3.5-turbo"
}
`, name)
}
