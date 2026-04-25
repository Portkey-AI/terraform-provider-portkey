package provider

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// skipIfTerraformOlderThan skips the test if the installed Terraform version
// is older than the specified minimum. Used for tests that require features
// like WriteOnly attributes (TF 1.11+).
func skipIfTerraformOlderThan(t *testing.T, minVersion string) {
	t.Helper()
	out, err := exec.Command("terraform", "version", "-json").Output()
	if err != nil {
		t.Skipf("Could not determine Terraform version: %v", err)
	}
	// Check if TF version meets minimum by looking for supported versions in JSON output
	versionStr := string(out)
	for _, v := range []string{"1.11", "1.12", "1.13", "1.14", "1.15", "2."} {
		if strings.Contains(versionStr, `"terraform_version":"`+v) {
			return // Version is sufficient
		}
	}
	t.Skipf("Test requires Terraform %s or later", minVersion)
}

// TestAccSecretReferenceResource_awsAccessKey exercises the AWS Secrets
// Manager path using static access-key credentials. It verifies create / read
// / import / update / delete and that every core attribute survives
// round-tripping.
func TestAccSecretReferenceResource_awsAccessKey(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-sr")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretReferenceResourceAWSAccessKey(rName, "prod/api-keys/openai"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_secret_reference.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_secret_reference.test", "slug"),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "manager_type", "aws_sm"),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "secret_path", "prod/api-keys/openai"),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "allow_all_workspaces", "true"),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "aws_access_key_auth.aws_region", "us-east-1"),
				),
			},
			{
				ResourceName:      "portkey_secret_reference.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Auth blocks carry credentials the API never returns on GET
				// (or returns masked). Ignore on import verify to avoid false
				// "not equivalent" failures. updated_at may differ by a second.
				ImportStateVerifyIgnore: []string{
					"aws_access_key_auth",
					"aws_assumed_role_auth",
					"aws_service_role_auth",
					"azure_entra_auth",
					"azure_managed_auth",
					"vault_token_auth",
					"vault_approle_auth",
					"vault_kubernetes_auth",
					"updated_at",
				},
			},
			{
				Config: testAccSecretReferenceResourceAWSAccessKey(rName+"-updated", "prod/api-keys/updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "secret_path", "prod/api-keys/updated"),
				),
			},
		},
	})
}

// TestAccSecretReferenceResource_vaultAppRole exercises the HashiCorp Vault
// path using AppRole auth, and also exercises allow_all_workspaces=false with
// a concrete allowed_workspaces set.
func TestAccSecretReferenceResource_vaultAppRole(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-sr-vault")
	wsID := getTestWorkspaceID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretReferenceResourceVaultAppRole(rName, wsID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "manager_type", "hashicorp_vault"),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "allow_all_workspaces", "false"),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "vault_approle_auth.vault_addr", "https://vault.example.internal"),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "vault_approle_auth.vault_role_id", "test-role-id"),
				),
			},
		},
	})
}

// TestAccSecretReferenceResource_planValidation_noAuthBlock verifies that
// ModifyPlan surfaces a clear error at plan time when no auth block is set.
// This is the core promise of the typed-blocks design: errors during
// `terraform plan`, not from the API at apply time.
func TestAccSecretReferenceResource_planValidation_noAuthBlock(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-sr-plan")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSecretReferenceResourceMissingAuth(rName),
				ExpectError: regexp.MustCompile(`Missing auth block`),
			},
		},
	})
}

// TestAccSecretReferenceResource_planValidation_mismatchedAuthBlock verifies
// that a Vault auth block paired with manager_type=aws_sm is rejected at
// plan time.
func TestAccSecretReferenceResource_planValidation_mismatchedAuthBlock(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-sr-plan")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSecretReferenceResourceMismatchedAuth(rName),
				ExpectError: regexp.MustCompile(`Auth block does not match manager_type`),
			},
		},
	})
}

// TestAccSecretReferenceResource_planValidation_conflictingWorkspaces
// verifies that allow_all_workspaces=true with a non-empty
// allowed_workspaces is rejected at plan time.
func TestAccSecretReferenceResource_planValidation_conflictingWorkspaces(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-sr-plan")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSecretReferenceResourceConflictingWorkspaces(rName),
				ExpectError: regexp.MustCompile(`Conflicting workspace-access attributes`),
			},
		},
	})
}

