package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUsageLimitsPolicyResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccUsageLimitsPolicyResourceConfig(rName, workspaceID, 1000.0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_usage_limits_policy.test", "id"),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "workspace_id", workspaceID),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "type", "cost"),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "credit_limit", "1000"),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "periodic_reset", "monthly"),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "status", "active"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_usage_limits_policy.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at"},
			},
			// Update credit_limit testing
			{
				Config: testAccUsageLimitsPolicyResourceConfig(rName, workspaceID, 2000.0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "credit_limit", "2000"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccUsageLimitsPolicyResource_updateName(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-rename")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsageLimitsPolicyResourceConfig(rName, workspaceID, 1000.0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "name", rName),
				),
			},
			{
				Config: testAccUsageLimitsPolicyResourceConfig(rName+"-updated", workspaceID, 1000.0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "name", rName+"-updated"),
				),
			},
		},
	})
}

func TestAccUsageLimitsPolicyResource_periodicResetDays(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-reset-days")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsageLimitsPolicyResourceConfigWithResetDays(rName, workspaceID, 1000.0, 30),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_usage_limits_policy.test", "id"),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "periodic_reset_days", "30"),
					resource.TestCheckNoResourceAttr("portkey_usage_limits_policy.test", "periodic_reset"),
					resource.TestCheckResourceAttrSet("portkey_usage_limits_policy.test", "next_usage_reset_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_usage_limits_policy.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at"},
			},
			// In-place update: credit_limit changes, reset days do not
			{
				Config: testAccUsageLimitsPolicyResourceConfigWithResetDays(rName, workspaceID, 2000.0, 30),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "credit_limit", "2000"),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "periodic_reset_days", "30"),
				),
			},
			// Changing the day-count must force replacement
			{
				Config: testAccUsageLimitsPolicyResourceConfigWithResetDays(rName, workspaceID, 2000.0, 60),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "periodic_reset_days", "60"),
				),
			},
		},
	})
}

func TestAccUsageLimitsPolicyResource_conflictingResetOptions(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-reset-conflict")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccUsageLimitsPolicyResourceConfigConflicting(rName, workspaceID),
				ExpectError: regexp.MustCompile(`mutually exclusive`),
			},
		},
	})
}

func TestAccUsageLimitsPolicyResource_excludes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-excludes")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with excludes in conditions
			{
				Config: testAccUsageLimitsPolicyResourceConfigWithExcludes(rName, workspaceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_usage_limits_policy.test_excludes", "id"),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test_excludes", "name", rName),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test_excludes", "type", "cost"),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test_excludes", "status", "active"),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test_excludes", "conditions", "[{\"excludes\":[\"gpt-4-mini\",\"gpt-4o-mini\"],\"key\":\"model\",\"value\":[\"gpt-4\",\"gpt-4o\"]}]"),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test_excludes", "group_by", "[{\"key\":\"api_key\"}]"),
				),
			},
			// ImportState testing — conditions (including excludes) should survive round-trip
			{
				ResourceName:            "portkey_usage_limits_policy.test_excludes",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccUsageLimitsPolicyResourceConfigWithResetDays(name, workspaceID string, creditLimit float64, resetDays int) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_usage_limits_policy" "test" {
  name         = %[1]q
  workspace_id = %[2]q
  conditions   = jsonencode([
    {
      key   = "workspace_id"
      value = %[2]q
    }
  ])
  group_by = jsonencode([
    {
      key = "api_key"
    }
  ])
  type                = "cost"
  credit_limit        = %[3]f
  periodic_reset_days = %[4]d
}
`, name, workspaceID, creditLimit, resetDays)
}

func testAccUsageLimitsPolicyResourceConfigConflicting(name, workspaceID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_usage_limits_policy" "test" {
  name         = %[1]q
  workspace_id = %[2]q
  conditions   = jsonencode([
    {
      key   = "workspace_id"
      value = %[2]q
    }
  ])
  group_by = jsonencode([
    {
      key = "api_key"
    }
  ])
  type                = "cost"
  credit_limit        = 1000.0
  periodic_reset      = "monthly"
  periodic_reset_days = 30
}
`, name, workspaceID)
}

