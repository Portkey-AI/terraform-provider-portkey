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
  auth_type   = "headers"
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
# MCP Integration Capabilities - Control which tools are available org-wide
# ============================================================================

resource "portkey_mcp_integration_capabilities" "github" {
  mcp_integration_id = portkey_mcp_integration.github.id

  capabilities = [
    {
      name    = "create_pull_request"
      type    = "tool"
      enabled = true
    },
    {
      name    = "delete_repository"
      type    = "tool"
      enabled = false # Disable dangerous tool at org level
    },
  ]
}

# ============================================================================
# Data Sources - Read MCP resources
# ============================================================================

data "portkey_mcp_integrations" "all" {
  depends_on = [portkey_mcp_integration.github]
}

# ============================================================================
# Variables
# ============================================================================

variable "workspace_id" {
  description = "Workspace ID for MCP integration provisioning"
  type        = string
}

# ============================================================================
# Outputs
# ============================================================================

output "mcp_integration_id" {
  value = portkey_mcp_integration.github.id
}

output "mcp_integrations_count" {
  value = length(data.portkey_mcp_integrations.all.integrations)
}
