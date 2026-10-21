package main

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

const (
	checkNameArch  = "arch"
	ruleUnlinted   = "UNLINTED"
	ruleUnselected = "UNSELECTED"
	ruleDepSource  = "DEP-SOURCE"
)

func checkArch(ctx context.Context, root string, list lister) ([]finding, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("absolute module root: %w", err)
	}
	findings, err := rootFindings(abs)
	if err != nil {
		return nil, err
	}
	goFiles, tree, err := walkTree(abs)
	if err != nil {
		return nil, err
	}
	listings, err := listTargets(ctx, abs, list)
	if err != nil {
		return nil, err
	}
	directives, err := directiveFindings(abs, goFiles)
	if err != nil {
		return nil, err
	}
	leaks, err := leakCheckFindings(abs, listings)
	if err != nil {
		return nil, err
	}
	findings = append(findings, leaks...)
	findings = append(findings, tree...)
	findings = append(findings, coverageFindings(abs, listings, goFiles)...)
	findings = append(findings, domainFindings(abs, listings)...)
	return sortFindings(append(findings, directives...)), nil
}

func listTargets(ctx context.Context, abs string, list lister) ([][]listedPackage, error) {
	var listings [][]listedPackage
	for _, t := range buildTargets {
		out, err := list(ctx, abs, t)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
		}
		packages, err := parseList(out)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
		}
		listings = append(listings, packages)
	}
	return listings, nil
}

func coverageFindings(abs string, listings [][]listedPackage, goFiles []string) []finding {
	lint, compiled := make(map[string]bool), make(map[string]bool)
	for i, packages := range listings {
		if buildTargets[i].name == hostTarget {
			lint = lintSet(abs, packages)
		}
	}
	var findings []finding
	for i, packages := range listings {
		for _, p := range packages {
			findings = append(findings, packageFindings(abs, buildTargets[i].name, p, lint, compiled)...)
		}
	}
	for _, path := range goFiles {
		if !compiled[path] {
			findings = append(findings, finding{path: path, rule: ruleUnselected, msg: "Go file compiled for no build target (host or a .goreleaser.yml target)"})
		}
	}
	return findings
}

func lintSet(abs string, host []listedPackage) map[string]bool {
	lint := make(map[string]bool)
	for _, p := range host {
		if !p.mainModule || p.depOnly || isTestMain(p) {
			continue
		}
		for _, file := range p.goFiles {
			path, inside := relPath(abs, filepath.Join(p.dir, file))
			if inside {
				lint[path] = true
			}
		}
	}
	return lint
}

func packageFindings(abs, t string, p listedPackage, lint, compiled map[string]bool) []finding {
	if p.standard {
		return nil
	}
	if !p.mainModule {
		return dependencyFindings(abs, p)
	}
	if isTestMain(p) {
		return nil
	}
	var findings []finding
	for _, file := range p.goFiles {
		path, inside := relPath(abs, filepath.Join(p.dir, file))
		compiled[path] = true
		switch {
		case !inside:
			findings = append(findings, finding{path: path, rule: ruleUnlinted, msg: "compiled from outside the module root"})
		case !lint[path]:
			findings = append(findings, finding{path: path, rule: ruleUnlinted, msg: "compiled for " + t + " but not linted by golangci-lint run ./... on the host"})
		}
	}
	return findings
}

func dependencyFindings(abs string, p listedPackage) []finding {
	if path, inside := relPath(abs, p.dir); inside {
		return []finding{{path: path, rule: ruleDepSource, msg: "dependency source inside the module root"}}
	}
	if p.localReplace {
		return []finding{{path: "go.mod", rule: ruleDepSource, msg: "filesystem-replaced dependency source"}}
	}
	return nil
}

func isTestMain(p listedPackage) bool {
	if p.name != "main" || !strings.HasSuffix(p.importPath, ".test") || len(p.goFiles) == 0 {
		return false
	}
	for _, file := range p.goFiles {
		if !filepath.IsAbs(file) {
			return false
		}
	}
	return true
}

func relPath(abs, target string) (string, bool) {
	rel, err := filepath.Rel(abs, target)
	if err != nil {
		return filepath.ToSlash(target), false
	}
	return filepath.ToSlash(rel), rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func sortFindings(fs []finding) []finding {
	compare := func(a, b finding) int {
		return cmp.Or(cmp.Compare(a.path, b.path), cmp.Compare(a.line, b.line), cmp.Compare(a.rule, b.rule))
	}
	slices.SortFunc(fs, compare)
	return slices.CompactFunc(fs, func(a, b finding) bool { return compare(a, b) == 0 })
}
