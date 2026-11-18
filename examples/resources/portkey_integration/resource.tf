terraform {
  required_providers {
    portkey = {
      source = "portkey-ai/portkey"
    }
  }
}

# ----------------------------------------------------------------------------
# Classic integration: provider API key stored directly in Terraform state.
# ----------------------------------------------------------------------------
resource "portkey_integration" "openai" {
  name           = "openai-production"
  ai_provider_id = "openai"
  key            = var.openai_api_key
  description    = "Production OpenAI integration"
}

# ----------------------------------------------------------------------------
# Secret-reference-backed integration.
#
# Instead of sending the provider key inline, resolve it at request time from
# a portkey_secret_reference that points at AWS Secrets Manager / Azure Key
# Vault / HashiCorp Vault. Nothing sensitive ever lives in Terraform state.
# ----------------------------------------------------------------------------
resource "portkey_secret_reference" "openai_key" {
  name         = "openai-prod-key"
  manager_type = "aws_sm"
  secret_path  = "portkey/openai/prod"
  secret_key   = "OPENAI_API_KEY"

  aws_access_key_auth = {
    aws_access_key_id     = var.aws_access_key_id
    aws_secret_access_key = var.aws_secret_access_key
    aws_region            = "us-east-1"
  }
}

resource "portkey_integration" "openai_from_secret_ref" {
  name           = "openai-production-sr"
  ai_provider_id = "openai"
  # `key` / `key_wo` intentionally omitted - the mapping below supplies it.

  secret_mappings = [
    {
      target_field        = "key"
      secret_reference_id = portkey_secret_reference.openai_key.slug
    },
  ]
}

# ----------------------------------------------------------------------------
# Composite example: some configuration fields inline, a sensitive field
# resolved from a secret reference. Example target is AWS Bedrock, where
# aws_secret_access_key is the sensitive field we want to keep out of state.
# ----------------------------------------------------------------------------
resource "portkey_integration" "bedrock_hybrid" {
  name           = "bedrock-production"
  ai_provider_id = "bedrock"

  configurations = jsonencode({
    aws_auth_type     = "accessKey"
    aws_region        = "us-east-1"
    aws_access_key_id = var.aws_access_key_id
    # aws_secret_access_key is supplied by the mapping below.
  })

  secret_mappings = [
    {
      target_field        = "configurations.aws_secret_access_key"
      secret_reference_id = portkey_secret_reference.openai_key.slug
      secret_key          = "AWS_SECRET_ACCESS_KEY" # override for multi-value secrets
    },
  ]
}

variable "openai_api_key" {
  type      = string
  sensitive = true
}

variable "aws_access_key_id" {
  type      = string
  sensitive = true
}

variable "aws_secret_access_key" {
  type      = string
  sensitive = true
}

# ----------------------------------------------------------------------------
# Google Vertex AI — Service Account JSON key
# ----------------------------------------------------------------------------
resource "portkey_integration" "vertex_service_account" {
  name           = "vertex-production-sa"
  ai_provider_id = "vertex-ai"

  configurations = jsonencode({
    vertex_auth_type             = "serviceAccount"
    vertex_region                = "us-central1"
    vertex_service_account_json  = jsondecode(var.vertex_service_account_json)
  })
}

# ----------------------------------------------------------------------------
# Google Vertex AI — Workload Identity Federation / Ambient GCP credentials
#
# Use this when the gateway pod already has a GCP identity attached (e.g. GKE
# Workload Identity, Cloud Run service account). No key is stored; the gateway
# fetches a token from the GCP metadata server at request time.
# vertex_auth_type = "workload" is the correct value — not "basic".
# ----------------------------------------------------------------------------
resource "portkey_integration" "vertex_workload_identity" {
  name           = "vertex-production-wif"
  ai_provider_id = "vertex-ai"
  # No `key` needed — authentication comes from the ambient GCP identity.

  configurations = jsonencode({
    vertex_auth_type = "workload"
    vertex_project_id = var.vertex_project_id
    vertex_region     = "us-central1"
  })
}

# ----------------------------------------------------------------------------
# Google Vertex AI — Basic (short-lived OAuth access token passed as api_key)
#
# Use this when you generate a GCP access token externally (e.g. via
# `gcloud auth print-access-token`) and pass it as the integration key.
# This is NOT Workload Identity Federation.
# ----------------------------------------------------------------------------
resource "portkey_integration" "vertex_basic" {
  name           = "vertex-production-basic"
  ai_provider_id = "vertex-ai"
  key            = var.vertex_access_token   # short-lived GCP OAuth2 access token

  configurations = jsonencode({
    vertex_auth_type  = "basic"
    vertex_project_id = var.vertex_project_id
    vertex_region     = "us-central1"
  })
}

variable "vertex_service_account_json" {
  description = "GCP service account key JSON (full file contents as a string)"
  type        = string
  sensitive   = true
}

variable "vertex_project_id" {
  description = "GCP project ID for Vertex AI"
  type        = string
}

variable "vertex_access_token" {
  description = "Short-lived GCP OAuth2 access token (for basic auth only)"
  type        = string
  sensitive   = true
}
