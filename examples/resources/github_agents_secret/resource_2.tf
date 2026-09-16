# Encrypted Secret Example

data "github_agents_public_key" "example" {
  repository = "example-repo"
}

resource "github_agents_secret" "example" {
  repository      = "example-repo"
  secret_name     = "EXAMPLE_SECRET_NAME"
  key_id          = data.github_agents_public_key.example.key_id
  value_encrypted = var.value_encrypted
}
