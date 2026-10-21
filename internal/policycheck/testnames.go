package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"regexp"
	"strings"
)

const ruleTestNames = "testnames"

// forbiddenTestName is the G-TESTNAMES rule, matched against FuncDecl.Name.
var forbiddenTestName = regexp.MustCompile(`^(Test|Benchmark|Fuzz).*(Calls|Uses).*`)

// checkTestNames is the G-TESTNAMES rule over every test file of the module.
func checkTestNames(fsys fs.FS) ([]finding, error) {
	paths, err := testFiles(fsys)
	if err != nil {
		return nil, err
	}
	var findings []finding
	for _, path := range paths {
		src, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		found, err := testNameFindings(path, src)
		if err != nil {
			return nil, err
		}
		findings = append(findings, found...)
	}
	return findings, nil
}

// testFiles lists every _test.go file in the module, skipping hidden, testdata, scratch and
// nested-module directories (go ignores the first two as well).
func testFiles(fsys fs.FS) ([]string, error) {
	var paths []string
	err := fs.WalkDir(fsys, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != "." && skipTestDir(fsys, path, entry.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk module: %w", err)
	}
	return paths, nil
}

func skipTestDir(fsys fs.FS, path, name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" || scratchDirs[path] {
		return true
	}
	_, err := fs.Stat(fsys, path+"/go.mod")
	return err == nil
}

func testNameFindings(path string, src []byte) ([]finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var findings []finding
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !forbiddenTestName.MatchString(fn.Name.Name) {
			continue
		}
		findings = append(findings, finding{
			path: path,
			line: fset.Position(fn.Name.Pos()).Line,
			rule: ruleTestNames,
			msg:  fmt.Sprintf("%s matches %s; name the asserted behaviour instead", fn.Name.Name, forbiddenTestName),
		})
	}
	return findings, nil
}
