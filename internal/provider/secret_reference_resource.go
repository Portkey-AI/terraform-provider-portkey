package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &secretReferenceResource{}
	_ resource.ResourceWithConfigure   = &secretReferenceResource{}
	_ resource.ResourceWithImportState = &secretReferenceResource{}
	_ resource.ResourceWithModifyPlan  = &secretReferenceResource{}
)

// Supported manager_type values.
const (
	secretManagerAWSSecretsManager = "aws_sm"
	secretManagerAzureKeyVault     = "azure_kv"
	secretManagerHashicorpVault    = "hashicorp_vault"
)

// Auth-block attribute names.
const (
	attrAWSAccessKeyAuth    = "aws_access_key_auth"
	attrAWSAssumedRoleAuth  = "aws_assumed_role_auth"
	attrAWSServiceRoleAuth  = "aws_service_role_auth"
	attrAzureEntraAuth      = "azure_entra_auth"
	attrAzureManagedAuth    = "azure_managed_auth"
	attrVaultTokenAuth      = "vault_token_auth"
	attrVaultAppRoleAuth    = "vault_approle_auth"
	attrVaultKubernetesAuth = "vault_kubernetes_auth"
)

// managerAuthBlocks maps each manager_type to the auth-block attributes that are legal for it.
var managerAuthBlocks = map[string]map[string]bool{
	secretManagerAWSSecretsManager: {
		attrAWSAccessKeyAuth:   true,
		attrAWSAssumedRoleAuth: true,
		attrAWSServiceRoleAuth: true,
	},
	secretManagerAzureKeyVault: {
		attrAzureEntraAuth:   true,
		attrAzureManagedAuth: true,
	},
	secretManagerHashicorpVault: {
		attrVaultTokenAuth:      true,
		attrVaultAppRoleAuth:    true,
		attrVaultKubernetesAuth: true,
	},
}

// NewSecretReferenceResource is a helper function to simplify the provider implementation.
func NewSecretReferenceResource() resource.Resource {
	return &secretReferenceResource{}
}

// secretReferenceResource is the resource implementation.
type secretReferenceResource struct {
	client *client.Client
}

// secretReferenceResourceModel maps the resource schema data.
type secretReferenceResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Slug               types.String `tfsdk:"slug"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	ManagerType        types.String `tfsdk:"manager_type"`
	SecretPath         types.String `tfsdk:"secret_path"`
	SecretKey          types.String `tfsdk:"secret_key"`
	AllowAllWorkspaces types.Bool   `tfsdk:"allow_all_workspaces"`
	AllowedWorkspaces  types.Set    `tfsdk:"allowed_workspaces"`
	Tags               types.Map    `tfsdk:"tags"`
	AuthVersion        types.Int64  `tfsdk:"auth_version"`

	AWSAccessKeyAuth    *awsAccessKeyAuthModel    `tfsdk:"aws_access_key_auth"`
	AWSAssumedRoleAuth  *awsAssumedRoleAuthModel  `tfsdk:"aws_assumed_role_auth"`
	AWSServiceRoleAuth  *awsServiceRoleAuthModel  `tfsdk:"aws_service_role_auth"`
	AzureEntraAuth      *azureEntraAuthModel      `tfsdk:"azure_entra_auth"`
	AzureManagedAuth    *azureManagedAuthModel    `tfsdk:"azure_managed_auth"`
	VaultTokenAuth      *vaultTokenAuthModel      `tfsdk:"vault_token_auth"`
	VaultAppRoleAuth    *vaultAppRoleAuthModel    `tfsdk:"vault_approle_auth"`
	VaultKubernetesAuth *vaultKubernetesAuthModel `tfsdk:"vault_kubernetes_auth"`

	Status    types.String `tfsdk:"status"`
	CreatedBy types.String `tfsdk:"created_by"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

type awsAccessKeyAuthModel struct {
	AWSAccessKeyID       types.String `tfsdk:"aws_access_key_id"`
	AWSAccessKeyIDWO     types.String `tfsdk:"aws_access_key_id_wo"`
	AWSSecretAccessKey   types.String `tfsdk:"aws_secret_access_key"`
	AWSSecretAccessKeyWO types.String `tfsdk:"aws_secret_access_key_wo"`
	AWSRegion            types.String `tfsdk:"aws_region"`
}

type awsAssumedRoleAuthModel struct {
	AWSRoleARN      types.String `tfsdk:"aws_role_arn"`
	AWSExternalID   types.String `tfsdk:"aws_external_id"`
	AWSExternalIDWO types.String `tfsdk:"aws_external_id_wo"`
	AWSRegion       types.String `tfsdk:"aws_region"`
}

type awsServiceRoleAuthModel struct {
	AWSRegion types.String `tfsdk:"aws_region"`
}

