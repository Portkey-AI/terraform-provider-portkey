# Testing the Portkey Terraform Provider

## Quick Start

```bash
export PORTKEY_API_KEY="your-admin-api-key"
make testacc
```

## How Acceptance Tests Work

The `terraform-plugin-testing` framework runs **real Terraform operations** against the **live Portkey API**:

```
1. terraform apply   → Provider Create() → POST to Portkey API
2. terraform refresh → Provider Read()   → GET from Portkey API  
3. Check functions   → Verify Terraform state matches expectations
4. terraform apply   → Provider Update() → PUT to Portkey API
5. terraform destroy → Provider Delete() → DELETE from Portkey API
```

Tests **create and destroy real resources**. Use a test/dev organization.

## Test File Structure

Tests live alongside the code they test (standard Go convention):

```
internal/provider/
├── workspace_resource.go
├── workspace_resource_test.go      # Tests for workspace_resource.go
├── user_invite_resource.go
├── user_invite_resource_test.go
└── ...
```

## Writing Tests

### Basic Test Template

```go
func TestAccResourceName_scenario(t *testing.T) {
    rName := acctest.RandomWithPrefix("tf-acc-test")

    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Step 1: Create
            {
                Config: testAccResourceConfig(rName),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet("portkey_resource.test", "id"),
                    resource.TestCheckResourceAttr("portkey_resource.test", "name", rName),
                ),
            },
            // Step 2: Import
            {
                ResourceName:            "portkey_resource.test",
                ImportState:             true,
                ImportStateVerify:       true,
                ImportStateVerifyIgnore: []string{"sensitive_field"},
            },
            // Step 3: Update
            {
                Config: testAccResourceConfigUpdated(rName),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("portkey_resource.test", "name", rName+"-updated"),
                ),
            },
            // Step 4: Delete happens automatically
        },
    })
}

func testAccResourceConfig(name string) string {
    return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_resource" "test" {
  name = %[1]q
}
`, name)
}
```

### Common Check Functions

```go
// Attribute exists
resource.TestCheckResourceAttrSet("portkey_workspace.test", "id")

// Exact value
resource.TestCheckResourceAttr("portkey_workspace.test", "name", "expected")

// Compare two resources
resource.TestCheckResourceAttrPair(
    "data.portkey_workspace.test", "name",
    "portkey_workspace.test", "name",
)

// List length
resource.TestCheckResourceAttr("portkey_user_invite.test", "workspaces.#", "2")

// Nested attribute
resource.TestCheckResourceAttr("portkey_user_invite.test", "workspaces.0.role", "admin")
```

### Using Data Sources in Tests

To get dynamic values (like existing user IDs), use data sources:

```go
func testAccWorkspaceMemberConfig(workspaceName, role string) string {
    return fmt.Sprintf(`
provider "portkey" {}

data "portkey_users" "all" {}

resource "portkey_workspace" "test" {
  name = %[1]q
}

resource "portkey_workspace_member" "test" {
  workspace_id = portkey_workspace.test.id
  user_id      = data.portkey_users.all.users[0].id
  role         = %[2]q
}
`, workspaceName, role)
}
```

### Skipping Tests

Skip tests when prerequisites aren't met or APIs have known issues:

```go
func TestAccFeature_basic(t *testing.T) {
    t.Skip("Skipping: API endpoint has known bug - see issue #123")
    // ...
}
```

## Running Tests

```bash
# All acceptance tests
make testacc

# Specific test
TF_ACC=1 go test ./internal/provider -v -run TestAccWorkspaceResource_basic

# With debug logging
TF_ACC=1 TF_LOG=DEBUG go test ./internal/provider -v -run TestAccWorkspaceResource_basic

# Unit tests only (no API calls)
go test ./internal/provider -v -run TestProvider_Has
```

## Best Practices

1. **Random names** - Use `acctest.RandomWithPrefix("tf-acc-")` to avoid conflicts
2. **Test full lifecycle** - Create → Read → Update → Import → Delete
3. **Isolate tests** - Each test creates/destroys its own resources
4. **Check computed fields** - Verify `id`, `created_at`, etc. are set
5. **Ignore timestamps on import** - Use `ImportStateVerifyIgnore` for fields that may differ

## Debugging

```bash
# Enable Terraform debug logs
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform.log

# Disable test caching
TF_ACC=1 go test ./internal/provider -v -run TestName -count=1
```

## Current Test Status

| Component | Status | Notes |
|-----------|--------|-------|
| Workspace Resource | ✅ Passing | Full CRUD + Import |
| User Invite Resource | ✅ Passing | With workspaces and scopes |
| Workspace Data Source | ✅ Passing | |
| Workspaces Data Source | ✅ Passing | |
| User Data Source | ✅ Passing | |
| Users Data Source | ✅ Passing | |
| Workspace Member Resource | ⏸️ Skipped | Blocked by API bug in `getMember` endpoint |
| Secret Reference Resource | ✅ Passing | CRUD + write-only credentials, `auth_version` rotation, and plan-time validation |
| Secret Reference Data Source | ✅ Passing | |
| Secret References Data Source | ✅ Passing | |

## Troubleshooting

**Provider not found:**
```bash
make clean && make install
rm -rf .terraform .terraform.lock.hcl && terraform init
```

**API auth errors:**
- Verify: `echo $PORTKEY_API_KEY`
- Ensure key has admin privileges

**Orphaned resources:**
- Check Portkey dashboard
- Delete manually via API/dashboard