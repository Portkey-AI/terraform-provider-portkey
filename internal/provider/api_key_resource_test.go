package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAPIKeyResource_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-apikey")
	nameUpdated := name + "-updated"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccAPIKeyResourceConfig(name, "organisation", "service"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "key"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", name),
					resource.TestCheckResourceAttr("portkey_api_key.test", "type", "organisation"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "sub_type", "service"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "status", "active"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"}, // Key is only returned on creation
			},
			// Update and Read testing
			{
				Config: testAccAPIKeyResourceConfigWithDescription(nameUpdated, "organisation", "service", "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("portkey_api_key.test", "description", "Updated description"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccAPIKeyResource_withScopes(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-apiscope")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with scopes
			{
				Config: testAccAPIKeyResourceConfigWithScopes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", name),
					resource.TestCheckResourceAttr("portkey_api_key.test", "scopes.#", "2"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccAPIKeyResource_withMetadata(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-apimeta")
	nameUpdated := name + "-upd"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with metadata
			{
				Config: testAccAPIKeyResourceConfigWithMetadata(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", name),
					resource.TestCheckResourceAttr("portkey_api_key.test", "metadata.%", "2"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "metadata._user", "test-service"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "metadata.service_uuid", "abc123"),
				),
			},
			// Update metadata
			{
				Config: testAccAPIKeyResourceConfigWithMetadataUpdated(nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("portkey_api_key.test", "metadata.%", "3"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "metadata._user", "updated-service"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "metadata.service_uuid", "xyz789"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "metadata.environment", "production"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccAPIKeyResource_withAlertEmails(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-apialert")
	nameUpdated := name + "-upd"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with alert_emails
			{
				Config: testAccAPIKeyResourceConfigWithAlertEmails(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", name),
					resource.TestCheckResourceAttr("portkey_api_key.test", "alert_emails.#", "1"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "alert_emails.0", "test@example.com"),
				),
			},
			// Update alert_emails
			{
				Config: testAccAPIKeyResourceConfigWithAlertEmailsUpdated(nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("portkey_api_key.test", "alert_emails.#", "2"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccAPIKeyResource_withMetadataAndAlertEmails(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-apifull")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with both metadata and alert_emails
			{
				Config: testAccAPIKeyResourceConfigWithMetadataAndAlertEmails(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", name),
					resource.TestCheckResourceAttr("portkey_api_key.test", "metadata.%", "2"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "metadata._user", "full-test"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "alert_emails.#", "2"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"}, // Key is only returned on creation
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccAPIKeyResourceConfig(name, keyType, subType string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = %[2]q
  sub_type = %[3]q
  scopes   = ["providers.list"]
}
`, name, keyType, subType)
}

func testAccAPIKeyResourceConfigWithDescription(name, keyType, subType, description string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name        = %[1]q
  type        = %[2]q
  sub_type    = %[3]q
  description = %[4]q
  scopes      = ["providers.list"]
}
`, name, keyType, subType, description)
}

func testAccAPIKeyResourceConfigWithScopes(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["logs.list", "logs.view"]
}
`, name)
}

func testAccAPIKeyResourceConfigWithMetadata(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  metadata = {
    "_user"        = "test-service"
    "service_uuid" = "abc123"
  }
}
`, name)
}

func testAccAPIKeyResourceConfigWithMetadataUpdated(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  metadata = {
    "_user"        = "updated-service"
    "service_uuid" = "xyz789"
    "environment"  = "production"
  }
}
`, name)
}

func testAccAPIKeyResourceConfigWithAlertEmails(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  alert_emails = ["test@example.com"]
}
`, name)
}

func testAccAPIKeyResourceConfigWithAlertEmailsUpdated(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  alert_emails = ["test@example.com", "admin@example.com"]
}
`, name)
}

func testAccAPIKeyResourceConfigWithMetadataAndAlertEmails(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  metadata = {
    "_user"        = "full-test"
    "service_uuid" = "full123"
  }

  alert_emails = ["test@example.com", "admin@example.com"]
}
`, name)
}

func TestAccAPIKeyResource_withUsageLimits(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-apilim")
	nameUpdated := name + "-upd"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with usage_limits
			{
				Config: testAccAPIKeyResourceConfigWithUsageLimits(name, 500, "monthly"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", name),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.credit_limit", "500"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.periodic_reset", "monthly"),
				),
			},
			// Update usage_limits
			{
				Config: testAccAPIKeyResourceConfigWithUsageLimits(nameUpdated, 1000, "weekly"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.credit_limit", "1000"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.periodic_reset", "weekly"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccAPIKeyResource_withRateLimits(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-apirl")
	nameUpdated := name + "-upd"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with rate_limits
			{
				Config: testAccAPIKeyResourceConfigWithRateLimits(name, "requests", "rpm", 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", name),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rate_limits.#", "1"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rate_limits.0.type", "requests"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rate_limits.0.unit", "rpm"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rate_limits.0.value", "100"),
				),
			},
			// Update rate_limits
			{
				Config: testAccAPIKeyResourceConfigWithRateLimits(nameUpdated, "requests", "rpd", 5000),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rate_limits.#", "1"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rate_limits.0.type", "requests"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rate_limits.0.unit", "rpd"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rate_limits.0.value", "5000"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccAPIKeyResourceConfigWithUsageLimits(name string, creditLimit int, periodicReset string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  usage_limits = {
    credit_limit   = %[2]d
    periodic_reset = %[3]q
  }
}
`, name, creditLimit, periodicReset)
}

func testAccAPIKeyResourceConfigWithRateLimits(name, rlType, rlUnit string, rlValue int) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  rate_limits {
    type  = %[2]q
    unit  = %[3]q
    value = %[4]d
  }
}
`, name, rlType, rlUnit, rlValue)
}
