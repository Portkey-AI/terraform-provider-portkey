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

### Secret Management
- **Secret References**: Register external secret managers (AWS Secrets Manager, Azure Key Vault, HashiCorp Vault) with plan-time validated, typed auth blocks

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
| `ai_provider_id` | String | Yes | Provider: `openai`, `anthropic`, `azure-openai`, `bedrock`, etc. |
| `key` | String | No | API key for the provider (write-only) |
| `configurations` | String (JSON) | No | Provider-specific configurations (write-only) |
| `description` | String | No | Description |

**Import**: `terraform import portkey_integration.example integration-slug`

**Note**: The `key` and `configurations` fields are write-only and cannot be retrieved from the API after creation. When importing, you must manually add these values to your configuration.

##### AWS Bedrock with IAM Role

```hcl
resource "portkey_integration" "bedrock" {
  name           = "AWS Bedrock Production"
  ai_provider_id = "bedrock"
  
  configurations = jsonencode({
    aws_auth_type   = "assumedRole"       # Required for assume-role integrations
    aws_role_arn    = "arn:aws:iam::123456789012:role/PortkeyBedrockRole"
    aws_region      = "us-east-1"
    aws_external_id = "your-external-id"  # Optional
  })
}
```

**Note**: When using Application Inference Profile ARNs as `model_slug` in `portkey_integration_model_access`, you must also set `base_model_slug` (format: `{region}.{bedrock_model_id}`, e.g. `global.anthropic.claude-sonnet-4-6`).

##### AWS Bedrock with Access Keys

```hcl
resource "portkey_integration" "bedrock_keys" {
  name           = "AWS Bedrock (Access Keys)"
  ai_provider_id = "bedrock"
  key            = var.aws_secret_access_key
  
  configurations = jsonencode({
    aws_access_key_id = var.aws_access_key_id
    aws_region        = "us-east-1"
  })
}
```

##### Azure OpenAI

Azure OpenAI requires a specific configuration format with authentication mode and deployment details:

```hcl
resource "portkey_integration" "azure_openai" {
  name           = "Azure OpenAI"
  ai_provider_id = "azure-openai"
  key            = var.azure_api_key

  configurations = jsonencode({
    azure_auth_mode     = "default"  # Required: "default", "entra", or "managed"
    azure_resource_name = "my-azure-resource"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"  # Model type: gpt-4, gpt-35-turbo, etc.
        is_default            = true
      }
    ]
  })
}
```

**Azure OpenAI Configuration Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `azure_auth_mode` | String | Yes | Authentication mode: "default", "entra", or "managed" |
| `azure_resource_name` | String | Yes | Your Azure OpenAI resource name |
| `azure_deployment_config` | Array | Yes | Array of deployment configurations (min 1) |
| `azure_deployment_config[].azure_deployment_name` | String | Yes | Deployment name from Azure OpenAI Studio |
| `azure_deployment_config[].azure_api_version` | String | Yes | API version (e.g., "2024-02-15-preview"), max 30 chars |
| `azure_deployment_config[].azure_model_slug` | String | Yes | Model identifier (e.g., "gpt-4", "gpt-35-turbo") |
| `azure_deployment_config[].is_default` | Boolean | No | Set to true for the default deployment |
| `azure_deployment_config[].alias` | String | No* | Alias for the deployment (*required if not default) |
| `azure_entra_tenant_id` | String | For entra | Azure AD tenant ID (required for entra auth) |
| `azure_entra_client_id` | String | For entra | Azure AD client ID (required for entra auth) |
| `azure_entra_client_secret` | String | For entra | Azure AD client secret (required for entra auth) |
| `azure_managed_client_id` | String | For managed | Managed identity client ID (optional for managed auth) |

**Authentication Modes:**

- `default` - Uses the API key provided in the `key` field
- `entra` - Uses Microsoft Entra ID for authentication
- `managed` - Uses Azure Managed Identity for authentication

**Multiple Deployments Example (API Key Auth):**

```hcl
resource "portkey_workspace" "production" {
  name        = "Production"
  description = "Production environment workspace"
}
resource "portkey_integration" "azure_openai_multi" {
  name           = "Azure OpenAI Multi-Model"
  ai_provider_id = "azure-openai"
  key            = var.azure_api_key

  configurations = jsonencode({
    azure_auth_mode     = "default"
    azure_resource_name = "my-azure-resource"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      },
      {
        alias                 = "gpt35"
        azure_deployment_name = "gpt-35-turbo-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-35-turbo"
      }
    ]
  })
}
```

