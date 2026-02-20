# Basic workspace
resource "portkey_workspace" "basic" {
  name        = "Development"
  description = "Development workspace"
}

# Workspace with usage limits and rate limits
resource "portkey_workspace" "with_limits" {
  name        = "Production"
  description = "Production workspace with budget controls"

  usage_limits = [{
    type            = "cost"
    credit_limit    = 1000
    alert_threshold = 800
    periodic_reset  = "monthly"
  }]

  rate_limits = [{
    type  = "requests"
    unit  = "rpm"
    value = 5000
  }]
}

# To clear limits, simply remove the usage_limits or rate_limits blocks
# from your config and re-apply.
