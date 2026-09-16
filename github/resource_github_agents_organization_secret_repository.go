package github

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/go-github/v92/github"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceGithubAgentsOrganizationSecretRepository() *schema.Resource {
	return &schema.Resource{
		Description: "Adds a repository to the allow list of a GitHub Agents organization secret.",

		Schema: map[string]*schema.Schema{
			"secret_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateSecretNameFunc,
				Description:      "Name of the existing secret.",
			},
			"repository_id": {
				Type:             schema.TypeInt,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
				Description:      "The repository ID that can access the organization secret.",
			},
		},

		CreateContext: resourceGithubAgentsOrganizationSecretRepositoryCreate,
		ReadContext:   resourceGithubAgentsOrganizationSecretRepositoryRead,
		DeleteContext: resourceGithubAgentsOrganizationSecretRepositoryDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGithubAgentsOrganizationSecretRepositoryImport,
		},
	}
}

func resourceGithubAgentsOrganizationSecretRepositoryCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	if err := checkOrganization(m); err != nil {
		return diag.FromErr(err)
	}

	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	secretName, _ := d.Get("secret_name").(string)
	repoIDInt, _ := d.Get("repository_id").(int)
	repoID := int64(repoIDInt)

	if _, err := client.Agents.AddSelectedRepoToOrgSecret(ctx, owner, secretName, repoID); err != nil {
		return diag.FromErr(err)
	}

	id, err := buildID(secretName, strconv.Itoa(repoIDInt))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)

	return nil
}

func resourceGithubAgentsOrganizationSecretRepositoryRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	if err := checkOrganization(m); err != nil {
		return diag.FromErr(err)
	}

	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	secretName, _ := d.Get("secret_name").(string)
	repoIDInt, _ := d.Get("repository_id").(int)
	repoID := int64(repoIDInt)

	for repo, err := range client.Agents.ListSelectedReposForOrgSecretIter(ctx, owner, secretName, &github.ListOptions{PerPage: meta.maxPerPage}) {
		if err != nil {
			if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok && ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "Removing agents organization secret repository association from state because the secret no longer exists in GitHub", map[string]any{"secret_name": secretName, "repository_id": repoIDInt})
				d.SetId("")
				return nil
			}
			return diag.FromErr(err)
		}

		if repo.GetID() == repoID {
			return nil
		}
	}

	tflog.Info(ctx, "Removing agents organization secret repository association from state because it no longer exists in GitHub", map[string]any{"secret_name": secretName, "repository_id": repoIDInt})
	d.SetId("")

	return nil
}

func resourceGithubAgentsOrganizationSecretRepositoryDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	if err := checkOrganization(m); err != nil {
		return diag.FromErr(err)
	}

	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	secretName, _ := d.Get("secret_name").(string)
	repoIDInt, _ := d.Get("repository_id").(int)
	repoID := int64(repoIDInt)

	if _, err := client.Agents.RemoveSelectedRepoFromOrgSecret(ctx, owner, secretName, repoID); err != nil {
		if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok && ghErr.Response.StatusCode == http.StatusNotFound {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubAgentsOrganizationSecretRepositoryImport(ctx context.Context, d *schema.ResourceData, _ any) ([]*schema.ResourceData, error) {
	secretName, repoIDStr, err := parseID2(d.Id())
	if err != nil {
		return nil, err
	}

	repoID, err := strconv.Atoi(repoIDStr)
	if err != nil {
		return nil, err
	}

	if err := d.Set("secret_name", secretName); err != nil {
		return nil, err
	}
	if err := d.Set("repository_id", repoID); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
