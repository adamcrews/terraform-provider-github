package github

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccGithubAgentsOrganizationSecretsDataSource(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		randomID := acctest.RandString(testRandomIDLength)
		secretName := strings.ToUpper(fmt.Sprintf("%s%s", strings.ReplaceAll(testResourcePrefix, "-", "_"), randomID))

		config := fmt.Sprintf(`
resource "github_agents_organization_secret" "test" {
  secret_name = "%s"
  value       = "super_secret_value"
  visibility  = "all"
}

data "github_agents_organization_secrets" "test" {
  depends_on = [github_agents_organization_secret.test]
}
`, secretName)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessHasOrgs(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("data.github_agents_organization_secrets.test", tfjsonpath.New("secrets"), knownvalueListContainsMapAttr("name", secretName)),
					},
				},
			},
		})
	})
}

// knownvalueListContainsMapAttr asserts a []any list contains a map with attr == want.
type knownvalueListContainsMapAttrCheck struct {
	attr string
	want string
}

func knownvalueListContainsMapAttr(attr, want string) knownvalue.Check {
	return knownvalueListContainsMapAttrCheck{attr: attr, want: want}
}

func (c knownvalueListContainsMapAttrCheck) CheckValue(other any) error {
	list, ok := other.([]any)
	if !ok {
		return fmt.Errorf("expected []any value for list contains check, got: %T", other)
	}

	for _, elem := range list {
		m, ok := elem.(map[string]any)
		if !ok {
			continue
		}
		if v, _ := m[c.attr].(string); v == c.want {
			return nil
		}
	}

	return fmt.Errorf("expected list to contain map with %s=%q", c.attr, c.want)
}

func (c knownvalueListContainsMapAttrCheck) String() string {
	return fmt.Sprintf("list containing map with %s=%q", c.attr, c.want)
}
