# Code quality standard

`make verify` is the single quality entry point. CI runs the same target as a blocking job, so a
green local run means a green gate. Every tool is pinned to an exact version in `GNUmakefile`.

## Gates

| Gate | Tool (pinned in `GNUmakefile`) | Threshold |
|---|---|---|
| Formatting | `gofmt -s -l` | no unformatted files |
| Static analysis | `go vet`, staticcheck, revive (golint successor, default rules) | clean |
| Cyclomatic complexity | golangci-lint `gocyclo` | fail above 15 |
| Cognitive complexity | golangci-lint `gocognit` | fail above 15 |
| Unhandled errors | `errcheck` | clean |
| Dead code | `unparam`, `ineffassign`, `unused` | clean |
| Duplication | `dupl` | clean |
| Layer boundaries | `depguard` (see below) | clean |
| SAST | `gosec` | clean |
| Dependency vulnerabilities | `govulncheck` | clean |
| Secrets | `gitleaks`: pre-commit hook (`make hooks`) and CI over the PR commit range | no findings |
| Races and order dependence | `go test -race -shuffle=on -count=3 ./...` | pass |
| Goroutine leaks | `goleak.VerifyTestMain` in every test package | pass |
| Fuzzing | `go test -fuzz` for each `Fuzz*` target, 30s each (`make fuzz`) | no crash |
| Coverage | `go tool cover -func` | reported, not gated |

There are no size limits (function, file, line or diff length). Split code by meaning.

## Layer rules (enforced by depguard)

- `main` imports `internal/provider` only.
- `internal/provider` imports `internal/providerdata` and `internal/vapiv1`; non-test code must
  not import `internal/vapi`.
- `internal/vapiv1` imports `internal/vapi` and `internal/providerdata`, never `internal/provider`.
- `internal/vapi` and `internal/providerdata` import no other internal package.

## Fuzz targets

Every parser of external input has a fuzz target (`internal/vapi/fuzz_test.go`): API response
decoding through `APIClient.Do`, `FileResponse.UnmarshalJSON` and
`ResponseFunctionParams.UnmarshalJSON`. New parsers of API responses need a target too.
Test-harness value conversion is test code and is not fuzzed.

## Not applicable

- Memory limit (`GOMEMLIMIT`): the provider is a short-lived Terraform plugin process, not a
  container with a memory limit.
- HTTP service baseline (timeouts, `/livez`, `/readyz`, rate limits, graceful shutdown): the
  provider is not a deployable HTTP service; it only acts as an HTTP client.

## Baseline

Violations that existed when the gates were introduced are listed in
[baseline.md](baseline.md) and excluded in `.golangci.yml` one by one (path, linter, text and a
reason). New and changed code gets no exclusions; bare `//nolint` is not allowed.

## Setup

```bash
make hooks    # enable the gitleaks pre-commit hook
make verify   # run every gate
```

## Previous state

Before this standard the repository only ran `golangci-lint` (errcheck, staticcheck, unparam,
unused and a few small linters) and Terraform acceptance tests in CI. There was no complexity,
duplication, SAST, vulnerability, secret, race, leak or fuzz gate. A local-only lint config with
`gocognit` at 20 existed outside version control; the committed threshold is 15.
