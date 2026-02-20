# Resource x Operation Matrix

*Last updated: February 2026*

## Summary

| Category | Resources | Data Sources | Test Status |
|----------|:---------:|:------------:|-------------|
| Organization | 3 | 4 | ⚠️ Workspace delete blocked |
| AI Gateway | 6 | 12 | ✅ All passing |
| Governance | 3 | 6 | ✅ All passing |
| Access Control | 1 | 2 | ✅ All passing |
| MCP Gateway | 3 | 2 | ✅ All passing (11 tests) |
| **Total** | **16** | **26** | **All passing** |

## Provider Resources

| Resource | Create | Read | Update | Delete | Import | API Status | Test Status |
|----------|:------:|:----:|:------:|:------:|:------:|------------|-------------|
| `portkey_workspace` | ✅ | ✅ | ✅ | ⚠️ | ✅ | Delete requires name in body | ⚠️ 7 tests, delete blocked by backend |
| `portkey_workspace_member` | ✅ | ⚠️ | ✅ | ✅ | ✅ | getMember API has issues | Skipped |
| `portkey_user_invite` | ✅ | ✅ | ❌ | ✅ | ✅ | Update API doesn't exist | ✅ Passing |
| `portkey_integration` | ✅ | ✅ | ✅ | ✅ | ✅ | Full CRUD working | ✅ Passing |
| `portkey_api_key` | ✅ | ✅ | ✅ | ✅ | ✅ | Full CRUD working | ✅ 7 tests |
| `portkey_provider` | ✅ | ✅ | ✅ | ✅ | ✅ | Full CRUD working | ✅ Passing |
| `portkey_config` | ✅ | ✅ | ✅ | ✅ | ✅ | Full CRUD working | ✅ Passing |
| `portkey_prompt` | ✅ | ✅ | ⚠️ | ✅ | ✅ | Template updates need versions | ✅ Passing |
| `portkey_prompt_partial` | ✅ | ✅ | ⚠️ | ✅ | ✅ | Content updates need versions | ✅ Passing |
| `portkey_prompt_collection` | ✅ | ✅ | ✅ | ✅ | ✅ | Full CRUD working | ✅ Passing |
| `portkey_guardrail` | ✅ | ✅ | ✅ | ✅ | ✅ | Full CRUD working | ✅ Passing |
| `portkey_usage_limits_policy` | ✅ | ✅ | ✅ | ✅ | ✅ | Full CRUD working | ✅ Passing |
| `portkey_rate_limits_policy` | ✅ | ✅ | ✅ | ✅ | ✅ | Full CRUD working | ✅ Passing |
| `portkey_mcp_integration` | ✅ | ✅ | ✅ | ✅ | ✅ | Full CRUD | ✅ Passing |
| `portkey_mcp_integration_workspace_access` | ✅ | ✅ | ✅ | ✅ | ✅ | Bulk PUT wrapping | ✅ Passing |
| `portkey_mcp_integration_capabilities` | ✅ | ✅ | ✅ | ✅ | ✅ | Bulk PUT | ✅ Passing |

## Data Sources

