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
				ExpectError: regexp.MustCompile(`allow_config_override can only be set to true`),
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
//
// depends_on is required here: without a direct attribute reference to
// portkey_config.test, Terraform would not know the destroy order and might
// attempt to delete the config before the API key, causing a 409 (config still
// bound to the key).
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
  depends_on            = [portkey_config.test]
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

// ----------------------------------------------------------------------------
// expires_at tests
// ----------------------------------------------------------------------------

// TestAccAPIKeyResource_withExpiresAt verifies:
//   - An API key can be created with expires_at set.
//   - The value is read back correctly from the API.
//   - expires_at can be updated in-place (no destroy/recreate).
//   - An invalid (non-RFC3339) value is rejected at plan time.
func TestAccAPIKeyResource_withExpiresAt(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-exp")
	nameUpdated := name + "-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with expires_at set.
			{
				Config: testAccAPIKeyResourceConfigWithExpiresAt(name, "2030-01-01T00:00:00Z"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", name),
					resource.TestCheckResourceAttr("portkey_api_key.test", "expires_at", "2030-01-01T00:00:00Z"),
				),
			},
			// Step 2: Update expires_at in-place — no replacement should occur.
			{
				Config: testAccAPIKeyResourceConfigWithExpiresAt(nameUpdated, "2031-06-30T23:59:59Z"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("portkey_api_key.test", "expires_at", "2031-06-30T23:59:59Z"),
				),
			},
		},
	})
}

// TestAccAPIKeyResource_expiresAtInvalidFormat verifies that a non-RFC3339
// expires_at value is rejected at plan time with a clear error.
func TestAccAPIKeyResource_expiresAtInvalidFormat(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-expinv")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAPIKeyResourceConfigWithExpiresAt(name, "not-a-date"),
				ExpectError: regexp.MustCompile(`(?i)invalid.*rfc3339|rfc3339.*invalid`),
			},
		},
	})
}

func testAccAPIKeyResourceConfigWithExpiresAt(name, expiresAt string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name       = %[1]q
  type       = "organisation"
  sub_type   = "service"
  scopes     = ["providers.list"]
  expires_at = %[2]q
}
`, name, expiresAt)
}

// ----------------------------------------------------------------------------
// rotation_policy tests
// ----------------------------------------------------------------------------

// TestAccAPIKeyResource_withRotationPolicy verifies:
//   - A key can be created with a rotation_policy (monthly, 1h transition).
//   - next_rotation_at is computed by the API and read back into state.
//   - The policy can be updated in-place (change to weekly).
//   - An empty rotation_policy block is rejected at plan time.
func TestAccAPIKeyResource_withRotationPolicy(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-rot")
	nameUpdated := name + "-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with monthly rotation, 1h transition.
			{
				Config: testAccAPIKeyResourceConfigWithRotationPolicy(name, "monthly", 3600000),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", name),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rotation_policy.rotation_period", "monthly"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rotation_policy.key_transition_period_ms", "3600000"),
					// next_rotation_at is computed by the API.
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "rotation_policy.next_rotation_at"),
				),
			},
			// Step 2: Update to weekly rotation in-place.
			{
				Config: testAccAPIKeyResourceConfigWithRotationPolicy(nameUpdated, "weekly", 1800000),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rotation_policy.rotation_period", "weekly"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "rotation_policy.key_transition_period_ms", "1800000"),
				),
			},
			// Step 3: Import round-trip.
			{
				ResourceName:            "portkey_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},
			},
		},
	})
}

// TestAccAPIKeyResource_rotationPolicyBelowMinTransition verifies that
// key_transition_period_ms < 1800000 is rejected at plan time.
func TestAccAPIKeyResource_rotationPolicyBelowMinTransition(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-rotmin")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAPIKeyResourceConfigWithRotationPolicy(name, "monthly", 60000), // 1 minute — below 30 min minimum
				ExpectError: regexp.MustCompile(`(?i)at least 1800000`),
			},
		},
	})
}

// TestAccAPIKeyResource_rotationPolicyEmptyBlock verifies that an all-null
// rotation_policy block is rejected at plan time with a clear error.
func TestAccAPIKeyResource_rotationPolicyEmptyBlock(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-rotempty")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAPIKeyResourceConfigWithEmptyRotationPolicy(name),
				ExpectError: regexp.MustCompile(`(?i)empty rotation_policy`),
			},
		},
	})
}

func testAccAPIKeyResourceConfigWithRotationPolicy(name, period string, transitionMs int) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  rotation_policy = {
    rotation_period          = %[2]q
    key_transition_period_ms = %[3]d
  }
}
`, name, period, transitionMs)
}

