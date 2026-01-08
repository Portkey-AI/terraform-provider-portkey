package provider

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAPIKeyResource_basic(t *testing.T) {
	rnd := rand.Int63()
	name := fmt.Sprintf("tf-acc-test-api-key-%d", rnd)
	nameUpdated := fmt.Sprintf("tf-acc-test-api-key-%d-updated", rnd)
	resource.Test(t, resource.TestCase{
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
	rnd := rand.Int63()
	name := fmt.Sprintf("tf-acc-test-api-key-scopes-%d", rnd)
	resource.Test(t, resource.TestCase{
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
	rnd := rand.Int63() % 1000000 // Keep random number shorter
	name := fmt.Sprintf("tf-acc-meta-%d", rnd)
	nameUpdated := fmt.Sprintf("tf-acc-meta-%d-upd", rnd)
	resource.Test(t, resource.TestCase{
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
	rnd := rand.Int63() % 1000000 // Keep random number shorter
	name := fmt.Sprintf("tf-acc-alerts-%d", rnd)
	nameUpdated := fmt.Sprintf("tf-acc-alerts-%d-upd", rnd)
	resource.Test(t, resource.TestCase{
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
	rnd := rand.Int63() % 1000000 // Keep random number shorter
	name := fmt.Sprintf("tf-acc-full-%d", rnd)
	resource.Test(t, resource.TestCase{
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
