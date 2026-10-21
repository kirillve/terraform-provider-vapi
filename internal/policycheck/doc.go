// Command policycheck checks the repository's narrow source-policy contracts.
//
//	policycheck -check testnames|arch -root <module-dir>
//
// Check testnames (`make gate-testnames`): no test, benchmark or fuzz function declared in a
// _test.go file of the module has a name matching ^(Test|Benchmark|Fuzz).*(Calls|Uses).*
// (ast.FuncDecl.Name; case-sensitive). A name like that describes which helper a test reaches, not
// the behaviour it asserts. Hidden, underscore, testdata, scratch (tmp) and nested-module
// directories are not walked.
//
// Check arch (first step of `make lint-arch` and of the pre-commit hook) refuses module source the
// architecture gate (.golangci-arch.yml) cannot see the way the compiler does: ARCH-GLOBMETA (a glob
// metacharacter in the module path, which golangci-lint splices into the depguard exemptions),
// SOURCE-LINK (a symlink, nested repository or special file), NON-GO-SOURCE (assembly, C, syso or
// cgo), WORKSPACE (go.work), NESTED-MODULE (a go.mod below the root other than tools/, the
// documentation generator that holds only a `//go:build generate` file) and, parsing every Go file
// as the compiler does, SEC-A1 (a nolint that silences depguard or forbidigo), SEC-A2 (//permit:),
// GEN-MARKER (a generated-file header), LINE-DIRECTIVE (//line, /*line), LINKNAME (//go:linkname)
// and CANON-IMPORT (an import path not spelled as a plain quoted string: depguard matches the
// spelling, so an escaped or raw-string import would pass it; legal Go, refused).
//
// The gate runs golangci-lint ./... once, on the host, so every module file the toolchain compiles
// must be in that run: arch asks the toolchain itself (`go list -deps -test`) for the host and for
// every goos/goarch pair released by .goreleaser.yml (buildTargets; CGO_ENABLED=0 as goreleaser
// builds) and refuses UNLINTED (compiled for a target but not linted on the host: platform files,
// imported _, . or testdata packages), UNSELECTED (a Go file compiled for no target) and DEP-SOURCE
// (vendored or filesystem-replaced dependency source), and DOMAIN-TRANSITIVE (a non-test package
// under internal/providerdata that links net/http or database/sql through any dependency, e.g.
// expvar -> net/http). buildTargets must match .goreleaser.yml.
//
// LEAKCHECK requires each test package in each build listing to declare TestMain with a call to
// go.uber.org/goleak.VerifyTestMain, checked through the Go AST.
//
// The source checks are syntactic (go/parser, go/ast) and never execute scanned code. A file that
// does not parse fails the run. -root must be a directory that holds go.mod.
//
// Exit codes: 0 clean; 1 violations, one "path:line: rule: message" line each on stdout (line 0 for
// a path-level finding); 2 usage error, unreadable or unparseable source, or a failed go command,
// reported on stderr. Standard library only (depguard rule policycheck-stdlib-only); arch runs the
// toolchain's own go list.
package main