func testAccAPIKeyResourceConfigWithEmptyRotationPolicy(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  rotation_policy = {}
}
`, name)
}

// ----------------------------------------------------------------------------
// reset_usage tests
// ----------------------------------------------------------------------------

// TestAccAPIKeyResource_resetUsage verifies the write-only trigger semantics:
//   - Creating a key with reset_usage = true is accepted.
//   - After apply, state stores null for reset_usage (trigger consumed).
//   - Removing reset_usage from config produces no perpetual diff (state is null,
//     config is null → no change in plan).
func TestAccAPIKeyResource_resetUsage(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-reset")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create a key with usage_limits so there is usage to reset.
			{
				Config: testAccAPIKeyResourceConfigWithUsageLimits(name, 1000, "monthly"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", name),
				),
			},
			// Step 2: Trigger a usage reset. State should store null for reset_usage
			// after apply (trigger consumed), so next plan shows no change.
			{
				Config: testAccAPIKeyResourceConfigWithResetUsage(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("portkey_api_key.test", "reset_usage"),
					// last_reset_at must be populated after a reset.
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "last_reset_at"),
				),
			},
			// Step 3: Remove reset_usage from config — must produce no diff.
			{
				Config:             testAccAPIKeyResourceConfigWithUsageLimits(name, 1000, "monthly"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccAPIKeyResourceConfigWithResetUsage(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  usage_limits = {
    credit_limit   = 1000
    periodic_reset = "monthly"
  }

  reset_usage = true
}
`, name)
}

// ----------------------------------------------------------------------------
// expanded usage_limits tests
// ----------------------------------------------------------------------------

// TestAccAPIKeyResource_usageLimitsExpanded verifies the new usage_limits fields:
//   - type ("tokens" or "cost")
//   - periodic_reset_days as an alternative to periodic_reset
//   - next_usage_reset_at computed by the API
//   - Mutual exclusivity of periodic_reset and periodic_reset_days is enforced.
//   - credit_limit is required when usage_limits is set.
func TestAccAPIKeyResource_usageLimitsWithType(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-ultype")
	nameUpdated := name + "-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with type="tokens" and periodic_reset="monthly".
			{
				Config: testAccAPIKeyResourceConfigWithUsageLimitsType(name, "tokens", 500000, "monthly"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.type", "tokens"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.credit_limit", "500000"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.periodic_reset", "monthly"),
				),
			},
			// Step 2: Update to type="cost" and a different credit_limit.
			{
				Config: testAccAPIKeyResourceConfigWithUsageLimitsType(nameUpdated, "cost", 1000, "weekly"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_api_key.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.type", "cost"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.credit_limit", "1000"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.periodic_reset", "weekly"),
				),
			},
		},
	})
}

// TestAccAPIKeyResource_usageLimitsPeriodicResetDays verifies periodic_reset_days.
func TestAccAPIKeyResource_usageLimitsPeriodicResetDays(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-uldays")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfigWithPeriodicResetDays(name, 30),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_api_key.test", "id"),
					resource.TestCheckResourceAttr("portkey_api_key.test", "usage_limits.periodic_reset_days", "30"),
				),
			},
		},
	})
}

// TestAccAPIKeyResource_usageLimitsMissingCreditLimit verifies that omitting
// credit_limit when usage_limits is set is rejected at plan time.
func TestAccAPIKeyResource_usageLimitsMissingCreditLimit(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-ulnocl")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAPIKeyResourceConfigUsageLimitsMissingCreditLimit(name),
				ExpectError: regexp.MustCompile(`(?i)credit_limit is required`),
			},
		},
	})
}

// TestAccAPIKeyResource_usageLimitsMutualExclusion verifies that setting both
// periodic_reset and periodic_reset_days is rejected at plan time.
func TestAccAPIKeyResource_usageLimitsMutualExclusion(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-ulmutex")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAPIKeyResourceConfigUsageLimitsBothResets(name),
				ExpectError: regexp.MustCompile(`(?i)mutually exclusive`),
			},
		},
	})
}

// TestAccAPIKeyResource_usageLimitsPeriodicResetDaysZero verifies that
// periodic_reset_days = 0 is rejected by the schema validator (minimum is 1).
func TestAccAPIKeyResource_usageLimitsPeriodicResetDaysZero(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-ulr0")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAPIKeyResourceConfigWithPeriodicResetDays(name, 0),
				ExpectError: regexp.MustCompile(`(?i)between 1 and 365|at least 1`),
			},
		},
	})
}

// TestAccAPIKeyResource_usageLimitsPeriodicResetDaysOverMax verifies that
// periodic_reset_days = 366 is rejected by the schema validator (maximum is 365).
func TestAccAPIKeyResource_usageLimitsPeriodicResetDaysOverMax(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ak-ulr366")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAPIKeyResourceConfigWithPeriodicResetDays(name, 366),
				ExpectError: regexp.MustCompile(`(?i)between 1 and 365|at most 365`),
			},
		},
	})
}

func testAccAPIKeyResourceConfigWithUsageLimitsType(name, limitType string, creditLimit int, periodicReset string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  usage_limits = {
    type           = %[2]q
    credit_limit   = %[3]d
    periodic_reset = %[4]q
  }
}
`, name, limitType, creditLimit, periodicReset)
}

func testAccAPIKeyResourceConfigWithPeriodicResetDays(name string, days int) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  usage_limits = {
    credit_limit        = 1000
    periodic_reset_days = %[2]d
  }
}
`, name, days)
}

func testAccAPIKeyResourceConfigUsageLimitsMissingCreditLimit(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  usage_limits = {
    periodic_reset = "monthly"
  }
}
`, name)
}

func testAccAPIKeyResourceConfigUsageLimitsBothResets(name string) string {
	return fmt.Sprintf(`
resource "portkey_api_key" "test" {
  name     = %[1]q
  type     = "organisation"
  sub_type = "service"
  scopes   = ["providers.list"]

  usage_limits = {
    credit_limit        = 1000
    periodic_reset      = "monthly"
    periodic_reset_days = 30
  }
}
`, name)
}
