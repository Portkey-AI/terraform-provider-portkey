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
  # api_key read from PORTKEY_API_KEY environment variable
}

variable "aws_access_key_id" {
  type      = string
  sensitive = true
}

variable "aws_secret_access_key" {
  type      = string
  sensitive = true
}

variable "vault_role_id" {
  type    = string
  default = "example-role-id"
}

variable "vault_secret_id" {
  type      = string
  sensitive = true
  default   = "example-secret-id"
}

# --- AWS Secrets Manager via static access keys -----------------------------

resource "portkey_secret_reference" "aws_prod_openai" {
  name         = "prod-openai-key"
  description  = "OpenAI API key for production traffic, backed by AWS Secrets Manager"
  manager_type = "aws_sm"
  secret_path  = "prod/api-keys/openai"
  # secret_key is optional — use it to pick a single key out of a JSON secret payload.
  # secret_key = "OPENAI_API_KEY"

  aws_access_key_auth = {
    aws_access_key_id     = var.aws_access_key_id
    aws_secret_access_key = var.aws_secret_access_key
    aws_region            = "us-east-1"
  }

  tags = {
    environment = "production"
    owner       = "platform"
  }
}

# --- HashiCorp Vault via AppRole --------------------------------------------

resource "portkey_secret_reference" "vault_staging_anthropic" {
  name         = "staging-anthropic-key"
  manager_type = "hashicorp_vault"
  secret_path  = "kv/data/staging/anthropic"
  secret_key   = "api_key"

  vault_approle_auth = {
    vault_addr      = "https://vault.example.internal"
    vault_role_id   = var.vault_role_id
    vault_secret_id = var.vault_secret_id
    vault_namespace = "staging"
  }

  # Restrict which workspaces may read this secret.
  allow_all_workspaces = false
  allowed_workspaces   = ["staging-ml"]
}

# --- Data sources -----------------------------------------------------------

# Fetch one by slug (or UUID) — auth_config is intentionally not exposed.
data "portkey_secret_reference" "by_slug" {
  slug = portkey_secret_reference.aws_prod_openai.slug
}

# List all AWS Secrets Manager references in the org.
data "portkey_secret_references" "aws" {
  manager_type = "aws_sm"
}

output "aws_secret_reference_id" {
  description = "UUID of the AWS-backed secret reference"
  value       = portkey_secret_reference.aws_prod_openai.id
}

output "vault_secret_reference_slug" {
  description = "Slug of the Vault-backed secret reference"
  value       = portkey_secret_reference.vault_staging_anthropic.slug
}

output "all_aws_secret_names" {
  description = "Names of every AWS-backed secret reference in the org"
  value       = [for s in data.portkey_secret_references.aws.secret_references : s.name]
}
