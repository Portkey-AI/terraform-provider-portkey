# Portkey Admin API - Available Endpoints

## ✅ Fully Implemented Resources

All primary Portkey resources are now implemented in the Terraform provider.

### Organization Management

| Resource | Endpoint | Operations | Terraform Resource |
|----------|----------|------------|-------------------|
| Workspaces | `/admin/workspaces` | CRUD | `portkey_workspace` |
| Users | `/admin/users` | Read, List | `portkey_user` (data source) |
| User Invites | `/admin/users/invites` | CRD | `portkey_user_invite` |
| Workspace Members | `/admin/workspaces/{id}/users` | CRUD | `portkey_workspace_member` |

### AI Gateway

| Resource | Endpoint | Operations | Terraform Resource |
|----------|----------|------------|-------------------|
| Integrations | `/integrations` | CRUD | `portkey_integration` |
| Providers (Virtual Keys) | `/providers` | CRUD | `portkey_provider` |
| Configs | `/configs` | CRUD | `portkey_config` |
| Prompts | `/prompts` | CRUD | `portkey_prompt` |

### Governance & Policies

| Resource | Endpoint | Operations | Terraform Resource |
|----------|----------|------------|-------------------|
| Guardrails | `/guardrails` | CRUD | `portkey_guardrail` |
| Usage Limits | `/policies/usage-limits` | CRUD | `portkey_usage_limits_policy` |
| Rate Limits | `/policies/rate-limits` | CRUD | `portkey_rate_limits_policy` |

### Access Control

| Resource | Endpoint | Operations | Terraform Resource |
|----------|----------|------------|-------------------|
| API Keys | `/api-keys` | CRUD | `portkey_api_key` |

### Secret Management

| Resource | Endpoint | Operations | Terraform Resource |
|----------|----------|------------|-------------------|
| Secret References | `/secret-references` | CRUD | `portkey_secret_reference` |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `portkey_workspace` | Fetch single workspace by ID |
| `portkey_workspaces` | List all workspaces |
| `portkey_user` | Fetch single user by ID |
| `portkey_users` | List all users |
| `portkey_integration` | Fetch single integration by slug |
| `portkey_integrations` | List all integrations |
| `portkey_provider` | Fetch single provider by ID |
| `portkey_providers` | List providers in workspace |
| `portkey_config` | Fetch single config by slug |
| `portkey_configs` | List configs |
| `portkey_prompt` | Fetch single prompt by ID/slug |
| `portkey_prompts` | List prompts |
| `portkey_guardrail` | Fetch single guardrail by ID/slug |
| `portkey_guardrails` | List guardrails |
| `portkey_usage_limits_policy` | Fetch single policy by ID |
| `portkey_usage_limits_policies` | List usage limits policies |
| `portkey_rate_limits_policy` | Fetch single policy by ID |
| `portkey_rate_limits_policies` | List rate limits policies |
| `portkey_api_key` | Fetch single API key by ID |
| `portkey_api_keys` | List API keys |
| `portkey_secret_reference` | Fetch single secret reference by slug |
| `portkey_secret_references` | List secret references |

## Known Limitations

### Workspace Deletion
- Workspaces with resources (virtual keys, configs, etc.) cannot be deleted
- Error: `409: Unable to delete. Please ensure that all Virtual Keys are deleted`
- Workaround: Delete all workspace resources before deleting the workspace

### User Updates
- User role update API rejects same-role updates
- User resource is read-only in the provider

### User Invite Updates
- No PUT endpoint exists for user invites
- To modify an invite, delete and recreate it

### Prompt Template Updates
- Template updates via API have validation issues
- Name updates work reliably
- Template changes create new versions (not default)

## API Architecture

```
Organization Level (Admin API Key)
├── /admin/workspaces ✅
├── /admin/users ✅ (read-only)
├── /admin/users/invites ✅
│
Organization-owned with per-workspace visibility (Admin API Key)
└── /secret-references ✅
    # Object is org-scoped (has organisation_id, no workspace_id). Visibility
    # to workspaces is controlled by allow_all_workspaces / allowed_workspaces
    # in the request body, not by the URL scope.
│
Workspace Level (Workspace or Admin API Key)
├── /integrations ✅
├── /providers (virtual-keys) ✅
├── /configs ✅
├── /prompts ✅
├── /guardrails ✅
├── /policies/usage-limits ✅
├── /policies/rate-limits ✅
└── /api-keys ✅
```

## Test Status

All implemented resources have passing acceptance tests:

| Category | Tests | Status |
|----------|-------|--------|
| Workspaces | 5 tests | ⚠️ Delete blocked by backend issue |
| Users & Invites | 5 tests | ✅ Passing |
| Integrations | 5 tests | ✅ Passing |
| Providers | 2 tests | ✅ Passing |
| Configs | 5 tests | ✅ Passing |
| Prompts | 5 tests | ✅ Passing |
| Guardrails | 4 tests | ✅ Passing |
| Usage Limits Policies | 4 tests | ✅ Passing |
| Rate Limits Policies | 4 tests | ✅ Passing |
| API Keys | 4 tests | ✅ Passing |
| Secret References | 11 tests | ✅ Passing |

## Related Documentation

- [Resource Matrix](./RESOURCE_MATRIX.md) - Detailed CRUD operation status
- [Testing Guide](./TESTING.md) - How to run tests
- [Adding New APIs](./ADDING_NEW_APIS.md) - Development playbook
