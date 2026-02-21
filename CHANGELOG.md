# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.13] - 2026-02-22

### Fixed
- **Prompt & Prompt Partial Drift Detection** - Fixed issues where external changes made in the Portkey console were invisible to Terraform:
  - Now detects when someone edits a prompt or partial outside Terraform (new versions or rollbacks)
  - Terraform will show the drift and overwrite back to config values on next apply
- **Version Description Perpetual Drift** - Fixed infinite plan loop when `version_description` was set via console but not in Terraform config
- **MakeDefault Version Lookup** - Fixed incorrect version targeting when versions are created outside Terraform (e.g., console edits creating gaps in version sequence). Now uses version list API to find the correct version number instead of assuming `+1` increment

## [0.2.12] - 2026-02-20

### Added
- **Prompt Partials** - New resource and data sources for reusable template fragments:
  - `portkey_prompt_partial` - Create and manage prompt partials with versioning support
  - `portkey_prompt_partial` (data source) - Look up a single partial by slug with optional version
  - `portkey_prompt_partials` (data source) - List all partials, optionally filtered by workspace
  - Partials can be referenced in prompts via Mustache syntax (`{{>partial-slug}}`)
- **Usage & Rate Limits** - Added budget controls and rate limiting to workspace and API key resources:
  - `portkey_workspace` - New `usage_limits` and `rate_limits` attributes for workspace-level controls
  - `portkey_api_key` - New `usage_limits` and `rate_limits` attributes for key-level controls
  - All related data sources now return limit configurations

### Fixed
- **Prompt Resource Eventual Consistency** - Fixed issues where Portkey API returns stale data after mutations:
  - Template, version, and version_id are now preserved from state during Read
  - Added `virtual_key` to all version-creating prompt updates (API requirement)
  - Fixed Parameters field handling for Optional+Computed attributes
- **Version Description Warning** - Added diagnostic warning when `version_description` changes without content (no-op against API)

### Changed
- Refactored shared limit conversion helpers into `limits_helpers.go` to eliminate duplication

## [0.2.11] - 2026-02-04

### Added
- **Prompt Collections** - New resource and data sources for organizing prompts within workspaces:
  - `portkey_prompt_collection` - Create and manage prompt collections with hierarchical nesting support
  - `portkey_prompt_collection` (data source) - Look up a single collection by ID
  - `portkey_prompt_collections` (data source) - List all collections, optionally filtered by workspace
  - Support for nested collections via `parent_collection_id`

### Documentation
- Added documentation and examples for prompt collection resource and data sources
- Updated RESOURCE_MATRIX.md with prompt collection support

## [0.2.10] - 2026-01-28

### Fixed
- **Azure OpenAI Integration Configuration** - Fixed incorrect field names in documentation that caused 400 "Invalid request" errors:
  - Changed `resource_name` to `azure_resource_name`
  - Changed flat `deployment_id`/`api_version` to nested `azure_deployment_config` array structure
  - Added required `azure_auth_mode` field (values: "default", "entra", "managed")
  - Added required `azure_model_slug` field in deployment config

### Added
- **Azure OpenAI Authentication Modes** - Full documentation and examples for all 3 authentication methods:
  - `default` - API key authentication
  - `entra` - Microsoft Entra ID (Azure AD) authentication
  - `managed` - Azure Managed Identity authentication
- **Azure OpenAI Acceptance Tests** - 5 new tests to prevent future regressions:
  - Basic configuration with default auth
  - Multiple deployments
  - Entra ID authentication
  - Managed Identity authentication
  - Configuration updates

## [0.2.9] - 2026-01-27

### Added
- **Integration Model Access** - New resource and data source for managing model access per integration:
  - `portkey_integration_model_access` - Enable/disable specific models for an integration with optional custom pricing
  - `portkey_integration_models` - Data source to list all models available for an integration
  - Support for custom/fine-tuned models (`is_custom`, `is_finetune`, `base_model_slug`)
  - Support for custom pricing configuration (`pricing_config` with `pay_as_you_go` token prices)
  - Built-in models are disabled on delete; custom models are fully removed

### Documentation
- Added documentation for integration model access resource and data source

## [0.2.8] - 2026-01-26

### Added
- **Write-Only API Key Support** - `portkey_integration` now supports write-only API keys using Terraform 1.11+'s `WriteOnly` attribute:
  - `key_wo` - API key that is never stored in Terraform state or shown in plan output
  - `key_version` - Trigger to control when the key is sent to the API (increment to update)
  - Provides enhanced security for teams with strict compliance requirements
  - Original `key` attribute remains available for simpler workflows
- **Integration Workspace Access** - New resource and data source for managing integration access per workspace:
  - `portkey_integration_workspace_access` - Enable integrations for specific workspaces with optional limits
  - `portkey_integration_workspaces` - Data source to list workspace access configurations
  - Support for `usage_limits` (cost/request limits with alerts and periodic reset)
  - Support for `rate_limits` (requests per minute/hour/day)
  - Enables full IaC workflows without manual UI enablement

### Documentation
- Updated `portkey_integration` documentation with write-only key examples
- Added documentation for integration workspace access resource and data source

## [0.2.7] - 2026-01-08

### Added
- **API Key Metadata & Alert Emails** - `portkey_api_key` now supports:
  - `metadata` - Custom metadata (map of strings) attached to the API key for tracking, observability, and service identification. Example: `{"_user": "service-name", "service_uuid": "abc123"}`
  - `alert_emails` - List of email addresses to receive alerts related to the API key's usage
