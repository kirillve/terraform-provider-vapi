package main

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWalkTree_LinksReposNativeAndWorkspace_Reported(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"ok.go", ".git/config", "nested/.git/config", "nested/ok.go", "go.work", "go.work.sum", "native.s", "native.syso", "tmp/scratch.go", "tools/tools.go", "tools/go.mod", "other/go.mod", "other/o.go"} {
		name := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for name, target := range map[string]string{"link.go": "ok.go", "dirlink": "nested"} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	files, findings, err := walkTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(files, []string{"nested/ok.go", "ok.go"}) {
		t.Fatalf("files=%v; want only regular Go files", files)
	}
	want := map[string]string{
		"dirlink": "SOURCE-LINK", "link.go": "SOURCE-LINK", "nested/.git": "SOURCE-LINK",
		"other": "NESTED-MODULE", "go.work": "WORKSPACE", "go.work.sum": "WORKSPACE", "native.s": "NON-GO-SOURCE", "native.syso": "NON-GO-SOURCE",
	}
	if len(findings) != len(want) {
		t.Fatalf("findings=%+v; want %v", findings, want)
	}
	for _, f := range findings {
		if rule, ok := want[f.path]; !ok || f.rule != rule || f.line != 0 {
			t.Fatalf("unexpected finding %+v; want %v", f, want)
		}
		delete(want, f.path)
	}
}

func TestRootFindings_GlobMetaInPath_Reported(t *testing.T) {
	root := filepath.Join(t.TempDir(), "a*b")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	findings, err := rootFindings(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings=%+v; want two glob path findings", findings)
	}
	for _, f := range findings {
		if f.path != "." || f.line != 0 || f.rule != "ARCH-GLOBMETA" {
			t.Fatalf("finding=%+v; want .:0: ARCH-GLOBMETA", f)
		}
	}
}

func TestCheckArch_SameListingEverywhere_Clean(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "p.go"), []byte("package p\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	list := func(context.Context, string, target) ([]byte, error) {
		return []byte("P p\x00N p\x00D " + root + "\x00F main\x00G p.go\x00E\x00\n"), nil
	}
	got, err := checkArch(t.Context(), root, list)
	if err != nil || len(got) != 0 {
		t.Fatalf("findings=%+v err=%v; want clean", got, err)
	}
	t.Run("goleak_test_main", func(t *testing.T) {
		root := t.TempDir()
		for name, source := range map[string]string{
			"go.mod":    "module example.com/p\n\ngo 1.27\n\nrequire go.uber.org/goleak v1.3.0\n",
			"go.sum":    "go.uber.org/goleak v1.3.0 h1:2K3zAYmnTNqV73imy9J1T3WC+gmCePx2hEGkimedGto=\ngo.uber.org/goleak v1.3.0/go.mod h1:CoHD4mav9JJNrW/WLlf7HGZPjdw8EucARQHekz1X6bE=\n",
			"p.go":      "package p\n",
			"p_test.go": "package p_test\n\nimport (\"testing\"; \"go.uber.org/goleak\")\n\nfunc TestMain(m *testing.M) { goleak.VerifyTestMain(m) }\n",
		} {
			if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		got, err := checkArch(t.Context(), root, goList)
		if err != nil || len(got) != 0 {
			t.Fatalf("findings=%+v err=%v; want test package with goleak clean", got, err)
		}
	})
}

func TestCheckArch_TargetOnlyFile_ReportsUnlinted(t *testing.T) {
	var all []string
	for _, tg := range buildTargets {
		all = append(all, tg.name)
	}
	for _, name := range all {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			var calls []string
			got, err := checkArch(t.Context(), root, onlyOnTarget(name, root, &calls))
			if err != nil || !slices.Equal(calls, all) {
				t.Fatalf("findings=%+v err=%v targets=%v", got, err, calls)
			}
			// A host-only file is linted, so nothing is reported for it.
			if wantFindings := name != hostTarget; len(got) != btoi(wantFindings) {
				t.Fatalf("findings=%+v; want %d for %s", got, btoi(wantFindings), name)
			}
			if len(got) == 1 && (got[0].path != "p.go" || got[0].rule != "UNLINTED" || !strings.Contains(got[0].msg, name)) {
				t.Fatalf("findings=%+v; want p.go UNLINTED for %s", got, name)
			}
		})
	}
}

// onlyOnTarget lists p.go on the named target only and records every target asked for.
func onlyOnTarget(name, root string, calls *[]string) lister {
	return func(_ context.Context, _ string, target target) ([]byte, error) {
		*calls = append(*calls, target.name)
		if target.name != name {
			return nil, nil
		}
		return []byte("P p\x00N p\x00D " + root + "\x00F main\x00G p.go\x00E\x00"), nil
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestCheckArch_FileNotInHostLint_Reported(t *testing.T) {
	for _, tc := range []struct{ name, constraint, rule string }{
		{"platform_only", "windows", "UNLINTED"},
		{"compiled_nowhere", "bogus", "UNSELECTED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{
				"go.mod":  "module example.com/p\n\ngo 1.27\n",
				"base.go": "package p\n",
				"p.go":    "//go:build " + tc.constraint + "\n\npackage p\n",
			})
			got, err := checkArch(t.Context(), root, goList)
			if err != nil || len(got) != 1 || got[0].path != "p.go" || got[0].rule != tc.rule {
				t.Fatalf("findings=%+v err=%v; want p.go %s", got, err, tc.rule)
			}
		})
	}
}

