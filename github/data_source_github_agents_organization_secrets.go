package github

import (
	"context"

	"github.com/google/go-github/v92/github"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGithubAgentsOrganizationSecrets() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGithubAgentsOrganizationSecretsRead,

		Description: "Get the list of GitHub Agents secrets for an organization.",

		Schema: map[string]*schema.Schema{
			"secrets": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of secrets for the organization.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the secret.",
						},
						"visibility": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Visibility of the secret.",
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Timestamp for when the secret was created.",
						},
						"updated_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Timestamp for when the secret was last updated.",
						},
					},
				},
			},
		},
	}
}

func dataSourceGithubAgentsOrganizationSecretsRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	var allSecrets []map[string]string
	for secret, err := range client.Agents.ListOrgSecretsIter(ctx, owner, &github.ListOptions{PerPage: meta.maxPerPage}) {
		if err != nil {
			return diag.FromErr(err)
		}

		allSecrets = append(allSecrets, map[string]string{
			"name":       secret.Name,
			"created_at": secret.CreatedAt.String(),
			"updated_at": secret.UpdatedAt.String(),
			"visibility": secret.Visibility,
		})
	}

	d.SetId(owner)
	if err := d.Set("secrets", allSecrets); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
