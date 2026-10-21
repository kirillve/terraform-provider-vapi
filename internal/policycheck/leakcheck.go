package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

const ruleLeakCheck = "LEAKCHECK"

func leakCheckFindings(abs string, listings [][]listedPackage) ([]finding, error) {
	var findings []finding
	for _, packages := range listings {
		for dir, files := range testPackageFiles(abs, packages) {
			verified, err := verifiedTestPackage(abs, files)
			if err != nil {
				return nil, err
			}
			if !verified {
				findings = append(findings, finding{path: dir, rule: ruleLeakCheck, msg: "test package lacks goleak.VerifyTestMain"})
			}
		}
	}
	return findings, nil
}

// Internal and external tests share one generated test binary and one TestMain.
func testPackageFiles(abs string, packages []listedPackage) map[string]map[string]bool {
	groups := make(map[string]map[string]bool)
	for _, p := range packages {
		if !p.mainModule || isTestMain(p) {
			continue
		}
		addTestFiles(abs, p, groups)
	}
	return groups
}

func addTestFiles(abs string, p listedPackage, groups map[string]map[string]bool) {
	for _, file := range p.goFiles {
		path, inside := relPath(abs, filepath.Join(p.dir, file))
		if !inside || !strings.HasSuffix(path, "_test.go") {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(path))
		if groups[dir] == nil {
			groups[dir] = make(map[string]bool)
		}
		groups[dir][path] = true
	}
}

func verifiedTestPackage(abs string, files map[string]bool) (bool, error) {
	for path := range files {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(abs, path), nil, 0)
		if err != nil {
			return false, fmt.Errorf("parse leak check %s: %w", path, err)
		}
		if fileVerifiesLeaks(file) {
			return true, nil
		}
	}
	return false, nil
}

func fileVerifiesLeaks(file *ast.File) bool {
	testingName := importName(file, "testing", "testing")
	goleakName := importName(file, "go.uber.org/goleak", "goleak")
	if testingName == "" || goleakName == "" {
		return false
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || !isLeakTestMain(function, testingName) {
			continue
		}
		parameter := function.Type.Params.List[0].Names[0].Name
		if bodyVerifiesLeaks(function.Body, goleakName, parameter) {
			return true
		}
	}
	return false
}

func importName(file *ast.File, path, defaultName string) string {
	for _, spec := range file.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil || value != path {
			continue
		}
		if spec.Name != nil {
			return spec.Name.Name
		}
		return defaultName
	}
	return ""
}

func isLeakTestMain(function *ast.FuncDecl, testingName string) bool {
	if function.Name.Name != "TestMain" || function.Recv != nil || function.Body == nil || function.Type.Results != nil {
		return false
	}
	parameters := function.Type.Params.List
	if len(parameters) != 1 || len(parameters[0].Names) != 1 {
		return false
	}
	pointer, ok := parameters[0].Type.(*ast.StarExpr)
	return ok && importedSymbol(pointer.X, testingName, "M")
}

func importedSymbol(expression ast.Expr, name, symbol string) bool {
	if name == "." {
		identifier, ok := expression.(*ast.Ident)
		return ok && identifier.Name == symbol
	}
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != symbol {
		return false
	}
	identifier, ok := selector.X.(*ast.Ident)
	return ok && identifier.Name == name
}

func bodyVerifiesLeaks(body *ast.BlockStmt, goleakName, parameter string) bool {
	verified := false
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 || !importedSymbol(call.Fun, goleakName, "VerifyTestMain") {
			return true
		}
		argument, ok := call.Args[0].(*ast.Ident)
		verified = verified || ok && argument.Name == parameter
		return true
	})
	return verified
}