// TestAccSecretReferenceResource_writeOnly_create verifies that creating a
// secret reference with only _wo sensitive fields populated succeeds and that
// neither the _wo attribute nor its plain sibling surface in state (the plain
// field stays null - the value flew straight through config -> wire -> API).
//
// This is the core "nothing in state" guarantee of the write-only pattern.
func TestAccSecretReferenceResource_writeOnly_create(t *testing.T) {
	skipIfTerraformOlderThan(t, "1.11")
	rName := acctest.RandomWithPrefix("tf-acc-sr-wo")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretReferenceResourceAWSWriteOnly(rName, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_secret_reference.test", "id"),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "manager_type", "aws_sm"),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "auth_version", "1"),
					// Neither the plain sibling nor the _wo value are stored.
					resource.TestCheckNoResourceAttr("portkey_secret_reference.test", "aws_access_key_auth.aws_access_key_id"),
					resource.TestCheckNoResourceAttr("portkey_secret_reference.test", "aws_access_key_auth.aws_secret_access_key"),
					resource.TestCheckNoResourceAttr("portkey_secret_reference.test", "aws_access_key_auth.aws_access_key_id_wo"),
					resource.TestCheckNoResourceAttr("portkey_secret_reference.test", "aws_access_key_auth.aws_secret_access_key_wo"),
					// Non-sensitive fields still round-trip.
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "aws_access_key_auth.aws_region", "us-east-1"),
				),
			},
		},
	})
}

// TestAccSecretReferenceResource_authVersionRotation drives the full rotation
// lifecycle on a single resource:
//
//  1. Create at auth_version=1 with a _wo credential.
//  2. Same auth_version, different _wo value  -> plan must be empty (the
//     provider does not re-send the credential without a version bump).
//  3. Bump auth_version=2  -> apply succeeds, auth_version updates in state,
//     plain sibling stays null throughout.
//  4. Keep auth_version=2, change only non-credential attributes (description)
//     -> apply succeeds, credential is preserved server-side via the API's
//     auth_config merge semantics, plain sibling still null.
//
// Together these steps prove the state-safety property (the credential never
// lands in .tfstate) and the rotation-gating property (the credential is only
// sent on the wire when the user explicitly bumps auth_version).
func TestAccSecretReferenceResource_authVersionRotation(t *testing.T) {
	skipIfTerraformOlderThan(t, "1.11")
	rName := acctest.RandomWithPrefix("tf-acc-sr-rotate")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1. Initial apply at v1.
			{
				Config: testAccSecretReferenceResourceVaultTokenWO(rName, "first-description", "hvs.test-token-v1", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "auth_version", "1"),
					resource.TestCheckNoResourceAttr("portkey_secret_reference.test", "vault_token_auth.vault_token"),
					resource.TestCheckNoResourceAttr("portkey_secret_reference.test", "vault_token_auth.vault_token_wo"),
				),
			},
			// 2. Changing only the _wo value without bumping auth_version is a
			//    no-op. The framework strips _wo from plan, auth_version stays
			//    at 1, so the provider detects no change.
			{
				Config:             testAccSecretReferenceResourceVaultTokenWO(rName, "first-description", "hvs.test-token-v2", 1),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// 3. Bump auth_version to trigger rotation.
			{
				Config: testAccSecretReferenceResourceVaultTokenWO(rName, "first-description", "hvs.test-token-v2", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "auth_version", "2"),
					resource.TestCheckNoResourceAttr("portkey_secret_reference.test", "vault_token_auth.vault_token"),
					resource.TestCheckNoResourceAttr("portkey_secret_reference.test", "vault_token_auth.vault_token_wo"),
				),
			},
			// 4. Change only a non-credential attribute (description). The
			//    provider builds a partial auth_config that omits vault_token;
			//    the backend keeps the previous token via its merge semantics.
			{
				Config: testAccSecretReferenceResourceVaultTokenWO(rName, "updated-description", "hvs.test-token-v2", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "auth_version", "2"),
					resource.TestCheckResourceAttr("portkey_secret_reference.test", "description", "updated-description"),
					resource.TestCheckNoResourceAttr("portkey_secret_reference.test", "vault_token_auth.vault_token"),
				),
			},
		},
	})
}

