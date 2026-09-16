package github

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/go-github/v92/github"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceGithubAgentsOrganizationSecretRepositories() *schema.Resource {
	return &schema.Resource{
		Description: "Manages the repository allow list for a GitHub Agents organization secret.",

		Schema: map[string]*schema.Schema{
			"secret_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateSecretNameFunc,
				Description:      "Name of the existing secret.",
			},
			"selected_repository_ids": {
				Type: schema.TypeSet,
				Set:  schema.HashInt,
				Elem: &schema.Schema{
					Type:             schema.TypeInt,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
				},
				Required:    true,
				Description: "An array of repository ids that can access the organization secret.",
			},
		},

		CreateContext: resourceGithubAgentsOrganizationSecretRepositoriesCreateOrUpdate,
		ReadContext:   resourceGithubAgentsOrganizationSecretRepositoriesRead,
		UpdateContext: resourceGithubAgentsOrganizationSecretRepositoriesCreateOrUpdate,
		DeleteContext: resourceGithubAgentsOrganizationSecretRepositoriesDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGithubAgentsOrganizationSecretRepositoriesImport,
		},
	}
}

func resourceGithubAgentsOrganizationSecretRepositoriesCreateOrUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	if err := checkOrganization(m); err != nil {
		return diag.FromErr(err)
	}

	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	secretName, _ := d.Get("secret_name").(string)
	repoIDs := []int64{}

	ids, _ := d.Get("selected_repository_ids").(*schema.Set)
	for _, id := range ids.List() {
		repoID, _ := id.(int)
		repoIDs = append(repoIDs, int64(repoID))
	}

	_, err := client.Agents.SetSelectedReposForOrgSecret(ctx, owner, secretName, repoIDs)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(secretName)

	return nil
}

func resourceGithubAgentsOrganizationSecretRepositoriesRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	if err := checkOrganization(m); err != nil {
		return diag.FromErr(err)
	}

	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	secretName, _ := d.Get("secret_name").(string)

	repoIDs := []int64{}
	for repo, err := range client.Agents.ListSelectedReposForOrgSecretIter(ctx, owner, secretName, &github.ListOptions{PerPage: meta.maxPerPage}) {
		if err != nil {
			if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok && ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "Removing agents organization secret repositories from state because the secret no longer exists in GitHub", map[string]any{"secret_name": secretName})
				d.SetId("")
				return nil
			}
			return diag.FromErr(err)
		}

		repoIDs = append(repoIDs, repo.GetID())
	}

	if err := d.Set("selected_repository_ids", repoIDs); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubAgentsOrganizationSecretRepositoriesDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	if err := checkOrganization(m); err != nil {
		return diag.FromErr(err)
	}

	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	_, err := client.Agents.SetSelectedReposForOrgSecret(ctx, owner, d.Id(), []int64{})
	if err != nil {
		if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok && ghErr.Response.StatusCode == http.StatusNotFound {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubAgentsOrganizationSecretRepositoriesImport(ctx context.Context, d *schema.ResourceData, m any) ([]*schema.ResourceData, error) {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	secretName := d.Id()

	if err := d.Set("secret_name", secretName); err != nil {
		return nil, err
	}

	repoIDs := []int64{}
	for repo, err := range client.Agents.ListSelectedReposForOrgSecretIter(ctx, owner, secretName, &github.ListOptions{PerPage: meta.maxPerPage}) {
		if err != nil {
			return nil, err
		}

		repoIDs = append(repoIDs, repo.GetID())
	}

	if err := d.Set("selected_repository_ids", repoIDs); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
