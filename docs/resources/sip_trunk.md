---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "vapi_sip_trunk Resource - vapi"
subcategory: ""
description: |-
  Manages a SIP trunk in the VAPI system.
---

# vapi_sip_trunk (Resource)

Manages a SIP trunk in the VAPI system.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `gateways` (Attributes List) (see [below for nested schema](#nestedatt--gateways))
- `name` (String)
- `outbound_leading_plus_enabled` (Boolean)
- `sip_provider` (String) The SIP trunk provider identifier (e.g., byo-sip-trunk).

### Optional

- `outbound_authentication_plan` (Attributes) (see [below for nested schema](#nestedatt--outbound_authentication_plan))
- `sip_diversion_header` (String)
- `tech_prefix` (String)

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedatt--gateways"></a>
### Nested Schema for `gateways`

Required:

- `ip` (String)


<a id="nestedatt--outbound_authentication_plan"></a>
### Nested Schema for `outbound_authentication_plan`

Required:

- `auth_password` (String)
- `auth_username` (String)

Optional:

- `sip_register_plan` (Attributes) (see [below for nested schema](#nestedatt--outbound_authentication_plan--sip_register_plan))

<a id="nestedatt--outbound_authentication_plan--sip_register_plan"></a>
### Nested Schema for `outbound_authentication_plan.sip_register_plan`

Required:

- `domain` (String)
- `realm` (String)
- `username` (String)
