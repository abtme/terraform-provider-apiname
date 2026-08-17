# Terraform Provider for ApiName / DomainNameAPI

A [Terraform](https://www.terraform.io) provider for managing domain
nameservers via the [DomainNameAPI reseller API](https://www.domainnameapi.com/api-integration)
(branded "ApiName").

## Using the provider

```hcl
terraform {
  required_providers {
    apiname = {
      source = "abtme/apiname"
    }
  }
}

provider "apiname" {
  # reseller_id/api_key can also be set via the APINAME_RESELLER_ID/APINAME_API_KEY environment variables
  reseller_id = "your-reseller-id"
  api_key     = "your-api-key"
}

resource "apiname_domain_nameservers" "example" {
  domain_name = "example.com"
  nameservers = ["ns1.yourdns.com", "ns2.yourdns.com"]
}
```

Full documentation, including the provider schema and resource import
syntax, is published on the
[Terraform Registry](https://registry.terraform.io/providers/abtme/apiname/latest/docs).

## Resources

| Resource | Description |
|---|---|
| [`apiname_domain_nameservers`](docs/resources/domain_nameservers.md) | Manages the nameservers for an existing domain registration. |

## Developing the provider

Requires [Go](https://go.dev/) (see `go.mod` for the version) and
[Terraform](https://www.terraform.io/downloads) locally.

```shell
go build ./...
```

### Generating docs

Documentation under `docs/` is generated from the provider's schema plus
the example `.tf` files in `examples/`, via
[tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs):

```shell
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate
```

### Releasing

Pushing a `v*` tag triggers `.github/workflows/release.yml`, which builds
and signs release artifacts with [GoReleaser](https://goreleaser.com/) and
publishes a GitHub Release. The Terraform Registry picks up new versions
automatically once connected.

## License

See [LICENSE](LICENSE).
