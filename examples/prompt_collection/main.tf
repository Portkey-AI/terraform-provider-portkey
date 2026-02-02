terraform {
  required_providers {
    portkey = {
      source = "portkey-ai/portkey"
    }
  }
}

provider "portkey" {
  # api_key = "your-api-key" # Or set PORTKEY_API_KEY env var
}

# First, create a workspace (or reference an existing one)
resource "portkey_workspace" "example" {
  name        = "Example Workspace"
  description = "Workspace for prompt collection example"
}

# Create a top-level collection
resource "portkey_prompt_collection" "main" {
  name         = "Main Prompts"
  workspace_id = portkey_workspace.example.id
}

# Create a nested collection
resource "portkey_prompt_collection" "terraform" {
  name                 = "Terraform Prompts"
  workspace_id         = portkey_workspace.example.id
  parent_collection_id = portkey_prompt_collection.main.id
}

# Output the collection IDs
output "main_collection_id" {
  value = portkey_prompt_collection.main.id
}

output "terraform_collection_id" {
  value = portkey_prompt_collection.terraform.id
}

output "terraform_collection_slug" {
  value = portkey_prompt_collection.terraform.slug
}
