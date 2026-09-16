# Un-encrypted Secret Example

resource "github_agents_secret" "example" {
  repository  = "example-repo"
  secret_name = "EXAMPLE_SECRET_NAME"
  value       = "example-value"
}
