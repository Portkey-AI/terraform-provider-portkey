package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWorkspaceResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccWorkspaceResourceConfig(rName, "Initial description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "created_at"),
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_workspace.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at"},
			},
			// Update testing
			{
				Config: testAccWorkspaceResourceConfig(rName+"-updated", "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "description", "Updated description"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccWorkspaceResource_minimal tests workspace creation without optional fields.
func TestAccWorkspaceResource_minimal(t *testing.T) {

	rName := acctest.RandomWithPrefix("tf-acc-minimal")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkspaceResourceConfigMinimal(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "created_at"),
				),
			},
		},
	})
}

func TestAccWorkspaceResource_updateName(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-rename")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkspaceResourceConfig(rName, "Initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
				),
			},
			{
				Config: testAccWorkspaceResourceConfig(rName+"-renamed", "Initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName+"-renamed"),
				),
			},
		},
	})
}

func TestAccWorkspaceResource_updateDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-desc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkspaceResourceConfig(rName, "Original description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "description", "Original description"),
				),
			},
			{
				Config: testAccWorkspaceResourceConfig(rName, "Modified description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "description", "Modified description"),
				),
			},
			{
				Config: testAccWorkspaceResourceConfig(rName, "Final description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "description", "Final description"),
				),
			},
		},
	})
}

func testAccWorkspaceResourceConfig(name, description string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = %[2]q
}
`, name, description)
}

func testAccWorkspaceResourceConfigMinimal(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name = %[1]q
}
`, name)
}

func TestAccWorkspaceResource_withMetadata(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-meta")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with metadata
			{
				Config: testAccWorkspaceResourceConfigWithMetadata(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "metadata.%", "2"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "metadata._user", "test-workspace"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "metadata.team", "engineering"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_workspace.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at"},
			},
			// Update metadata
			{
				Config: testAccWorkspaceResourceConfigWithMetadataUpdated(rName + "-upd"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName+"-upd"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "metadata.%", "3"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "metadata._user", "updated-workspace"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "metadata.team", "platform"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "metadata.environment", "production"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccWorkspaceResourceConfigWithMetadata(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Workspace with metadata"
  
  metadata = {
    "_user" = "test-workspace"
    "team"  = "engineering"
  }
}
`, name)
}

func testAccWorkspaceResourceConfigWithMetadataUpdated(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Workspace with updated metadata"

  metadata = {
    "_user"       = "updated-workspace"
    "team"        = "platform"
    "environment" = "production"
  }
}
`, name)
}

func TestAccWorkspaceResource_withUsageLimits(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-wslim")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with usage_limits
			{
				Config: testAccWorkspaceResourceConfigWithUsageLimits(rName, 500, 400),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.#", "1"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.type", "cost"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.credit_limit", "500"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.alert_threshold", "400"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.periodic_reset", "monthly"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_workspace.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at"},
			},
			// Update usage_limits — changes credit_limit and alert_threshold.
			// This is the exact path that used to fail with "Provider produced
			// inconsistent result after apply" when the Portkey API briefly
			// returned stale values in the PUT response. The Update handler
			// now trusts the plan; the post-apply state must match the new
			// values without depending on API echo timing.
			{
				Config: testAccWorkspaceResourceConfigWithUsageLimits(rName+"-upd", 1000, 800),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName+"-upd"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.credit_limit", "1000"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.alert_threshold", "800"),
				),
			},
			// Re-apply the same config: no drift is allowed. Catches the case
			// where Read after Update flips the trusted plan values back to a
			// stale API response on the next refresh.
			{
				Config:   testAccWorkspaceResourceConfigWithUsageLimits(rName+"-upd", 1000, 800),
				PlanOnly: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccWorkspaceResource_withRateLimits(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-wsrl")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with rate_limits
			{
				Config: testAccWorkspaceResourceConfigWithRateLimits(rName, 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.#", "1"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.0.type", "requests"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.0.unit", "rpm"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.0.value", "100"),
				),
			},
			// Update rate_limits — changes value. Same stale-echo risk as the
			// usage_limits update path: the Portkey API can briefly return the
			// prior value in the PUT response, which would have triggered
			// "Provider produced inconsistent result after apply" before the
			// Update handler started trusting the plan for rate_limits.
			{
				Config: testAccWorkspaceResourceConfigWithRateLimits(rName, 250),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.0.value", "250"),
				),
			},
			// Re-apply the same config: no drift allowed. Catches the case
			// where Read after Update flips trusted plan values back to a
			// stale API response on the next refresh.
			{
				Config:   testAccWorkspaceResourceConfigWithRateLimits(rName, 250),
				PlanOnly: true,
			},
		},
	})
}