type azureEntraAuthModel struct {
	AzureEntraTenantID       types.String `tfsdk:"azure_entra_tenant_id"`
	AzureEntraClientID       types.String `tfsdk:"azure_entra_client_id"`
	AzureEntraClientSecret   types.String `tfsdk:"azure_entra_client_secret"`
	AzureEntraClientSecretWO types.String `tfsdk:"azure_entra_client_secret_wo"`
	AzureVaultURL            types.String `tfsdk:"azure_vault_url"`
}

type azureManagedAuthModel struct {
	AzureManagedClientID types.String `tfsdk:"azure_managed_client_id"`
	AzureVaultURL        types.String `tfsdk:"azure_vault_url"`
}

type vaultTokenAuthModel struct {
	VaultAddr      types.String `tfsdk:"vault_addr"`
	VaultToken     types.String `tfsdk:"vault_token"`
	VaultTokenWO   types.String `tfsdk:"vault_token_wo"`
	VaultNamespace types.String `tfsdk:"vault_namespace"`
}

type vaultAppRoleAuthModel struct {
	VaultAddr       types.String `tfsdk:"vault_addr"`
	VaultRoleID     types.String `tfsdk:"vault_role_id"`
	VaultSecretID   types.String `tfsdk:"vault_secret_id"`
	VaultSecretIDWO types.String `tfsdk:"vault_secret_id_wo"`
	VaultNamespace  types.String `tfsdk:"vault_namespace"`
}

type vaultKubernetesAuthModel struct {
	VaultAddr      types.String `tfsdk:"vault_addr"`
	VaultRole      types.String `tfsdk:"vault_role"`
	VaultNamespace types.String `tfsdk:"vault_namespace"`
}

// Metadata returns the resource type name.
func (r *secretReferenceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_reference"
}