func testAccUsageLimitsPolicyResourceConfig(name, workspaceID string, creditLimit float64) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_usage_limits_policy" "test" {
  name         = %[1]q
  workspace_id = %[2]q
  conditions   = jsonencode([
    {
      key   = "workspace_id"
      value = %[2]q
    }
  ])
  group_by = jsonencode([
    {
      key = "api_key"
    }
  ])
  type           = "cost"
  credit_limit   = %[3]f
  alert_threshold = %[3]f * 0.8
  periodic_reset = "monthly"
}
`, name, workspaceID, creditLimit)
}

func testAccUsageLimitsPolicyResourceConfigWithExcludes(name, workspaceID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_usage_limits_policy" "test_excludes" {
  name         = %[1]q
  workspace_id = %[2]q
  conditions   = jsonencode([
    {
      key      = "model"
      value    = ["gpt-4", "gpt-4o"]
      excludes = ["gpt-4-mini", "gpt-4o-mini"]
    }
  ])
  group_by = jsonencode([
    { key = "api_key" }
  ])
  type           = "cost"
  credit_limit   = 100.0
  alert_threshold = 80.0
  periodic_reset = "monthly"
}
`, name, workspaceID)
}

// TestAccUsageLimitsPolicyResource_updateConditionsInPlace is the regression test for
// the budget-counter reset bug. `conditions` used to carry a RequiresReplace plan
// modifier, so adding a user to a policy destroyed it and created a new one. The
// replacement got a fresh UUID and, because spend counters are keyed by policy ID
// server-side, an accumulated usage of zero — silently handing every affected user
// their full budget again.
//
// The plan check is the assertion that matters: it fails if Terraform ever goes back to
// planning a replacement. The ID check confirms the same policy survived the apply.
func TestAccUsageLimitsPolicyResource_updateConditionsInPlace(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-cond")
	workspaceID := getTestWorkspaceID()

	var policyID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsageLimitsPolicyResourceConfigWithUsers(rName, workspaceID, []string{"aqua-agent-bot"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCapturePolicyID("portkey_usage_limits_policy.test", &policyID),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "conditions",
						`[{"key":"metadata._user","value":["aqua-agent-bot"]}]`),
				),
			},
			// Adding a user must update in place, not replace.
			{
				Config: testAccUsageLimitsPolicyResourceConfigWithUsers(rName, workspaceID, []string{"aqua-agent-bot", "blue-agent-bot"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("portkey_usage_limits_policy.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPolicyIDUnchanged("portkey_usage_limits_policy.test", &policyID),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "conditions",
						`[{"key":"metadata._user","value":["aqua-agent-bot","blue-agent-bot"]}]`),
				),
			},
			// Removing a user must also update in place.
			{
				Config: testAccUsageLimitsPolicyResourceConfigWithUsers(rName, workspaceID, []string{"blue-agent-bot"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("portkey_usage_limits_policy.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPolicyIDUnchanged("portkey_usage_limits_policy.test", &policyID),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "conditions",
						`[{"key":"metadata._user","value":["blue-agent-bot"]}]`),
				),
			},
		},
	})
}

// TestAccUsageLimitsPolicyResource_clearAlertThreshold covers the companion bug: Update
// skipped alert_threshold entirely when the plan value was null, so removing the
// attribute from HCL left the old threshold live on the server.
func TestAccUsageLimitsPolicyResource_clearAlertThreshold(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-thresh")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsageLimitsPolicyResourceConfigThreshold(rName, workspaceID, "alert_threshold = 800.0"),
				Check: resource.TestCheckResourceAttr(
					"portkey_usage_limits_policy.test", "alert_threshold", "800"),
			},
			{
				Config: testAccUsageLimitsPolicyResourceConfigThreshold(rName, workspaceID, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("portkey_usage_limits_policy.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckNoResourceAttr(
					"portkey_usage_limits_policy.test", "alert_threshold"),
			},
		},
	})
}

