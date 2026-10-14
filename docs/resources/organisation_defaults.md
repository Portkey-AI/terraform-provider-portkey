---
page_title: "portkey_organisation_defaults Resource - portkey"
subcategory: ""
description: |-
  Manages organisation-level default input/output guardrails for Portkey.
---

# portkey_organisation_defaults (Resource)

Manages organisation-level default input/output guardrails for Portkey. Every request routed through the organisation's API keys will pass through the configured guardrails in order, unless a workspace overrides them via `portkey_workspace_defaults`.

The Portkey Admin API scopes these defaults to the organisation that owns the configured Admin API key (`GET`/`PUT /admin/organisation/defaults`), so there is no `organisation_id` argument. Exactly one `portkey_organisation_defaults` may exist per provider configuration — declaring two blocks is a config bug, and the last apply wins.

## Example Usage

```terraform
# Organisation-scoped guardrail: note the absence of workspace_id.
resource "portkey_guardrail" "org_pii" {
  name = "org-pii-check"
  checks = jsonencode([{
    id = "default.wordCount"
    parameters = {
      minWords = 1
      maxWords = 4000
    }
  }])
  actions = jsonencode({
    onFail  = "log"
    message = "guardrail triggered"
  })
}

resource "portkey_organisation_defaults" "this" {
  input_guardrails  = [portkey_guardrail.org_pii.slug]
  output_guardrails = [portkey_guardrail.org_pii.slug]
}
```

~> **Note:** Only organisation-scoped guardrails may be used here. Passing a workspace-scoped guardrail (one created with `workspace_id`) fails with `Workspace-scoped guardrails cannot be set as organisation defaults`. Create org-scoped guardrails by omitting `workspace_id` on `portkey_guardrail`.

~> **Note:** Reference guardrails by `slug` for stable plans. The Admin API accepts either guardrail IDs or slugs on write and returns both on read; the provider stores slugs in state. Storing `portkey_guardrail.foo.slug` keeps state and HCL in sync; using `portkey_guardrail.foo.id` (a UUID) produces a permanent plan diff.

~> **Note:** At least one of `input_guardrails` or `output_guardrails` must be set. The API rejects a write that carries neither, so the provider enforces this at plan time.

~> **Note:** Omitting a list means Terraform does not manage it, while setting it to `[]` **always** clears every attached guardrail of that kind.

An omitted attribute is never sent to the API, so guardrails of that kind attached outside Terraform — for example through the Portkey UI — are **preserved**, and Terraform adopts whatever the API reports into state. This holds uniformly on create and on update: removing an attribute you previously managed leaves those guardrails in place rather than clearing them. It is what makes it safe to adopt this resource against an organisation that already has defaults, and to manage only one of the two lists.

~> **Note:** Because an omitted list is not in the configuration, Terraform has no dependency edge from this resource to the guardrails in it. If a `portkey_guardrail` you manage is attached to a list you do not manage, `terraform destroy` may try to delete the guardrail before the defaults are cleared and fail with `AB01 Guardrail is being used in organisation defaults`. Re-running the destroy succeeds. To get a proper ordering guarantee, reference the guardrail in the list so Terraform can see the dependency.

Destroying the resource clears both lists via the update endpoint; there is no separate delete endpoint for organisation defaults.

## Permissions

The Admin API key must carry the `organisation_settings.read` and `organisation_settings.update` scopes. Keys without them receive a 403 from the defaults endpoints.

## Schema

### Optional

- `input_guardrails` (List of String, Computed) Guardrails applied to inbound requests across the organisation, as a list of guardrail slugs (or IDs — the API accepts both). Prefer `portkey_guardrail.foo.slug` to avoid a permanent plan diff. Only organisation-scoped guardrails are accepted. Omitting this attribute leaves any existing input guardrails untouched and adopts them into state; set it to `[]` to clear them.
- `output_guardrails` (List of String, Computed) Guardrails applied to model responses across the organisation, as a list of guardrail slugs (or IDs — the API accepts both). Prefer `portkey_guardrail.foo.slug` to avoid a permanent plan diff. Only organisation-scoped guardrails are accepted. Omitting this attribute leaves any existing output guardrails untouched and adopts them into state; set it to `[]` to clear them.

### Read-Only

- `id` (String) Resource identifier. Always `organisation_defaults`.

## Import

The organisation is derived from the configured Admin API key, so the import ID is ignored. Use the resource's fixed identifier for clarity:

```
terraform import portkey_organisation_defaults.this organisation_defaults
```
