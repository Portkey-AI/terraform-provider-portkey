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
  # Alternatively, set it explicitly:
  # api_key = var.portkey_api_key
}

# Create a workspace
resource "portkey_workspace" "engineering" {
  name        = "Engineering Team"
  description = "Workspace for the engineering team"
}

# Create another workspace
resource "portkey_workspace" "ml_research" {
  name        = "ML Research"
  description = "Machine Learning research and experimentation"
}

# Invite a user with workspace access
resource "portkey_user_invite" "data_scientist" {
  email = "scientist@example.com"
  role  = "member"

  workspaces = [
    {
      id   = portkey_workspace.ml_research.id
      role = "admin"
    },
    {
      id   = portkey_workspace.engineering.id
      role = "member"
    }
  ]

  scopes = [
    "logs.export",
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

# Look up an existing user by email and add them to a workspace
data "portkey_users" "senior_engineer" {
  email = "senior-engineer@example.com"
}

resource "portkey_workspace_member" "senior_engineer" {
  workspace_id = portkey_workspace.engineering.id
  user_id      = data.portkey_users.senior_engineer.users[0].id
  role         = "manager"
}

# Query all workspaces
data "portkey_workspaces" "all" {
  depends_on = [
    portkey_workspace.engineering,
    portkey_workspace.ml_research
  ]
}

# Query all users
data "portkey_users" "all" {
  depends_on = [
    portkey_user_invite.data_scientist
  ]
}

# Output workspace information
output "engineering_workspace_id" {
  description = "The ID of the engineering workspace"
  value       = portkey_workspace.engineering.id
}

output "ml_workspace_id" {
  description = "The ID of the ML research workspace"
  value       = portkey_workspace.ml_research.id
}

output "workspace_count" {
  description = "Total number of workspaces"
  value       = length(data.portkey_workspaces.all.workspaces)
}

output "invitation_status" {
  description = "Status of the data scientist invitation"
  value       = portkey_user_invite.data_scientist.status
}

