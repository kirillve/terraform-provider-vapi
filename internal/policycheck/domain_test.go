package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDomainFindings_TransitiveDeniedDep_Reported(t *testing.T) {
	root := t.TempDir()
	for _, tt := range []struct {
		name, dir, dep, target string
		deps                   []string
		targetIndex            int
	}{
		{"deny_order", "internal/providerdata", "net/http", "host", []string{"database/sql", "net/http"}, 0},
		{"nested_prefix", "cmd/x/internal/providerdata", "net/http/httptrace", "host", []string{"net/http/httptrace"}, 0},
		{"windows_only", "internal/providerdata", "net/http", "windows/amd64", []string{"net/http"}, targetIndex("windows/amd64")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			listings := make([][]listedPackage, len(buildTargets))
			if tt.targetIndex < 0 || tt.targetIndex >= len(listings) {
				t.Fatalf("target index=%d; listings=%d", tt.targetIndex, len(listings))
			}
			listings[tt.targetIndex] = []listedPackage{{
				importPath: "example.com/p/" + tt.dir, name: "domain", dir: filepath.Join(root, tt.dir),
				mainModule: true, deps: tt.deps,
			}}
			got := domainFindings(root, listings)
			want := []finding{{path: tt.dir, rule: "DOMAIN-TRANSITIVE", msg: "links " + tt.dep + " for " + tt.target}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("findings=%+v; want %+v", got, want)
			}
		})
	}
}

func TestDomainFindings_AllowedOrOutsideDomain_Clean(t *testing.T) {
	root := t.TempDir()
	for _, tt := range []struct {
		name string
		pkg  listedPackage
	}{
		{"stdlib_only", listedPackage{dir: filepath.Join(root, "internal", "book"), mainModule: true, deps: []string{"strings"}}},
		{"vapi_http", listedPackage{dir: filepath.Join(root, "internal", "vapi"), mainModule: true, deps: []string{"net/http"}}},
		{"test_variant", listedPackage{importPath: "example.com/p/internal/providerdata [example.com/p/internal/providerdata.test]", dir: filepath.Join(root, "internal", "book"), mainModule: true, deps: []string{"net/http"}}},
		{"test_main", listedPackage{importPath: "example.com/p/internal/providerdata.test", name: "main", dir: filepath.Join(root, "internal", "book"), mainModule: true, goFiles: []string{filepath.Join(root, "cache.go")}, deps: []string{"net/http"}}},
		{"providerdata_json", listedPackage{dir: filepath.Join(root, "internal", "providerdata"), mainModule: true, deps: []string{"encoding/json"}}},
		{"dependency", listedPackage{dir: filepath.Join(root, "internal", "book"), deps: []string{"net/http"}}},
		{"outside_root", listedPackage{dir: filepath.Join(t.TempDir(), "internal", "book"), mainModule: true, deps: []string{"net/http"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := domainFindings(root, [][]listedPackage{{tt.pkg}}); len(got) != 0 {
				t.Fatalf("findings=%+v; want none", got)
			}
		})
	}
}

func targetIndex(name string) int {
	for i, t := range buildTargets {
		if t.name == name {
			return i
		}
	}
	panic("unknown build target " + name)
}
