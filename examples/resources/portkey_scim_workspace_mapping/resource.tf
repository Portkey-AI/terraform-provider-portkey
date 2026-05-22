# Manage SCIM-group → workspace + role bindings alongside the workspace they
# belong to. The Portkey API has no PATCH endpoint for SCIM mappings — changing
# role, workspace_id, or the SCIM group reference forces resource replacement.

resource "portkey_workspace" "example" {
  name        = "Example Workspace"
  description = "Example workspace"
}

# Bind by SCIM group display name (pushed from Okta with a stable name).
resource "portkey_scim_workspace_mapping" "example_admins" {
  workspace_id    = portkey_workspace.example.id
  scim_group_name = "app-portkey-stage-ws-example-admins"
  role            = "admin"
}

resource "portkey_scim_workspace_mapping" "example_members" {
  workspace_id    = portkey_workspace.example.id
  scim_group_name = "app-portkey-stage-ws-example-members"
  role            = "member"
}

# Alternatively, bind by SCIM group ID if you already know it.
resource "portkey_scim_workspace_mapping" "alt_admins" {
  workspace_id  = portkey_workspace.example.id
  scim_group_id = "d290f1ee-6c54-4b01-90e6-d701748f0851"
  role          = "admin"
}