**Microsoft Entra ID Authentication Example:**

```hcl
resource "portkey_integration" "azure_openai_entra" {
  name           = "Azure OpenAI (Entra ID)"
  ai_provider_id = "azure-openai"

  configurations = jsonencode({
    azure_auth_mode           = "entra"
    azure_resource_name       = "my-azure-resource"
    azure_entra_tenant_id     = var.azure_tenant_id
    azure_entra_client_id     = var.azure_client_id
    azure_entra_client_secret = var.azure_client_secret
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      }
    ]
  })
}
```

**Managed Identity Authentication Example:**

```hcl
resource "portkey_integration" "azure_openai_managed" {
  name           = "Azure OpenAI (Managed Identity)"
  ai_provider_id = "azure-openai"

  configurations = jsonencode({
    azure_auth_mode          = "managed"
    azure_resource_name      = "my-azure-resource"
    azure_managed_client_id  = var.azure_managed_client_id  # Optional
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      }
    ]
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

  usage_limits = [{
    type            = "cost"
    credit_limit    = 100
    alert_threshold = 80
    periodic_reset  = "monthly"
  }]

  rate_limits = [{
    type  = "requests"
    unit  = "rpm"
    value = 1000
  }]
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

Manages Portkey API keys (organization-scoped or workspace-scoped).

**Arguments**

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Human-readable name |
| `type` | String | Yes | `organisation` or `workspace` |
| `sub_type` | String | Yes | `service` or `user` |
| `workspace_id` | String | No | Required for `type = "workspace"` keys |
| `user_id` | String | No | Required for `sub_type = "user"` keys |
| `description` | String | No | Optional description |
| `scopes` | List(String) | No | Permission scopes |
| `metadata` | Map(String) | No | Custom string metadata attached to every request |
| `alert_emails` | List(String) | No | Emails to notify on usage alerts |
| `config_id` | String | No | ID of a Portkey config to bind as the default for all requests |
| `allow_config_override` | Bool | No | Allow callers to override `config_id` at request time (default `true`) |
| `expires_at` | String | No | RFC3339 datetime when this key expires; can be updated after creation |
| `reset_usage` | Bool | No | Set `true` to trigger an immediate usage counter reset (write-only trigger) |
| `usage_limits` | Object | No | See [usage_limits](#usage_limits) below |
| `rate_limits` | List(Object) | No | See [rate_limits](#rate_limits) below |
| `rotation_policy` | Object | No | See [rotation_policy](#rotation_policy) below |

**`usage_limits` nested object**

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `credit_limit` | Number | Yes (if block present) | Maximum credits (tokens or cost units) |
| `type` | String | No | `tokens` or `cost` — what the limit counts |
| `alert_threshold` | Number | No | Trigger alert emails at this usage level |
| `periodic_reset` | String | No | Reset cadence: `monthly` or `weekly` |
| `periodic_reset_days` | Number | No | Custom reset interval in days (1–365); alternative to `periodic_reset` |
| `next_usage_reset_at` | String | No/Computed | ISO8601 datetime for the next scheduled reset |

**`rate_limits` nested object** (list)

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `type` | String | Yes | `requests` or `tokens` |
| `unit` | String | Yes | `rpm`, `rph`, `rpd`, `rps`, `rpw` |
| `value` | Number | Yes | Limit value |

**`rotation_policy` nested object**

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `rotation_period` | String | No | `weekly` or `monthly` |
| `next_rotation_at` | String | No/Computed | ISO8601 datetime for the next scheduled rotation; computed by the API |
| `key_transition_period_ms` | Number | No | Overlap window in ms after rotation during which both old and new keys are valid (minimum 1800000 — 30 min) |

**Read-only attributes**: `id`, `key` (sensitive), `organisation_id`, `status`, `created_at`, `updated_at`, `last_reset_at`

**Import**: `terraform import portkey_api_key.example api-key-id`

---

### Secret Management Resources

#### `portkey_secret_reference`

Registers an external secret manager (AWS Secrets Manager, Azure Key Vault, or HashiCorp Vault) with Portkey. Instead of embedding credentials in Portkey, you reference a secret stored in your existing vault. Credentials inside the auth blocks are marked `Sensitive` and are never returned by the API after creation (the API returns them masked; the provider preserves the user-supplied values in state to avoid spurious diffs).

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Display name |
| `description` | String | No | Optional description |
| `manager_type` | String | Yes (ForceNew) | `aws_sm`, `azure_kv`, or `hashicorp_vault` |
| `secret_path` | String | Yes | Path to the secret in the external manager |
| `secret_key` | String | No | Optional key within the secret payload |
| `tags` | Map(String) | No | Arbitrary string tags |
| `allow_all_workspaces` | Bool | No | If `true` (default), reference is usable in every workspace. Mutually exclusive with a non-empty `allowed_workspaces` |
| `allowed_workspaces` | Set(String) | No | Restrict usage to these workspace IDs/slugs |
| Exactly one of the 9 `*_auth` blocks (see below) | | Yes | Plan-time validated |

**Plan-time validation (enforced in `ModifyPlan`):**
- Exactly one `*_auth` block must be configured.
- The configured `*_auth` block must match the declared `manager_type` family.
- `allow_all_workspaces = true` cannot be combined with a non-empty `allowed_workspaces`.

These errors are raised during `terraform plan`, not `terraform apply`, for fast feedback.

**Supported auth blocks by `manager_type`:**

| `manager_type` | Valid auth blocks |
|----------------|--------------------|
| `aws_sm` | `aws_access_key_auth`, `aws_assumed_role_auth`, `aws_service_role_auth` |
| `azure_kv` | `azure_entra_auth`, `azure_managed_auth` |
| `hashicorp_vault` | `vault_token_auth`, `vault_approle_auth`, `vault_kubernetes_auth` |

**Import**: `terraform import portkey_secret_reference.example secret-reference-slug`

##### AWS Secrets Manager (Access Keys)

```hcl
resource "portkey_secret_reference" "aws_prod" {
  name         = "AWS Prod Secrets"
  manager_type = "aws_sm"
  secret_path  = "prod/openai/api-key"

  aws_access_key_auth = {
    aws_access_key_id     = var.aws_access_key_id
    aws_secret_access_key = var.aws_secret_access_key
    aws_region            = "us-east-1"
  }

  allow_all_workspaces = true
}
```

##### AWS Secrets Manager (Assumed Role)

```hcl
resource "portkey_secret_reference" "aws_role" {
  name         = "AWS Role-Based"
  manager_type = "aws_sm"
  secret_path  = "prod/openai/api-key"

  aws_assumed_role_auth = {
    aws_role_arn    = "arn:aws:iam::123456789012:role/PortkeySecretsRole"
    aws_region      = "us-east-1"
    aws_external_id = "optional-external-id"
  }
}
```

##### HashiCorp Vault (AppRole)

```hcl
resource "portkey_secret_reference" "vault_approle" {
  name         = "Vault AppRole"
  manager_type = "hashicorp_vault"
  secret_path  = "secret/data/openai"

  vault_approle_auth = {
    vault_addr      = "https://vault.example.com"
    vault_role_id   = var.vault_role_id
    vault_secret_id = var.vault_secret_id
    vault_namespace = "admin" # Optional (Vault Enterprise)
  }

  allow_all_workspaces = false
  allowed_workspaces = [
    portkey_workspace.production.id,
  ]
}
```

##### Azure Key Vault (Entra Service Principal)

```hcl
resource "portkey_secret_reference" "azure_entra" {
  name         = "Azure Key Vault (Entra)"
  manager_type = "azure_kv"
  secret_path  = "openai-api-key"

  azure_entra_auth = {
    azure_vault_url           = "https://my-vault.vault.azure.net/"
    azure_entra_tenant_id     = var.azure_tenant_id
    azure_entra_client_id     = var.azure_client_id
    azure_entra_client_secret = var.azure_client_secret
  }
}
```

See `examples/secret_reference/` for a complete, runnable example covering AWS, Azure, and Vault variants plus data source usage.

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

### Secret Management Data Sources

| Data Source | Description | Key Arguments |
|-------------|-------------|---------------|
| `portkey_secret_reference` | Fetch a single secret reference by slug (auth credentials not exposed) | `slug` |
| `portkey_secret_references` | List secret references (paginated, filterable by `search`, `manager_type`) | - |

> **Note:** Neither data source exposes the `auth_config` block. The API returns credential fields masked, so surfacing them in a data source would only leak placeholder values and invite state drift.

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

#### Adding Usage Limits and Rate Limits

```hcl
resource "portkey_api_key" "controlled" {
  name         = "Budget-Controlled Key"
  type         = "workspace"
  sub_type     = "service"
  workspace_id = portkey_workspace.production.id
  scopes       = ["completions.write", "providers.list"]

  usage_limits = {
    type            = "tokens"      # count tokens (or "cost" for dollar spend)
    credit_limit    = 1000000       # 1M tokens
    alert_threshold = 800000        # email alert at 800K
    periodic_reset  = "monthly"     # auto-reset every month
  }

  rate_limits = [
    {
      type  = "requests"
      unit  = "rpm"    # requests per minute (also: rph, rpd, rps, rpw)
      value = 500
    }
  ]

  alert_emails = ["platform-team@example.com"]
}
```

To clear limits, remove the `usage_limits` or `rate_limits` block and re-apply. The provider sends a `null` to the API which clears the limits.

#### Adding a Rotation Policy

A rotation policy tells Portkey to automatically rotate this key on a schedule. After rotation, both the old and new key remain valid for `key_transition_period_ms` milliseconds so in-flight requests are not disrupted.

```hcl
resource "portkey_api_key" "rotating" {
  name         = "Auto-Rotating Service Key"
  type         = "workspace"
  sub_type     = "service"
  workspace_id = portkey_workspace.production.id
  scopes       = ["completions.write", "providers.list"]

  rotation_policy = {
    rotation_period          = "monthly"
    key_transition_period_ms = 3600000  # 1 hour overlap (minimum is 1800000 = 30 min)
  }
}
```

You can also set a specific `next_rotation_at` datetime to control exactly when the first rotation fires:

```hcl
  rotation_policy = {
    rotation_period          = "weekly"
    next_rotation_at         = "2026-05-01T00:00:00Z"
    key_transition_period_ms = 1800000  # 30 min minimum
  }
