package github

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/google/go-github/v92/github"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceGithubAgentsOrganizationSecret() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGithubAgentsOrganizationSecretCreate,
		ReadContext:   resourceGithubAgentsOrganizationSecretRead,
		UpdateContext: resourceGithubAgentsOrganizationSecretUpdate,
		DeleteContext: resourceGithubAgentsOrganizationSecretDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGithubAgentsOrganizationSecretImport,
		},

		CustomizeDiff: diffSecret,

		Description: "Resource to manage a GitHub Agents secret for an organization.",

		Schema: map[string]*schema.Schema{
			"secret_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Name of the secret.",
				ValidateDiagFunc: validateSecretNameFunc,
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
			"visibility": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"all", "private", "selected"}, false)),
				Description:      "Configures the access that repositories have to the organization secret. Must be one of 'all', 'private', or 'selected'.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp for when the secret was created.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp for when the secret was last updated by the provider.",
			},
			"remote_updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp for when the secret was last updated.",
			},
		},
	}
}

func resourceGithubAgentsOrganizationSecretCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	secretName, _ := d.Get("secret_name").(string)
	keyID, _ := d.Get("key_id").(string)
	encryptedValue, _ := d.Get("value_encrypted").(string)
	visibility, _ := d.Get("visibility").(string)

	var publicKey string
	if len(keyID) == 0 || len(encryptedValue) == 0 {
		ki, pk, err := getAgentsOrganizationPublicKeyDetails(ctx, meta)
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

	secretReq := github.SecretOrgRequest{
		KeyID:          keyID,
		EncryptedValue: encryptedValue,
		Visibility:     visibility,
	}

	if _, err := client.Agents.CreateOrUpdateOrgSecret(ctx, owner, secretName, secretReq); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(secretName)

	if err := d.Set("key_id", keyID); err != nil {
		return diag.FromErr(err)
	}

	// GitHub API does not return on create so we have to lookup the secret to get timestamps.
	if secret, err := retryUntilResourceFound(ctx, func() (*github.Secret, error) {
		val, _, err := client.Agents.GetOrgSecret(ctx, owner, secretName)
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

func resourceGithubAgentsOrganizationSecretRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	secretName, _ := d.Get("secret_name").(string)

	secret, _, err := client.Agents.GetOrgSecret(ctx, owner, secretName)
	if err != nil {
		if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok && ghErr.Response.StatusCode == http.StatusNotFound {
			tflog.Info(ctx, "Removing agents organization secret from state because it no longer exists in GitHub", map[string]any{"secret_name": secretName})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

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
	if err = d.Set("visibility", secret.Visibility); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubAgentsOrganizationSecretUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	secretName, _ := d.Get("secret_name").(string)
	keyID, _ := d.Get("key_id").(string)
	encryptedValue, _ := d.Get("value_encrypted").(string)
	visibility, _ := d.Get("visibility").(string)

	var publicKey string
	if len(keyID) == 0 || len(encryptedValue) == 0 {
		ki, pk, err := getAgentsOrganizationPublicKeyDetails(ctx, meta)
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

	secretReq := github.SecretOrgRequest{
		KeyID:          keyID,
		EncryptedValue: encryptedValue,
		Visibility:     visibility,
	}

	if _, err := client.Agents.CreateOrUpdateOrgSecret(ctx, owner, secretName, secretReq); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("key_id", keyID); err != nil {
		return diag.FromErr(err)
	}

	// GitHub API does not return on update so we have to lookup the secret to get timestamps.
	if secret, err := retryUntilResourceFound(ctx, func() (*github.Secret, error) {
		val, _, err := client.Agents.GetOrgSecret(ctx, owner, secretName)
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

func resourceGithubAgentsOrganizationSecretDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	secretName, _ := d.Get("secret_name").(string)

	tflog.Info(ctx, "Deleting agents organization secret", map[string]any{"secret_name": secretName})

	if _, err := client.Agents.DeleteOrgSecret(ctx, owner, secretName); err != nil {
		if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok && ghErr.Response.StatusCode == http.StatusNotFound {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubAgentsOrganizationSecretImport(ctx context.Context, d *schema.ResourceData, m any) ([]*schema.ResourceData, error) {
	meta, _ := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	secretName := d.Id()

	secret, _, err := client.Agents.GetOrgSecret(ctx, owner, secretName)
	if err != nil {
		return nil, err
	}

	if err := d.Set("secret_name", secretName); err != nil {
		return nil, err
	}
	if err := d.Set("visibility", secret.Visibility); err != nil {
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

func getAgentsOrganizationPublicKeyDetails(ctx context.Context, meta *Owner) (string, string, error) {
	client := meta.v3client
	owner := meta.name

	publicKey, _, err := client.Agents.GetOrgPublicKey(ctx, owner)
	if err != nil {
		return "", "", err
	}

	return publicKey.GetKeyID(), publicKey.GetKey(), err
}
