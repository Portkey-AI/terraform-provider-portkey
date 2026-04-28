package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccUsersDataSource_basic verifies the data source returns the full list
// of users with no filters applied. The data source auto-paginates, so the
// returned count should equal the API-reported total.
func TestAccUsersDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_users.all", "users.#"),
					resource.TestCheckResourceAttrSet("data.portkey_users.all", "total"),
					testAccCheckUsersDataSourceCountMatchesTotal("data.portkey_users.all"),
				),
			},
		},
	})
}

// TestAccUsersDataSource_pagination forces a tiny upstream page size so the
// data source must make multiple round-trips. The auto-paginated result must
// still equal the reported total.
func TestAccUsersDataSource_pagination(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersDataSourceConfigPageSize(1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.portkey_users.paged", "page_size", "1"),
					resource.TestCheckResourceAttrSet("data.portkey_users.paged", "users.#"),
					resource.TestCheckResourceAttrSet("data.portkey_users.paged", "total"),
					testAccCheckUsersDataSourceCountMatchesTotal("data.portkey_users.paged"),
				),
			},
		},
	})
}

// TestAccUsersDataSource_roleFilter verifies the role server-side filter is
// forwarded to the API and every returned user has the requested role.
func TestAccUsersDataSource_roleFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersDataSourceConfigRole("owner"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.portkey_users.by_role", "role", "owner"),
					resource.TestCheckResourceAttrSet("data.portkey_users.by_role", "total"),
					testAccCheckUsersDataSourceAllHaveRole("data.portkey_users.by_role", "owner"),
				),
			},
		},
	})
}

// TestAccUsersDataSource_invalidRole verifies the schema validator rejects
// values outside the {admin, member, owner} enum at plan time.
func TestAccUsersDataSource_invalidRole(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccUsersDataSourceConfigRole("superuser"),
				ExpectError: regexp.MustCompile(`(?i)attribute role value must be one of`),
			},
		},
	})
}

// TestAccUsersDataSource_pageSizeOutOfRange verifies the validator catches
// page_size values outside the allowed 1-1000 window.
func TestAccUsersDataSource_pageSizeOutOfRange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccUsersDataSourceConfigPageSize(0),
				ExpectError: regexp.MustCompile(`(?i)attribute page_size value must be between`),
			},
		},
	})
}

// ----------------------------------------------------------------------------
// HCL builders
// ----------------------------------------------------------------------------

func testAccUsersDataSourceConfig() string {
	return `
provider "portkey" {}

data "portkey_users" "all" {}
`
}

func testAccUsersDataSourceConfigPageSize(size int) string {
	return fmt.Sprintf(`
provider "portkey" {}

data "portkey_users" "paged" {
  page_size = %d
}
`, size)
}

func testAccUsersDataSourceConfigRole(role string) string {
	return fmt.Sprintf(`
provider "portkey" {}

data "portkey_users" "by_role" {
  role = %q
}
`, role)
}

// ----------------------------------------------------------------------------
// Custom check functions
// ----------------------------------------------------------------------------

// testAccCheckUsersDataSourceCountMatchesTotal asserts that the auto-paginated
// users list length matches the API-reported total. This catches both
// "we paginated wrong" and "we set total from the wrong response" bugs.
func testAccCheckUsersDataSourceCountMatchesTotal(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("data source %q not found in state", name)
		}
		count := rs.Primary.Attributes["users.#"]
		total := rs.Primary.Attributes["total"]
		if count == "" || total == "" {
			return fmt.Errorf("data source %q missing users.# or total in state", name)
		}
		if count != total {
			return fmt.Errorf("data source %q: returned users.# = %s but total = %s — pagination did not return all users", name, count, total)
		}
		return nil
	}
}

// testAccCheckUsersDataSourceAllHaveRole asserts every returned user has the
// expected role (proving the server-side filter is forwarded correctly).
func testAccCheckUsersDataSourceAllHaveRole(name, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("data source %q not found in state", name)
		}
		countStr := rs.Primary.Attributes["users.#"]
		count, err := strconv.Atoi(countStr)
		if err != nil {
			return fmt.Errorf("data source %q: could not parse users.# = %q: %w", name, countStr, err)
		}
		for i := 0; i < count; i++ {
			gotRole := rs.Primary.Attributes["users."+strconv.Itoa(i)+".role"]
			if gotRole != expected {
				return fmt.Errorf("data source %q: users[%d].role = %q, expected %q", name, i, gotRole, expected)
			}
		}
		return nil
	}
}