// TestAccSecretReferenceResource_planValidation_plainAndWOConflict verifies
// that setting both the plain sensitive field and its _wo sibling on the same
// auth block fails at plan time with the "Conflicting credential attributes"
// diagnostic.
func TestAccSecretReferenceResource_planValidation_plainAndWOConflict(t *testing.T) {
	skipIfTerraformOlderThan(t, "1.11")
	rName := acctest.RandomWithPrefix("tf-acc-sr-plan")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSecretReferenceResourcePlainAndWOConflict(rName),
				ExpectError: regexp.MustCompile(`Conflicting credential attributes`),
			},
		},
	})
}

// TestAccSecretReferenceResource_planValidation_missingCredential verifies
// that omitting both the plain sensitive field and its _wo sibling for a
// required credential fails at plan time.
func TestAccSecretReferenceResource_planValidation_missingCredential(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-sr-plan")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSecretReferenceResourceMissingCredential(rName),
				ExpectError: regexp.MustCompile(`Missing required credential`),
			},
		},
	})
}

// TestAccSecretReferenceDataSource_singular verifies the by-slug data source
// returns the expected metadata and never exposes credentials.
func TestAccSecretReferenceDataSource_singular(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-sr-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretReferenceDataSourceSingular(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_secret_reference.test", "id"),
					resource.TestCheckResourceAttr("data.portkey_secret_reference.test", "name", rName),
					resource.TestCheckResourceAttr("data.portkey_secret_reference.test", "manager_type", "aws_sm"),
				),
			},
		},
	})
}

// TestAccSecretReferenceDataSource_plural verifies the list data source
// returns at least one item after creating a resource.
func TestAccSecretReferenceDataSource_plural(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-sr-ds-list")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretReferenceDataSourcePlural(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.portkey_secret_references.aws", "secret_references.#"),
				),
			},
		},
	})
}

// --- Config helpers ---

func testAccSecretReferenceResourceAWSAccessKey(name, secretPath string) string {
	return fmt.Sprintf(`
%s

resource "portkey_secret_reference" "test" {
  name         = %q
  manager_type = "aws_sm"
  secret_path  = %q

  aws_access_key_auth = {
    aws_access_key_id     = "AKIATEST"
    aws_secret_access_key = "test-secret"
    aws_region            = "us-east-1"
  }
}
`, providerConfig, name, secretPath)
}

func testAccSecretReferenceResourceVaultAppRole(name, workspaceID string) string {
	return fmt.Sprintf(`
%s

resource "portkey_secret_reference" "test" {
  name         = %q
  manager_type = "hashicorp_vault"
  secret_path  = "kv/data/test/path"

  vault_approle_auth = {
    vault_addr      = "https://vault.example.internal"
    vault_role_id   = "test-role-id"
    vault_secret_id = "test-secret-id"
  }

  allow_all_workspaces = false
  allowed_workspaces   = [%q]
}
`, providerConfig, name, workspaceID)
}

func testAccSecretReferenceResourceMissingAuth(name string) string {
	return fmt.Sprintf(`
%s

resource "portkey_secret_reference" "test" {
  name         = %q
  manager_type = "aws_sm"
  secret_path  = "prod/anything"
}
`, providerConfig, name)
}

func testAccSecretReferenceResourceMismatchedAuth(name string) string {
	return fmt.Sprintf(`
%s

resource "portkey_secret_reference" "test" {
  name         = %q
  manager_type = "aws_sm"
  secret_path  = "prod/anything"

  vault_approle_auth = {
    vault_addr      = "https://vault.example.internal"
    vault_role_id   = "test-role-id"
    vault_secret_id = "test-secret-id"
  }
}
`, providerConfig, name)
}

func testAccSecretReferenceResourceConflictingWorkspaces(name string) string {
	return fmt.Sprintf(`
%s

resource "portkey_secret_reference" "test" {
  name         = %q
  manager_type = "aws_sm"
  secret_path  = "prod/anything"

  aws_access_key_auth = {
    aws_access_key_id     = "AKIATEST"
    aws_secret_access_key = "test"
    aws_region            = "us-east-1"
  }

  allow_all_workspaces = true
  allowed_workspaces   = ["some-workspace"]
}
`, providerConfig, name)
}

