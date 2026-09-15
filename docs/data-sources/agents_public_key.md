---
page_title: "github_agents_public_key (Data Source) - GitHub"
description: |-
  Get information on a GitHub Agents Public Key.
---

# github_agents_public_key (Data Source)

Use this data source to retrieve information about a GitHub Agents public key. This data source is required to be used with other GitHub secrets interagents. Note that the provider `token` must have admin rights to a repository to retrieve it's action public key.

## Example Usage

```terraform
data "github_agents_public_key" "example" {
  repository = "example_repo"
}
```

## Argument Reference

- `repository` - (Required) Name of the repository to get public key from.

## Attributes Reference

- `key_id` - ID of the key that has been retrieved.
- `key` - Actual key retrieved.
