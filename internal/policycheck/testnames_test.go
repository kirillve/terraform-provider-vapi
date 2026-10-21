package main

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestTestNames_ForbiddenVerbInName_Flagged(t *testing.T) {
	for _, name := range []string{"TestFooCallsBar", "TestFooUsesBar", "BenchmarkXUses", "FuzzXCalls"} {
		t.Run(name, func(t *testing.T) {
			fsys := fstest.MapFS{"internal/x/x_test.go": {Data: []byte("package x\n\nimport \"testing\"\n\nfunc " + name + "(t *testing.T) {}\n")}}
			findings, err := checkTestNames(fsys)
			if err != nil {
				t.Fatal(err)
			}
			if len(findings) != 1 {
				t.Fatalf("findings=%v; want one", findings)
			}
			got := findings[0]
			if got.path != "internal/x/x_test.go" || got.line != 5 || got.rule != "testnames" {
				t.Fatalf("finding=%+v; want internal/x/x_test.go:5: testnames", got)
			}
		})
	}
}

func TestTestNames_CleanNames_Pass(t *testing.T) {
	fsys := fstest.MapFS{"internal/x/x_test.go": {Data: []byte("package x\nfunc TestRecallsList() {}\nfunc TestCausesX() {}\n")}}
	findings, err := checkTestNames(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings=%v; want none", findings)
	}
}

func TestTestNames_NonTestFile_Ignored(t *testing.T) {
	src := []byte("package x\nimport \"testing\"\nfunc TestFooUsesBar(t *testing.T) {}\n")
	fsys := fstest.MapFS{
		"internal/x/x.go":               {Data: src},
		"internal/x/x_test.go_extra.go": {Data: src},
	}
	findings, err := checkTestNames(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings=%v; want none", findings)
	}
}

// failingFS injects a ReadDir error for dir and a ReadFile error for file.
type failingFS struct {
	fstest.MapFS
	dir, file string
}

func (f failingFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == f.dir {
		return nil, errors.New("injected readdir error")
	}
	return f.MapFS.ReadDir(name)
}

func (f failingFS) ReadFile(name string) ([]byte, error) {
	if name == f.file {
		return nil, errors.New("injected read error")
	}
	return f.MapFS.ReadFile(name)
}

func TestTestNames_UnreadableEntry_Errors(t *testing.T) {
	src := fstest.MapFS{"internal/x/x_test.go": {Data: []byte("package x\n")}}
	for _, tc := range []struct{ name, dir, file, want string }{
		{"readdir", "internal/x", "", "injected readdir error"},
		{"readfile", "", "internal/x/x_test.go", "injected read error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings, err := checkTestNames(failingFS{MapFS: src, dir: tc.dir, file: tc.file})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("findings=%+v err=%v; want error containing %q", findings, err, tc.want)
			}
		})
	}
}