// Schema defines the schema for the resource.
func (r *secretReferenceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Portkey secret reference to an external secret manager " +
			"(AWS Secrets Manager, Azure Key Vault, or HashiCorp Vault). Exactly one auth " +
			"block must be set, and the auth block family must match manager_type.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Secret reference identifier (UUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Description: "URL-friendly identifier. Auto-generated from name if not provided. Used as the primary identifier for Read/Update/Delete and import.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"description": schema.StringAttribute{
				Description: "Optional description (max 1024 chars).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
			},
			"manager_type": schema.StringAttribute{
				Description: "Secret manager type. Available options: `aws_sm`, `azure_kv`, `hashicorp_vault`. Immutable after creation.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						secretManagerAWSSecretsManager,
						secretManagerAzureKeyVault,
						secretManagerHashicorpVault,
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"secret_path": schema.StringAttribute{
				Description: "Path to the secret in the external manager (max 1024 chars).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1024),
				},
			},
			"secret_key": schema.StringAttribute{
				Description: "Optional key within the secret payload (max 255 chars).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"allow_all_workspaces": schema.BoolAttribute{
				Description: "When true (default), all workspaces can use this secret reference. When false, only workspaces listed in allowed_workspaces have access. Cannot be true simultaneously with allowed_workspaces being non-empty.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"allowed_workspaces": schema.SetAttribute{
				Description: "Set of workspace UUIDs or slugs that are allowed to use this secret reference. Mutually exclusive with allow_all_workspaces=true. When set, the API automatically sets allow_all_workspaces=false.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"tags": schema.MapAttribute{
				Description: "Custom metadata tags to attach to the secret reference (string -> string).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"auth_version": schema.Int64Attribute{
				Description: "Rotation trigger for write-only credentials (`*_wo`). The provider re-sends the write-only values only when this value changes. Bump to rotate.",
				Optional:    true,
			},

			attrAWSAccessKeyAuth: schema.SingleNestedAttribute{
				Description: "Static AWS access key credentials (manager_type must be 'aws_sm').",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"aws_access_key_id": schema.StringAttribute{
						Description: "AWS access key ID. Stored in state. Mutually exclusive with `aws_access_key_id_wo`.",
						Optional:    true,
						Sensitive:   true,
					},
					"aws_access_key_id_wo": schema.StringAttribute{
						Description: "Write-only AWS access key ID. Never stored in state. Requires Terraform >= 1.11; rotate by bumping `auth_version`.",
						Optional:    true,
						WriteOnly:   true,
					},
					"aws_secret_access_key": schema.StringAttribute{
						Description: "AWS secret access key. Stored in state. Mutually exclusive with `aws_secret_access_key_wo`.",
						Optional:    true,
						Sensitive:   true,
					},
					"aws_secret_access_key_wo": schema.StringAttribute{
						Description: "Write-only AWS secret access key. Never stored in state. Requires Terraform >= 1.11; rotate by bumping `auth_version`.",
						Optional:    true,
						WriteOnly:   true,
					},
					"aws_region": schema.StringAttribute{
						Description: "AWS region (e.g. 'us-east-1').",
						Required:    true,
					},
				},
			},
			attrAWSAssumedRoleAuth: schema.SingleNestedAttribute{
				Description: "AWS STS AssumeRole credentials (manager_type must be 'aws_sm').",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"aws_role_arn": schema.StringAttribute{
						Description: "ARN of the IAM role to assume.",
						Required:    true,
					},
					"aws_external_id": schema.StringAttribute{
						Description: "Optional external ID for the AssumeRole call. Stored in state. Mutually exclusive with `aws_external_id_wo`.",
						Optional:    true,
						Sensitive:   true,
					},
					"aws_external_id_wo": schema.StringAttribute{
						Description: "Write-only external ID. Never stored in state. Requires Terraform >= 1.11; rotate by bumping `auth_version`.",
						Optional:    true,
						WriteOnly:   true,
					},
					"aws_region": schema.StringAttribute{
						Description: "AWS region.",
						Required:    true,
					},
				},
			},
			attrAWSServiceRoleAuth: schema.SingleNestedAttribute{
				Description: "AWS service role (instance profile / IRSA) credentials, resolved from the environment where Portkey runs (manager_type must be 'aws_sm').",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"aws_region": schema.StringAttribute{
						Description: "AWS region.",
						Optional:    true,
					},
				},
			},
			attrAzureEntraAuth: schema.SingleNestedAttribute{
				Description: "Azure Entra (AAD) service principal credentials for Azure Key Vault (manager_type must be 'azure_kv').",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"azure_entra_tenant_id": schema.StringAttribute{
						Description: "Azure Entra tenant ID.",
						Required:    true,
					},
					"azure_entra_client_id": schema.StringAttribute{
						Description: "Azure Entra application (client) ID.",
						Required:    true,
					},
					"azure_entra_client_secret": schema.StringAttribute{
						Description: "Azure Entra client secret. Stored in state. Mutually exclusive with `azure_entra_client_secret_wo`.",
						Optional:    true,
						Sensitive:   true,
					},
					"azure_entra_client_secret_wo": schema.StringAttribute{
						Description: "Write-only Azure Entra client secret. Never stored in state. Requires Terraform >= 1.11; rotate by bumping `auth_version`.",
						Optional:    true,
						WriteOnly:   true,
					},
					"azure_vault_url": schema.StringAttribute{
						Description: "Full URL of the Azure Key Vault, e.g. https://my-vault.vault.azure.net.",
						Required:    true,
					},
				},
			},
			attrAzureManagedAuth: schema.SingleNestedAttribute{
				Description: "Azure Managed Identity credentials for Azure Key Vault (manager_type must be 'azure_kv').",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"azure_managed_client_id": schema.StringAttribute{
						Description: "Optional client ID of the user-assigned managed identity.",
						Optional:    true,
					},
					"azure_vault_url": schema.StringAttribute{
						Description: "Full URL of the Azure Key Vault.",
						Required:    true,
					},
				},
			},
			attrVaultTokenAuth: schema.SingleNestedAttribute{
				Description: "HashiCorp Vault token-based auth (manager_type must be 'hashicorp_vault').",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"vault_addr": schema.StringAttribute{
						Description: "Vault server address (e.g. https://vault.example.com).",
						Required:    true,
					},
					"vault_token": schema.StringAttribute{
						Description: "Vault token. Stored in state. Mutually exclusive with `vault_token_wo`.",
						Optional:    true,
						Sensitive:   true,
					},
					"vault_token_wo": schema.StringAttribute{
						Description: "Write-only Vault token. Never stored in state. Requires Terraform >= 1.11; rotate by bumping `auth_version`.",
						Optional:    true,
						WriteOnly:   true,
					},
					"vault_namespace": schema.StringAttribute{
						Description: "Optional Vault Enterprise namespace.",
						Optional:    true,
					},
				},
			},
			attrVaultAppRoleAuth: schema.SingleNestedAttribute{
				Description: "HashiCorp Vault AppRole auth (manager_type must be 'hashicorp_vault').",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"vault_addr": schema.StringAttribute{
						Description: "Vault server address.",
						Required:    true,
					},
					"vault_role_id": schema.StringAttribute{
						Description: "AppRole role ID.",
						Required:    true,
					},
					"vault_secret_id": schema.StringAttribute{
						Description: "AppRole secret ID. Stored in state. Mutually exclusive with `vault_secret_id_wo`.",
						Optional:    true,
						Sensitive:   true,
					},
					"vault_secret_id_wo": schema.StringAttribute{
						Description: "Write-only AppRole secret ID. Never stored in state. Requires Terraform >= 1.11; rotate by bumping `auth_version`.",
						Optional:    true,
						WriteOnly:   true,
					},
					"vault_namespace": schema.StringAttribute{
						Description: "Optional Vault Enterprise namespace.",
						Optional:    true,
					},
				},
			},
			attrVaultKubernetesAuth: schema.SingleNestedAttribute{
				Description: "HashiCorp Vault Kubernetes auth (manager_type must be 'hashicorp_vault'). Portkey uses its in-cluster service-account token to authenticate.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"vault_addr": schema.StringAttribute{
						Description: "Vault server address.",
						Required:    true,
					},
					"vault_role": schema.StringAttribute{
						Description: "Vault Kubernetes auth role name.",
						Required:    true,
					},
					"vault_namespace": schema.StringAttribute{
						Description: "Optional Vault Enterprise namespace.",
						Optional:    true,
					},
				},
			},

			"status": schema.StringAttribute{
				Description: "Status of the secret reference (e.g. 'ACTIVE').",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Description: "Identity that created the secret reference.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the secret reference was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the secret reference was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *secretReferenceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// ModifyPlan enforces cross-attribute rules at plan time: exactly one auth_* block set,
// auth block family matches manager_type, allow_all_workspaces XOR allowed_workspaces,
// allowed_workspaces transitions the API supports, and plain vs _wo credential rules.
func (r *secretReferenceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to validate on destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	var config secretReferenceResourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	set := authBlocksSetInConfig(&config)

	if len(set) == 0 {
		resp.Diagnostics.AddError(
			"Missing auth block",
			"Exactly one auth block must be set. Pick one of: "+strings.Join(allAuthBlockAttrs(), ", ")+
				", matching manager_type.",
		)
	} else if len(set) > 1 {
		resp.Diagnostics.AddError(
			"Conflicting auth blocks",
			fmt.Sprintf("Exactly one auth block may be set, but got: %s. Remove all but one.",
				strings.Join(set, ", ")),
		)
	} else if !config.ManagerType.IsNull() && !config.ManagerType.IsUnknown() {
		mt := config.ManagerType.ValueString()
		allowed, ok := managerAuthBlocks[mt]
		if ok && !allowed[set[0]] {
			resp.Diagnostics.AddAttributeError(
				path.Root(set[0]),
				"Auth block does not match manager_type",
				fmt.Sprintf("manager_type %q only accepts these auth blocks: %s. Got: %s.",
					mt, strings.Join(sortedKeys(allowed), ", "), set[0]),
			)
		}
	}

	if !config.AllowAllWorkspaces.IsNull() && !config.AllowAllWorkspaces.IsUnknown() &&
		config.AllowAllWorkspaces.ValueBool() &&
		!config.AllowedWorkspaces.IsNull() && !config.AllowedWorkspaces.IsUnknown() &&
		len(config.AllowedWorkspaces.Elements()) > 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("allow_all_workspaces"),
			"Conflicting workspace-access attributes",
			"allow_all_workspaces cannot be true when allowed_workspaces is non-empty. "+
				"Set allow_all_workspaces = false (or omit it) when providing allowed_workspaces.",
		)
	}

	// When the user omits allow_all_workspaces but provides a non-empty
	// allowed_workspaces list, mirror the server's behaviour (it implicitly
	// sets allow_all_workspaces=false) in the plan. Without this, the schema
	// default pushes plan=true while the API stores false, producing perpetual
	// drift on every refresh.
	if config.AllowAllWorkspaces.IsNull() &&
		!config.AllowedWorkspaces.IsNull() && !config.AllowedWorkspaces.IsUnknown() &&
		len(config.AllowedWorkspaces.Elements()) > 0 {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("allow_all_workspaces"), types.BoolValue(false))...)
	}

	// API rejects allowed_workspaces=[] with AB01. Catch at plan time instead of apply time.
	if !config.AllowedWorkspaces.IsNull() && !config.AllowedWorkspaces.IsUnknown() &&
		len(config.AllowedWorkspaces.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("allowed_workspaces"),
			"Empty allowed_workspaces is not supported",
			"The API rejects an empty list (errorCode AB01). Omit the attribute entirely, or provide at least one workspace ID.",
		)
	}

	// Dropping a non-empty allowed_workspaces on an existing resource is only reachable via
	// allow_all_workspaces=true — the API merge preserves the stored list on omission, and
	// an explicit [] is rejected. Force the user to be explicit to avoid silent divergence.
	if !req.State.Raw.IsNull() {
		var state secretReferenceResourceModel
		if d := req.State.Get(ctx, &state); !d.HasError() {
			var plan secretReferenceResourceModel
			if d := req.Plan.Get(ctx, &plan); !d.HasError() {
				hadList := !state.AllowedWorkspaces.IsNull() && !state.AllowedWorkspaces.IsUnknown() &&
					len(state.AllowedWorkspaces.Elements()) > 0
				dropsList := plan.AllowedWorkspaces.IsNull() ||
					(!plan.AllowedWorkspaces.IsUnknown() && len(plan.AllowedWorkspaces.Elements()) == 0)
				openingGate := !plan.AllowAllWorkspaces.IsNull() && !plan.AllowAllWorkspaces.IsUnknown() &&
					plan.AllowAllWorkspaces.ValueBool()
				if hadList && dropsList && !openingGate {
					resp.Diagnostics.AddAttributeError(
						path.Root("allowed_workspaces"),
						"Cannot clear allowed_workspaces on an existing resource",
						"The API has no transition from a restricted list to an empty list. "+
							"Either keep at least one workspace in allowed_workspaces, set allow_all_workspaces = true "+
							"to open the gate, or taint/recreate the resource.",
					)
				}
			}
		}
	}

	validateAuthSecretPairs(&config, &resp.Diagnostics)
}

