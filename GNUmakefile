default: fmt lint install generate

# Pinned tool versions: the only place they are defined (CI calls `make verify`).
GOLANGCI_LINT = github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
GOVULNCHECK   = golang.org/x/vuln/cmd/govulncheck@v1.9.0
GITLEAKS      = github.com/zricethezav/gitleaks/v8@v8.30.1
# Binary tool (not go-runnable): CI installs exactly this version; locally it must be on PATH.
TRIVY_VERSION = 0.76.0
FUZZTIME     ?= 30s

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	go run $(GOLANGCI_LINT) run

# Architecture gate: source policy first (policycheck refuses source the gate cannot see), then
# depguard layering + tagged forbidigo rules from .golangci-arch.yml.
lint-arch: gate-testnames
	go run -buildvcs=false ./internal/policycheck -check arch -root .
	go run $(GOLANGCI_LINT) run -c .golangci-arch.yml

lint-security:
	go run $(GOLANGCI_LINT) run -c .golangci-security.yml

lint-complexity:
	go run $(GOLANGCI_LINT) run -c .golangci-complexity.yml

# No test, benchmark or fuzz function name contains Calls or Uses (internal/policycheck).
gate-testnames:
	go run -buildvcs=false ./internal/policycheck -check testnames -root .

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

# verify is the single quality entry point; see docs/quality/STANDARD.md. All steps block.
verify: fmt-check gate-mod vet lint lint-arch lint-security lint-complexity vuln trivy-fs secrets test-race fuzz

fmt-check:
	@out="$$(gofmt -s -l $$(git ls-files '*.go'))"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

# Symbol scan proves nothing reachable is affected; the package scan fails on ANY known
# vulnerability in an imported package or the standard library, reachable or not.
vuln:
	go run $(GOVULNCHECK) ./...
	go run $(GOVULNCHECK) -scan package ./...

# Module checksums match go.sum and go.mod/go.sum are tidy.
gate-mod:
	go mod verify
	go mod tidy -diff

# Vulnerability, secret and misconfiguration scan of the tree, all severities. Waivers live in
# .trivyignore (dated and justified); tmp/ is gitignored scratch, not part of the tree. `make print-trivy-version` feeds the CI install step.
trivy-fs:
	@command -v trivy >/dev/null 2>&1 || { echo "trivy-fs: FAIL: trivy not installed (want $(TRIVY_VERSION)): 'brew install trivy' or https://trivy.dev/docs/latest/getting-started/installation/" >&2; exit 1; }
	trivy fs --quiet --exit-code 1 --scanners vuln,secret,misconfig --ignorefile .trivyignore --skip-dirs tmp .

print-trivy-version:
	@echo $(TRIVY_VERSION)

secrets:
	go run $(GITLEAKS) git --redact --no-banner .

# Race detector, random order, 3 runs; coverage is reported, not gated.
test-race:
	go test -race -shuffle=on -count=3 -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

fuzz:
	FUZZTIME=$(FUZZTIME) ./scripts/fuzz.sh

hooks:
	git config core.hooksPath scripts/hooks

.PHONY: lint-arch lint-security lint-complexity gate-testnames gate-mod trivy-fs print-trivy-version fmt fmt-check vet vuln secrets test-race fuzz hooks verify lint test testacc build install generate
