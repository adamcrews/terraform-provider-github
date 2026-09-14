terraform {
  required_providers {
    github = {
      source = "integrations/github"
    }
  }
}

# Auth and owner come from the environment:
#   export GITHUB_TOKEN=...
#   export GITHUB_OWNER=your-org
provider "github" {}