| Data Source | Read | List | API Status | Test Status |
|-------------|:----:|:----:|------------|-------------|
| `portkey_workspace` | ✅ | - | Working | ✅ 3 tests |
| `portkey_workspaces` | - | ✅ | Working | ✅ 2 tests |
| `portkey_user` | ✅ | - | Working | Passing |
| `portkey_users` | - | ✅ | Working | Passing |
| `portkey_integration` | ✅ | - | Working | ✅ Passing |
| `portkey_integrations` | - | ✅ | Working | ✅ Passing |
| `portkey_api_key` | ✅ | - | Working | ✅ 3 tests |
| `portkey_api_keys` | - | ✅ | Working | ✅ 1 test |
| `portkey_provider` | ✅ | - | Working | ✅ Passing |
| `portkey_providers` | - | ✅ | Working | ✅ Passing |
| `portkey_config` | ✅ | - | Working | ✅ Passing |
| `portkey_configs` | - | ✅ | Working | ✅ Passing |
| `portkey_prompt` | ✅ | - | Working | ✅ Passing |
| `portkey_prompts` | - | ✅ | Working | ✅ Passing |
| `portkey_prompt_partial` | ✅ | - | Working | ✅ Passing |
| `portkey_prompt_partials` | - | ✅ | Working | ✅ Passing |
| `portkey_prompt_collection` | ✅ | - | Working | ✅ Passing |
| `portkey_prompt_collections` | - | ✅ | Working | ✅ Passing |
| `portkey_guardrail` | ✅ | - | Working | ✅ Passing |
| `portkey_guardrails` | - | ✅ | Working | ✅ Passing |
| `portkey_usage_limits_policy` | ✅ | - | Working | ✅ Passing |
| `portkey_usage_limits_policies` | - | ✅ | Working | ✅ Passing |
| `portkey_rate_limits_policy` | ✅ | - | Working | ✅ Passing |
| `portkey_rate_limits_policies` | - | ✅ | Working | ✅ Passing |
| `portkey_mcp_integration` | ✅ | - | Working | ✅ Passing |
| `portkey_mcp_integrations` | - | ✅ | Working | ✅ Passing |

## Not Implemented (API Available)

None - all primary resources are now implemented!

*Note: `portkey_provider` is the same as Virtual Keys (`/providers` aliases `/virtual-keys`)*

## Not Implemented (API Issues)

| Resource | Create | Read | Update | Delete | Issue |
|----------|:------:|:----:|:------:|:------:|-------|
| `portkey_user` | N/A | ✅ | ⚠️ | ✅ | Update rejects same-role updates |

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Working |
| ⚠️ | Has known issues |
| ❌ | Not available in API |
| 🟡 | API available, not implemented |
| - | Not applicable |

## API Endpoints by Resource

### Workspaces
```
POST   /admin/workspaces           → Create
GET    /admin/workspaces           → List
GET    /admin/workspaces/{id}      → Read
PUT    /admin/workspaces/{id}      → Update
DELETE /admin/workspaces/{id}      → Delete (requires {"name": "..."} in body)
```

### Users
```
GET    /admin/users                → List
GET    /admin/users/{id}           → Read
PUT    /admin/users/{id}           → Update (rejects if role unchanged)
DELETE /admin/users/{id}           → Delete
```

### User Invites
```
POST   /admin/users/invites        → Create
GET    /admin/users/invites        → List
GET    /admin/users/invites/{id}   → Read
DELETE /admin/users/invites/{id}   → Delete
```
*Note: PUT endpoint does not exist*

### Workspace Members
```
POST   /admin/workspaces/{id}/users           → Add member
GET    /admin/workspaces/{id}/users           → List members
GET    /admin/workspaces/{id}/users/{userId}  → Get member (buggy)
PUT    /admin/workspaces/{id}/users/{userId}  → Update role
DELETE /admin/workspaces/{id}/users/{userId}  → Remove member
```

### Integrations
```
POST   /integrations               → Create (requires `key` for most providers)
GET    /integrations               → List
GET    /integrations/{slug}        → Read
PUT    /integrations/{slug}        → Update
DELETE /integrations/{slug}        → Delete
```

### API Keys
```
POST   /api-keys/{type}/{sub-type} → Create
GET    /api-keys                   → List (optional ?workspace_id=xxx filter)
GET    /api-keys/{id}              → Read
PUT    /api-keys/{id}              → Update
DELETE /api-keys/{id}              → Delete
```

**API Key Types:**

| Type | Sub-Type | Name | Use Case |
|------|----------|------|----------|
| `organisation` | `service` | Admin API Key | Access to Admin APIs (org management) |
| `workspace` | `service` | Workspace Service Key | Workspace-scoped service access |
| `workspace` | `user` | Workspace User Key | User-specific workspace access (requires `user_id`) |

*Note: Workspace keys require `workspace_id` in the request body*

