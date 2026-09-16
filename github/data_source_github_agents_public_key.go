package github

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGithubAgentsPublicKey() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGithubAgentsPublicKeyRead,

		Description: "Get information on a GitHub Agents repository public key.",

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the repository to get the public key from.",
			},
			"key_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the public key.",
			},
			"key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Base64 encoded public key.",
			},
		},
	}
}

func dataSourceGithubAgentsPublicKeyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	m, _ := meta.(*Owner)
	client := m.v3client
	owner := m.name

	repository, _ := d.Get("repository").(string)

	publicKey, _, err := client.Agents.GetRepoPublicKey(ctx, owner, repository)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(publicKey.GetKeyID())
	err = d.Set("key_id", publicKey.GetKeyID())
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("key", publicKey.GetKey())
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
