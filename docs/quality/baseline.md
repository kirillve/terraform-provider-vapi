# Quality baseline

Existing violations accepted when `make verify` was introduced. Each has a targeted exclusion
with a reason in `.golangci.yml`. Fix them by refactoring, then delete the exclusion and the row.

| ID | Gate | Location | Finding |
|---|---|---|---|
| B1 | dupl | `internal/vapiv1/vapi_assistant_resource.go:543` | Create handler duplicates `vapi_sip_trunk_phone_number_resource.go:104` |
| B1 | dupl | `internal/vapiv1/vapi_sip_trunk_phone_number_resource.go:104` | Create handler duplicates `vapi_twilio_phone_number_resource.go:152` |
| B1 | dupl | `internal/vapiv1/vapi_twilio_phone_number_resource.go:152` | Create handler duplicates `vapi_assistant_resource.go:543` |
| B2 | gocognit | `internal/provider/tfvalue_test.go:26` | `jsonToValue`, complexity 16 |
| B3 | gocognit | `internal/provider/tfvalue_test.go:187` | `tfFingerprintWalk`, complexity 18 |
| B4 | gocognit | `internal/vapiv1/vapi_assistant_resource.go:659` | `flatten`, complexity 18 |

Counts per gate: dupl 3, gocognit 3, gocyclo 0, gosec 0, depguard 0, errcheck 0, staticcheck 0,
unparam 0, ineffassign 0, unused 0, govulncheck 0, gitleaks 0.
