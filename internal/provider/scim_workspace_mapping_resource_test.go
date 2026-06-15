package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccScimWorkspaceMappingResource_basic creates a fresh workspace and
// binds a SCIM group to it by name, then verifies attribute values and
// round-trips through import.
//
// scim_group_name is unique-prefixed per run so the test can run repeatedly
// without colliding with prior runs or auto-provisioning patterns.
func TestAccScimWorkspaceMappingResource_basic(t *testing.T) {
	wsName := acctest.RandomWithPrefix("tf-acc-scim-ws")
	groupName := acctest.RandomWithPrefix("tf-acc-scim-grp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create + Read
			{
				Config: testAccScimWorkspaceMappingByName(wsName, groupName, "member"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_scim_workspace_mapping.test", "id"),
					resource.TestCheckResourceAttrPair(
						"portkey_scim_workspace_mapping.test", "workspace_id",
						"portkey_workspace.test", "id",
					),
					resource.TestCheckResourceAttr("portkey_scim_workspace_mapping.test", "scim_group_name", groupName),
					resource.TestCheckResourceAttrSet("portkey_scim_workspace_mapping.test", "scim_group_id"),
					resource.TestCheckResourceAttr("portkey_scim_workspace_mapping.test", "role", "member"),
				),
			},
			// Import
			{
				ResourceName:            "portkey_scim_workspace_mapping.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"scim_group_name"},
				ImportStateIdFunc:       scimWorkspaceMappingImportStateIdFunc("portkey_scim_workspace_mapping.test"),
			},
		},
	})
}

// TestAccScimWorkspaceMappingResource_roleReplace verifies that changing the
// role attribute forces resource replacement (new mapping ID) — required
// because the Portkey API has no update endpoint for SCIM mappings.
func TestAccScimWorkspaceMappingResource_roleReplace(t *testing.T) {
	wsName := acctest.RandomWithPrefix("tf-acc-scim-ws")
	groupName := acctest.RandomWithPrefix("tf-acc-scim-grp")

	var idBefore, idAfter string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccScimWorkspaceMappingByName(wsName, groupName, "member"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_scim_workspace_mapping.test", "role", "member"),
					captureResourceAttr("portkey_scim_workspace_mapping.test", "id", &idBefore),
				),
			},
			{
				Config: testAccScimWorkspaceMappingByName(wsName, groupName, "admin"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_scim_workspace_mapping.test", "role", "admin"),
					captureResourceAttr("portkey_scim_workspace_mapping.test", "id", &idAfter),
					func(_ *terraform.State) error {
						if idBefore == "" || idAfter == "" {
							return fmt.Errorf("captured IDs missing (before=%q after=%q)", idBefore, idAfter)
						}
						if idBefore == idAfter {
							return fmt.Errorf("expected role change to replace the mapping (new ID), got same ID %q", idBefore)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestAccScimWorkspaceMappingResource_byID maps an already-existing SCIM
// group to a workspace using scim_group_id. Requires the caller to provide
// a real SCIM group ID via TEST_SCIM_GROUP_ID; skipped otherwise.
func TestAccScimWorkspaceMappingResource_byID(t *testing.T) {
	scimGroupID := os.Getenv("TEST_SCIM_GROUP_ID")
	if scimGroupID == "" {
		t.Skip("TEST_SCIM_GROUP_ID must be set to test mapping by scim_group_id")
	}

	wsName := acctest.RandomWithPrefix("tf-acc-scim-ws")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccScimWorkspaceMappingByID(wsName, scimGroupID, "admin"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_scim_workspace_mapping.test", "scim_group_id", scimGroupID),
					resource.TestCheckResourceAttr("portkey_scim_workspace_mapping.test", "role", "admin"),
				),
			},
		},
	})
}

func testAccScimWorkspaceMappingByName(workspaceName, groupName, role string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for SCIM workspace mapping"
}

resource "portkey_scim_workspace_mapping" "test" {
  workspace_id    = portkey_workspace.test.id
  scim_group_name = %[2]q
  role            = %[3]q
}
`, workspaceName, groupName, role)
}

func testAccScimWorkspaceMappingByID(workspaceName, scimGroupID, role string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Test workspace for SCIM workspace mapping (by ID)"
}

resource "portkey_scim_workspace_mapping" "test" {
  workspace_id  = portkey_workspace.test.id
  scim_group_id = %[2]q
  role          = %[3]q
}
`, workspaceName, scimGroupID, role)
}

func scimWorkspaceMappingImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		workspaceID := rs.Primary.Attributes["workspace_id"]
		id := rs.Primary.Attributes["id"]
		return fmt.Sprintf("%s/%s", workspaceID, id), nil
	}
}

// captureResourceAttr stores the named attribute into the provided string
// pointer so a later step can compare across applies.
func captureResourceAttr(resourceName, attr string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		*dst = rs.Primary.Attributes[attr]
		return nil
	}
}
