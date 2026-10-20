# terraform-provider-vapi

```shell
go install
```

## Using the provider

```hcl
terraform {
  required_providers {
    vapi = {
      source  = "kirillve/vapi"
      version = "0.11.0"
    }
  }
}

provider "vapi" {
  url   = "https://api.vapi.ai"
  token = "some-token"
}
```

## Requirements

Go 1.27.2 or newer is required to build the provider.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

## Security notes

Terraform state stores every attribute in plaintext. The assistant attribute `server_url_secret` and the SIP trunk credential attribute `auth_password` are not marked Sensitive yet, so their values can appear in plan output. API error bodies are echoed in diagnostics.