func testAccWorkspaceResourceConfigWithUsageLimits(name string, creditLimit, alertThreshold int) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Workspace with usage limits"

  usage_limits = [{
    type            = "cost"
    credit_limit    = %[2]d
    alert_threshold = %[3]d
    periodic_reset  = "monthly"
  }]
}
`, name, creditLimit, alertThreshold)
}

func testAccWorkspaceResourceConfigWithRateLimits(name string, value int) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Workspace with rate limits"

  rate_limits = [{
    type  = "requests"
    unit  = "rpm"
    value = %[2]d
  }]
}
`, name, value)
}

func testAccWorkspaceResourceConfigNoLimits(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Workspace without limits"
}
`, name)
}

// TestAccWorkspaceResource_clearUsageLimits verifies that removing usage_limits
// from the Terraform config sends null to the API and clears the limits.
func TestAccWorkspaceResource_clearUsageLimits(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-wsclear")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with usage_limits
			{
				Config: testAccWorkspaceResourceConfigWithUsageLimits(rName, 500, 400),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.#", "1"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.credit_limit", "500"),
				),
			},
			// Step 2: Remove usage_limits — should clear them
			{
				Config: testAccWorkspaceResourceConfigNoLimits(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.#", "0"),
				),
			},
		},
	})
}

// TestAccWorkspaceResource_clearRateLimits verifies that removing rate_limits
// from the Terraform config sends null to the API and clears the limits.
func TestAccWorkspaceResource_clearRateLimits(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-wsrlcl")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with rate_limits
			{
				Config: testAccWorkspaceResourceConfigWithRateLimits(rName, 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.#", "1"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.0.type", "requests"),
				),
			},
			// Step 2: Remove rate_limits — should clear them
			{
				Config: testAccWorkspaceResourceConfigNoLimits(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.#", "0"),
				),
			},
		},
	})
}

// TestAccWorkspaceResource_usageLimitsExplicitNullPeriodicReset is a
// regression test for a "Provider produced inconsistent result after apply"
// bug where the workspace usage_limits helper unconditionally wrapped the
// API's `periodic_reset` value in types.StringValue, including when the API
// returned "" for an unset field. With `periodic_reset = null` in HCL the
// plan said null but the post-apply state held cty.StringVal(""), tripping
// Terraform's consistency check.
//
// The test exercises Create AND Update (the path most users hit) AND a
// PlanOnly step to confirm no spurious diffs after rotation through Read.
func TestAccWorkspaceResource_usageLimitsExplicitNullPeriodicReset(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-wsulnp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with periodic_reset explicitly null.
			{
				Config: testAccWorkspaceResourceConfigUsageLimitsNullPeriodicReset(rName, 500, 400),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.#", "1"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.type", "cost"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.credit_limit", "500"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.alert_threshold", "400"),
					// The bug: this attribute used to come back as "" instead of being absent.
					resource.TestCheckNoResourceAttr("portkey_workspace.test", "usage_limits.0.periodic_reset"),
				),
			},
			// Step 2: Re-apply the same config — must produce an empty plan.
			// Confirms Read does not flip null → "" and re-introduce drift.
			{
				Config:   testAccWorkspaceResourceConfigUsageLimitsNullPeriodicReset(rName, 500, 400),
				PlanOnly: true,
			},
			// Step 3: Update credit_limit/alert_threshold while keeping
			// periodic_reset = null. Exercises the Update → Read path that
			// triggered the original "inconsistent result" report.
			{
				Config: testAccWorkspaceResourceConfigUsageLimitsNullPeriodicReset(rName, 1000, 800),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.credit_limit", "1000"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.alert_threshold", "800"),
					resource.TestCheckNoResourceAttr("portkey_workspace.test", "usage_limits.0.periodic_reset"),
				),
			},
		},
	})
}

// testAccWorkspaceResourceConfigUsageLimitsNullPeriodicReset is the user-
// reported config that triggered the bug: usage_limits set, but
// periodic_reset explicitly nulled to opt out of monthly/weekly cadence.
func testAccWorkspaceResourceConfigUsageLimitsNullPeriodicReset(name string, creditLimit, alertThreshold int) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Workspace with explicit-null periodic_reset"

  usage_limits = [{
    type            = "cost"
    credit_limit    = %[2]d
    alert_threshold = %[3]d
    periodic_reset  = null
  }]
}
`, name, creditLimit, alertThreshold)
}

