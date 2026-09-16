package github

import (
	"context"

	"github.com/google/go-github/v92/github"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGithubAgentsSecrets() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGithubAgentsSecretsRead,

		Description: "Get the list of GitHub Agents secrets for a repository.",

		Schema: map[string]*schema.Schema{
			"full_name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"name"},
				Description:   "Full name of the repository (in `org/name` format).",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"full_name"},
				Description:   "Name of the repository.",
			},
			"secrets": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of secrets for the repository.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the secret.",
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

func dataSourceGithubAgentsSecretsRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name
	var repoName string

	if fullName, ok := d.GetOk("full_name"); ok {
		var err error
		fullNameStr, _ := fullName.(string)
		owner, repoName, err = splitRepoFullName(fullNameStr)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if name, ok := d.GetOk("name"); ok {
		repoName, _ = name.(string)
	}

	if repoName == "" {
		return diag.Errorf("one of %q or %q has to be provided", "full_name", "name")
	}

	var allSecrets []map[string]string
	for secret, err := range client.Agents.ListRepoSecretsIter(ctx, owner, repoName, &github.ListOptions{PerPage: meta.maxPerPage}) {
		if err != nil {
			return diag.FromErr(err)
		}

		allSecrets = append(allSecrets, map[string]string{
			"name":       secret.Name,
			"created_at": secret.CreatedAt.String(),
			"updated_at": secret.UpdatedAt.String(),
		})
	}

	d.SetId(repoName)
	if err := d.Set("secrets", allSecrets); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
