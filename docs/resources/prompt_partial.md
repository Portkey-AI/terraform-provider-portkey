---
page_title: "portkey_prompt_partial Resource - terraform-provider-portkey"
subcategory: ""
description: |-
  Manages a Portkey prompt partial for reusable template fragments within a workspace.
---

# portkey_prompt_partial (Resource)

Manages a Portkey prompt partial.

Prompt partials are reusable template fragments that can be referenced in prompts via Mustache syntax (`{{>partial-slug}}`). They allow you to share common content across multiple prompts without duplication.

## Example Usage

```terraform
# Create a workspace
resource "portkey_workspace" "example" {
  name        = "Example Workspace"
  description = "Workspace for prompt partial example"
}

# Create a prompt partial
resource "portkey_prompt_partial" "system_context" {
  name                = "System Context"
  content             = "You are a helpful assistant. Always respond in a professional tone."
  workspace_id        = portkey_workspace.example.id
  version_description = "Initial version"
}

# Reference the partial in a prompt using {{>system-context}}
resource "portkey_prompt" "example" {
  name         = "Example Prompt"
  workspace_id = portkey_workspace.example.id

  # The partial can be included via {{>system-context}}
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Human-readable name for the prompt partial.
* `content` - (Required) The partial template content. Maps to the API `string` field.
* `workspace_id` - (Optional) Workspace ID to scope the prompt partial to. Required when using an org-level API key. Changing this forces a new resource.
* `version_description` - (Optional) Description for the prompt partial version. Only takes effect when `content` changes in the same apply.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Prompt partial identifier (UUID).
* `slug` - URL-friendly identifier for the prompt partial. Auto-generated based on name.
* `version` - Current version number of the prompt partial.
* `prompt_partial_version_id` - Current version ID of the prompt partial.
* `status` - Status of the prompt partial (active, archived).
* `created_at` - Timestamp when the prompt partial was created.
* `updated_at` - Timestamp when the prompt partial was last updated.

## Known Limitations

* **Drift detection:** Due to eventual consistency in the Portkey API, this resource preserves content, version, and version_id from Terraform state rather than refreshing from the API. External changes to these fields (via UI, API, or another Terraform workspace) will not be detected by `terraform plan`. Terraform should be treated as the source of truth for prompt partial content.
* **Version description:** `version_description` only takes effect when `content` changes in the same apply. Changing `version_description` alone will not create a new version.

## Import

Prompt partials can be imported using the workspace ID and slug:

```shell
terraform import portkey_prompt_partial.example <workspace_id>/<slug>
```
