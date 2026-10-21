package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	ruleGlobMeta   = "ARCH-GLOBMETA"
	ruleWorkspace  = "WORKSPACE"
	ruleSourceLink = "SOURCE-LINK"
	ruleNonGo      = "NON-GO-SOURCE"
	ruleNestedMod  = "NESTED-MODULE"
)

// knownNestedModules are top-level directories with their own go.mod that the root module's
// lint run does not cover by design: tools/ holds only the `//go:build generate` documentation
// generator. Any other nested module is refused, because golangci-lint ./... would never see it.
var knownNestedModules = map[string]bool{"tools": true}

// scratchDirs are top-level, gitignored scratch directories that never hold module source.
var scratchDirs = map[string]bool{"tmp": true}

const globMetachars = `[]{}*?\`

var nonGoSources = map[string]bool{
	".s": true, ".S": true, ".sx": true, ".c": true, ".cc": true, ".cpp": true, ".cxx": true,
	".h": true, ".hh": true, ".hpp": true, ".hxx": true, ".m": true,
	".f": true, ".F": true, ".for": true, ".f90": true, ".syso": true, ".swig": true, ".swigcxx": true,
}

func rootFindings(abs string) ([]finding, error) {
	physical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve module root: %w", err)
	}
	var findings []finding
	for _, path := range []string{abs, physical} {
		if strings.ContainsAny(path, globMetachars) {
			findings = append(findings, finding{path: ".", rule: ruleGlobMeta, msg: "module path contains glob metacharacters: " + path})
		}
	}
	return findings, nil
}

func walkTree(abs string) ([]string, []finding, error) {
	var files []string
	var findings []finding
	err := fs.WalkDir(os.DirFS(abs), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		found, goFile, walkErr := visit(abs, path, d)
		if found != nil {
			findings = append(findings, *found)
		}
		if goFile {
			files = append(files, path)
		}
		return walkErr
	})
	if err != nil {
		return nil, nil, fmt.Errorf("walk module tree: %w", err)
	}
	return files, findings, nil
}

// visit classifies one tree entry: a finding, a regular Go source file, or a directory to skip
// (reported as fs.SkipDir).
func visit(abs, path string, d fs.DirEntry) (found *finding, goFile bool, err error) {
	if path == ".git" {
		// The repository's own metadata; only a nested .git is a finding.
		if d.IsDir() {
			return nil, false, fs.SkipDir
		}
		return nil, false, nil
	}
	if d.IsDir() && path != "." {
		skip, f, bad, err := classifyDir(abs, path)
		if err != nil {
			return nil, false, err
		}
		if bad {
			found = &f
		}
		if skip {
			return found, false, fs.SkipDir
		}
	}
	if f, bad := classifyEntry(path, d); bad {
		return &f, false, nil
	}
	return found, d.Type().IsRegular() && strings.HasSuffix(path, ".go"), nil
}

// classifyDir decides whether the walk descends into a directory: scratch directories and the
// known nested module are skipped, any other nested module is reported and skipped.
func classifyDir(abs, path string) (skip bool, f finding, bad bool, err error) {
	if scratchDirs[path] {
		return true, finding{}, false, nil
	}
	_, statErr := os.Lstat(filepath.Join(abs, filepath.FromSlash(path), "go.mod"))
	switch {
	case errors.Is(statErr, fs.ErrNotExist):
		return false, finding{}, false, nil
	case statErr != nil:
		return false, finding{}, false, fmt.Errorf("stat nested go.mod: %w", statErr)
	case knownNestedModules[path]:
		return true, finding{}, false, nil
	}
	return true, finding{path: path, rule: ruleNestedMod, msg: "nested module is not covered by the root lint run"}, true, nil
}

func classifyEntry(path string, d fs.DirEntry) (finding, bool) {
	f := finding{path: path}
	switch {
	case d.Name() == ".git":
		f.rule, f.msg = ruleSourceLink, "nested .git entry is forbidden"
	case d.Type()&fs.ModeSymlink != 0:
		f.rule, f.msg = ruleSourceLink, "source tree symlink is forbidden"
	case d.IsDir():
		return finding{}, false
	case !d.Type().IsRegular():
		f.rule, f.msg = ruleSourceLink, "non-regular source tree entry is forbidden"
	case d.Name() == "go.work" || d.Name() == "go.work.sum":
		f.rule, f.msg = ruleWorkspace, "workspace file is forbidden"
	case nonGoSources[filepath.Ext(path)]:
		f.rule, f.msg = ruleNonGo, "non-Go source file is forbidden"
	default:
		return finding{}, false
	}
	return f, true
}
