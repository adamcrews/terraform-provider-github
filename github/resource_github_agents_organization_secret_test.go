package github

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccGithubAgentsOrganizationSecret(t *testing.T) {
	t.Parallel()

	skipUnlessHasOrgs(t)

	t.Run("with_value", func(t *testing.T) {
		t.Parallel()

		randomID := acctest.RandString(testRandomIDLength)
		secretName := strings.ToUpper(fmt.Sprintf("%s%s", strings.ReplaceAll(testResourcePrefix, "-", "_"), randomID))

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "%%s"
  visibility  = "all"
}
`, secretName)

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(config, "super_secret_value"),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_agents_organization_secret.test", tfjsonpath.New("key_id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue("github_agents_organization_secret.test", tfjsonpath.New("created_at"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue("github_agents_organization_secret.test", tfjsonpath.New("updated_at"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue("github_agents_organization_secret.test", tfjsonpath.New("remote_updated_at"), knownvalue.NotNull()),
					},
				},
				{
					Config: fmt.Sprintf(config, "super_secret_value_2"),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("github_agents_organization_secret.test", plancheck.ResourceActionUpdate),
						},
					},
				},
				{
					ResourceName:            "github_agents_organization_secret.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"key_id", "value"},
				},
			},
		})
	})

	t.Run("with_value_encrypted", func(t *testing.T) {
		t.Parallel()

		key := mustGetOrganizationAgentsPublicKey(t)

		encryptedValue, err := encryptPlaintext("super_secret_value", key.GetKey())
		if err != nil {
			t.Fatalf("failed to encrypt plaintext: %v", err)
		}
		encryptedValue2, err := encryptPlaintext("super_secret_value_2", key.GetKey())
		if err != nil {
			t.Fatalf("failed to encrypt plaintext: %v", err)
		}

		randomID := acctest.RandString(testRandomIDLength)
		secretName := strings.ToUpper(fmt.Sprintf("%s%s", strings.ReplaceAll(testResourcePrefix, "-", "_"), randomID))

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name     = "%s"
  key_id          = "%s"
  value_encrypted = "%%s"
  visibility      = "all"
}
`, secretName, key.GetKeyID())

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(config, base64.StdEncoding.EncodeToString(encryptedValue)),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_agents_organization_secret.test", tfjsonpath.New("created_at"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue("github_agents_organization_secret.test", tfjsonpath.New("updated_at"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue("github_agents_organization_secret.test", tfjsonpath.New("remote_updated_at"), knownvalue.NotNull()),
					},
				},
				{
					Config: fmt.Sprintf(config, base64.StdEncoding.EncodeToString(encryptedValue2)),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("github_agents_organization_secret.test", plancheck.ResourceActionUpdate),
						},
					},
				},
				{
					ResourceName:            "github_agents_organization_secret.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"key_id", "value_encrypted"},
				},
			},
		})
	})

	t.Run("with_visibility_all", func(t *testing.T) {
		t.Parallel()

		randomID := acctest.RandString(testRandomIDLength)
		secretName := strings.ToUpper(fmt.Sprintf("%s%s", strings.ReplaceAll(testResourcePrefix, "-", "_"), randomID))

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "all"
}
`, secretName)

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
				},
			},
		})
	})

	t.Run("with_visibility_private", func(t *testing.T) {
		t.Parallel()

		randomID := acctest.RandString(testRandomIDLength)
		secretName := strings.ToUpper(fmt.Sprintf("%s%s", strings.ReplaceAll(testResourcePrefix, "-", "_"), randomID))

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "private"
}
`, secretName)

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
				},
			},
		})
	})

	t.Run("with_visibility_selected_no_repos", func(t *testing.T) {
		t.Parallel()

		randomID := acctest.RandString(testRandomIDLength)
		secretName := strings.ToUpper(fmt.Sprintf("%s%s", strings.ReplaceAll(testResourcePrefix, "-", "_"), randomID))

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "selected"
}
`, secretName)

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
				},
			},
		})
	})

	t.Run("with_visibility_selected_with_repos", func(t *testing.T) {
		t.Parallel()

		repo := mustCreateTestRepository(t)
		repo2 := mustCreateTestRepository(t)

		randomID := acctest.RandString(testRandomIDLength)
		secretName := strings.ToUpper(fmt.Sprintf("%s%s", strings.ReplaceAll(testResourcePrefix, "-", "_"), randomID))

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "selected"
}

resource "github_agents_organization_secret_repositories" "test" {
  secret_name             = github_agents_organization_secret.test.secret_name
  selected_repository_ids = [%%s]
}
`, secretName)

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(config, fmt.Sprintf(`"%v"`, repo.GetID())),
				},
				{
					Config: fmt.Sprintf(config, fmt.Sprintf(`"%v", "%v"`, repo.GetID(), repo2.GetID())),
				},
			},
		})
	})

	t.Run("can_change_visibility", func(t *testing.T) {
		t.Parallel()

		randomID := acctest.RandString(testRandomIDLength)
		secretName := strings.ToUpper(fmt.Sprintf("%s%s", strings.ReplaceAll(testResourcePrefix, "-", "_"), randomID))

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "%%v"
}
`, secretName)

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(config, "all"),
				},
				{
					Config: fmt.Sprintf(config, "private"),
				},
				{
					Config: fmt.Sprintf(config, "selected"),
				},
			},
		})
	})

	t.Run("updates_on_drift", func(t *testing.T) {
		t.Parallel()

		randomID := acctest.RandString(testRandomIDLength)
		secretName := strings.ToUpper(fmt.Sprintf("%s%s", strings.ReplaceAll(testResourcePrefix, "-", "_"), randomID))

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "all"
}
`, secretName)

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
				},
				{
					PreConfig: func() {
						mustUpdateOrganizationAgentsSecret(t, secretName, "super_secret_value_2")
					},
					Config: config,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("github_agents_organization_secret.test", plancheck.ResourceActionUpdate),
						},
					},
				},
			},
		})
	})

	t.Run("lifecycle_ignore_suppresses_drift", func(t *testing.T) {
		t.Parallel()

		randomID := acctest.RandString(testRandomIDLength)
		secretName := strings.ToUpper(fmt.Sprintf("%s%s", strings.ReplaceAll(testResourcePrefix, "-", "_"), randomID))

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "all"

  lifecycle {
    ignore_changes = [updated_at]
  }
}
`, secretName)

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
				},
				{
					PreConfig: func() {
						mustUpdateOrganizationAgentsSecret(t, secretName, "super_secret_value_2")
					},
					Config: config,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("github_agents_organization_secret.test", plancheck.ResourceActionNoop),
						},
					},
				},
			},
		})
	})
}
