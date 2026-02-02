---
page_title: "portkey_prompt_collection Resource - terraform-provider-portkey"
subcategory: ""
description: |-
  Manages a Portkey prompt collection for organizing prompts within a workspace.
---

# portkey_prompt_collection (Resource)

Manages a Portkey prompt collection.

Collections are used to organize prompts within a workspace. They can be nested using `parent_collection_id` to create hierarchical organization structures.

## Example Usage

```terraform
# Create a workspace
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
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the collection.
* `workspace_id` - (Required) Workspace ID (UUID) where this collection belongs. Changing this forces a new resource.
* `parent_collection_id` - (Optional) Parent collection ID for nested collections. Leave empty for top-level collections. Changing this forces a new resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Collection identifier (UUID).
* `slug` - URL-friendly identifier for the collection. Auto-generated from name.
* `is_default` - Whether this is the default collection for the workspace.
* `status` - Collection status (active, archived).
* `created_at` - Timestamp when the collection was created.
* `last_updated_at` - Timestamp when the collection was last updated.

## Import

Prompt collections can be imported using the collection ID:

```shell
terraform import portkey_prompt_collection.example <collection-id>
```
