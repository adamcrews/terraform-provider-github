package github

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/google/go-github/v92/github"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceGithubAgentsSecret() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGithubAgentsSecretCreate,
		ReadContext:   resourceGithubAgentsSecretRead,
		UpdateContext: resourceGithubAgentsSecretUpdate,
		DeleteContext: resourceGithubAgentsSecretDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGithubAgentsSecretImport,
		},

		CustomizeDiff: customdiff.All(
			diffRepository,
			diffSecret,
		),

		Description: "Resource to manage a GitHub Agents secret for a repository.",

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the repository.",
			},
			"repository_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "ID of the repository.",
			},
			"secret_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateSecretNameFunc,
				Description:      "Name of the secret.",
			},
			"key_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				RequiredWith:  []string{"value_encrypted"},
				ConflictsWith: []string{"value"},
				Description:   "ID of the public key used to encrypt the secret.",
			},
			"value": {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ExactlyOneOf: []string{"value", "value_encrypted"},
				Description:  "Plaintext value to be encrypted.",
			},
			"value_encrypted": {
				Type:             schema.TypeString,
				Optional:         true,
				Sensitive:        true,
				ExactlyOneOf:     []string{"value", "value_encrypted"},
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsBase64),
				Description:      "Value encrypted with the GitHub public key, defined by key_id, in Base64 format.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of when the secret was created.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of when the secret was last updated by the provider.",
			},
			"remote_updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp for when the secret was last updated.",
			},
		},
	}
}

func resourceGithubAgentsSecretCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName, _ := d.Get("repository").(string)
	secretName, _ := d.Get("secret_name").(string)
	keyID, _ := d.Get("key_id").(string)
	encryptedValue, _ := d.Get("value_encrypted").(string)

	repo, _, err := client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		return diag.FromErr(err)
	}
	repoID := int(repo.GetID())

	var publicKey string
	if len(keyID) == 0 || len(encryptedValue) == 0 {
		ki, pk, err := getAgentsPublicKeyDetails(ctx, meta, repoName)
		if err != nil {
			return diag.FromErr(err)
		}

		keyID = ki
		publicKey = pk
	}

	if len(encryptedValue) == 0 {
		plaintextValue, _ := d.Get("value").(string)

		encryptedBytes, err := encryptPlaintext(plaintextValue, publicKey)
		if err != nil {
			return diag.FromErr(err)
		}
		encryptedValue = base64.StdEncoding.EncodeToString(encryptedBytes)
	}

	secretReq := github.SecretRequest{
		KeyID:          keyID,
		EncryptedValue: encryptedValue,
	}

	if _, err = client.Agents.CreateOrUpdateRepoSecret(ctx, owner, repoName, secretName, secretReq); err != nil {
		return diag.FromErr(err)
	}

	id, err := buildID(repoName, secretName)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)

	if err := d.Set("repository_id", repoID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("key_id", keyID); err != nil {
		return diag.FromErr(err)
	}

	// GitHub API does not return on create so we have to lookup the secret to get timestamps.
	if secret, err := retryUntilResourceFound(ctx, func() (*github.Secret, error) {
		val, _, err := client.Agents.GetRepoSecret(ctx, owner, repoName, secretName)
		return val, err
	}, nil); err == nil {
		if err := d.Set("created_at", secret.CreatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("updated_at", secret.UpdatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("remote_updated_at", secret.UpdatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGithubAgentsSecretRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName, _ := d.Get("repository").(string)
	secretName, _ := d.Get("secret_name").(string)

	secret, _, err := client.Agents.GetRepoSecret(ctx, owner, repoName, secretName)
	if err != nil {
		if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok && ghErr.Response.StatusCode == http.StatusNotFound {
			tflog.Info(ctx, "Removing agents secret from state because it no longer exists in GitHub", map[string]any{"secret_name": secretName, "repository": repoName})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	id, err := buildID(repoName, secretName)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)

	// Due to the eventually consistent behavior of this API we may not get created_at/updated_at
	// values on the first read after creation, so we only set them here if they are not already set.
	if createdAt, _ := d.Get("created_at").(string); len(createdAt) == 0 {
		if err = d.Set("created_at", secret.CreatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
	}
	if updatedAt, _ := d.Get("updated_at").(string); len(updatedAt) == 0 {
		if err = d.Set("updated_at", secret.UpdatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
	}
	if err = d.Set("remote_updated_at", secret.UpdatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubAgentsSecretUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName, _ := d.Get("repository").(string)
	secretName, _ := d.Get("secret_name").(string)
	keyID, _ := d.Get("key_id").(string)
	encryptedValue, _ := d.Get("value_encrypted").(string)

	var publicKey string
	if len(keyID) == 0 || len(encryptedValue) == 0 {
		ki, pk, err := getAgentsPublicKeyDetails(ctx, meta, repoName)
		if err != nil {
			return diag.FromErr(err)
		}

		keyID = ki
		publicKey = pk
	}

	if len(encryptedValue) == 0 {
		plaintextValue, _ := d.Get("value").(string)

		encryptedBytes, err := encryptPlaintext(plaintextValue, publicKey)
		if err != nil {
			return diag.FromErr(err)
		}
		encryptedValue = base64.StdEncoding.EncodeToString(encryptedBytes)
	}

	secretReq := github.SecretRequest{
		KeyID:          keyID,
		EncryptedValue: encryptedValue,
	}

	if _, err := client.Agents.CreateOrUpdateRepoSecret(ctx, owner, repoName, secretName, secretReq); err != nil {
		return diag.FromErr(err)
	}

	id, err := buildID(repoName, secretName)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)

	if err := d.Set("key_id", keyID); err != nil {
		return diag.FromErr(err)
	}

	// GitHub API does not return on update so we have to lookup the secret to get timestamps.
	if secret, err := retryUntilResourceFound(ctx, func() (*github.Secret, error) {
		val, _, err := client.Agents.GetRepoSecret(ctx, owner, repoName, secretName)
		return val, err
	}, nil); err == nil {
		if err := d.Set("created_at", secret.CreatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("updated_at", secret.UpdatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("remote_updated_at", secret.UpdatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGithubAgentsSecretDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName, _ := d.Get("repository").(string)
	secretName, _ := d.Get("secret_name").(string)

	tflog.Info(ctx, "Deleting agents repo secret", map[string]any{"secret_name": secretName, "repository": repoName})
	if _, err := client.Agents.DeleteRepoSecret(ctx, owner, repoName, secretName); err != nil {
		if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok && ghErr.Response.StatusCode == http.StatusNotFound {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubAgentsSecretImport(ctx context.Context, d *schema.ResourceData, m any) ([]*schema.ResourceData, error) {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName, secretName, err := parseID2(d.Id())
	if err != nil {
		return nil, err
	}

	repo, _, err := client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	repoID := int(repo.GetID())

	secret, _, err := client.Agents.GetRepoSecret(ctx, owner, repoName, secretName)
	if err != nil {
		return nil, err
	}

	if err := d.Set("repository", repoName); err != nil {
		return nil, err
	}
	if err := d.Set("repository_id", repoID); err != nil {
		return nil, err
	}
	if err := d.Set("secret_name", secretName); err != nil {
		return nil, err
	}
	if err := d.Set("created_at", secret.CreatedAt.String()); err != nil {
		return nil, err
	}
	if err := d.Set("updated_at", secret.UpdatedAt.String()); err != nil {
		return nil, err
	}
	if err := d.Set("remote_updated_at", secret.UpdatedAt.String()); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func getAgentsPublicKeyDetails(ctx context.Context, meta *Owner, repository string) (string, string, error) {
	client := meta.v3client
	owner := meta.name

	publicKey, _, err := client.Agents.GetRepoPublicKey(ctx, owner, repository)
	if err != nil {
		return "", "", err
	}

	return publicKey.GetKeyID(), publicKey.GetKey(), err
}
