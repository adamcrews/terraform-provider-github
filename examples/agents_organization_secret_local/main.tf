resource "github_agents_organization_secret" "adam_test" {
  secret_name = "ADAM_TEST"
  value       = "adam_test_value"
  visibility  = "all"
}

data "github_agents_organization_secrets" "all" {
  depends_on = [github_agents_organization_secret.adam_test]
}

output "adam_test_secret_name" {
  value = github_agents_organization_secret.adam_test.secret_name
}

output "adam_test_created_at" {
  value = github_agents_organization_secret.adam_test.created_at
}

output "org_agents_secret_names" {
  value = [for s in data.github_agents_organization_secrets.all.secrets : s.name]
}
