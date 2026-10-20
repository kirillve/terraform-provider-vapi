default: fmt lint install generate

# Pinned tool versions: the only place they are defined (CI calls `make verify`).
GOLANGCI_LINT = github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
GOVULNCHECK   = golang.org/x/vuln/cmd/govulncheck@v1.9.0
GITLEAKS      = github.com/zricethezav/gitleaks/v8@v8.30.1
FUZZTIME     ?= 30s

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	go run $(GOLANGCI_LINT) run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

# verify is the single quality entry point; see docs/quality/STANDARD.md. All steps block.
verify: fmt-check vet lint vuln secrets test-race fuzz

fmt-check:
	@out="$$(gofmt -s -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

vuln:
	go run $(GOVULNCHECK) ./...

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

.PHONY: fmt fmt-check vet vuln secrets test-race fuzz hooks verify lint test testacc build install generate