// sensitiveFieldSpec describes one (plain, _wo) attribute pair within an auth block.
type sensitiveFieldSpec struct {
	blockAttr string
	plainAttr string
	woAttr    string
	plain     types.String
	wo        types.String
	required  bool
}

// collectSensitiveFieldSpecs returns every (plain, _wo) pair present for the selected auth block.
func collectSensitiveFieldSpecs(m *secretReferenceResourceModel) []sensitiveFieldSpec {
	out := []sensitiveFieldSpec{}
	if b := m.AWSAccessKeyAuth; b != nil {
		out = append(out,
			sensitiveFieldSpec{attrAWSAccessKeyAuth, "aws_access_key_id", "aws_access_key_id_wo", b.AWSAccessKeyID, b.AWSAccessKeyIDWO, true},
			sensitiveFieldSpec{attrAWSAccessKeyAuth, "aws_secret_access_key", "aws_secret_access_key_wo", b.AWSSecretAccessKey, b.AWSSecretAccessKeyWO, true},
		)
	}
	if b := m.AWSAssumedRoleAuth; b != nil {
		out = append(out,
			sensitiveFieldSpec{attrAWSAssumedRoleAuth, "aws_external_id", "aws_external_id_wo", b.AWSExternalID, b.AWSExternalIDWO, false},
		)
	}
	if b := m.AzureEntraAuth; b != nil {
		out = append(out,
			sensitiveFieldSpec{attrAzureEntraAuth, "azure_entra_client_secret", "azure_entra_client_secret_wo", b.AzureEntraClientSecret, b.AzureEntraClientSecretWO, true},
		)
	}
	if b := m.VaultTokenAuth; b != nil {
		out = append(out,
			sensitiveFieldSpec{attrVaultTokenAuth, "vault_token", "vault_token_wo", b.VaultToken, b.VaultTokenWO, true},
		)
	}
	if b := m.VaultAppRoleAuth; b != nil {
		out = append(out,
			sensitiveFieldSpec{attrVaultAppRoleAuth, "vault_secret_id", "vault_secret_id_wo", b.VaultSecretID, b.VaultSecretIDWO, true},
		)
	}
	return out
}

