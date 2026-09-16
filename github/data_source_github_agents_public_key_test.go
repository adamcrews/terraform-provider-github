package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccGithubAgentsPublicKeyDataSource(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		skipUnauthenticated(t)

		repo := mustCreateTestRepository(t)

		config := fmt.Sprintf(`
data "github_agents_public_key" "test" {
  repository = "%s"
}
`, repo.GetName())

		resource.Test(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("data.github_agents_public_key.test", tfjsonpath.New("key_id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue("data.github_agents_public_key.test", tfjsonpath.New("key"), knownvalue.NotNull()),
					},
				},
			},
		})
	})
}
