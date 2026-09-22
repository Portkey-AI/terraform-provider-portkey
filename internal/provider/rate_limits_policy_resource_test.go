package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccRateLimitsPolicyResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRateLimitsPolicyResourceConfig(rName, workspaceID, 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_rate_limits_policy.test", "id"),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test", "workspace_id", workspaceID),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test", "type", "requests"),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test", "unit", "rpm"),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test", "value", "100"),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test", "status", "active"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "portkey_rate_limits_policy.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at"},
			},
			// Update value testing
			{
				Config: testAccRateLimitsPolicyResourceConfig(rName, workspaceID, 200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test", "value", "200"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccRateLimitsPolicyResource_updateName(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-rename")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRateLimitsPolicyResourceConfig(rName, workspaceID, 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test", "name", rName),
				),
			},
			{
				Config: testAccRateLimitsPolicyResourceConfig(rName+"-updated", workspaceID, 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test", "name", rName+"-updated"),
				),
			},
		},
	})
}

func TestAccRateLimitsPolicyResource_excludes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-excludes")
	workspaceID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with excludes in conditions
			{
				Config: testAccRateLimitsPolicyResourceConfigWithExcludes(rName, workspaceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_rate_limits_policy.test_excludes", "id"),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test_excludes", "name", rName),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test_excludes", "type", "requests"),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test_excludes", "unit", "rpm"),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test_excludes", "status", "active"),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test_excludes", "conditions", "[{\"excludes\":[\"gpt-4-mini\",\"gpt-4o-mini\"],\"key\":\"model\",\"value\":[\"gpt-4\",\"gpt-4o\"]}]"),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test_excludes", "group_by", "[{\"key\":\"api_key\"}]"),
				),
			},
			// ImportState testing — conditions (including excludes) should survive round-trip
			{
				ResourceName:            "portkey_rate_limits_policy.test_excludes",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccRateLimitsPolicyResourceConfig(name, workspaceID string, value int) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_rate_limits_policy" "test" {
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
  type  = "requests"
  unit  = "rpm"
  value = %[3]d
}
`, name, workspaceID, value)
}

func testAccRateLimitsPolicyResourceConfigWithExcludes(name, workspaceID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_rate_limits_policy" "test_excludes" {
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
  type  = "requests"
  unit  = "rpm"
  value = 50
}
`, name, workspaceID)
}

// TestAccRateLimitsPolicyResource_updateConditionsInPlace mirrors the usage-limits
// regression test. `conditions` carried the same RequiresReplace marker here, and the
// API accepts it on PUT, so a targeting change must update in place rather than tear
// the policy down and rebuild it.
func TestAccRateLimitsPolicyResource_updateConditionsInPlace(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-rl-cond")
	workspaceID := getTestWorkspaceID()

	var policyID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRateLimitsPolicyResourceConfigWithUsers(rName, workspaceID, []string{"aqua-agent-bot"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCapturePolicyID("portkey_rate_limits_policy.test", &policyID),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test", "conditions",
						`[{"key":"metadata._user","value":["aqua-agent-bot"]}]`),
				),
			},
			{
				Config: testAccRateLimitsPolicyResourceConfigWithUsers(rName, workspaceID, []string{"aqua-agent-bot", "blue-agent-bot"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("portkey_rate_limits_policy.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPolicyIDUnchanged("portkey_rate_limits_policy.test", &policyID),
					resource.TestCheckResourceAttr("portkey_rate_limits_policy.test", "conditions",
						`[{"key":"metadata._user","value":["aqua-agent-bot","blue-agent-bot"]}]`),
				),
			},
		},
	})
}

func testAccRateLimitsPolicyResourceConfigWithUsers(name, workspaceID string, users []string) string {
	quoted := make([]string, 0, len(users))
	for _, u := range users {
		quoted = append(quoted, fmt.Sprintf("%q", u))
	}

	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_rate_limits_policy" "test" {
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
  type  = "requests"
  unit  = "rpm"
  value = 100
}
`, name, workspaceID, strings.Join(quoted, ", "))
}