// validateAuthSecretPairs enforces mutual exclusion between plain and _wo attributes,
// requires at least one for historically-required credentials, and warns when a _wo
// attribute is used without auth_version.
func validateAuthSecretPairs(config *secretReferenceResourceModel, diags *diag.Diagnostics) {
	specs := collectSensitiveFieldSpecs(config)
	hasWO := false
	for _, s := range specs {
		plainSet := !s.plain.IsNull() && !s.plain.IsUnknown()
		woSet := !s.wo.IsNull() && !s.wo.IsUnknown()
		if woSet {
			hasWO = true
		}
		if plainSet && woSet {
			diags.AddAttributeError(
				path.Root(s.blockAttr).AtName(s.plainAttr),
				"Conflicting credential attributes",
				fmt.Sprintf("Only one of %q or %q may be set inside %s. Pick the plain attribute (simpler, in state) or the _wo attribute (write-only, out of state).",
					s.plainAttr, s.woAttr, s.blockAttr),
			)
		}
		if s.required && !plainSet && !woSet {
			diags.AddAttributeError(
				path.Root(s.blockAttr).AtName(s.plainAttr),
				"Missing required credential",
				fmt.Sprintf("Either %q or %q must be set inside %s.",
					s.plainAttr, s.woAttr, s.blockAttr),
			)
		}
	}

	if hasWO && config.AuthVersion.IsNull() {
		diags.AddAttributeWarning(
			path.Root("auth_version"),
			"Missing auth_version",
			"A write-only credential attribute (*_wo) is set but auth_version is not. Without auth_version the write-only value is sent to the API on every apply (not gated by a rotation trigger), and you lose the ability to explicitly control when rotation happens. Set auth_version to any integer (typically 1 on first apply, bump to rotate).",
		)
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *secretReferenceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan secretReferenceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write-only attributes live only in config.
	var config secretReferenceResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always rotate on create; nothing on the server side to preserve yet.
	authConfig, authErr := buildAuthConfig(&plan, &config, true)
	if authErr != nil {
		resp.Diagnostics.AddError("Invalid auth block", authErr.Error())
		return
	}

	createReq := client.CreateSecretReferenceRequest{
		Name:        plan.Name.ValueString(),
		ManagerType: plan.ManagerType.ValueString(),
		SecretPath:  plan.SecretPath.ValueString(),
		AuthConfig:  authConfig,
	}

	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() {
		createReq.Slug = plan.Slug.ValueString()
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}
	if !plan.SecretKey.IsNull() && !plan.SecretKey.IsUnknown() {
		createReq.SecretKey = plan.SecretKey.ValueString()
	}
	if !plan.AllowAllWorkspaces.IsNull() && !plan.AllowAllWorkspaces.IsUnknown() {
		v := plan.AllowAllWorkspaces.ValueBool()
		createReq.AllowAllWorkspaces = &v
	}
	if !plan.AllowedWorkspaces.IsNull() && !plan.AllowedWorkspaces.IsUnknown() {
		var ws []string
		resp.Diagnostics.Append(plan.AllowedWorkspaces.ElementsAs(ctx, &ws, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.AllowedWorkspaces = ws
	}
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		tags := map[string]string{}
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Tags = tags
	}

	createResp, err := r.client.CreateSecretReference(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret reference",
			"Could not create secret reference, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch full details by slug; create response only returns id/slug/object.
	secretRef, err := r.client.GetSecretReference(ctx, createResp.Slug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading secret reference after creation",
			"Could not read secret reference, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(applyReadToState(ctx, &plan, secretRef)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *secretReferenceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state secretReferenceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretRef, err := r.client.GetSecretReference(ctx, state.Slug.ValueString())
	if err != nil {
		// Treat 404 (AB08) as drift: resource is gone server-side, so remove it
		// from state and let Terraform plan a recreate instead of erroring.
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Portkey Secret Reference",
			"Could not read Portkey secret reference slug "+state.Slug.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(applyReadToState(ctx, &state, secretRef)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *secretReferenceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan secretReferenceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state secretReferenceResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write-only attributes live only in config.
	var config secretReferenceResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rotate := authVersionChanged(&plan, &state)
	authConfig, authErr := buildAuthConfig(&plan, &config, rotate)
	if authErr != nil {
		resp.Diagnostics.AddError("Invalid auth block", authErr.Error())
		return
	}

	updateReq := client.UpdateSecretReferenceRequest{
		Name:       plan.Name.ValueString(),
		SecretPath: plan.SecretPath.ValueString(),
		AuthConfig: authConfig,
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		updateReq.Description = plan.Description.ValueString()
	}
	// Tri-state secret_key: omit if unchanged, send null to clear, send value to set.
	switch {
	case plan.SecretKey.IsUnknown():
		// leave RawMessage nil → field omitted
	case plan.SecretKey.IsNull() && !state.SecretKey.IsNull() && state.SecretKey.ValueString() != "":
		updateReq.SecretKey = client.JSONNull
	case !plan.SecretKey.IsNull():
		b, err := json.Marshal(plan.SecretKey.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to encode secret_key", err.Error())
			return
		}
		updateReq.SecretKey = b
	}
	if !plan.AllowAllWorkspaces.IsNull() && !plan.AllowAllWorkspaces.IsUnknown() {
		v := plan.AllowAllWorkspaces.ValueBool()
		updateReq.AllowAllWorkspaces = &v
	}
	if !plan.AllowedWorkspaces.IsNull() && !plan.AllowedWorkspaces.IsUnknown() {
		var ws []string
		resp.Diagnostics.Append(plan.AllowedWorkspaces.ElementsAs(ctx, &ws, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.AllowedWorkspaces = ws
	}
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		tags := map[string]string{}
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Tags = tags
	}

	secretRef, err := r.client.UpdateSecretReference(ctx, state.Slug.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Portkey Secret Reference",
			"Could not update secret reference, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(applyReadToState(ctx, &plan, secretRef)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *secretReferenceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state secretReferenceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSecretReference(ctx, state.Slug.ValueString()); err != nil {
		// Already gone server-side — treat as successful delete.
		if strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Portkey Secret Reference",
			"Could not delete secret reference, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state by slug.
func (r *secretReferenceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("slug"), req, resp)
}

// authBlocksSetInConfig returns the non-null auth block attribute names in the given model.
func authBlocksSetInConfig(m *secretReferenceResourceModel) []string {
	pairs := []struct {
		name string
		set  bool
	}{
		{attrAWSAccessKeyAuth, m.AWSAccessKeyAuth != nil},
		{attrAWSAssumedRoleAuth, m.AWSAssumedRoleAuth != nil},
		{attrAWSServiceRoleAuth, m.AWSServiceRoleAuth != nil},
		{attrAzureEntraAuth, m.AzureEntraAuth != nil},
		{attrAzureManagedAuth, m.AzureManagedAuth != nil},
		{attrVaultTokenAuth, m.VaultTokenAuth != nil},
		{attrVaultAppRoleAuth, m.VaultAppRoleAuth != nil},
		{attrVaultKubernetesAuth, m.VaultKubernetesAuth != nil},
	}
	out := []string{}
	for _, p := range pairs {
		if p.set {
			out = append(out, p.name)
		}
	}
	return out
}

func allAuthBlockAttrs() []string {
	return []string{
		attrAWSAccessKeyAuth, attrAWSAssumedRoleAuth, attrAWSServiceRoleAuth,
		attrAzureEntraAuth, attrAzureManagedAuth,
		attrVaultTokenAuth, attrVaultAppRoleAuth, attrVaultKubernetesAuth,
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// buildAuthConfig converts the set auth block into the API payload, injecting the
// discriminator key. Assumes ModifyPlan already enforced "exactly one auth block set".
// Sensitive fields are routed through resolveSensitive, which may omit them so the
// backend preserves the existing server-side value via merge semantics.
func buildAuthConfig(plan, config *secretReferenceResourceModel, rotate bool) (client.SecretReferenceAuthConfig, error) {
	switch {
	case plan.AWSAccessKeyAuth != nil:
		b := plan.AWSAccessKeyAuth
		var cb *awsAccessKeyAuthModel
		if config != nil {
			cb = config.AWSAccessKeyAuth
		}
		out := client.SecretReferenceAuthConfig{
			"aws_auth_type": "accessKey",
			"aws_region":    b.AWSRegion.ValueString(),
		}
		if v, ok := resolveSensitive(b.AWSAccessKeyID, woOrZero(cb, func(c *awsAccessKeyAuthModel) types.String { return c.AWSAccessKeyIDWO }), rotate); ok {
			out["aws_access_key_id"] = v
		}
		if v, ok := resolveSensitive(b.AWSSecretAccessKey, woOrZero(cb, func(c *awsAccessKeyAuthModel) types.String { return c.AWSSecretAccessKeyWO }), rotate); ok {
			out["aws_secret_access_key"] = v
		}
		return out, nil
	case plan.AWSAssumedRoleAuth != nil:
		b := plan.AWSAssumedRoleAuth
		var cb *awsAssumedRoleAuthModel
		if config != nil {
			cb = config.AWSAssumedRoleAuth
		}
		out := client.SecretReferenceAuthConfig{
			"aws_auth_type": "assumedRole",
			"aws_role_arn":  b.AWSRoleARN.ValueString(),
			"aws_region":    b.AWSRegion.ValueString(),
		}
		if v, ok := resolveSensitive(b.AWSExternalID, woOrZero(cb, func(c *awsAssumedRoleAuthModel) types.String { return c.AWSExternalIDWO }), rotate); ok {
			out["aws_external_id"] = v
		}
		return out, nil
	case plan.AWSServiceRoleAuth != nil:
		out := client.SecretReferenceAuthConfig{"aws_auth_type": "serviceRole"}
		if !plan.AWSServiceRoleAuth.AWSRegion.IsNull() && !plan.AWSServiceRoleAuth.AWSRegion.IsUnknown() {
			out["aws_region"] = plan.AWSServiceRoleAuth.AWSRegion.ValueString()
		}
		return out, nil
	case plan.AzureEntraAuth != nil:
		b := plan.AzureEntraAuth
		var cb *azureEntraAuthModel
		if config != nil {
			cb = config.AzureEntraAuth
		}
		out := client.SecretReferenceAuthConfig{
			"azure_auth_mode":       "entra",
			"azure_entra_tenant_id": b.AzureEntraTenantID.ValueString(),
			"azure_entra_client_id": b.AzureEntraClientID.ValueString(),
			"azure_vault_url":       b.AzureVaultURL.ValueString(),
		}
		if v, ok := resolveSensitive(b.AzureEntraClientSecret, woOrZero(cb, func(c *azureEntraAuthModel) types.String { return c.AzureEntraClientSecretWO }), rotate); ok {
			out["azure_entra_client_secret"] = v
		}
		return out, nil
	case plan.AzureManagedAuth != nil:
		b := plan.AzureManagedAuth
		out := client.SecretReferenceAuthConfig{
			"azure_auth_mode": "managed",
			"azure_vault_url": b.AzureVaultURL.ValueString(),
		}
		if !b.AzureManagedClientID.IsNull() && !b.AzureManagedClientID.IsUnknown() {
			out["azure_managed_client_id"] = b.AzureManagedClientID.ValueString()
		}
		return out, nil
	case plan.VaultTokenAuth != nil:
		b := plan.VaultTokenAuth
		var cb *vaultTokenAuthModel
		if config != nil {
			cb = config.VaultTokenAuth
		}
		out := client.SecretReferenceAuthConfig{
			"vault_auth_type": "token",
			"vault_addr":      b.VaultAddr.ValueString(),
		}
		if v, ok := resolveSensitive(b.VaultToken, woOrZero(cb, func(c *vaultTokenAuthModel) types.String { return c.VaultTokenWO }), rotate); ok {
			out["vault_token"] = v
		}
		if !b.VaultNamespace.IsNull() && !b.VaultNamespace.IsUnknown() {
			out["vault_namespace"] = b.VaultNamespace.ValueString()
		}
		return out, nil
	case plan.VaultAppRoleAuth != nil:
		b := plan.VaultAppRoleAuth
		var cb *vaultAppRoleAuthModel
		if config != nil {
			cb = config.VaultAppRoleAuth
		}
		out := client.SecretReferenceAuthConfig{
			"vault_auth_type": "approle",
			"vault_addr":      b.VaultAddr.ValueString(),
			"vault_role_id":   b.VaultRoleID.ValueString(),
		}
		if v, ok := resolveSensitive(b.VaultSecretID, woOrZero(cb, func(c *vaultAppRoleAuthModel) types.String { return c.VaultSecretIDWO }), rotate); ok {
			out["vault_secret_id"] = v
		}
		if !b.VaultNamespace.IsNull() && !b.VaultNamespace.IsUnknown() {
			out["vault_namespace"] = b.VaultNamespace.ValueString()
		}
		return out, nil
	case plan.VaultKubernetesAuth != nil:
		b := plan.VaultKubernetesAuth
		out := client.SecretReferenceAuthConfig{
			"vault_auth_type": "kubernetes",
			"vault_addr":      b.VaultAddr.ValueString(),
			"vault_role":      b.VaultRole.ValueString(),
		}
		if !b.VaultNamespace.IsNull() && !b.VaultNamespace.IsUnknown() {
			out["vault_namespace"] = b.VaultNamespace.ValueString()
		}
		return out, nil
	}
	return nil, fmt.Errorf("no auth block set; this should have been caught by ModifyPlan")
}

// resolveSensitive picks the source for a (plain, _wo) pair: plain if set, else _wo
// when rotating, else omit (backend preserves existing value via merge).
func resolveSensitive(plain, wo types.String, rotate bool) (string, bool) {
	if !plain.IsNull() && !plain.IsUnknown() {
		return plain.ValueString(), true
	}
	if rotate && !wo.IsNull() && !wo.IsUnknown() {
		return wo.ValueString(), true
	}
	return "", false
}

// woOrZero safely extracts a _wo value from a possibly-nil config auth block.
func woOrZero[T any](block *T, f func(*T) types.String) types.String {
	if block == nil {
		return types.StringNull()
	}
	return f(block)
}

// authVersionChanged returns true when write-only credentials should be re-sent on Update.
// Rotates when auth_version differs from state, or when it is unset in either side.
func authVersionChanged(plan, state *secretReferenceResourceModel) bool {
	if plan.AuthVersion.IsNull() || plan.AuthVersion.IsUnknown() {
		return true
	}
	if state.AuthVersion.IsNull() || state.AuthVersion.IsUnknown() {
		return true
	}
	return plan.AuthVersion.ValueInt64() != state.AuthVersion.ValueInt64()
}

// applyReadToState maps API response fields onto the Terraform state model.
// Skips auth_* blocks (returned masked → spurious diffs) and allowed_workspaces
// (never echoed by GET → would clobber user config).
func applyReadToState(ctx context.Context, m *secretReferenceResourceModel, s *client.SecretReference) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(s.ID)
	if m.Slug.IsNull() || m.Slug.IsUnknown() {
		m.Slug = types.StringValue(s.Slug)
	}
	m.Name = types.StringValue(s.Name)
	m.ManagerType = types.StringValue(s.ManagerType)
	m.SecretPath = types.StringValue(s.SecretPath)

	if s.Description != "" {
		m.Description = types.StringValue(s.Description)
	} else if m.Description.IsUnknown() {
		m.Description = types.StringNull()
	}

	if s.SecretKey != "" {
		m.SecretKey = types.StringValue(s.SecretKey)
	} else if m.SecretKey.IsUnknown() {
		m.SecretKey = types.StringNull()
	}

	m.AllowAllWorkspaces = types.BoolValue(s.AllowAllWorkspaces)

	if len(s.Tags) > 0 {
		tags, d := types.MapValueFrom(ctx, types.StringType, s.Tags)
		diags.Append(d...)
		m.Tags = tags
	} else if m.Tags.IsUnknown() {
		m.Tags = types.MapNull(types.StringType)
	}

	if m.Status.IsUnknown() || s.Status != "" {
		m.Status = types.StringValue(s.Status)
	}
	if m.CreatedBy.IsUnknown() || s.CreatedBy != "" {
		m.CreatedBy = types.StringValue(s.CreatedBy)
	}
	m.CreatedAt = types.StringValue(s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if !s.UpdatedAt.IsZero() {
		m.UpdatedAt = types.StringValue(s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		m.UpdatedAt = m.CreatedAt
	}

	return diags
}
