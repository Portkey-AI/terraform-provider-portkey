# Basic integration workspace access
resource "portkey_integration_workspace_access" "basic" {
  integration_id = portkey_integration.openai.slug
  workspace_id   = portkey_workspace.dev.id
}

# With usage and rate limits
resource "portkey_integration_workspace_access" "with_limits" {
  integration_id = portkey_integration.openai.slug
  workspace_id   = portkey_workspace.dev.id
  enabled        = true

  usage_limits = [{
    type            = "cost"
    credit_limit    = 100
    alert_threshold = 80
    periodic_reset  = "monthly"
  }]

  rate_limits = [{
    type  = "requests"
    unit  = "rpm"
    value = 1000
  }]
}

# To clear limits, simply remove the usage_limits or rate_limits blocks
# from your config and re-apply.
