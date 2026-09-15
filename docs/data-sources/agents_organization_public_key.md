---
page_title: "github_agents_organization_public_key (Data Source) - GitHub"
description: |-
  Get information on a GitHub Agents Organization Public Key.
---

# github_agents_organization_public_key (Data Source)

Use this data source to retrieve information about a GitHub Agents Organization public key. This data source is required to be used with other GitHub secrets interagents. Note that the provider `token` must have admin rights to an organization to retrieve it's action public key.

## Example Usage

```terraform
data "github_agents_organization_public_key" "example" {}
```

## Attributes Reference

- `key_id` - ID of the key that has been retrieved.
- `key` - Actual key retrieved.
