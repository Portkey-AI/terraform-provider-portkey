# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.6...HEAD
[0.2.6]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.5...v0.2.6
[0.2.5]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.4...v0.2.5
[0.2.4]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/Portkey-AI/terraform-provider-portkey/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Portkey-AI/terraform-provider-portkey/releases/tag/v0.1.0

