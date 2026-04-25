# AWS Secrets Manager via static access keys
resource "portkey_secret_reference" "aws_access_key" {
  name         = "AWS Prod Secrets"
  description  = "Prod OpenAI key stored in AWS Secrets Manager"
  manager_type = "aws_sm"
  secret_path  = "prod/openai/api-key"

  aws_access_key_auth = {
    aws_access_key_id     = var.aws_access_key_id
    aws_secret_access_key = var.aws_secret_access_key
    aws_region            = "us-east-1"
  }

  allow_all_workspaces = true

  tags = {
    env   = "prod"
    owner = "platform"
  }
}

# AWS Secrets Manager via AssumeRole
resource "portkey_secret_reference" "aws_assumed_role" {
  name         = "AWS Assumed Role"
  manager_type = "aws_sm"
  secret_path  = "prod/openai/api-key"

  aws_assumed_role_auth = {
    aws_role_arn    = "arn:aws:iam::123456789012:role/PortkeySecretsRole"
    aws_region      = "us-east-1"
    aws_external_id = "optional-external-id"
  }

  allowed_workspaces = [
    portkey_workspace.production.id,
  ]
}

# Write-only credentials with rotation trigger. Bump auth_version to rotate.
resource "portkey_secret_reference" "aws_wo" {
  name         = "AWS Rotatable Secrets"
  manager_type = "aws_sm"
  secret_path  = "prod/openai/api-key"
  auth_version = 1

  aws_access_key_auth = {
    aws_access_key_id_wo     = var.aws_access_key_id
    aws_secret_access_key_wo = var.aws_secret_access_key
    aws_region               = "us-east-1"
  }
}

# HashiCorp Vault via AppRole
resource "portkey_secret_reference" "vault_approle" {
  name         = "Vault AppRole"
  manager_type = "hashicorp_vault"
  secret_path  = "secret/data/openai"

  vault_approle_auth = {
    vault_addr      = "https://vault.example.com"
    vault_role_id   = var.vault_role_id
    vault_secret_id = var.vault_secret_id
    vault_namespace = "admin"
  }
}

# Azure Key Vault via Entra service principal
resource "portkey_secret_reference" "azure_entra" {
  name         = "Azure Key Vault (Entra)"
  manager_type = "azure_kv"
  secret_path  = "openai-api-key"

  azure_entra_auth = {
    azure_vault_url           = "https://my-vault.vault.azure.net"
    azure_entra_tenant_id     = var.azure_tenant_id
    azure_entra_client_id     = var.azure_client_id
    azure_entra_client_secret = var.azure_client_secret
  }
}
