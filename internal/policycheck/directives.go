package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

const (
	ruleNolint   = "SEC-A1"
	rulePermit   = "SEC-A2"
	ruleGen      = "GEN-MARKER"
	ruleLine     = "LINE-DIRECTIVE"
	ruleLinkname = "LINKNAME"
	ruleCanon    = "CANON-IMPORT"
)

var silencedLinters = map[string]bool{"depguard": true, "forbidigo": true, "all": true}

func directiveFindings(abs string, goFiles []string) ([]finding, error) {
	var findings []finding
	for _, path := range goFiles {
		src, err := fs.ReadFile(os.DirFS(abs), path)
		if err != nil {
			return nil, err
		}
		fileFindings, err := fileDirectiveFindings(path, src)
		if err != nil {
			return nil, err
		}
		findings = append(findings, fileFindings...)
	}
	return findings, nil
}

func fileDirectiveFindings(path string, src []byte) ([]finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var findings []finding
	if ast.IsGenerated(file) {
		findings = append(findings, finding{path: path, line: 1, rule: ruleGen, msg: "generated-file header hides source from lint"})
	}
	findings = append(findings, importFindings(fset, path, file)...)
	return append(findings, commentFindings(fset, path, file)...), nil
}

func importFindings(fset *token.FileSet, path string, file *ast.File) []finding {
	var findings []finding
	for _, spec := range file.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		switch {
		case err != nil || strconv.Quote(value) != spec.Path.Value:
			findings = append(findings, finding{
				path: path, line: fset.PositionFor(spec.Pos(), false).Line,
				rule: ruleCanon, msg: "import path not spelled as a plain quoted string",
			})
		case value == "C":
			findings = append(findings, finding{
				path: path, line: fset.PositionFor(spec.Pos(), false).Line,
				rule: ruleNonGo, msg: "cgo source is forbidden",
			})
		}
	}
	return findings
}

func commentFindings(fset *token.FileSet, path string, file *ast.File) []finding {
	var findings []finding
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if rule, msg, ok := commentFinding(comment.Text); ok {
				findings = append(findings, finding{
					path: path, line: fset.PositionFor(comment.Pos(), false).Line,
					rule: rule, msg: msg,
				})
			}
		}
	}
	return findings
}

func commentFinding(text string) (rule, msg string, ok bool) {
	if strings.HasPrefix(text, "//go:linkname") {
		return ruleLinkname, "linkname redirects source symbols", true
	}
	if strings.HasPrefix(text, "//line") || strings.HasPrefix(text, "/*line") {
		return ruleLine, "line directive redirects source positions", true
	}
	body := strings.ToLower(strings.TrimLeft(strings.TrimSuffix(text, "*/"), "/ \t*"))
	if strings.HasPrefix(body, "permit:") {
		return rulePermit, "permit directive silences source policy", true
	}
	if rest, found := strings.CutPrefix(body, "nolint"); found && silencesArch(rest) {
		return ruleNolint, "nolint directive silences architecture checks", true
	}
	return "", "", false
}

func silencesArch(rest string) bool {
	if rest == "" || strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "\t") {
		return true
	}
	list, ok := strings.CutPrefix(rest, ":")
	if !ok {
		return false
	}
	list, _, _ = strings.Cut(list, "//")
	if strings.HasPrefix(list, "all") {
		return true
	}
	for name := range strings.SplitSeq(list, ",") {
		if silencedLinters[strings.TrimSpace(name)] {
			return true
		}
	}
	return false
}
