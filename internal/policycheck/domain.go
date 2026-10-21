package main

import "strings"

const ruleDomain = "DOMAIN-TRANSITIVE"

// domainDeny applies the deny list of the internal/providerdata depguard rule (.golangci-arch.yml)
// to every dependency, not only to direct imports: a $gostd package such as expvar or net/rpc
// links net/http itself. Change both together.
var domainDeny = []struct {
	dir  string
	deny []string
}{
	{"internal/providerdata", []string{"net/http", "database/sql"}},
}

func domainFindings(abs string, listings [][]listedPackage) []finding {
	var findings []finding
	for i, packages := range listings {
		for _, p := range packages {
			path, inside := relPath(abs, p.dir)
			if !p.mainModule || !inside || isTestMain(p) || strings.Contains(p.importPath, " [") {
				continue
			}
			if dep, ok := deniedDep(path, p.deps); ok {
				findings = append(findings, finding{path: path, rule: ruleDomain, msg: "links " + dep + " for " + buildTargets[i].name})
			}
		}
	}
	return findings
}

func deniedDep(path string, deps []string) (string, bool) {
	for _, rule := range domainDeny {
		if strings.Contains("/"+path+"/", "/"+rule.dir+"/") {
			if dep, ok := firstDenied(rule.deny, deps); ok {
				return dep, true
			}
		}
	}
	return "", false
}

func firstDenied(deny, deps []string) (string, bool) {
	for _, d := range deny {
		for _, dep := range deps {
			if dep == d || strings.HasPrefix(dep, d+"/") {
				return dep, true
			}
		}
	}
	return "", false
}