// testAccSecretReferenceResourceAWSWriteOnly produces a config that populates
// the AWS access-key auth block exclusively via _wo attributes plus an
// auth_version trigger. Used to prove nothing lands in state.
func testAccSecretReferenceResourceAWSWriteOnly(name string, authVersion int) string {
	return fmt.Sprintf(`
%s

resource "portkey_secret_reference" "test" {
  name         = %q
  manager_type = "aws_sm"
  secret_path  = "prod/api-keys/openai"
  auth_version = %d

  aws_access_key_auth = {
    aws_access_key_id_wo     = "AKIAWOTEST000000"
    aws_secret_access_key_wo = "wo-secret-never-stored"
    aws_region               = "us-east-1"
  }
}
`, providerConfig, name, authVersion)
}

// testAccSecretReferenceResourceVaultTokenWO builds a Vault token-auth config
// with a write-only token and a parameterised auth_version. The description
// is parameterised so tests can exercise non-credential-attribute updates.
func testAccSecretReferenceResourceVaultTokenWO(name, description, token string, authVersion int) string {
	return fmt.Sprintf(`
%s

resource "portkey_secret_reference" "test" {
  name         = %q
  description  = %q
  manager_type = "hashicorp_vault"
  secret_path  = "kv/data/test/path"
  auth_version = %d

  vault_token_auth = {
    vault_addr     = "https://vault.example.internal"
    vault_token_wo = %q
  }
}
`, providerConfig, name, description, authVersion, token)
}

// testAccSecretReferenceResourcePlainAndWOConflict sets both aws_access_key_id
// and aws_access_key_id_wo on the same block - rejected at plan time.
func testAccSecretReferenceResourcePlainAndWOConflict(name string) string {
	return fmt.Sprintf(`
%s

resource "portkey_secret_reference" "test" {
  name         = %q
  manager_type = "aws_sm"
  secret_path  = "prod/anything"

  aws_access_key_auth = {
    aws_access_key_id        = "AKIACONFLICT0000"
    aws_access_key_id_wo     = "AKIACONFLICT0001"
    aws_secret_access_key    = "plain"
    aws_region               = "us-east-1"
  }
}
`, providerConfig, name)
}

// testAccSecretReferenceResourceMissingCredential leaves aws_secret_access_key
// (and its _wo sibling) unset on a block that historically required it -
// rejected at plan time.
func testAccSecretReferenceResourceMissingCredential(name string) string {
	return fmt.Sprintf(`
%s

resource "portkey_secret_reference" "test" {
  name         = %q
  manager_type = "aws_sm"
  secret_path  = "prod/anything"

  aws_access_key_auth = {
    aws_access_key_id = "AKIATEST00000000"
    aws_region        = "us-east-1"
  }
}
`, providerConfig, name)
}

func testAccSecretReferenceDataSourceSingular(name string) string {
	return fmt.Sprintf(`
%s

resource "portkey_secret_reference" "test" {
  name         = %q
  manager_type = "aws_sm"
  secret_path  = "prod/api-keys/openai"

  aws_access_key_auth = {
    aws_access_key_id     = "AKIATEST"
    aws_secret_access_key = "test"
    aws_region            = "us-east-1"
  }
}

data "portkey_secret_reference" "test" {
  slug = portkey_secret_reference.test.slug
}
`, providerConfig, name)
}

func testAccSecretReferenceDataSourcePlural(name string) string {
	return fmt.Sprintf(`
%s

resource "portkey_secret_reference" "test" {
  name         = %q
  manager_type = "aws_sm"
  secret_path  = "prod/api-keys/openai"

  aws_access_key_auth = {
    aws_access_key_id     = "AKIATEST"
    aws_secret_access_key = "test"
    aws_region            = "us-east-1"
  }
}

data "portkey_secret_references" "aws" {
  manager_type = "aws_sm"
  depends_on   = [portkey_secret_reference.test]
}
`, providerConfig, name)
}
