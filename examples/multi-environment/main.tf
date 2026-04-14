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
  api_key = var.portkey_api_key
}

# ============================================================================
# WORKSPACES - Organized by Environment and Team
# ============================================================================

# Production Workspaces
resource "portkey_workspace" "prod_api" {
  name        = "Production - API"
  description = "Production API services with LLM integrations"
}

resource "portkey_workspace" "prod_ml" {
  name        = "Production - ML"
  description = "Production machine learning workloads"
}

resource "portkey_workspace" "prod_analytics" {
  name        = "Production - Analytics"
  description = "Production analytics and reporting"
}

# Staging Workspaces
resource "portkey_workspace" "staging_api" {
  name        = "Staging - API"
  description = "Staging environment for API testing"
}

resource "portkey_workspace" "staging_ml" {
  name        = "Staging - ML"
  description = "Staging ML model testing"
}

# Development Workspace
resource "portkey_workspace" "dev_shared" {
  name        = "Development - Shared"
  description = "Shared development environment for all teams"
}

# ============================================================================
# USER INVITATIONS - Team Members
# ============================================================================

# Engineering Team Lead
resource "portkey_user_invite" "eng_lead" {
  email = var.engineering_lead_email
  role  = "admin"

  workspaces = [
    {
      id   = portkey_workspace.prod_api.id
      role = "admin"
    },
    {
      id   = portkey_workspace.staging_api.id
      role = "admin"
    },
    {
      id   = portkey_workspace.dev_shared.id
      role = "admin"
    }
  ]

  scopes = [
    "logs.export",
    "logs.list",
    "logs.view",
    "configs.create",
    "configs.update",
    "configs.delete",
    "configs.read",
    "configs.list",
    "virtual_keys.create",
    "virtual_keys.update",
    "virtual_keys.delete",
    "virtual_keys.read",
    "virtual_keys.list",
    "virtual_keys.copy"
  ]
}

# ML Team Lead
resource "portkey_user_invite" "ml_lead" {
  email = var.ml_lead_email
  role  = "member"

  workspaces = [
    {
      id   = portkey_workspace.prod_ml.id
      role = "admin"
    },
    {
      id   = portkey_workspace.staging_ml.id
      role = "admin"
    },
    {
      id   = portkey_workspace.dev_shared.id
      role = "manager"
    }
  ]

  scopes = [
    "logs.export",
    "logs.list",
    "logs.view",
    "configs.create",
    "configs.update",
    "configs.read",
    "configs.list",
    "virtual_keys.create",
    "virtual_keys.update",
    "virtual_keys.read",
    "virtual_keys.list"
  ]
}

# Analytics Team Member
resource "portkey_user_invite" "analyst" {
  email = var.analyst_email
  role  = "member"

  workspaces = [
    {
      id   = portkey_workspace.prod_analytics.id
      role = "admin"
    },
    {
      id   = portkey_workspace.staging_api.id
      role = "member"
    }
  ]

  scopes = [
    "logs.export",
    "logs.list",
    "logs.view",
    "configs.read",
    "configs.list"
  ]
}

# Backend Engineer (Read-Only Production Access)
resource "portkey_user_invite" "backend_engineer" {
  email = var.backend_engineer_email
  role  = "member"

  workspaces = [
    {
      id   = portkey_workspace.prod_api.id
      role = "member" # Read-only in production
    },
    {
      id   = portkey_workspace.staging_api.id
      role = "admin" # Full access in staging
    },
    {
      id   = portkey_workspace.dev_shared.id
      role = "admin" # Full access in dev
    }
  ]

  scopes = [
    "logs.list",
    "logs.view",
    "configs.read",
    "configs.list",
    "configs.create",
    "configs.update",
    "virtual_keys.read",
    "virtual_keys.list",
    "virtual_keys.create"
  ]
}

# ============================================================================
# DATA SOURCES - Query Existing Resources
# ============================================================================

data "portkey_workspaces" "all" {
  depends_on = [
    portkey_workspace.prod_api,
    portkey_workspace.prod_ml,
    portkey_workspace.prod_analytics,
    portkey_workspace.staging_api,
    portkey_workspace.staging_ml,
    portkey_workspace.dev_shared
  ]
}

data "portkey_users" "all" {
  depends_on = [
    portkey_user_invite.eng_lead,
    portkey_user_invite.ml_lead,
    portkey_user_invite.analyst,
    portkey_user_invite.backend_engineer
  ]
}

# Look up a specific user by email (avoids iterating over all users)
data "portkey_users" "eng_lead_lookup" {
  email = var.engineering_lead_email
  depends_on = [portkey_user_invite.eng_lead]
}

# ============================================================================
# LOCAL VALUES - For Organization
# ============================================================================

locals {
  # Map of environment to workspace IDs
  workspace_map = {
    production = {
      api       = portkey_workspace.prod_api.id
      ml        = portkey_workspace.prod_ml.id
      analytics = portkey_workspace.prod_analytics.id
    }
    staging = {
      api = portkey_workspace.staging_api.id
      ml  = portkey_workspace.staging_ml.id
    }
    development = {
      shared = portkey_workspace.dev_shared.id
    }
  }

  # Team access summary
  team_structure = {
    engineering = [
      portkey_workspace.prod_api.id,
      portkey_workspace.staging_api.id,
      portkey_workspace.dev_shared.id
    ]
    ml = [
      portkey_workspace.prod_ml.id,
      portkey_workspace.staging_ml.id,
      portkey_workspace.dev_shared.id
    ]
    analytics = [
      portkey_workspace.prod_analytics.id
    ]
  }

  # Common tags/metadata
  common_labels = {
    managed_by  = "terraform"
    environment = var.environment
    team        = var.team
  }
}