// --- Icon tests ---

// TestAccWorkspaceResource_withIcon tests creating a workspace with an icon,
// verifying that the icon is stored separately and the name in state does not
// include the icon prefix.
func TestAccWorkspaceResource_withIcon(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-icon")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with icon
			{
				Config: testAccWorkspaceResourceConfigWithIcon(rName, "🧪"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "icon", "🧪"),
				),
			},
			// ImportState testing — icon and name differ after import because
			// import uses backwards-compatible behavior (icon=null, full name).
			// The subsequent plan step (re-applying the config) corrects this.
			{
				ResourceName:            "portkey_workspace.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at", "icon", "name"},
			},
			// Re-apply config after import — should converge to clean state
			{
				Config: testAccWorkspaceResourceConfigWithIcon(rName, "🧪"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "icon", "🧪"),
				),
			},
		},
	})
}

// TestAccWorkspaceResource_withIcon_update tests changing the icon on a workspace.
func TestAccWorkspaceResource_withIcon_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-iconup")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with icon
			{
				Config: testAccWorkspaceResourceConfigWithIcon(rName, "🧪"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "icon", "🧪"),
				),
			},
			// Update to different icon
			{
				Config: testAccWorkspaceResourceConfigWithIcon(rName, "🔬"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "icon", "🔬"),
				),
			},
		},
	})
}

// TestAccWorkspaceResource_withIcon_clear tests removing the icon from a workspace.
func TestAccWorkspaceResource_withIcon_clear(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-iconclr")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with icon
			{
				Config: testAccWorkspaceResourceConfigWithIcon(rName, "🧪"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "icon", "🧪"),
				),
			},
			// Clear icon by setting to empty string
			{
				Config: testAccWorkspaceResourceConfigWithIcon(rName, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "icon", ""),
				),
			},
		},
	})
}

// TestAccWorkspaceResource_withoutIcon_noChange verifies that workspaces created
// WITHOUT an icon field behave identically to the pre-icon behavior — the name
// includes any emoji prefix the API returns, and no stripping occurs.
func TestAccWorkspaceResource_withoutIcon_noChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-noicon")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without icon — backwards compatible behavior
			{
				Config: testAccWorkspaceResourceConfigMinimal(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
				),
			},
		},
	})
}

// TestAccWorkspaceResource_emojiInName_noIcon verifies that creating a workspace
// with an emoji in the name but no icon field does NOT cause flip-flop drift
// on subsequent refreshes. The API auto-extracts the emoji into an icon field,
// but the provider must NOT store that in state.
func TestAccWorkspaceResource_emojiInName_noIcon(t *testing.T) {
	rName := "🎯 " + acctest.RandomWithPrefix("tf-acc-emojiname")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with emoji in name, no icon field
			{
				Config: testAccWorkspaceResourceConfigMinimal(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
				),
			},
			// Second plan/apply — must show NO changes (no flip-flop)
			{
				Config: testAccWorkspaceResourceConfigMinimal(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
				),
			},
		},
	})
}

// TestAccWorkspaceResource_optInToIcon tests the transition from emoji-in-name
// (no icon field) to explicit icon management. The user changes their config
// from name="🎯 Target" to name="Target", icon="🎯".
func TestAccWorkspaceResource_optInToIcon(t *testing.T) {
	rSuffix := acctest.RandomWithPrefix("tf-acc-optin")
	emojiName := "🎯 " + rSuffix
	cleanName := rSuffix

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with emoji in name, no icon
			{
				Config: testAccWorkspaceResourceConfig(emojiName, "Target workspace"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", emojiName),
				),
			},
			// Step 2: Opt in — change to clean name + icon
			{
				Config: testAccWorkspaceResourceConfigWithIcon(cleanName, "🎯"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", cleanName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "icon", "🎯"),
				),
			},
			// Step 3: Verify stable — no drift on re-plan
			{
				Config: testAccWorkspaceResourceConfigWithIcon(cleanName, "🎯"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", cleanName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "icon", "🎯"),
				),
			},
		},
	})
}

func testAccWorkspaceResourceConfigWithIcon(name, icon string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  icon        = %[2]q
  description = "Workspace with icon"
}
`, name, icon)
}
