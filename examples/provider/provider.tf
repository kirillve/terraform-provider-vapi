terraform {
  required_providers {
    vapi = {
      source  = "kirillve/vapi"
      version = "0.3.4"
    }
  }
}

provider "vapi" {
  url   = "https://api.vapi.ai"
  token = "some-token"
}
