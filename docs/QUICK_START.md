# Quick Start Guide - Portkey Terraform Provider

This guide will help you get started with the Portkey Terraform Provider in 5 minutes.

## Prerequisites

1. A Portkey account (sign up at https://portkey.ai)
2. Terraform >= 1.0 installed
3. A Portkey Admin API key

## Step 1: Get Your Admin API Key

1. Log in to Portkey Dashboard
2. Navigate to **Admin Settings** → **API Keys**
3. Click **Create API Key**
4. Select:
   - Type: **Organization**
   - Sub-type: **Service**
   - Scopes: Select all or specific ones you need
5. Copy and save your API key securely

## Step 2: Set Up Your Environment

```bash
# Set your API key as an environment variable
export PORTKEY_API_KEY="your-admin-api-key-here"
```

## Step 3: Create Your First Terraform Configuration

Create a file named `main.tf`:

```hcl
terraform {
  required_providers {
    portkey = {
      source = "portkey-ai/portkey"
    }
  }
}

provider "portkey" {
  # API key is read from PORTKEY_API_KEY environment variable
}

# Create a workspace
resource "portkey_workspace" "my_first_workspace" {
  name        = "My First Workspace"
  description = "Created with Terraform"
}

# Output the workspace ID
output "workspace_id" {
  value = portkey_workspace.my_first_workspace.id
}
```

## Step 4: Initialize and Apply

```bash
# Initialize Terraform (downloads the provider)
terraform init

# Preview changes
terraform plan

# Create the workspace
terraform apply
```

Type `yes` when prompted.

## Step 5: Verify

You should see output similar to:

```
Apply complete! Resources: 1 added, 0 changed, 0 destroyed.

Outputs:

workspace_id = "ws-abc123..."
```

Visit your Portkey dashboard to see the new workspace!

## What's Next?

### Invite a User

Add this to your `main.tf`:

```hcl
resource "portkey_user_invite" "team_member" {
  email = "teammate@company.com"
  role  = "member"
  
  workspaces = [{
    id   = portkey_workspace.my_first_workspace.id
    role = "admin"
  }]
  
  scopes = [
    "logs.view",
    "logs.list",
    "configs.read"
  ]
}
```

Then run:
```bash
terraform apply
```

### Query Existing Resources

Add data sources to query existing workspaces and users:

```hcl
# Get all workspaces
data "portkey_workspaces" "all" {}

# Get all users
data "portkey_users" "all" {}

output "total_workspaces" {
  value = length(data.portkey_workspaces.all.workspaces)
}

output "total_users" {
  value = length(data.portkey_users.all.users)
}
```

### Look Up a User by Email

Use the `email` filter to resolve an email address to a user ID without fetching all users:

```hcl
data "portkey_users" "manager" {
  email = "manager@example.com"
}

output "manager_id" {
  value = data.portkey_users.manager.users[0].id
}
```

### Add a Workspace Member

Look up a user by email and add them to a workspace:

```hcl
data "portkey_users" "member" {
  email = "teammate@example.com"
}

resource "portkey_workspace_member" "existing_user" {
  workspace_id = portkey_workspace.my_first_workspace.id
  user_id      = data.portkey_users.member.users[0].id
  role         = "manager"
}
```

## Common Commands

```bash
# See what would change
terraform plan

# Apply changes
terraform apply

# Show current state
terraform show

# List resources
terraform state list

# Get specific output
terraform output workspace_id

# Destroy everything
terraform destroy
```

## Tips

1. **Version Control**: Add `.terraform/` and `*.tfstate` to `.gitignore`
2. **State Management**: Use remote state for team collaboration
3. **Secrets**: Never commit API keys - use environment variables
4. **Modules**: Organize complex setups using Terraform modules
5. **Workspaces**: Use Terraform workspaces for multiple environments

## Example: Production Setup

For a production-ready setup:

```hcl
terraform {
  required_version = ">= 1.0"
  
  required_providers {
    portkey = {
      source  = "portkey-ai/portkey"
      version = "~> 0.1"
    }
  }
  
  # Use remote state
  backend "s3" {
    bucket = "my-terraform-state"
    key    = "portkey/terraform.tfstate"
    region = "us-east-1"
    encrypt = true
  }
}

provider "portkey" {
  api_key = var.portkey_api_key  # Pass via -var flag
}

variable "portkey_api_key" {
  type      = string
  sensitive = true
}
```

Run with:
```bash
terraform plan -var="portkey_api_key=$PORTKEY_API_KEY"
terraform apply -var="portkey_api_key=$PORTKEY_API_KEY"
```

## Troubleshooting

### "Unable to Create Portkey API Client"
- Check that `PORTKEY_API_KEY` is set
- Verify the API key is an Organization Admin key

### "API request failed with status 403"
- Ensure your API key has the necessary scopes
- Check that you have Organization Owner or Admin role

### "Provider not found"
- Run `terraform init` to download the provider
- Check that your `required_providers` block is correct

## Resources

- [Full Documentation](../README.md)
- [Provider Configuration](../docs/index.md)
- [Advanced Examples](../examples/multi-environment/README.md)
- [Portkey Admin API Docs](https://portkey.ai/docs/api-reference/admin-api/introduction)

## Support

- GitHub Issues: [Report a bug or request a feature]
- Portkey Discord: https://portkey.sh/discord-1
- Portkey Docs: https://portkey.ai/docs

Happy Terraforming! 🚀
