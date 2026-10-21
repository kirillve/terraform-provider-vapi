# Quality baseline

Existing violations accepted when `make verify` was introduced. Each has a targeted exclusion
with a reason in `.golangci.yml`. Fix them by refactoring, then delete the exclusion and the row.

| ID | Gate | Location | Finding |
|---|---|---|---|
| B1 | dupl | `internal/vapiv1/vapi_*_resource.go` (19 blocks, 9-16 lines each) | CRUD handlers repeat one request/decode-error shape across the v1 resources; dupl threshold is 50 tokens (reference parity). Fix by extracting shared handler helpers |
| B2 | gocognit | `internal/provider/tfvalue_test.go:26` | `jsonToValue`, complexity 16 |
| B3 | gocognit | `internal/provider/tfvalue_test.go:187` | `tfFingerprintWalk`, complexity 18 |
| B4 | funlen | `internal/vapiv1/vapi_*_resource.go` | `Schema` methods (78-374 lines): declarative Terraform schemas, splitting adds no clarity |
| B5 | errorlint | `internal/vapi/vapi-client.go` (`UploadData`) | transport error is formatted with `%v` on purpose: `TestUploadData_TransportErrorText` pins it as unwrapped text |

Counts per gate: dupl 19 blocks (B1), funlen 5 (B4), gocognit 2, errorlint 1, gocyclo 0, gosec 0,
errcheck 0, staticcheck 0, unparam 0, ineffassign 0, unused 0, govulncheck 0, gitleaks 0.
