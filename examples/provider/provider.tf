terraform {
  required_providers {
    vapi = {
      source  = "kirillve/vapi"
      version = "0.9.0"
    }
  }
}

provider "vapi" {
  url   = "https://api.vapi.ai"
  token = "some-token"
}
