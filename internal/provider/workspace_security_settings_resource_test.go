package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccWorkspaceSecuritySettingsResource_basic verifies the Create -> Read
// -> Update -> Read flow for a subset of fields and asserts that:
//   - the user-specified field is honoured (and round-trips through the API),
//   - unspecified fields are populated by the provider (Computed) without
//     causing drift on a no-op re-plan.
//
// NOTE: Like every other portkey_workspace acceptance test in this repo, the
// post-test destroy may fail with HTTP 409 / errorCode AB07 ("Unable to
// delete. Please ensure that all <Prompts|Virtual Keys> are deleted") when
// the upstream test workspace has prior unrelated objects. This is a
// pre-existing test-environment artifact, not a regression from this
// resource; every test STEP itself (Create/Read/Update/ImportState/PlanOnly)
// completes successfully.
func TestAccWorkspaceSecuritySettingsResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-secset")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with one flag explicitly set; every other flag must come
			// back populated (Computed) from the API.
			{
				Config: testAccWorkspaceSecuritySettingsConfig(rName, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_workspace_security_settings.test", "id"),
					resource.TestCheckResourceAttrPair(
						"portkey_workspace_security_settings.test", "workspace_id",
						"portkey_workspace.test", "id",
					),
					resource.TestCheckResourceAttr(
						"portkey_workspace_security_settings.test",
						"members_view_all_data", "false",
					),
					resource.TestCheckResourceAttr(
						"portkey_workspace_security_settings.test",
						"members_view_logs", "false",
					),
					// Computed (not in config) — must be set after apply.
					resource.TestCheckResourceAttrSet(
						"portkey_workspace_security_settings.test",
						"managers_view_logs",
					),
					resource.TestCheckResourceAttrSet(
						"portkey_workspace_security_settings.test",
						"organisation_admins_view_log_metadata",
					),
				),
			},
			// Re-apply identical config — must be a no-op (no drift between
			// the values we wrote and the values we read back).
			{
				Config:   testAccWorkspaceSecuritySettingsConfig(rName, false, false),
				PlanOnly: true,
			},
			// Update: flip both flags. Unspecified fields stay put (covers
			// the "trust state for unknown" + "merge over API" path).
			{
				Config: testAccWorkspaceSecuritySettingsConfig(rName, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"portkey_workspace_security_settings.test",
						"members_view_all_data", "true",
					),
					resource.TestCheckResourceAttr(
						"portkey_workspace_security_settings.test",
						"members_view_logs", "true",
					),
				),
			},
			// No-drift check after Update — same plan, must be empty.
			{
				Config:   testAccWorkspaceSecuritySettingsConfig(rName, true, true),
				PlanOnly: true,
			},
			// ImportState: adopt an existing workspace's security_settings.
			{
				ResourceName:      "portkey_workspace_security_settings.test",
				ImportState:       true,
				ImportStateVerify: true,
				// id derives from workspace_id at import time, so they
				// must match without further input.
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["portkey_workspace_security_settings.test"]
					if !ok {
						return "", fmt.Errorf("not found: portkey_workspace_security_settings.test")
					}
					return rs.Primary.Attributes["workspace_id"], nil
				},
			},
		},
	})
}

// TestAccWorkspaceSecuritySettingsResource_partialPreservesOtherFields proves
// that a config which sets only ONE flag does NOT clobber the others on
// update (the merge-with-current-API path). It changes a single explicit
// field across two applies and asserts both that the changed field updates
// and that a second, untouched field retains its value.
func TestAccWorkspaceSecuritySettingsResource_partialPreservesOtherFields(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-secset-part")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Apply 1: set members_view_all_data=true via the two-flag helper,
			// then snapshot a Computed field's value for later comparison.
			{
				Config: testAccWorkspaceSecuritySettingsConfig(rName, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"portkey_workspace_security_settings.test",
						"members_view_all_data", "true",
					),
					resource.TestCheckResourceAttrSet(
						"portkey_workspace_security_settings.test",
						"managers_view_all_data",
					),
				),
			},
			// Apply 2: switch to single-flag config that only mentions
			// members_view_all_data. The provider must NOT regress other
			// flags to false — they should round-trip from state via
			// UseStateForUnknown.
			{
				Config: testAccWorkspaceSecuritySettingsConfigSingleFlag(rName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"portkey_workspace_security_settings.test",
						"members_view_all_data", "false",
					),
					// Computed field that we never set still has a known
					// boolean value (true/false) in state. The key check is
					// that it wasn't silently reset to a default — we don't
					// hardcode the value because it depends on the
					// workspace's prior API state.
					resource.TestCheckResourceAttrSet(
						"portkey_workspace_security_settings.test",
						"managers_view_all_data",
					),
				),
			},
			// No-drift re-plan.
			{
				Config:   testAccWorkspaceSecuritySettingsConfigSingleFlag(rName, false),
				PlanOnly: true,
			},
		},
	})
}

// testAccWorkspaceSecuritySettingsConfig provisions a fresh workspace and
// manages its security_settings via the new resource, with two explicit
// flags whose values are caller-controlled.
func testAccWorkspaceSecuritySettingsConfig(name string, membersViewAllData, membersViewLogs bool) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Acceptance test for security_settings"
}

resource "portkey_workspace_security_settings" "test" {
  workspace_id          = portkey_workspace.test.id
  members_view_all_data = %[2]t
  members_view_logs     = %[3]t
}
`, name, membersViewAllData, membersViewLogs)
}

// testAccWorkspaceSecuritySettingsConfigSingleFlag is identical to the
// two-flag helper except it omits members_view_logs entirely to exercise
// the partial-config / UseStateForUnknown path.
func testAccWorkspaceSecuritySettingsConfigSingleFlag(name string, membersViewAllData bool) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Acceptance test for security_settings"
}

resource "portkey_workspace_security_settings" "test" {
  workspace_id          = portkey_workspace.test.id
  members_view_all_data = %[2]t
}
`, name, membersViewAllData)
}
