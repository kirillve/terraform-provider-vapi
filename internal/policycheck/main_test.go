package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_UnknownCheck_Exits2(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/policy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"unknown_check", []string{"-check", "nope", "-root", root}, `unknown -check "nope"`},
		{"missing_check", []string{"-root", root}, `unknown -check ""`},
		{"missing_root", []string{"-check", "testnames"}, "-root is required"},
		{"root_without_go_mod", []string{"-check", "testnames", "-root", t.TempDir()}, "not a module directory"},
		{"extra_argument", []string{"-check", "testnames", "-root", root, "x"}, `unexpected argument "x"`},
		{"undefined_flag", []string{"-bogus"}, "flag provided but not defined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, &stdout, &stderr)
			if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("code=%d stdout=%q stderr=%q; want code 2, empty stdout, stderr containing %q", code, stdout.String(), stderr.String(), tc.want)
			}
		})
	}
}

func TestCLI_UnparseableFile_Exits2(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/policy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "x"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "x", "bad_test.go"), []byte("package x\n\nfunc TestOK( {\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-check", "testnames", "-root", root}, &stdout, &stderr)
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "internal/x/bad_test.go:3:") {
		t.Fatalf("code=%d stdout=%q stderr=%q; want parse error on line 3", code, stdout.String(), stderr.String())
	}
}

func TestCLI_CleanTree_Exits0(t *testing.T) {
	root := t.TempDir()
	for path, src := range map[string]string{
		"go.mod":               "module example.com/policy\n",
		"internal/x/x_test.go": "package x\nfunc TestRecallsList() {}\n",
		"tools/y/y.go":         "package y\nfunc TestFooUsesBar() {}\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-check", "testnames", "-root", root}, &stdout, &stderr)
	if code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q; want clean", code, stdout.String(), stderr.String())
	}
}

func TestCLI_Violation_Exits1WithFileLineRule(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/policy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd", "app"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "app", "app_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestFooUsesBar(t *testing.T) {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-check", "testnames", "-root", root}, &stdout, &stderr)
	const want = "cmd/app/app_test.go:5: testnames: TestFooUsesBar matches ^(Test|Benchmark|Fuzz).*(Calls|Uses).*; name the asserted behaviour instead\n"
	if code != 1 || stderr.Len() != 0 || stdout.String() != want {
		t.Fatalf("code=%d stdout=%q stderr=%q; want code 1 and %q", code, stdout.String(), stderr.String(), want)
	}
}

func TestCLI_ArchCleanModule_Exits0(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/policy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "clean.go"), []byte("package policy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-check", "arch", "-root", root}, &stdout, &stderr)
	if code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q; want clean", code, stdout.String(), stderr.String())
	}
}

func TestCLI_ArchSymlinkedPackageDir_Exits1(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/policy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "internal"), 0o700); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "u.go"), []byte("package u\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "p.go"), []byte("package policy\nimport _ \"example.com/policy/internal/u\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "internal", "u")); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-check", "arch", "-root", root}, &stdout, &stderr)
	if code != 1 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "internal/u:0: SOURCE-LINK:") || !strings.Contains(stdout.String(), "internal/u/u.go:0: UNLINTED:") {
		t.Fatalf("code=%d stdout=%q stderr=%q; want source link violation", code, stdout.String(), stderr.String())
	}
}

func TestCLI_ArchImportedUnderscoreDir_Exits1(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "_hidden"), 0o700); err != nil {
		t.Fatal(err)
	}
	for path, src := range map[string]string{
		"go.mod":       "module example.com/policy\n\ngo 1.27\n",
		"p.go":         "package policy\nimport _ \"example.com/policy/_hidden\"\n",
		"_hidden/p.go": "package hidden\n",
	} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-check", "arch", "-root", root}, &stdout, &stderr)
	if code != 1 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "_hidden/p.go:0: UNLINTED:") {
		t.Fatalf("code=%d stdout=%q stderr=%q; want UNLINTED", code, stdout.String(), stderr.String())
	}
}

func TestParseArgs_KnownChecks_Dispatched(t *testing.T) {
	for _, name := range []string{"testnames", "arch"} {
		t.Run(name, func(t *testing.T) {
			check, root, err := parseArgs([]string{"-check", name, "-root", "mod"}, io.Discard)
			if err != nil || check == nil || root != "mod" {
				t.Fatalf("parseArgs(%s)=(%v,%q,%v); want a check for root mod", name, check != nil, root, err)
			}
		})
	}
}

func TestCLI_ArchFilesystemReplace_Exits1(t *testing.T) {
	root, dep := t.TempDir(), t.TempDir()
	for path, src := range map[string]string{
		filepath.Join(dep, "go.mod"):  "module example.com/dep\n\ngo 1.27\n",
		filepath.Join(dep, "dep.go"):  "package dep\n",
		filepath.Join(root, "go.mod"): "module example.com/policy\n\ngo 1.27\n\nrequire example.com/dep v0.0.0\n\nreplace example.com/dep => " + dep + "\n",
		filepath.Join(root, "p.go"):   "package policy\n\nimport _ \"example.com/dep\"\n",
	} {
		if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr strings.Builder
	code := run([]string{"-check", "arch", "-root", root}, &stdout, &stderr)
	if code != 1 || stderr.Len() != 0 || stdout.String() != "go.mod:0: DEP-SOURCE: filesystem-replaced dependency source\n" {
		t.Fatalf("code=%d stdout=%q stderr=%q; want go.mod:0: DEP-SOURCE", code, stdout.String(), stderr.String())
	}
}

func TestCLI_ArchDomainLinksHTTPTransitively_Exits1(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "providerdata"), 0o700); err != nil {
		t.Fatal(err)
	}
	for path, src := range map[string]string{
		"go.mod":                        "module example.com/policy\n\ngo 1.27\n",
		"internal/providerdata/data.go": "package providerdata\n\nimport _ \"expvar\"\n",
	} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr strings.Builder
	code := run([]string{"-check", "arch", "-root", root}, &stdout, &stderr)
	const want = "internal/providerdata:0: DOMAIN-TRANSITIVE: links net/http for host\n"
	if code != 1 || stderr.Len() != 0 || stdout.String() != want {
		t.Fatalf("code=%d stdout=%q stderr=%q; want code 1 and %q", code, stdout.String(), stderr.String(), want)
	}
}
