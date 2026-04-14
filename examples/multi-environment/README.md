# Multi-Environment Portkey Setup Example

This example demonstrates a complete Portkey organization setup with multiple environments, teams, and access controls.

## Directory Structure

```
.
├── main.tf
├── variables.tf
├── outputs.tf
├── environments/
│   ├── production.tfvars
│   ├── staging.tfvars
│   └── development.tfvars
└── modules/
    └── team-workspace/
        ├── main.tf
        ├── variables.tf
        └── outputs.tf
```

## main.tf

```hcl
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
      role = "member"  # Read-only in production
    },
    {
      id   = portkey_workspace.staging_api.id
      role = "admin"   # Full access in staging
    },
    {
      id   = portkey_workspace.dev_shared.id
      role = "admin"   # Full access in dev
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
```

## variables.tf

```hcl
variable "portkey_api_key" {
  description = "Portkey Admin API Key"
  type        = string
  sensitive   = true
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"

  validation {
    condition     = contains(["production", "staging", "development"], var.environment)
    error_message = "Environment must be production, staging, or development."
  }
}

variable "team" {
  description = "Team name"
  type        = string
  default     = "platform"
}

variable "engineering_lead_email" {
  description = "Email for engineering team lead"
  type        = string
}

variable "ml_lead_email" {
  description = "Email for ML team lead"
  type        = string
}

variable "analyst_email" {
  description = "Email for analytics team member"
  type        = string
}

variable "backend_engineer_email" {
  description = "Email for backend engineer"
  type        = string
}
```

## outputs.tf

```hcl
# ============================================================================
# WORKSPACE OUTPUTS
# ============================================================================

output "production_workspaces" {
  description = "Production workspace IDs"
  value = {
    api       = portkey_workspace.prod_api.id
    ml        = portkey_workspace.prod_ml.id
    analytics = portkey_workspace.prod_analytics.id
  }
}

output "staging_workspaces" {
  description = "Staging workspace IDs"
  value = {
    api = portkey_workspace.staging_api.id
    ml  = portkey_workspace.staging_ml.id
  }
}

output "development_workspace" {
  description = "Development workspace ID"
  value       = portkey_workspace.dev_shared.id
}

# ============================================================================
# USER INVITATION OUTPUTS
# ============================================================================

output "pending_invitations" {
  description = "Status of all user invitations"
  value = {
    engineering_lead  = portkey_user_invite.eng_lead.status
    ml_lead          = portkey_user_invite.ml_lead.status
    analyst          = portkey_user_invite.analyst.status
    backend_engineer = portkey_user_invite.backend_engineer.status
  }
}

output "invitation_details" {
  description = "Details of all invitations"
  value = {
    total_invitations = 4
    invitations = [
      {
        email  = portkey_user_invite.eng_lead.email
        role   = portkey_user_invite.eng_lead.role
        status = portkey_user_invite.eng_lead.status
      },
      {
        email  = portkey_user_invite.ml_lead.email
        role   = portkey_user_invite.ml_lead.role
        status = portkey_user_invite.ml_lead.status
      },
      {
        email  = portkey_user_invite.analyst.email
        role   = portkey_user_invite.analyst.role
        status = portkey_user_invite.analyst.status
      },
      {
        email  = portkey_user_invite.backend_engineer.email
        role   = portkey_user_invite.backend_engineer.role
        status = portkey_user_invite.backend_engineer.status
      }
    ]
  }
  sensitive = true
}

# ============================================================================
# ORGANIZATION SUMMARY
# ============================================================================

output "organization_summary" {
  description = "Summary of Portkey organization structure"
  value = {
    total_workspaces = length(data.portkey_workspaces.all.workspaces)
    workspace_breakdown = {
      production  = 3
      staging     = 2
      development = 1
    }
    environment_access = local.workspace_map
  }
}

output "workspace_urls" {
  description = "Portkey dashboard URLs for each workspace"
  value = {
    prod_api       = "https://app.portkey.ai/workspaces/${portkey_workspace.prod_api.id}"
    prod_ml        = "https://app.portkey.ai/workspaces/${portkey_workspace.prod_ml.id}"
    prod_analytics = "https://app.portkey.ai/workspaces/${portkey_workspace.prod_analytics.id}"
    staging_api    = "https://app.portkey.ai/workspaces/${portkey_workspace.staging_api.id}"
    staging_ml     = "https://app.portkey.ai/workspaces/${portkey_workspace.staging_ml.id}"
    dev_shared     = "https://app.portkey.ai/workspaces/${portkey_workspace.dev_shared.id}"
  }
}
```

## environments/production.tfvars

```hcl
environment = "production"
team        = "platform"

engineering_lead_email  = "eng-lead@company.com"
ml_lead_email          = "ml-lead@company.com"
analyst_email          = "analyst@company.com"
backend_engineer_email = "backend@company.com"
```

## Usage

```bash
# Initialize Terraform
terraform init

# Plan for production
terraform plan -var-file="environments/production.tfvars" -var="portkey_api_key=$PORTKEY_API_KEY"

# Apply configuration
terraform apply -var-file="environments/production.tfvars" -var="portkey_api_key=$PORTKEY_API_KEY"

# View outputs
terraform output

# Get specific output
terraform output production_workspaces
```

## Key Features Demonstrated

1. **Multi-Environment Setup**: Separate workspaces for production, staging, and development
2. **Team-Based Access Control**: Different permissions for different roles
3. **Granular Scope Management**: Specific API scopes per user
4. **Role Hierarchy**: From admin to read-only access
5. **Comprehensive Outputs**: Easy access to workspace IDs and invitation status
6. **Environment Variables**: Secure API key management
7. **Data Sources**: Query existing resources for validation

## Best Practices

1. **Separate Workspaces by Environment**: Keep prod, staging, and dev isolated
2. **Principle of Least Privilege**: Grant minimum required scopes
3. **Team-Based Access**: Organize access by team and role
4. **Read-Only Production**: Give engineers read access to prod, write to staging
5. **Shared Development**: Single dev workspace for collaboration
6. **Output Documentation**: Clear outputs for team reference

