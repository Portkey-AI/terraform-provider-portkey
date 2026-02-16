terraform {
  required_providers {
    portkey = {
      source = "portkey-ai/portkey"
    }
  }
}

provider "portkey" {}

# Look up existing workspace
data "portkey_workspace" "dev" {
  id = var.workspace_id
}

# ============================================================================
# MCP Integration (org-level) - Register an MCP server
# ============================================================================

resource "portkey_mcp_integration" "github" {
  name        = "GitHub MCP Server"
  description = "GitHub tools via MCP"
  url         = "https://mcp.github.com/sse"
  auth_type   = "bearer"
  transport   = "sse"

  # Optional: JSON string of auth configuration (sensitive)
  # configurations = jsonencode({
  #   token = var.github_token
  # })
}

# ============================================================================
# MCP Integration Workspace Access - Grant workspace access to the integration
# ============================================================================

resource "portkey_mcp_integration_workspace_access" "github_dev" {
  mcp_integration_id = portkey_mcp_integration.github.id
  workspace_id       = data.portkey_workspace.dev.id
  enabled            = true
}

# ============================================================================
# MCP Server (workspace-level) - Provision the integration to a workspace
# ============================================================================

resource "portkey_mcp_server" "github_dev" {
  name               = "GitHub Dev"
  description        = "GitHub MCP for dev workspace"
  mcp_integration_id = portkey_mcp_integration.github.id
  workspace_id       = data.portkey_workspace.dev.id

  depends_on = [portkey_mcp_integration_workspace_access.github_dev]
}

# ============================================================================
# MCP Integration Capabilities - Control which tools are available org-wide
# ============================================================================

resource "portkey_mcp_integration_capabilities" "github" {
  mcp_integration_id = portkey_mcp_integration.github.id

  capabilities {
    name    = "create_pull_request"
    type    = "tool"
    enabled = true
  }

  capabilities {
    name    = "delete_repository"
    type    = "tool"
    enabled = false # Disable dangerous tool at org level
  }
}

# ============================================================================
# MCP Server Capabilities - Further restrict at workspace level
# ============================================================================

resource "portkey_mcp_server_capabilities" "github_dev" {
  mcp_server_id = portkey_mcp_server.github_dev.id

  capabilities {
    name    = "create_pull_request"
    type    = "tool"
    enabled = true
  }
}

# ============================================================================
# MCP Server User Access - Control per-user access
# ============================================================================

# resource "portkey_mcp_server_user_access" "github_dev_admin" {
#   mcp_server_id = portkey_mcp_server.github_dev.id
#   user_id       = var.admin_user_id
#   enabled       = true
# }

# ============================================================================
# Data Sources - Read MCP resources
# ============================================================================

data "portkey_mcp_integrations" "all" {
  depends_on = [portkey_mcp_integration.github]
}

data "portkey_mcp_servers" "dev" {
  workspace_id = data.portkey_workspace.dev.id
  depends_on   = [portkey_mcp_server.github_dev]
}

# ============================================================================
# Variables
# ============================================================================

variable "workspace_id" {
  description = "Workspace ID for MCP server provisioning"
  type        = string
}

# ============================================================================
# Outputs
# ============================================================================

output "mcp_integration_id" {
  value = portkey_mcp_integration.github.id
}

output "mcp_server_id" {
  value = portkey_mcp_server.github_dev.id
}

output "mcp_integrations_count" {
  value = length(data.portkey_mcp_integrations.all.integrations)
}

output "mcp_servers_count" {
  value = length(data.portkey_mcp_servers.dev.servers)
}