- **Workspace Metadata** - `portkey_workspace` now supports:
  - `metadata` - Custom metadata (map of strings) attached to the workspace for tracking teams, environments, and services

### Documentation
- Updated documentation for `portkey_api_key` resource and data sources
- Updated documentation for `portkey_workspace` resource and data sources

## [0.2.6] - 2026-01-05

### Documentation
- Added Terraform Registry documentation for all resources and data sources
- Documentation auto-generated using `tfplugindocs`

## [0.2.5] - 2026-01-05

### Added
- **AWS Bedrock IAM Role Support** - `portkey_integration` now supports a `configurations` field for provider-specific settings:
  - AWS Bedrock with IAM Role authentication (`aws_role_arn`, `aws_region`, `aws_external_id`)
  - AWS Bedrock with Access Keys (`aws_access_key_id`, `aws_region`)
  - Azure OpenAI configurations (`resource_name`, `deployment_id`, `api_version`)

### Documentation
- Added comprehensive examples for AWS Bedrock and Azure OpenAI integrations
- Updated `portkey_integration` documentation with `configurations` field

## [0.2.4] - 2026-01-05

### Fixed
- Fixed lint errors (gofmt, unused function, errcheck)
- **Critical: Fixed "Provider produced inconsistent result after apply" errors** - Resolved issues where Terraform would report inconsistent results due to state handling

## [0.2.3] - 2026-01-04

### Documentation
- Added known issue for workspace deletion with emoji names in README

## [0.2.2] - 2026-01-04

### Fixed
- **Critical: Resources no longer unnecessarily recreated on every apply** - Fixed a bug where `RequiresReplace` attributes (like `workspace_id`) were being overwritten during `Read()` operations, causing Terraform to detect false changes and trigger destroy/create cycles. Affected resources:
  - `portkey_config`
  - `portkey_guardrail`
  - `portkey_provider`
  - `portkey_prompt`
  - `portkey_integration`
  - `portkey_api_key`
  - `portkey_user_invite`
  - `portkey_rate_limits_policy`
  - `portkey_usage_limits_policy`
- Fixed CI linting issues and code formatting
- Reverted golangci-lint config to v1 format for CI compatibility

## [0.2.1] - 2026-01-03

### Documentation
- Added Prerequisites section to README
- Added Troubleshooting section to README
- Added Known Issues section to README
- Fixed README examples to use `jsonencode()` for JSON fields

### Fixed
- Fixed provider unit tests with correct resource counts
- Added Terraform setup to CI and formatted example files
- Fixed gofmt formatting and removed unused functions

## [0.2.0] - 2026-01-02

### Added
- **AI Gateway Resources:**
  - `portkey_integration` - Manage AI provider integrations (OpenAI, Anthropic, Azure, etc.)
  - `portkey_provider` - Manage providers/virtual keys for workspace-scoped AI access
  - `portkey_config` - Manage gateway configurations with routing and fallbacks
  - `portkey_prompt` - Manage versioned prompt templates
- **Governance Resources:**
  - `portkey_guardrail` - Set up content validation and safety checks
  - `portkey_usage_limits_policy` - Control costs with spending limits
  - `portkey_rate_limits_policy` - Manage request rate limiting
- **Access Control Resources:**
  - `portkey_api_key` - Create and manage Portkey API keys
- **Data Sources for all new resources:**
  - `portkey_integration`, `portkey_integrations`
  - `portkey_provider`, `portkey_providers`
  - `portkey_config`, `portkey_configs`
  - `portkey_prompt`, `portkey_prompts`
  - `portkey_guardrail`, `portkey_guardrails`
  - `portkey_usage_limits_policy`, `portkey_usage_limits_policies`
  - `portkey_rate_limits_policy`, `portkey_rate_limits_policies`
  - `portkey_api_key`, `portkey_api_keys`

### Documentation
- Added guide for adding new APIs to the Terraform provider
- Added Registry and CI badges to README

## [0.1.0] - 2026-01-01

### Added
- Initial release of the Portkey Terraform Provider
- **Organization Resources:**
  - `portkey_workspace` - Manage Portkey workspaces
  - `portkey_workspace_member` - Manage workspace membership
  - `portkey_user_invite` - Send user invitations with workspace access and scopes
- **Data Sources:**
  - `portkey_workspace` - Query single workspace by ID
  - `portkey_workspaces` - List all workspaces in organization
  - `portkey_user` - Query single user by ID
  - `portkey_users` - List all users in organization
- Provider configuration with API key authentication
- Support for environment variable `PORTKEY_API_KEY`
- Import functionality for all resources
- Comprehensive documentation and examples
- Multi-environment setup example

### Supported Operations
- Full CRUD operations for workspaces
- User invitation with granular scope management
- Workspace member role assignment
- Organization and workspace role management

### Known Limitations
- User invitations cannot be updated (must delete and recreate)
- Workspace deletion may be blocked by existing resources
- Prompt template updates create new versions (use makeDefault to promote)

[Unreleased]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.13...HEAD
[0.2.13]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.12...v0.2.13
[0.2.12]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.11...v0.2.12
[0.2.11]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.10...v0.2.11
[0.2.10]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.9...v0.2.10
[0.2.9]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.8...v0.2.9
[0.2.8]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.7...v0.2.8
[0.2.7]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.6...v0.2.7
[0.2.6]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.5...v0.2.6
[0.2.5]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.4...v0.2.5
[0.2.4]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Portkey-AI/terraform-provider-portkey/releases/tag/v0.1.0

