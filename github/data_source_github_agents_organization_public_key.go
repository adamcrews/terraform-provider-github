package github

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGithubAgentsOrganizationPublicKey() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGithubAgentsOrganizationPublicKeyRead,

		Description: "Get information on a GitHub Agents organization public key.",

		Schema: map[string]*schema.Schema{
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

func dataSourceGithubAgentsOrganizationPublicKeyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	err := checkOrganization(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	m, _ := meta.(*Owner)
	client := m.v3client
	owner := m.name

	publicKey, _, err := client.Agents.GetOrgPublicKey(ctx, owner)
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