func TestCheckArch_DepOnlyMainPackage_ReportsUnlinted(t *testing.T) {
	root := t.TempDir()
	list := func(context.Context, string, target) ([]byte, error) {
		return []byte("P p/_hidden\x00N hidden\x00D " + root + "/_hidden\x00F main\x00F dep\x00G p.go\x00E\x00"), nil
	}
	got, err := checkArch(t.Context(), root, list)
	if err != nil || len(got) != 1 || got[0].path != "_hidden/p.go" || got[0].rule != "UNLINTED" {
		t.Fatalf("findings=%+v err=%v; want one deduplicated UNLINTED", got, err)
	}
}

func TestCheckArch_TestMainRecord_Ignored(t *testing.T) {
	root := t.TempDir()
	list := func(_ context.Context, _ string, target target) ([]byte, error) {
		return []byte("P p.test\x00N main\x00D " + root + "\x00F main\x00G /go-build/" + target.name + "/main.go\x00E\x00"), nil
	}
	got, err := checkArch(t.Context(), root, list)
	if err != nil || len(got) != 0 {
		t.Fatalf("findings=%+v err=%v; want clean", got, err)
	}
}

func TestCheckArch_ListerError_FailsClosed(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		list := func(context.Context, string, target) ([]byte, error) {
			if malformed {
				return []byte("malformed"), nil
			}
			return nil, errors.New("listing failed")
		}
		if got, err := checkArch(t.Context(), t.TempDir(), list); err == nil || !strings.Contains(err.Error(), "host") {
			t.Fatalf("findings=%+v err=%v; want target-prefixed error", got, err)
		}
	}
}

func TestCheckArch_DependencySource_ReportsDepSource(t *testing.T) {
	for _, tc := range []struct {
		dir, flags, path string
	}{
		{"vendor/dep", "", "vendor/dep"},
		{"_dep", "", "_dep"},
		{"../external", "F localreplace\x00", "go.mod"},
		{"vendor/standard", "F std\x00", ""},
		{"../cache", "", ""},
	} {
		t.Run(tc.dir, func(t *testing.T) {
			root := t.TempDir()
			list := func(context.Context, string, target) ([]byte, error) {
				return []byte("P dep\x00N dep\x00D " + filepath.Join(root, tc.dir) + "\x00F dep\x00" + tc.flags + "G p.go\x00E\x00"), nil
			}
			got, err := checkArch(t.Context(), root, list)
			if tc.path == "" {
				if err != nil || len(got) != 0 {
					t.Fatalf("findings=%+v err=%v; want clean dependency source", got, err)
				}
				return
			}
			if err != nil || len(got) != 1 || got[0].path != tc.path || got[0].line != 0 || got[0].rule != "DEP-SOURCE" {
				t.Fatalf("findings=%+v err=%v; want %s:0: DEP-SOURCE", got, err, tc.path)
			}
		})
	}
}

func TestWalkTree_MissingRoot_Errors(t *testing.T) {
	files, findings, err := walkTree(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatalf("files=%v findings=%+v; want walk error", files, findings)
	}
}

func TestClassifyEntry_SpecialFile_ReportsSourceLink(t *testing.T) {
	entries, err := fs.ReadDir(fstest.MapFS{
		"dev": {Mode: fs.ModeDevice}, "fifo": {Mode: fs.ModeNamedPipe}, "sock": {Mode: fs.ModeSocket},
	}, ".")
	if err != nil || len(entries) != 3 {
		t.Fatalf("entries=%v err=%v; want three special entries", entries, err)
	}
	for _, d := range entries {
		f, bad := classifyEntry(d.Name(), d)
		if !bad || f.path != d.Name() || f.line != 0 || f.rule != "SOURCE-LINK" {
			t.Fatalf("classifyEntry(%s)=(%+v,%t); want %s:0: SOURCE-LINK", d.Name(), f, bad, d.Name())
		}
	}
}

func TestCheckArch_DirectiveViolation_Reported(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "p.go"), []byte("package p\n//nolint:depguard\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	list := func(context.Context, string, target) ([]byte, error) {
		return []byte("P p\x00N p\x00D " + root + "\x00F main\x00G p.go\x00E\x00"), nil
	}
	got, err := checkArch(t.Context(), root, list)
	if err != nil || len(got) != 1 || got[0].path != "p.go" || got[0].line != 2 || got[0].rule != "SEC-A1" {
		t.Fatalf("findings=%+v err=%v; want p.go:2: SEC-A1", got, err)
	}
}

func TestCheckArch_UnparsableGoFile_FailsClosed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "p.go"), []byte("package p\nfunc (\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	list := func(context.Context, string, target) ([]byte, error) {
		return []byte("P p\x00N p\x00D " + root + "\x00F main\x00G p.go\x00E\x00"), nil
	}
	if got, err := checkArch(t.Context(), root, list); err == nil || !strings.Contains(err.Error(), "p.go:2") {
		t.Fatalf("findings=%+v err=%v; want parse error at p.go:2", got, err)
	}
}

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
