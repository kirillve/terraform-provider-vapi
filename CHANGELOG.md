## Unreleased

CHANGES:
- a refresh that finds the object missing in VAPI (HTTP 404) now removes the resource from the state instead of failing, for `vapi_assistant`, `vapi_file`, `vapi_twilio_phone_number`, `vapi_sip_trunk_phone_number` and `vapi_tool_function`; the same applies to an import of the four that support it (`vapi_sip_trunk` and `vapi_tool_query_function` already did)
- destroying a resource that is already missing in VAPI (HTTP 404 on delete) now succeeds, for all seven resources
- the provider reports the version of the release it was built from instead of a fixed 0.11.0, and `go install` of a tagged release reports that tag; a build without release information reports dev
- updating a `vapi_file` whose previous file is already missing in VAPI (HTTP 404 on the delete that precedes the upload) now uploads the new content instead of failing; any other delete failure still fails the update
- changing an argument of a `vapi_sip_trunk_phone_number` now plans an in-place update; before, the computed `id` forced a replacement of the phone number
- `vapi_file` no longer plans a replacement because of its computed `id` and `org_id`; changing `content` or `filename` still replaces the file
- updating a `vapi_tool_query_function` no longer fails with an unknown `id` and `org_id` after apply; the plans of `vapi_assistant` and `vapi_sip_trunk` keep the existing `id` instead of showing it as known after apply
- creating or updating a `vapi_assistant` without `phone_number_id` no longer fails with an unknown value after apply; the attribute stays null until it is set

## v0.14.0

CHANGES:
- added optional server_url for Twilio phone numbers

## v0.13.0

CHANGES:
- added optional sms_enabled for Twilio phone numbers with in-place updates

## v0.12.0

CHANGES:
- added tests
- added parameters_json for nested JSON schema tool parameters
- added server_timeout_seconds for function tools

## v0.11.0

CHANGES:
  - added SIP trunk phone number support

## v0.10.0

CHANGES:
  - added SIP trunk support

## v0.9.5

CHANGES:
  - fixed tool provider bug

## v0.9.4

CHANGES:
  - fixed tool provider bug

## v0.9.3

CHANGES:
  - fixed tool provider bug

## v0.9.2

CHANGES:
  - fixed tool provider bug

## v0.9.1

CHANGES:
  - fixed assistant provider bug

## v0.9.0

CHANGES:
  - added extension to destinations 
  - fixed assistant provider bug

## v0.8.3

CHANGES:
  - fixed tool query 

## v0.8.2

CHANGES:
  - added artifacts_plan with recording format 

## v0.8.1

CHANGES:
  - hot-fixed AI Assistant Server URL, Secret, and TimeoutSeconds 

## v0.8.0

CHANGES:
  - added tool transferCall support

## v0.7.0

CHANGES:
  - added tool query support
  - fixed AI Assistant creation issue

## v0.6.1

CHANGES:
  - minor fix to avoid unnecessary alerts

## v0.6.0

CHANGES:
  - added DTMF function support

## v0.5.10

FIXES:
  - fixed VAPI Tools update

## v0.5.9

FIXES:
  - fixed VAPI Assistant update support

## v0.5.8

FIXES:
  - fixed VAPI Assistant update support

## v0.5.7

FIXES:
  - fixed VAPI Assistant update support

## v0.5.6

FIXES:
  - fixed VAPI Assistant update support

## v0.5.5

FIXES:
  - fixed VAPI Assistant update support

## v0.5.4

FIXES:
  - fixed VAPI Twilio AssistantID support

## v0.5.3

FIXES:
  - fixed VAPI Twilio AssistantID support

## v0.5.2

FEATURES:
  - added VAPI Twilio AssistantID support

## v0.5.1

FIXES:
  - fixed VAPI Assistant support

## v0.5.0

FEATURES:
  - added VAPI Assistant support

## v0.4.0

FEATURES:
  - added VAPI tool support (function call only)

## v0.3.7

FIXES:
  - improved VAPI files support
  - improved VAPI Twilio Phone Number import with fallback implemented

## v0.3.6

FIXES:
  - improved VAPI files support - issue when target file manually deleted from the VAPI service

## v0.3.5

FIXES:
  - improved VAPI Twilio Phone Number import with fallback implemented

## v0.3.4

FIXES:
  - improved VAPI files support - issue when target file manually deleted from the VAPI service

## v0.3.3

FIXES:
  - improved VAPI files support - issue when target file manually deleted from the VAPI service
  - improved VAPI Twilio Phone Number import - no fallback implemented

## v0.3.2

FIXES:
  - fixed VAPI files support - issue when target file manually deleted from the VAPI service

## v0.3.1

FIXES:
  - fixed VAPI files support - issue when target file manually deleted from the VAPI service

## v0.3.0

FEATURES:
  - added VAPI Twilio Phone Number import - no fallback implemented

FIXES:
  - fixed VAPI files support - issue when target file manually deleted from the VAPI service

## v0.2.0

FEATURES:
  - added VAPI files support -- buggy as hell
