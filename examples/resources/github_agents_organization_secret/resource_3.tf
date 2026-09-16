# Ignore Drift Secret Example

resource "github_agents_organization_secret" "example" {
  secret_name = "EXAMPLE_SECRET_NAME"
  value       = "example-value"
  visibility  = "all"

  lifecycle {
    ignore_changes = [updated_at]
  }
}
