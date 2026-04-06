package provider

import (
	"fmt"
	"regexp"
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

// TestAccAPIKeyResource_clearUsageLimits verifies that removing usage_limits
// from the Terraform config sends null to the API and clears the limits.
func TestAccAPIKeyResource_clearUsageLimits(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-clear")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with usage_limits
			{
				Config: testAccAPIKeyResourceConfigWithUsageLimits(name, 500, "monthly"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.credit_limit", "500"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.periodic_reset", "monthly"),
				),
			},
			// Step 2: Remove usage_limits from config — should clear them
			{
				Config: testAccAPIKeyResourceConfigNoLimits(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("portkey_api_key.test", "usage_limits.credit_limit"),
				),
			},
		},
	})
}

// TestAccAPIKeyResource_clearRateLimits verifies that removing rate_limits
// from the Terraform config sends null to the API and clears the limits.
func TestAccAPIKeyResource_clearRateLimits(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-clrl")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with rate_limits
			{
				Config: testAccAPIKeyResourceConfigWithRateLimits(name, "requests", "rpm", 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_api_key.test", "rate_limits.#", "1"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rate_limits.0.type", "requests"),
				),
			},
			// Step 2: Remove rate_limits from config — should clear them
			{
				Config: testAccAPIKeyResourceConfigNoLimits(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("portkey_api_key.test", "rate_limits.0.type"),
				),
			},
		},
	})
}

func testAccAPIKeyResourceConfigNoLimits(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]
}
`, name)
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

  rate_limits = [{
    type  = %[2]q
    unit  = %[3]q
    value = %[4]d
  }]
}
`, name, rlType, rlUnit, rlValue)
}

// TestAccAPIKeyResource_withConfigID verifies that:
//   - An API key can be created with config_id and allow_config_override = false,
//     and the attributes are read back correctly from the API.
//   - allow_config_override can be flipped to true in a subsequent apply while
//     config_id is intentionally omitted from HCL, relying on the Computed
//     attribute to preserve the existing API binding from state. This exercises
//     the Optional+Computed readback path and confirms that ModifyPlan accepts
//     the update because state already has a non-empty config_id.
func TestAccAPIKeyResource_withConfigID(t *testing.T) {
	keyName := acctest.RandomWithPrefix("tf-acc-ak-cfg")
	configName := acctest.RandomWithPrefix("tf-acc-cfg")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create an API key bound to a freshly created config,
			// with allow_config_override explicitly disabled.
			// Both config_id and allow_config_override are present in HCL.
			{
				Config: testAccAPIKeyResourceConfigWithConfigID(configName, workspaceID, keyName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "key"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", keyName),
					// config_id must match the ID of the portkey_config we created.
					resource.TestCheckResourceAttrPair(
						"portkey_api_key.test", "config_id",
						"portkey_config.test", "id",
					),
					resource.TestCheckResourceAttr("portkey_api_key.test", "allow_config_override", "false"),
				),
			},
			// Step 2: Flip allow_config_override to true while OMITTING config_id
			// from the HCL config entirely. The Computed attribute must preserve
			// the existing binding from state, and ModifyPlan must accept this
			// because state.ConfigID is non-empty.
			{
				Config: testAccAPIKeyResourceConfigUpdateOverrideOnly(configName, workspaceID, keyName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", keyName),
					resource.TestCheckResourceAttr("portkey_api_key.test", "allow_config_override", "true"),
					// config_id must still be readable from state even though it
					// was omitted from HCL — Computed reads it back from the API.
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "config_id"),
				),
			},
			// Step 3: Import — verify the state round-trips cleanly.
			{
				ResourceName:            "portkey_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},
			},
		},
	})
}

// TestAccAPIKeyResource_allowConfigOverrideWithoutConfigID verifies that the
// provider rejects allow_config_override = true when config_id is not set,
// producing a clear attribute-level error at plan time before any API call is made.
func TestAccAPIKeyResource_allowConfigOverrideWithoutConfigID(t *testing.T) {
	keyName := acctest.RandomWithPrefix("tf-acc-ak-noconf")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAPIKeyResourceConfigWithAllowOverrideNoConfigID(keyName),
				ExpectError: regexp.MustCompile(`allow_config_override can only be set to true when config_id is also specified`),
			},
		},
	})
}

// testAccAPIKeyResourceConfigWithConfigID builds a Terraform config that
// creates both a portkey_config and a portkey_api_key bound to it.
// Both config_id and allow_config_override are explicitly present in HCL.
func testAccAPIKeyResourceConfigWithConfigID(configName, workspaceID, keyName string, allowOverride bool) string {
	return fmt.Sprintf(`
resource "portkey_config" "test" {
  name         = %[1]q
  workspace_id = %[2]q
  config       = "{\"retry\":{\"attempts\":3}}"
}

resource "portkey_api_key" "test" {
  name                  = %[3]q
  type                  = "organisation"
  sub_type              = "service"
  scopes                = ["providers.list"]
  config_id             = portkey_config.test.id
  allow_config_override = %[4]t
}
`, configName, workspaceID, keyName, allowOverride)
}

// testAccAPIKeyResourceConfigUpdateOverrideOnly builds a Terraform config that
// sets allow_config_override = true but intentionally OMITS config_id from HCL.
// This exercises the Optional+Computed readback path: config_id is not in HCL,
// so Terraform reads it from state (the prior API binding). ModifyPlan must
// accept this because state already holds a non-empty config_id.
func testAccAPIKeyResourceConfigUpdateOverrideOnly(configName, workspaceID, keyName string) string {
	return fmt.Sprintf(`
resource "portkey_config" "test" {
  name         = %[1]q
  workspace_id = %[2]q
  config       = "{\"retry\":{\"attempts\":3}}"
}

resource "portkey_api_key" "test" {
  name                  = %[3]q
  type                  = "organisation"
  sub_type              = "service"
  scopes                = ["providers.list"]
  allow_config_override = true
  # config_id intentionally omitted: Computed preserves the existing API binding from state.
}
`, configName, workspaceID, keyName)
}

// testAccAPIKeyResourceConfigWithAllowOverrideNoConfigID builds a config that
// sets allow_config_override = true without a config_id — this must be rejected
// by the provider at plan time.
func testAccAPIKeyResourceConfigWithAllowOverrideNoConfigID(keyName string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name                 = %[1]q
  type                 = "organisation"
  sub_type             = "service"
  scopes               = ["providers.list"]
  allow_config_override = true
}
`, keyName)
}
