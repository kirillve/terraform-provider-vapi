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

## Resources deleted outside Terraform

When VAPI answers HTTP 404 (not found) to the read of a resource, the provider removes the resource from the Terraform state, so the next plan proposes to create it again. When VAPI answers 404 to a delete, the resource counts as deleted and the destroy succeeds. Only an exact 404 does this: a 401, 403 or 5xx response is still an error and never drops state.

If the provider `url` points at a server that answers 404 to every request, a refresh removes resources from the state while the objects stay in VAPI, and `terraform destroy -refresh=false` reports success without deleting anything. Check the `url` before you apply.

## Security notes

Terraform state stores every attribute in plaintext. The assistant attribute `server_url_secret` and the SIP trunk credential attribute `auth_password` are not marked Sensitive yet, so their values can appear in plan output. API error bodies are echoed in diagnostics.
