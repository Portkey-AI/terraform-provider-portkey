# Secret Reference Example

This example demonstrates managing Portkey secret references — pointers to
secrets stored in external secret managers (AWS Secrets Manager, Azure Key
Vault, or HashiCorp Vault). Secret references let other Portkey objects
(integrations, virtual keys) resolve credentials dynamically at request time
instead of embedding them in Portkey state.

## What's Included

- An AWS Secrets Manager reference using static access keys
- A HashiCorp Vault reference using AppRole auth, scoped to specific workspaces
- A singular data source lookup by slug
- A plural data source lookup filtered by `manager_type`

## Auth Block Rules

Exactly one auth block must be set per resource, and it must match
`manager_type`:

| `manager_type`     | Valid auth blocks                                                |
| ------------------ | ---------------------------------------------------------------- |
| `aws_sm`           | `aws_access_key_auth`, `aws_assumed_role_auth`, `aws_service_role_auth` |
| `azure_kv`         | `azure_entra_auth`, `azure_managed_auth`   |
| `hashicorp_vault`  | `vault_token_auth`, `vault_approle_auth`, `vault_kubernetes_auth` |

These rules are enforced at `terraform plan` time — if you pick the wrong
combination you get a clean error before any API call.

## Usage

1. Set your Portkey Admin API key:

   ```bash
   export PORTKEY_API_KEY="your-admin-api-key"
   ```

2. Provide the AWS credentials expected by the example:

   ```bash
   export TF_VAR_aws_access_key_id="AKIA..."
   export TF_VAR_aws_secret_access_key="..."
   ```

3. Initialize and apply:

   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

4. Clean up when done:

   ```bash
   terraform destroy
   ```

## Notes on Credentials and State

- Credential fields inside the auth blocks are marked `Sensitive`. Terraform
  still stores the value in state (consistent with `portkey_integration.key`);
  protect your state backend accordingly.
- The Portkey API returns `auth_config` on read with sensitive fields masked.
  The provider deliberately does not overwrite your configured auth block from
  that masked payload, so you will not see spurious diffs on subsequent plans.
- `secret_path`, `name`, `description`, `secret_key`, `allow_all_workspaces`,
  `allowed_workspaces`, and `tags` are all updatable in place. `manager_type`
  is immutable — changing it forces replacement.
- The delete call fails if the secret reference is still referenced by any
  integration or virtual key. Remove those dependencies first.