// TestAccUsageLimitsPolicyResource_switchResetCadence exercises the two other
// attributes that used to force replacement. Switching between periodic_reset and
// periodic_reset_days is supported by the API in place, and the provider must send the
// cleared side as an explicit null so the two never end up set at once.
func TestAccUsageLimitsPolicyResource_switchResetCadence(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-cadence")
	workspaceID := getTestWorkspaceID()

	var policyID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsageLimitsPolicyResourceConfigCadence(rName, workspaceID, `periodic_reset = "monthly"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCapturePolicyID("portkey_usage_limits_policy.test", &policyID),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "periodic_reset", "monthly"),
				),
			},
			// monthly -> weekly, in place
			{
				Config: testAccUsageLimitsPolicyResourceConfigCadence(rName, workspaceID, `periodic_reset = "weekly"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("portkey_usage_limits_policy.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPolicyIDUnchanged("portkey_usage_limits_policy.test", &policyID),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "periodic_reset", "weekly"),
				),
			},
			// periodic_reset -> periodic_reset_days, in place; the old field must clear
			{
				Config: testAccUsageLimitsPolicyResourceConfigCadence(rName, workspaceID, `periodic_reset_days = 45`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("portkey_usage_limits_policy.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPolicyIDUnchanged("portkey_usage_limits_policy.test", &policyID),
					resource.TestCheckResourceAttr("portkey_usage_limits_policy.test", "periodic_reset_days", "45"),
					resource.TestCheckNoResourceAttr("portkey_usage_limits_policy.test", "periodic_reset"),
				),
			},
		},
	})
}

// testAccCapturePolicyID records the resource's id so a later step can assert the same
// object survived, rather than a replacement wearing the same name.
func testAccCapturePolicyID(resourceName string, target *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource %s has no ID set", resourceName)
		}
		*target = rs.Primary.ID
		return nil
	}
}

// testAccCheckPolicyIDUnchanged fails if the policy was recreated. A new ID means the
// server-side usage counter was left behind with the archived policy.
func testAccCheckPolicyIDUnchanged(resourceName string, want *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		if *want == "" {
			return fmt.Errorf("no baseline ID captured for %s", resourceName)
		}
		if rs.Primary.ID != *want {
			return fmt.Errorf(
				"policy was replaced instead of updated in place: id was %s, now %s (accumulated usage is lost on replacement)",
				*want, rs.Primary.ID,
			)
		}
		return nil
	}
}

func testAccUsageLimitsPolicyResourceConfigWithUsers(name, workspaceID string, users []string) string {
	quoted := make([]string, 0, len(users))
	for _, u := range users {
		quoted = append(quoted, fmt.Sprintf("%q", u))
	}

	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_usage_limits_policy" "test" {
  name         = %[1]q
  workspace_id = %[2]q
  conditions   = jsonencode([
    {
      key   = "metadata._user"
      value = [%[3]s]
    }
  ])
  group_by = jsonencode([
    {
      key = "metadata._user"
    }
  ])
  type           = "cost"
  credit_limit   = 1000.0
  periodic_reset = "monthly"
}
`, name, workspaceID, strings.Join(quoted, ", "))
}

func testAccUsageLimitsPolicyResourceConfigThreshold(name, workspaceID, thresholdLine string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_usage_limits_policy" "test" {
  name         = %[1]q
  workspace_id = %[2]q
  conditions   = jsonencode([
    {
      key   = "workspace_id"
      value = %[2]q
    }
  ])
  group_by = jsonencode([
    {
      key = "api_key"
    }
  ])
  type           = "cost"
  credit_limit   = 1000.0
  periodic_reset = "monthly"
  %[3]s
}
`, name, workspaceID, thresholdLine)
}

func testAccUsageLimitsPolicyResourceConfigCadence(name, workspaceID, cadenceLine string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_usage_limits_policy" "test" {
  name         = %[1]q
  workspace_id = %[2]q
  conditions   = jsonencode([
    {
      key   = "workspace_id"
      value = %[2]q
    }
  ])
  group_by = jsonencode([
    {
      key = "api_key"
    }
  ])
  type         = "cost"
  credit_limit = 1000.0
  %[3]s
}
`, name, workspaceID, cadenceLine)
}