### Providers (Virtual Keys)
```
POST   /providers                  → Create (requires workspace_id + integration_id in body)
GET    /providers                  → List (requires ?workspace_id=xxx)
GET    /providers/{id}             → Read (requires ?workspace_id=xxx)
PUT    /providers/{id}             → Update (requires workspace_id in body)
DELETE /providers/{id}             → Delete (requires ?workspace_id=xxx)
```

**Important Requirements:**
- `workspace_id` must be the UUID, not the slug
- `integration_id` must reference an integration enabled for the workspace
- Import format: `workspace_id:provider_id`

### Prompts
```
POST   /prompts                    → Create (requires collection_id, string, parameters, virtual_key, model)
GET    /prompts                    → List (optional ?workspace_id=xxx or ?collection_id=xxx)
GET    /prompts/{slugOrId}         → Read (optional ?version=latest|default|N)
PUT    /prompts/{slugOrId}         → Update (name-only works; template updates create new versions)
DELETE /prompts/{slugOrId}         → Delete
PUT    /prompts/{slugOrId}/makeDefault → Set default version
```

**Prompt Versioning:**
- Each prompt has multiple versions
- Updates to template/model/parameters create a NEW version (not default)
- Use `makeDefault` to promote a version to default
- `?version=latest` gets the newest version, `?version=default` (or omit) gets the default

**Important Notes:**
- API may return different values than provided (virtual_key ID → slug, adds model to parameters)
- Template updates via API have validation issues - name updates work reliably

### Prompt Partials
```
POST   /prompts/partials                      → Create (requires name, string)
GET    /prompts/partials                      → List (optional ?workspace_id=xxx)
GET    /prompts/partials/{slugOrId}           → Read (optional ?version=latest|default|N)
PUT    /prompts/partials/{slugOrId}           → Update (name-only works; content updates create new versions)
DELETE /prompts/partials/{slugOrId}           → Delete
PUT    /prompts/partials/{slugOrId}/makeDefault → Set default version
```

**Prompt Partial Versioning:**
- Same versioning model as prompts
- Updates to content (`string` field) create a NEW version (not default)
- Use `makeDefault` to promote a version to default
- Reference in prompts via Mustache syntax: `{{>partial-slug}}`

### Guardrails
```
POST   /guardrails                 → Create (requires workspace_id or organisation_id, checks, actions)
GET    /guardrails                 → List (requires ?workspace_id=xxx)
GET    /guardrails/{slugOrId}      → Read
PUT    /guardrails/{slugOrId}      → Update
DELETE /guardrails/{slugOrId}      → Delete
```

**Guardrail Checks:**
Checks define what to validate. Each check has an `id` and optional `parameters`:
- `default.wordCount` - min/max word counts
- `default.characterCount` - min/max character counts
- `default.regexMatch` - pattern matching
- `default.jsonSchema` - JSON structure validation
- `default.contains` - keyword detection
- `default.webhook` - custom webhook validation
- `portkey.moderateContent` - OpenAI moderation (Pro)
- And many more...

**Actions Configuration:**
```json
{
  "onFail": "block|log|warn",
  "message": "Error message",
  "deny": false,
  "async": false
}
```

### Usage Limits Policies
```
POST   /policies/usage-limits      → Create
GET    /policies/usage-limits      → List (requires ?workspace_id=xxx)
GET    /policies/usage-limits/{id} → Read
PUT    /policies/usage-limits/{id} → Update
DELETE /policies/usage-limits/{id} → Delete
```

**Usage Limits Configuration:**
- `conditions`: Array of conditions (key/value pairs) to match requests
- `group_by`: Array of fields to aggregate usage by (e.g., `api_key`, `metadata.user_id`)
- `type`: `cost` or `tokens`
- `credit_limit`: Maximum usage allowed
- `alert_threshold`: Optional threshold for alerts
- `periodic_reset`: `monthly` or `weekly` (optional, cumulative if not set)