```

`next_rotation_at` is computed by the API after the first rotation, so Terraform will read it back automatically on subsequent plans.

#### Combining Expiry with a Rotation Policy

```hcl
resource "portkey_api_key" "short_lived" {
  name         = "Short-Lived Key"
  type         = "workspace"
  sub_type     = "service"
  workspace_id = portkey_workspace.production.id
  scopes       = ["completions.write"]

  # Hard expiry — key is invalidated after this date regardless of rotation
  expires_at = "2026-12-31T23:59:59Z"

  rotation_policy = {
    rotation_period          = "monthly"
    key_transition_period_ms = 3600000
  }
}
```

`expires_at` can be updated after creation — change the value and re-apply without destroying the key. To create a non-expiring key, simply omit the field.

#### Updating an API Key

All mutable fields can be updated in-place with `terraform apply` — no destroy/recreate is required (except for `type`, `sub_type`, `workspace_id`, and `user_id` which force replacement):

```hcl
resource "portkey_api_key" "backend" {
  name        = "Backend Service — updated name"   # update name
  type        = "workspace"
  sub_type    = "service"
  workspace_id = portkey_workspace.production.id
  scopes       = ["completions.write", "providers.list", "logs.view"]  # add scope

  # Bind to a config after the fact
  config_id             = portkey_config.routing.id
  allow_config_override = false   # lock callers to this config

  # Extend the expiry
  expires_at = "2027-06-30T23:59:59Z"

  usage_limits = {
    credit_limit = 2000000   # increase limit
    periodic_reset = "monthly"
  }

  rotation_policy = {
    rotation_period          = "monthly"
    key_transition_period_ms = 3600000
  }
}
```

#### Resetting Usage Counters

To immediately reset a key's usage counters without waiting for the next scheduled reset, set `reset_usage = true` and apply:

```hcl
resource "portkey_api_key" "backend" {
  name         = "Backend Service"
  type         = "workspace"
  sub_type     = "service"
  workspace_id = portkey_workspace.production.id
  scopes       = ["completions.write"]

  usage_limits = {
    credit_limit   = 1000000
    periodic_reset = "monthly"
  }

  reset_usage = true   # triggers immediate counter reset on next apply
}
```

`reset_usage` is a **write-only trigger** — it is never stored in Terraform state. After the reset is performed, state records `null` for this field. If you leave `reset_usage = true` in your config, the next `terraform plan` will show it as a pending change and every subsequent `apply` will reset the counters again. Remove the line (or set `reset_usage = false`) once the reset is done.

After a reset, `last_reset_at` (a read-only attribute) is updated by the API and will be visible in your state.

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
