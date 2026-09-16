resource "github_agents_organization_secret" "example" {
  secret_name = "EXAMPLE_SECRET_NAME"
  value       = "example-value"
  visibility  = "selected"
}

resource "github_repository" "example" {
  name       = "example-repo"
  visibility = "public"
}

resource "github_agents_organization_secret_repositories" "example" {
  secret_name             = github_agents_organization_secret.example.secret_name
  selected_repository_ids = [github_repository.example.repo_id]
}