### Rate Limits Policies
```
POST   /policies/rate-limits       → Create
GET    /policies/rate-limits       → List (requires ?workspace_id=xxx)
GET    /policies/rate-limits/{id}  → Read
PUT    /policies/rate-limits/{id}  → Update
DELETE /policies/rate-limits/{id}  → Delete
```

**Rate Limits Configuration:**
- `conditions`: Array of conditions (key/value pairs) to match requests
- `group_by`: Array of fields to apply rate limiting by
- `type`: `requests` or `tokens`
- `unit`: `rpm` (per minute), `rph` (per hour), or `rpd` (per day)
- `value`: Rate limit value

### Configs
```
POST   /configs                    → Create
GET    /configs                    → List (optional ?workspace_id=xxx filter)
GET    /configs/{slug}             → Read
PUT    /configs/{slug}             → Update
DELETE /configs/{slug}             → Delete
```

**Config Object Example:**
```json
{
  "name": "My Config",
  "workspace_id": "uuid",
  "config": {
    "retry": { "attempts": 3 },
    "cache": { "mode": "simple" }
  }
}
```
*Note: Config field is returned as a JSON string by the API, handled automatically*

### MCP Integrations
```
POST   /mcp-integrations                          → Create
GET    /mcp-integrations                          → List (optional ?workspace_id=xxx)
GET    /mcp-integrations/{id}                     → Read
PUT    /mcp-integrations/{id}                     → Update
DELETE /mcp-integrations/{id}                     → Delete
GET    /mcp-integrations/{id}/capabilities        → List capabilities
PUT    /mcp-integrations/{id}/capabilities        → Update capabilities (bulk)
GET    /mcp-integrations/{id}/workspaces          → List workspace access
PUT    /mcp-integrations/{id}/workspaces          → Update workspace access (bulk)
```

**MCP Integration Fields:**
- `name`: Display name
- `url`: MCP server URL
- `auth_type`: `none`, `headers`, `oauth_auto`
- `transport`: `http` (Streamable HTTP), `sse` (Server-Sent Events)
- `configurations`: JSON auth config (sensitive)
- `workspace_id`: Optional workspace scope

**Sub-resource Patterns:**
- Capabilities and access control use bulk PUT endpoints
- Single-item operations wrap a single item in the bulk request array
- Delete operations set `enabled=false` (no real DELETE endpoint)

> **Note:** `portkey_mcp_server` resources are not implemented — the `/mcp-servers` API returns 403 and appears to require additional permissions not available via the Admin API key. MCP integrations + workspace access is sufficient for the primary use case.

## Known Issues

### 1. Workspace Delete - Virtual Keys Block
- **Status**: Fixed provider code, but workspaces with resources can't be deleted
- **Error**: `409: Unable to delete. Please ensure that all Virtual Keys are deleted`
- **Cause**: Fresh workspaces appear to have auto-provisioned virtual keys
- **Action**: Report to Portkey backend team

### 2. Workspace Member Get - API Bug
- **Status**: Skipped in tests
- **Error**: getMember endpoint returns incomplete data
- **Action**: Report to Portkey backend team

### 3. User Update - Same Role Rejection
- **Status**: Not implemented as resource
- **Error**: `AB01: Invalid request` when updating to same role
- **Cause**: API rejects no-op updates
- **Action**: Provider would need to check if role changed before calling API

### 4. User Invite Update - No Endpoint
- **Status**: Not implemented
- **Error**: `404: Not Found`
- **Cause**: PUT endpoint doesn't exist
- **Action**: None - API design choice

## Test Commands

```bash
# Run all acceptance tests
export PORTKEY_API_KEY="your-key"
make testacc

# Run specific resource tests
TF_ACC=1 go test ./internal/provider -v -run TestAccWorkspaceResource
TF_ACC=1 go test ./internal/provider -v -run TestAccUserInviteResource

# Run with debug logging
TF_ACC=1 TF_LOG=DEBUG go test ./internal/provider -v -run TestName
```

