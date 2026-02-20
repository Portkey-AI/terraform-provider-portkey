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
			// Update usage_limits
			{
				Config: testAccWorkspaceResourceConfigWithUsageLimits(rName+"-upd", 1000, 800),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName+"-upd"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.credit_limit", "1000"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "usage_limits.0.alert_threshold", "800"),
				),
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
			{
				Config: testAccWorkspaceResourceConfigWithRateLimits(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_workspace.test", "id"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.#", "1"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.0.type", "requests"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.0.unit", "rpm"),
					resource.TestCheckResourceAttr("portkey_workspace.test", "rate_limits.0.value", "100"),
				),
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

func testAccWorkspaceResourceConfigWithRateLimits(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_workspace" "test" {
  name        = %[1]q
  description = "Workspace with rate limits"

  rate_limits = [{
    type  = "requests"
    unit  = "rpm"
    value = 100
  }]
}
`, name)
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
				Config: testAccWorkspaceResourceConfigWithRateLimits(rName),
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
