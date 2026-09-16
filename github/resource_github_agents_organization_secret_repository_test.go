package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccGithubAgentsOrganizationSecretRepository(t *testing.T) {
	t.Parallel()

	t.Run("create_import", func(t *testing.T) {
		t.Parallel()

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		secretName := fmt.Sprintf("test_%s", randomID)
		repoName := fmt.Sprintf("%s%s", testResourcePrefix, randomID)

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "selected"
}

resource "github_repository" "test" {
  name       = "%s"
  visibility = "public"
}

resource "github_agents_organization_secret_repository" "test" {
  secret_name   = github_agents_organization_secret.test.secret_name
  repository_id = github_repository.test.repo_id
}
`, secretName, repoName)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessHasOrgs(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_agents_organization_secret_repository.test", tfjsonpath.New("secret_name"), knownvalue.StringExact(secretName)),
						statecheck.ExpectKnownValue("github_agents_organization_secret_repository.test", tfjsonpath.New("repository_id"), knownvalue.NotNull()),
					},
				},
				{
					ResourceName:      "github_agents_organization_secret_repository.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})

	t.Run("recreates_on_external_removal", func(t *testing.T) {
		t.Parallel()

		skipUnlessHasOrgs(t)

		repo := mustCreateTestRepository(t)
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		secretName := fmt.Sprintf("test_%s", randomID)

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "selected"
}

resource "github_agents_organization_secret_repository" "test" {
  secret_name   = github_agents_organization_secret.test.secret_name
  repository_id = %d
}
`, secretName, repo.GetID())

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_agents_organization_secret_repository.test", tfjsonpath.New("repository_id"), knownvalue.Int64Exact(repo.GetID())),
					},
				},
				{
					PreConfig: func() {
						mustRemoveAgentsOrgSecretSelectedRepo(t, secretName, repo.GetID())
					},
					Config: config,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("github_agents_organization_secret_repository.test", plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_agents_organization_secret_repository.test", tfjsonpath.New("repository_id"), knownvalue.Int64Exact(repo.GetID())),
					},
				},
			},
		})
	})

	t.Run("handles_parent_secret_deleted", func(t *testing.T) {
		t.Parallel()

		skipUnlessHasOrgs(t)

		repo := mustCreateTestRepository(t)
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		secretName := fmt.Sprintf("test_%s", randomID)

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "selected"
}

resource "github_agents_organization_secret_repository" "test" {
  secret_name   = github_agents_organization_secret.test.secret_name
  repository_id = %d
}
`, secretName, repo.GetID())

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
				},
				{
					PreConfig: func() {
						mustDeleteOrganizationAgentsSecret(t, secretName)
					},
					RefreshState:       true,
					ExpectNonEmptyPlan: true,
				},
				{
					Config:  config,
					Destroy: true,
				},
			},
		})
	})
}
