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

func TestAccGithubAgentsOrganizationSecretRepositories(t *testing.T) {
	t.Parallel()

	t.Run("create_update_import", func(t *testing.T) {
		t.Parallel()

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		secretName := fmt.Sprintf("test_%s", randomID)
		repoName0 := fmt.Sprintf("%s%s-0", testResourcePrefix, randomID)
		repoName1 := fmt.Sprintf("%s%s-1", testResourcePrefix, randomID)
		repoName2 := fmt.Sprintf("%s%s-2", testResourcePrefix, randomID)

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "selected"
}

resource "github_repository" "test_0" {
  name       = "%s"
  visibility = "public"
}

resource "github_repository" "test_1" {
  name       = "%s"
  visibility = "public"
}

resource "github_repository" "test_2" {
  name       = "%s"
  visibility = "public"
}

resource "github_agents_organization_secret_repositories" "test" {
  secret_name = github_agents_organization_secret.test.secret_name
  selected_repository_ids = [
    github_repository.test_0.repo_id,
    github_repository.test_1.repo_id,
  ]
}
`, secretName, repoName0, repoName1, repoName2)

		configUpdated := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "selected"
}

resource "github_repository" "test_0" {
  name       = "%s"
  visibility = "public"
}

resource "github_repository" "test_1" {
  name       = "%s"
  visibility = "public"
}

resource "github_repository" "test_2" {
  name       = "%s"
  visibility = "public"
}

resource "github_agents_organization_secret_repositories" "test" {
  secret_name = github_agents_organization_secret.test.secret_name
  selected_repository_ids = [
    github_repository.test_0.repo_id,
    github_repository.test_2.repo_id,
  ]
}
`, secretName, repoName0, repoName1, repoName2)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessHasOrgs(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_agents_organization_secret_repositories.test", tfjsonpath.New("secret_name"), knownvalue.StringExact(secretName)),
						statecheck.ExpectKnownValue("github_agents_organization_secret_repositories.test", tfjsonpath.New("selected_repository_ids"), knownvalue.SetSizeExact(2)),
					},
				},
				{
					Config: configUpdated,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("github_agents_organization_secret_repositories.test", plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_agents_organization_secret_repositories.test", tfjsonpath.New("selected_repository_ids"), knownvalue.SetSizeExact(2)),
					},
				},
				{
					ResourceName:      "github_agents_organization_secret_repositories.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})

	t.Run("updates_on_external_drift", func(t *testing.T) {
		t.Parallel()

		skipUnlessHasOrgs(t)

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		secretName := fmt.Sprintf("test_%s", randomID)
		repoName0 := fmt.Sprintf("%s%s-0", testResourcePrefix, randomID)
		repoName1 := fmt.Sprintf("%s%s-1", testResourcePrefix, randomID)

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "selected"
}

resource "github_repository" "test_0" {
  name       = "%s"
  visibility = "public"
}

resource "github_repository" "test_1" {
  name       = "%s"
  visibility = "public"
}

resource "github_agents_organization_secret_repositories" "test" {
  secret_name = github_agents_organization_secret.test.secret_name
  selected_repository_ids = [
    github_repository.test_0.repo_id,
    github_repository.test_1.repo_id,
  ]
}
`, secretName, repoName0, repoName1)

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_agents_organization_secret_repositories.test", tfjsonpath.New("selected_repository_ids"), knownvalue.SetSizeExact(2)),
					},
				},
				{
					PreConfig: func() {
						mustSetAgentsOrgSecretSelectedRepos(t, secretName, []int64{})
					},
					Config: config,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("github_agents_organization_secret_repositories.test", plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_agents_organization_secret_repositories.test", tfjsonpath.New("selected_repository_ids"), knownvalue.SetSizeExact(2)),
					},
				},
			},
		})
	})

	t.Run("handles_parent_secret_deleted", func(t *testing.T) {
		t.Parallel()

		skipUnlessHasOrgs(t)

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

resource "github_agents_organization_secret_repositories" "test" {
  secret_name             = github_agents_organization_secret.test.secret_name
  selected_repository_ids = [github_repository.test.repo_id]
}
`, secretName, repoName)

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
