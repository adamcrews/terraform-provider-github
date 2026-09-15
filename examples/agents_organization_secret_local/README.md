# Local test: org Agents secret `ADAM_TEST`

Creates an organization-level GitHub Agents secret named `ADAM_TEST` with value `adam_test_value`.

## Run

Copy/paste:

```bash
cd ~/sandbox/personal/terraform-provider-github-agents-secrets
go build -o ~/go/bin/terraform-provider-github .

export TF_CLI_CONFIG_FILE="$PWD/examples/dev.tfrc"
export GITHUB_TOKEN="$(gh auth token)"
export GITHUB_OWNER=lovevery-digital

cd examples/agents_organization_secret_local
terraform init
terraform plan
terraform apply
```

You should see the `Provider development overrides are in effect` warning, confirming the local binary is used.

## Cleanup

```bash
cd ~/sandbox/personal/terraform-provider-github-agents-secrets/examples/agents_organization_secret_local
export TF_CLI_CONFIG_FILE="$HOME/sandbox/personal/terraform-provider-github-agents-secrets/examples/dev.tfrc"
export GITHUB_TOKEN="$(gh auth token)"
export GITHUB_OWNER=lovevery-digital
terraform destroy
```
