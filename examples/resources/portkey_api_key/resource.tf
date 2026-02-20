# Basic workspace API key
resource "portkey_api_key" "basic" {
  name         = "My Service Key"
  type         = "workspace"
  sub_type     = "service"
  workspace_id = portkey_workspace.dev.id

  scopes = [
    "completions.write",
    "providers.read",
    "providers.list",
  ]
}

# API key with usage limits and rate limits
resource "portkey_api_key" "with_limits" {
  name         = "Budget-Controlled Key"
  type         = "workspace"
  sub_type     = "service"
  workspace_id = portkey_workspace.dev.id

  scopes = ["completions.write", "providers.list"]

  usage_limits = {
    credit_limit    = 500
    alert_threshold = 400
    periodic_reset  = "monthly"
  }

  rate_limits = [{
    type  = "requests"
    unit  = "rpm"
    value = 1000
  }]
}

# To clear limits, simply remove the usage_limits or rate_limits blocks
# from your config and re-apply.
