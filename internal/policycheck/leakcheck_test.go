package main

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed testdata/leakcheck
var leakFixtures embed.FS

func leakFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	err := fs.WalkDir(leakFixtures, "testdata/leakcheck", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		contents, err := leakFixtures.ReadFile(path)
		if err != nil {
			return err
		}
		destination := filepath.Join(root, strings.TrimSuffix(strings.TrimPrefix(path, "testdata/leakcheck/"), ".txt"))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		return os.WriteFile(destination, contents, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCheckArch_LeakCheckFixtures_ReportMissingVerification(t *testing.T) {
	got, err := checkArch(t.Context(), leakFixture(t), goList)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"missing", "wrongfunction", "wrongimport", "wrongsignature"}
	if len(got) != len(want) {
		t.Fatalf("findings=%+v; want LEAKCHECK for %v", got, want)
	}
	for i, path := range want {
		if got[i].path != path || got[i].line != 0 || got[i].rule != "LEAKCHECK" || got[i].msg != "test package lacks goleak.VerifyTestMain" {
			t.Fatalf("finding=%+v; want %s:0: LEAKCHECK: test package lacks goleak.VerifyTestMain", got[i], path)
		}
	}
}

func TestCLI_LeakCheckMissing_Exits1(t *testing.T) {
	root := leakFixture(t)
	var stdout, stderr strings.Builder
	code := run([]string{"-check", "arch", "-root", root}, &stdout, &stderr)
	if code != 1 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "missing:0: LEAKCHECK: test package lacks goleak.VerifyTestMain\n") {
		t.Fatalf("code=%d stdout=%q stderr=%q; want LEAKCHECK exit 1", code, stdout.String(), stderr.String())
	}
}
