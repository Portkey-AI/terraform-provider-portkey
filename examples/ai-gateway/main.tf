terraform {
  required_version = ">= 1.0"

  required_providers {
    portkey = {
      source  = "portkey-ai/portkey"
      version = "~> 0.1"
    }
  }
}

provider "portkey" {
  # API key read from PORTKEY_API_KEY environment variable
}

variable "openai_api_key" {
  description = "OpenAI API key"
  type        = string
  sensitive   = true
}

# Create a workspace for production with budget controls
resource "portkey_workspace" "production" {
  name        = "Production"
  description = "Production AI Gateway environment"

  usage_limits {
    type            = "cost"
    credit_limit    = 1000
    alert_threshold = 800
    periodic_reset  = "monthly"
  }
}

# Create an integration for OpenAI
resource "portkey_integration" "openai" {
  name           = "OpenAI Production"
  ai_provider_id = "openai"
  key            = var.openai_api_key
  description    = "OpenAI API integration for production workloads"
}

# Create a provider (virtual key) linked to the integration
resource "portkey_provider" "openai_prod" {
  name           = "OpenAI Production Key"
  workspace_id   = portkey_workspace.production.id
  integration_id = portkey_integration.openai.id
  note           = "Main OpenAI key for production"
}

# Create a gateway config with retry and caching
resource "portkey_config" "production" {
  name         = "Production Gateway Config"
  workspace_id = portkey_workspace.production.id

  config = jsonencode({
    retry = {
      attempts        = 3
      on_status_codes = [429, 500, 502, 503]
    }
    cache = {
      mode = "simple"
    }
  })
}

# Create a guardrail for content moderation
resource "portkey_guardrail" "content_filter" {
  name         = "Production Content Filter"
  workspace_id = portkey_workspace.production.id

  # checks and actions must be JSON-encoded
  checks = jsonencode([
    {
      id = "default.wordCount"
      parameters = {
        minWords = 1
        maxWords = 10000
      }
    }
  ])

  actions = jsonencode({
    onFail  = "block"
    message = "Content validation failed. Please check your input."
  })
}

# Create a usage limits policy
resource "portkey_usage_limits_policy" "monthly_budget" {
  name            = "Monthly Cost Limit"
  workspace_id    = portkey_workspace.production.id
  type            = "cost"
  credit_limit    = 1000.0
  alert_threshold = 800.0
  periodic_reset  = "monthly"

  # conditions and group_by must be JSON-encoded arrays
  conditions = jsonencode([
    { key = "workspace_id", value = portkey_workspace.production.id }
  ])
  group_by = jsonencode([
    { key = "api_key" }
  ])
}

# Create a rate limits policy
resource "portkey_rate_limits_policy" "api_throttle" {
  name         = "API Request Throttle"
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

# Create a Portkey API key for your backend application with budget controls
resource "portkey_api_key" "backend_service" {
  name         = "Backend Service Key"
  type         = "workspace"
  sub_type     = "service"
  workspace_id = portkey_workspace.production.id

  scopes = [
    "logs.list",
    "logs.view",
    "configs.read",
    "configs.list",
    "providers.list",
    "providers.read"
  ]

  usage_limits = {
    credit_limit    = 500
    alert_threshold = 400
    periodic_reset  = "monthly"
  }

  alert_emails = ["platform-team@example.com"]
}

# Outputs
output "workspace_id" {
  description = "Production workspace ID"
  value       = portkey_workspace.production.id
}

output "integration_slug" {
  description = "OpenAI integration slug"
  value       = portkey_integration.openai.slug
}

output "provider_id" {
  description = "OpenAI provider (virtual key) ID"
  value       = portkey_provider.openai_prod.id
}

output "config_slug" {
  description = "Gateway config slug"
  value       = portkey_config.production.slug
}

output "guardrail_slug" {
  description = "Guardrail slug"
  value       = portkey_guardrail.content_filter.slug
}

output "api_key_id" {
  description = "Backend service API key ID"
  value       = portkey_api_key.backend_service.id
}

