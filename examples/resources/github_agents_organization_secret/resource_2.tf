# Encrypted Secret Example

data "github_agents_organization_public_key" "example" {}

resource "github_agents_organization_secret" "example" {
  secret_name     = "EXAMPLE_SECRET_NAME"
  key_id          = data.github_agents_organization_public_key.example.key_id
  value_encrypted = var.value_encrypted
  visibility      = "all"
}
