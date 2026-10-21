# Code quality standard

`make verify` is the single quality entry point. CI runs the same target as a blocking job, so a
green local run means a green gate. Every tool is pinned to an exact version in `GNUmakefile`.

## Gates

| Gate | Tool (pinned in `GNUmakefile`) | Threshold |
|---|---|---|
| Formatting | golangci-lint formatters `gofumpt` (extra rules) + `goimports` | no unformatted files |
| Static analysis | `govet` (enable-all minus fieldalignment/shadow), staticcheck, revive (explicit rule list), gocritic (diagnostic, performance, style), errorlint, nilnil, exhaustive, bodyclose, noctx, contextcheck, containedctx, fatcontext, usestdlibvars, errname, nolintlint (explanation and linter name required) | clean |
| Cyclomatic complexity | golangci-lint `gocyclo` | base config fails at 15; `.golangci-complexity.yml` gate fails at 15 or more (min 14), tests excluded |
| Cognitive complexity | golangci-lint `gocognit` | same as above |
| Function size | `funlen` | 60 lines / 50 statements |
| Security | `.golangci-security.yml`: `gosec`, severity and confidence low, audit on, `#nosec` ignored | clean |
| Unhandled errors | `errcheck` | clean |
| Dead code | `unparam`, `ineffassign`, `unused` | clean |
| Duplication | `dupl` | threshold 50, clean |
| Architecture | `make lint-arch`: `policycheck -check arch` (source policy), then `.golangci-arch.yml` (`depguard` layering, tagged `forbidigo`) | clean |
| Test names | `policycheck -check testnames` (`make gate-testnames`) | no `Test*Calls*` / `*Uses*` names |
| SAST | `gosec` | clean |
| Dependency vulnerabilities | `govulncheck` (symbol scan, then `-scan package`), `trivy fs` (vuln, secret, misconfig; waivers in `.trivyignore`) | clean |
| Module integrity | `go mod verify`, `go mod tidy -diff` (`make gate-mod`) | clean |
| Secrets | `gitleaks`: pre-commit hook (`make hooks`) and CI over the PR commit range | no findings |
| Races and order dependence | `go test -race -shuffle=on -count=3 ./...` | pass |
| Goroutine leaks | `goleak.VerifyTestMain` in every test package; `policycheck` LEAKCHECK refuses a test package without it | pass |
| Fuzzing | `go test -fuzz` for each `Fuzz*` target, 30s each (`make fuzz`) | no crash |
| Coverage | `go tool cover -func` | reported, not gated |

There are no size limits (function, file, line or diff length). Split code by meaning.

## Layer rules (enforced by `.golangci-arch.yml`)

- `main.go` imports the standard library, `providerserver` and `internal/provider` only.
- `internal/provider` imports the plugin framework, `internal/providerdata` and `internal/vapiv1`;
  non-test code must not import `internal/vapi` or `net/http`.
- `internal/vapiv1` imports the plugin framework, `terraform-plugin-log`, `internal/vapi` and
  `internal/providerdata`, never `internal/provider`.
- `internal/vapi` imports the standard library only. `internal/providerdata` imports the standard
  library only and, transitively, never `net/http` or `database/sql`.
- Terraform plugin libraries live only in `main.go`, `internal/provider` and `internal/vapiv1`;
  `net/http` only in `internal/vapi` and `internal/vapiv1`; testify and goleak only in `_test.go`.
- `forbidigo` tags: `[prod]` (no `fmt.Print*`, `println`, `http.Get`/`DefaultClient`/`DefaultTransport`
  in production code) and `[entrypoint]` (`os.Exit`, `log.Fatal*`, `context.Background/TODO` only in
  `main.go` and `internal/policycheck/main.go`).
- A `//nolint` naming `depguard` or `forbidigo` (or bare) is refused by `policycheck`; the layout
  change goes into `.golangci-arch.yml` in the same commit.

### Source policy (`internal/policycheck`)

`policycheck` is a standard-library-only command in the root module, so it is built, linted and
tested like the provider (the separate `tools/` module holds only the documentation generator and
is the one nested module it tolerates). `-check arch` refuses source the architecture lint cannot
see as the compiler does (nolint silencing, `//permit:`, generated headers, `//line`,
`//go:linkname`, non-canonical imports, symlinks, assembly/cgo, `go.work`, other nested modules),
files compiled for a release target but not linted on the host, files compiled nowhere, and test
packages without `goleak.VerifyTestMain`. Release targets mirror `.goreleaser.yml`
(`freebsd`, `windows`, `linux`, `darwin`; `windows/arm` is skipped because the toolchain cannot
list it without cgo).

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
# trivy must be on PATH for `make verify` (brew install trivy); CI installs the pinned version
make verify   # run every gate
```

## Previous state

Before this standard the repository only ran `golangci-lint` (errcheck, staticcheck, unparam,
unused and a few small linters) and Terraform acceptance tests in CI. There was no complexity,
duplication, SAST, vulnerability, secret, race, leak or fuzz gate. A local-only lint config with
`gocognit` at 20 existed outside version control; the committed threshold is 15.
