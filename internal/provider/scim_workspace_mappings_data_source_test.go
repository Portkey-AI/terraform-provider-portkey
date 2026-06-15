package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccScimWorkspaceMappingsDataSource_basic lists every mapping the API
// returns and asserts the response shape. The org may legitimately have
// zero mappings — we only assert the attribute is set (i.e. the data source
// resolved at all), mirroring portkey_workspaces' permissive check.
func TestAccScimWorkspaceMappingsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "portkey" {}

data "portkey_scim_workspace_mappings" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_scim_workspace_mappings.all", "mappings.#"),
				),
			},
		},
	})
}

// TestAccScimWorkspaceMappingsDataSource_withCreatedMapping creates a
// workspace + a mapping, then queries the data source filtered by the
// workspace id, and asserts the new mapping appears in the result.
func TestAccScimWorkspaceMappingsDataSource_withCreatedMapping(t *testing.T) {
	wsName := acctest.RandomWithPrefix("tf-acc-scim-ds-ws")
	groupName := acctest.RandomWithPrefix("tf-acc-scim-ds-grp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccScimWorkspaceMappingsDataSourceConfig(wsName, groupName, "member"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_scim_workspace_mappings.filtered", "mappings.#"),
					// The filtered data source should see exactly the mapping we just created.
					resource.TestCheckResourceAttr("data.portkey_scim_workspace_mappings.filtered", "mappings.#", "1"),
					resource.TestCheckResourceAttr("data.portkey_scim_workspace_mappings.filtered", "mappings.0.role", "member"),
					resource.TestCheckResourceAttrPair(
						"data.portkey_scim_workspace_mappings.filtered", "mappings.0.workspace_id",
						"portkey_workspace.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.portkey_scim_workspace_mappings.filtered", "mappings.0.id",
						"portkey_scim_workspace_mapping.test", "id",
					),
				),
			},
		},
	})
}

func testAccScimWorkspaceMappingsDataSourceConfig(workspaceName, groupName, role string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Workspace for SCIM mappings data source test"
}

resource "portkey_scim_workspace_mapping" "test" {
  workspace_id    = portkey_workspace.test.id
  scim_group_name = %[2]q
  role            = %[3]q
}

data "portkey_scim_workspace_mappings" "filtered" {
  workspace_id = portkey_workspace.test.id
  depends_on   = [portkey_scim_workspace_mapping.test]
}
`, workspaceName, groupName, role)
}
