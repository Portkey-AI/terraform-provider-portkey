# Terraform Provider for Portkey

[![Terraform Registry](https://img.shields.io/badge/registry-terraform-blue.svg)](https://registry.terraform.io/providers/portkey-ai/portkey/latest)
[![CI](https://github.com/Portkey-AI/terraform-provider-portkey/actions/workflows/ci.yml/badge.svg)](https://github.com/Portkey-AI/terraform-provider-portkey/actions/workflows/ci.yml)
[![Acceptance Tests](https://github.com/Portkey-AI/terraform-provider-portkey/actions/workflows/acc-tests.yml/badge.svg)](https://github.com/Portkey-AI/terraform-provider-portkey/actions/workflows/acc-tests.yml)

A Terraform provider for managing Portkey resources through the [Portkey Admin API](https://portkey.ai/docs/api-reference/admin-api/introduction).

## Features

This provider enables you to manage:

### Organization Management
- **Workspaces**: Create, update, and manage workspaces for organizing teams and projects
- **Workspace Members**: Assign users to workspaces with specific roles
- **User Invitations**: Send invitations to users with organization and workspace access
- **Users**: Query existing users in your organization (read-only)

### AI Gateway Resources
- **Integrations**: Manage AI provider connections (OpenAI, Anthropic, Azure, etc.)
- **Integration Workspace Access**: Enable integrations for specific workspaces with optional usage/rate limits
- **Providers (Virtual Keys)**: Create workspace-scoped API keys for AI providers
- **Configs**: Define gateway configurations with routing, fallbacks, and load balancing
- **Prompts**: Manage versioned prompt templates

### Governance & Policies
- **Guardrails**: Set up content moderation, validation rules, and safety checks
- **Usage Limits Policies**: Control costs with spending limits and thresholds
- **Rate Limits Policies**: Manage request rates per minute/hour/day

### Access Control
- **API Keys**: Create and manage Portkey API keys (organization and workspace-scoped)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for building from source)
- A Portkey account with Admin API access

## Installation

### From Terraform Registry (Recommended)

```hcl
terraform {
  required_providers {
    portkey = {
      source  = "portkey-ai/portkey"
      version = "~> 0.1"
    }
  }
}
```

### Building from Source

```bash
git clone https://github.com/Portkey-AI/terraform-provider-portkey
cd terraform-provider-portkey
make install
```

This will build and install the provider in your local Terraform plugins directory.

## Authentication

The provider requires a Portkey Admin API key. You can provide this in one of two ways:

### Environment Variable (Recommended)

```bash
export PORTKEY_API_KEY="your-admin-api-key"
```

### Provider Configuration

```hcl
provider "portkey" {
  api_key = "your-admin-api-key"
}
```

### Getting Your Admin API Key

1. Log in to your Portkey dashboard
2. Navigate to Admin Settings
3. Create an Organization Admin API key
4. Ensure you have Organization Owner or Admin role

**Note**: Admin API keys provide broad access to your organization. Store them securely and never commit them to version control.

## Usage Examples

### Basic Configuration

```hcl
terraform {
  required_providers {
    portkey = {
      source = "portkey-ai/portkey"
    }
  }
}

provider "portkey" {
  # API key read from PORTKEY_API_KEY environment variable
}

# Create a workspace
resource "portkey_workspace" "production" {
  name        = "Production"
  description = "Production environment workspace"
}
```

### Complete AI Gateway Setup

```hcl
# Create a workspace
resource "portkey_workspace" "production" {
  name        = "Production"
  description = "Production environment"
}

# Create an integration for OpenAI
resource "portkey_integration" "openai" {
  name           = "OpenAI Production"
  ai_provider_id = "openai"
  key            = var.openai_api_key
}

# Create a provider (virtual key) linked to the integration
resource "portkey_provider" "openai_prod" {
  name           = "OpenAI Production Key"
  workspace_id   = portkey_workspace.production.id
  integration_id = portkey_integration.openai.id
}

# Create a gateway config with retry logic
resource "portkey_config" "production" {
  name         = "Production Config"
  workspace_id = portkey_workspace.production.id
  
  config = jsonencode({
    retry = {
      attempts = 3
      on_status_codes = [429, 500, 502, 503]
    }
    cache = {
      mode = "simple"
    }
  })
}

# Create a prompt template
resource "portkey_prompt" "assistant" {
  name          = "AI Assistant"
  collection_id = "your-collection-id"
  virtual_key   = portkey_provider.openai_prod.id
  model         = "gpt-4"
  
  string = "You are a helpful assistant. User: {{user_input}}"
  
  parameters = jsonencode({
    temperature = 0.7
    max_tokens  = 1000
  })
}

# Create a guardrail for content moderation
resource "portkey_guardrail" "content_filter" {
  name         = "Content Filter"
  workspace_id = portkey_workspace.production.id
  
  # checks and actions must be JSON-encoded
  checks = jsonencode([
    {
      id = "default.wordCount"
      parameters = {
        minWords = 1
        maxWords = 5000
      }
    }
  ])
  
  actions = jsonencode({
    onFail  = "block"
    message = "Content validation failed"
  })
}

# Create a usage limits policy
resource "portkey_usage_limits_policy" "cost_limit" {
  name           = "Monthly Cost Limit"
  workspace_id   = portkey_workspace.production.id
  type           = "cost"
  credit_limit   = 1000.0
  alert_threshold = 800.0
  periodic_reset = "monthly"
  
  # conditions and group_by must be JSON-encoded arrays
  conditions = jsonencode([
    { key = "workspace_id", value = portkey_workspace.production.id }
  ])
  group_by = jsonencode([
    { key = "api_key" }
  ])
}

# Create a rate limits policy
resource "portkey_rate_limits_policy" "api_rate" {
  name         = "API Rate Limit"
  workspace_id = portkey_workspace.production.id
  type         = "requests"
  unit         = "rpm"
  value        = 100
  
  # conditions and group_by must be JSON-encoded arrays
  conditions = jsonencode([
    { key = "workspace_id", value = portkey_workspace.production.id }
  ])
  group_by = jsonencode([
    { key = "api_key" }
  ])
}

# Create a Portkey API key for your application
resource "portkey_api_key" "backend" {
  name         = "Backend Service Key"
  type         = "workspace"
  sub_type     = "service"
  workspace_id = portkey_workspace.production.id
  
  scopes = [
    "logs.list",
    "logs.view",
    "configs.read",
    "providers.list"
  ]
}
```

### Organization User Management

```hcl
# Invite a user with workspace access
resource "portkey_user_invite" "data_scientist" {
  email = "scientist@example.com"
  role  = "member"

  workspaces = [
    {
      id   = portkey_workspace.production.id
      role = "admin"
    }
  ]

  scopes = [
    "logs.export",
    "logs.list",
    "logs.view",
    "configs.read",
    "configs.list",
    "virtual_keys.read",
    "virtual_keys.list"
  ]
}

# Add an existing user to a workspace
resource "portkey_workspace_member" "senior_engineer" {
  workspace_id = portkey_workspace.production.id
  user_id      = "existing-user-id"
  role         = "manager"
}

# Query all workspaces
data "portkey_workspaces" "all" {}

# Query all users
data "portkey_users" "all" {}
```

### Self-Hosted Portkey

For self-hosted Portkey deployments, configure the base URL:

```hcl
provider "portkey" {
  api_key  = var.portkey_api_key
  base_url = "https://your-portkey-instance.com/v1"
}
```

## Resources

### Organization Resources

#### `portkey_workspace`

Manages a Portkey workspace.

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Name of the workspace |
| `description` | String | No | Description of the workspace |

**Import**: `terraform import portkey_workspace.example workspace-id`

#### `portkey_workspace_member`

Manages workspace membership for users.

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `workspace_id` | String | Yes | ID of the workspace |
| `user_id` | String | Yes | ID of the user |
| `role` | String | Yes | Role: `admin`, `manager`, `member` |

**Import**: `terraform import portkey_workspace_member.example workspace-id/member-id`

#### `portkey_user_invite`

Sends invitations to users.

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `email` | String | Yes | Email address to invite |
| `role` | String | Yes | Organization role: `admin`, `member` |
| `workspaces` | List | No | Workspaces to add user to |
| `scopes` | List | No | API scopes for the user |

**Note**: User invitations cannot be updated. To change an invitation, delete and recreate it.

---

### AI Gateway Resources

#### `portkey_integration`

Manages AI provider integrations (organization-level).

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Name of the integration |
| `ai_provider_id` | String | Yes | Provider: `openai`, `anthropic`, `azure-openai`, `aws-bedrock`, etc. |
| `key` | String | No | API key for the provider (write-only) |
| `configurations` | String (JSON) | No | Provider-specific configurations (write-only) |
| `description` | String | No | Description |

**Import**: `terraform import portkey_integration.example integration-slug`

**Note**: The `key` and `configurations` fields are write-only and cannot be retrieved from the API after creation. When importing, you must manually add these values to your configuration.

##### AWS Bedrock with IAM Role

```hcl
resource "portkey_integration" "bedrock" {
  name           = "AWS Bedrock Production"
  ai_provider_id = "aws-bedrock"
  
  configurations = jsonencode({
    aws_role_arn    = "arn:aws:iam::123456789012:role/PortkeyBedrockRole"
    aws_region      = "us-east-1"
    aws_external_id = "your-external-id"  # Optional
  })
}
```

##### AWS Bedrock with Access Keys

```hcl
resource "portkey_integration" "bedrock_keys" {
  name           = "AWS Bedrock (Access Keys)"
  ai_provider_id = "aws-bedrock"
  key            = var.aws_secret_access_key
  
  configurations = jsonencode({
    aws_access_key_id = var.aws_access_key_id
    aws_region        = "us-east-1"
  })
}
```

##### Azure OpenAI

```hcl
resource "portkey_integration" "azure_openai" {
  name           = "Azure OpenAI"
  ai_provider_id = "azure-openai"
  key            = var.azure_api_key

  configurations = jsonencode({
    resource_name = "my-azure-resource"
    deployment_id = "gpt-4-deployment"
    api_version   = "2024-02-15-preview"
  })
}
```

#### `portkey_integration_workspace_access`

Manages workspace access for integrations. Enables an integration to be used within a specific workspace.

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `integration_id` | String | Yes | Integration slug or ID |
| `workspace_id` | String | Yes | Workspace ID to grant access to |
| `enabled` | Boolean | No | Whether access is enabled (default: true) |
| `usage_limits` | Block | No | Usage limits for this workspace |
| `rate_limits` | Block | No | Rate limits for this workspace |

**Import**: `terraform import portkey_integration_workspace_access.example integration-slug/workspace-id`

##### Basic Example

```hcl
resource "portkey_integration_workspace_access" "openai_dev" {
  integration_id = portkey_integration.openai.slug
  workspace_id   = portkey_workspace.dev.id
}
```

##### With Limits

```hcl
resource "portkey_integration_workspace_access" "openai_dev" {
  integration_id = portkey_integration.openai.slug
  workspace_id   = portkey_workspace.dev.id
  enabled        = true

  usage_limits {
    type            = "cost"
    credit_limit    = 100
    alert_threshold = 80
    periodic_reset  = "monthly"
  }

  rate_limits {
    type  = "requests"
    unit  = "rpm"
    value = 1000
  }
}
```

#### `portkey_provider`

Manages providers (virtual keys) - workspace-scoped API keys for AI providers.

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Name of the provider |
| `workspace_id` | String | Yes | Workspace ID (UUID) |
| `integration_id` | String | Yes | Integration ID to link to |
| `note` | String | No | Notes |

**Import**: `terraform import portkey_provider.example workspace-id:provider-id`

#### `portkey_config`

Manages gateway configurations with routing, fallbacks, and caching.

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Name of the config |
| `workspace_id` | String | Yes | Workspace ID |
| `config` | String (JSON) | Yes | Configuration object |
| `is_default` | Number | No | Whether this is the default config |

**Import**: `terraform import portkey_config.example config-slug`

#### `portkey_prompt`

Manages versioned prompt templates.

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Name of the prompt |
| `collection_id` | String | Yes | Collection ID |
| `string` | String | Yes | Prompt template |
| `virtual_key` | String | Yes | Provider ID to use |
| `model` | String | Yes | Model name |
| `parameters` | String (JSON) | No | Model parameters |

**Import**: `terraform import portkey_prompt.example prompt-slug`

---

### Governance Resources

#### `portkey_guardrail`

Manages content validation and safety checks.

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Name of the guardrail |
| `workspace_id` | String | No | Workspace ID (or use organisation_id) |
| `organisation_id` | String | No | Organisation ID |
| `checks` | List | Yes | Validation checks to perform |
| `actions` | Object | Yes | Actions on check failure |

**Import**: `terraform import portkey_guardrail.example guardrail-slug`

#### `portkey_usage_limits_policy`

Manages spending limits and cost controls.

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Name of the policy |
| `workspace_id` | String | Yes | Workspace ID |
| `type` | String | Yes | `cost` or `tokens` |
| `credit_limit` | Number | Yes | Maximum usage allowed |
| `alert_threshold` | Number | No | Threshold for alerts |
| `periodic_reset` | String | No | `monthly` or `weekly` |
| `conditions` | List | Yes | Conditions to match |
| `group_by` | List | Yes | Fields to group usage by |

**Import**: `terraform import portkey_usage_limits_policy.example policy-id`

#### `portkey_rate_limits_policy`

Manages request rate limiting.

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Name of the policy |
| `workspace_id` | String | Yes | Workspace ID |
| `type` | String | Yes | `requests` or `tokens` |
| `unit` | String | Yes | `rpm`, `rph`, or `rpd` |
| `value` | Number | Yes | Rate limit value |
| `conditions` | List | Yes | Conditions to match |
| `group_by` | List | Yes | Fields to apply limits by |

**Import**: `terraform import portkey_rate_limits_policy.example policy-id`

---

### Access Control Resources

#### `portkey_api_key`

Manages Portkey API keys.

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Name of the API key |
| `type` | String | Yes | `organisation` or `workspace` |
| `sub_type` | String | Yes | `service` or `user` |
| `workspace_id` | String | No | Required for workspace keys |
| `scopes` | List | Yes | API scopes |

**Import**: `terraform import portkey_api_key.example api-key-id`

## Data Sources

### Organization Data Sources

| Data Source | Description | Key Arguments |
|-------------|-------------|---------------|
| `portkey_workspace` | Fetch a single workspace | `id` |
| `portkey_workspaces` | List all workspaces | - |
| `portkey_user` | Fetch a single user | `id` |
| `portkey_users` | List all users | - |

### AI Gateway Data Sources

| Data Source | Description | Key Arguments |
|-------------|-------------|---------------|
| `portkey_integration` | Fetch a single integration | `slug` |
| `portkey_integrations` | List all integrations | - |
| `portkey_integration_workspaces` | List workspace access for an integration | `integration_id` |
| `portkey_provider` | Fetch a single provider | `id`, `workspace_id` |
| `portkey_providers` | List providers in workspace | `workspace_id` |
| `portkey_config` | Fetch a single config | `slug` |
| `portkey_configs` | List configs | `workspace_id` (optional) |
| `portkey_prompt` | Fetch a single prompt | `id_or_slug` |
| `portkey_prompts` | List prompts | `workspace_id`, `collection_id` (optional) |

### Governance Data Sources

| Data Source | Description | Key Arguments |
|-------------|-------------|---------------|
| `portkey_guardrail` | Fetch a single guardrail | `id_or_slug` |
| `portkey_guardrails` | List guardrails | `workspace_id` or `organisation_id` |
| `portkey_usage_limits_policy` | Fetch a usage limits policy | `id` |
| `portkey_usage_limits_policies` | List usage limits policies | `workspace_id` |
| `portkey_rate_limits_policy` | Fetch a rate limits policy | `id` |
| `portkey_rate_limits_policies` | List rate limits policies | `workspace_id` |

### Access Control Data Sources

| Data Source | Description | Key Arguments |
|-------------|-------------|---------------|
| `portkey_api_key` | Fetch a single API key | `id` |
| `portkey_api_keys` | List API keys | `workspace_id` (optional) |

## Development

### Building

```bash
make build
```

### Testing

```bash
# Run unit tests
go test ./...

# Run acceptance tests (requires valid API key)
make testacc
```

### Installing Locally

```bash
make install
```

### Generating Documentation

```bash
make generate
```

## API Scopes

When creating API keys or inviting users, you can grant scopes from these categories:

### Logs & Analytics
- `logs.list`, `logs.view`, `logs.export`
- `analytics.view`

### Configs
- `configs.create`, `configs.read`, `configs.update`, `configs.delete`, `configs.list`

### Providers (Virtual Keys)
- `providers.create`, `providers.read`, `providers.update`, `providers.delete`, `providers.list`
- `virtual_keys.create`, `virtual_keys.read`, `virtual_keys.update`, `virtual_keys.delete`, `virtual_keys.list`, `virtual_keys.copy`

### Prompts
- `prompts.create`, `prompts.read`, `prompts.update`, `prompts.delete`, `prompts.list`, `prompts.publish`

### Guardrails
- `guardrails.create`, `guardrails.read`, `guardrails.update`, `guardrails.delete`, `guardrails.list`

### Policies
- `policies.create`, `policies.read`, `policies.update`, `policies.delete`, `policies.list`

### Workspaces & Users
- `workspaces.create`, `workspaces.read`, `workspaces.update`, `workspaces.delete`, `workspaces.list`
- `workspace_users.create`, `workspace_users.read`, `workspace_users.update`, `workspace_users.delete`, `workspace_users.list`
- `organisation_users.create`, `organisation_users.read`, `organisation_users.update`, `organisation_users.delete`, `organisation_users.list`

### API Keys
- `organisation_service_api_keys.create`, `organisation_service_api_keys.read`, `organisation_service_api_keys.update`, `organisation_service_api_keys.delete`, `organisation_service_api_keys.list`
- `workspace_service_api_keys.create`, `workspace_service_api_keys.read`, `workspace_service_api_keys.update`, `workspace_service_api_keys.delete`, `workspace_service_api_keys.list`
- `workspace_user_api_keys.create`, `workspace_user_api_keys.read`, `workspace_user_api_keys.update`, `workspace_user_api_keys.delete`, `workspace_user_api_keys.list`

### Integrations
- `organisation_integrations.create`, `organisation_integrations.read`, `organisation_integrations.update`, `organisation_integrations.delete`, `organisation_integrations.list`
- `workspace_integrations.create`, `workspace_integrations.read`, `workspace_integrations.update`, `workspace_integrations.delete`, `workspace_integrations.list`

## Roles

### Organization Roles

- `admin` - Full organization access
- `member` - Standard user access

### Workspace Roles

- `admin` - Full workspace access
- `manager` - Manage workspace resources and members
- `member` - Standard workspace access

## Resource Prerequisites

Some resources have specific prerequisites that must be met before they can be created.

### `portkey_provider` (Virtual Key)

To create a provider, you need:

1. **A workspace ID** (UUID format, not slug)
2. **An integration that is enabled for that workspace**

```hcl
# First, create or reference an integration
resource "portkey_integration" "openai" {
  name           = "OpenAI"
  ai_provider_id = "openai"
  key            = var.openai_api_key
}

# Create a workspace
resource "portkey_workspace" "production" {
  name = "Production"
}

# Enable the integration for the workspace
resource "portkey_integration_workspace_access" "openai_prod" {
  integration_id = portkey_integration.openai.slug
  workspace_id   = portkey_workspace.production.id
}

# Then create a provider linked to the integration
resource "portkey_provider" "main" {
  name           = "Production OpenAI"
  workspace_id   = portkey_workspace.production.id
  integration_id = portkey_integration.openai.slug

  depends_on = [portkey_integration_workspace_access.openai_prod]
}
```

**Common error**: `403 Forbidden` - The integration is not enabled for the specified workspace. Use `portkey_integration_workspace_access` to enable it.

### `portkey_api_key`

API keys require at least one scope:

```hcl
resource "portkey_api_key" "backend" {
  name     = "Backend Service"
  type     = "organisation"  # or "workspace"
  sub_type = "service"       # or "user"
  scopes   = ["providers.list", "logs.view"]  # Required - at least one scope
}
```

**Common error**: `502 Bad Gateway` - No scopes provided. Always include at least one scope.

### `portkey_guardrail`, `portkey_usage_limits_policy`, `portkey_rate_limits_policy`

These resources require JSON-encoded fields:

```hcl
# Use jsonencode() for complex fields
resource "portkey_guardrail" "example" {
  name         = "Content Filter"
  workspace_id = var.workspace_id
  
  checks = jsonencode([
    {
      id = "default.wordCount"
      parameters = { minWords = 1, maxWords = 5000 }
    }
  ])
  
  actions = jsonencode({
    onFail  = "block"
    message = "Validation failed"
  })
}
```

**Common error**: `400 Bad Request` - Using HCL syntax instead of `jsonencode()`.

## Troubleshooting

### 403 Forbidden Errors

| Resource | Cause | Solution |
|----------|-------|----------|
| `portkey_provider` | Integration not enabled for workspace | Use `portkey_integration_workspace_access` resource to enable |
| `portkey_api_key` | Admin key lacks required scopes | Use an admin key with full permissions |
| Any resource | Workspace access not granted | Ensure your admin key has workspace access |

### 404 Not Found Errors

| Resource | Cause | Solution |
|----------|-------|----------|
| `portkey_api_key` | Self-hosted API route differences | Verify `base_url` is correct for your deployment |
| Any resource | Resource was deleted externally | Run `terraform refresh` to sync state |

### 400 Bad Request Errors

| Resource | Cause | Solution |
|----------|-------|----------|
| Policies | Empty `conditions` array | Provide at least one condition: `jsonencode([{key="workspace_id", value="..."}])` |
| Guardrails | Invalid check format | Use `jsonencode()` for `checks` and `actions` |
| Configs | Invalid config JSON | Ensure `config` field contains valid JSON |

### State Drift Issues

If Terraform shows unexpected diffs on every plan:

1. **Config resource**: The API may normalize JSON differently. Use consistent formatting.
2. **Prompt resource**: Template updates create new versions. Use name-only updates.

### Self-Hosted Portkey

For self-hosted deployments, ensure `base_url` points to your instance:

```hcl
provider "portkey" {
  api_key  = var.portkey_api_key
  base_url = "https://your-portkey-instance.com/v1"
}
```

## Known Issues

### Workspace Deletion - Virtual Keys Block (API Issue)
Workspaces may fail to delete with error: `409: Unable to delete. Please ensure that all Virtual Keys are deleted`. This occurs even for newly created workspaces due to auto-provisioned resources on the backend.

**Workaround**: Manually delete all providers/virtual keys in the workspace before destroying.

### Workspace Deletion - Emoji Names (API Issue)
Workspaces with emoji characters in the name may fail to delete with error: `Invalid value` for the `name` parameter. The API's DELETE endpoint appears to have stricter validation than create/update endpoints.

**Workaround**: Rename the workspace to remove emoji characters before deleting, or delete via the Portkey UI.

**Note**: If deletion fails with "Invalid value" for name, first try running `terraform refresh` to sync state, then retry the destroy.

### Workspace Member Read (API Issue)
The workspace member `getMember` endpoint returns inconsistent data, which can cause Terraform state drift.

**Status**: Tests are skipped; resource works for create/update/delete operations.

### User Resource (API Limitation)
The user update API rejects requests when updating to the same role value.

**Impact**: User resource is read-only (data source only). Use `portkey_user_invite` to manage user access.

### User Invite Updates (API Design)
User invitations cannot be updated - there is no PUT endpoint.

**Workaround**: Delete and recreate the invitation to modify it.

### Prompt Template Updates (API Behavior)
Updating a prompt's template or parameters creates a new version rather than updating in place. The new version is not automatically set as the default.

**Workaround**: Name updates work reliably. For template changes, use the Portkey UI or API directly to manage versions.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## License

This provider is distributed under the Mozilla Public License 2.0. See `LICENSE` for more information.

## Support

- Documentation: [Portkey Docs](https://portkey.ai/docs)
- Admin API Reference: [Admin API Docs](https://portkey.ai/docs/api-reference/admin-api/introduction)
- Issues: [GitHub Issues](https://github.com/Portkey-AI/terraform-provider-portkey/issues)
- Community: [Discord](https://portkey.sh/discord-1)

## Related Projects

- [Portkey Gateway](https://github.com/Portkey-AI/gateway) - Open-source AI Gateway
- [Portkey Python SDK](https://github.com/Portkey-AI/portkey-python-sdk)
- [Portkey Node SDK](https://github.com/Portkey-AI/portkey-node-sdk)
