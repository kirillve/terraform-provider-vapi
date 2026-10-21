package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"
)

const archTimeout = 5 * time.Minute

const (
	exitClean     = 0
	exitViolation = 1
	exitUsage     = 2
)

// finding is one policy violation at a source position.
type finding struct {
	path string // slash-separated, relative to -root
	line int
	rule string
	msg  string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	check, root, err := parseArgs(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "policycheck: %v\n", err)
		return exitUsage
	}
	fsys := os.DirFS(root)
	if _, err := fs.Stat(fsys, "go.mod"); err != nil {
		fmt.Fprintf(stderr, "policycheck: -root %q is not a module directory: %v\n", root, err)
		return exitUsage
	}
	findings, err := check(root)
	if err != nil {
		fmt.Fprintf(stderr, "policycheck: %v\n", err)
		return exitUsage
	}
	for _, f := range findings {
		fmt.Fprintf(stdout, "%s:%d: %s: %s\n", f.path, f.line, f.rule, f.msg)
	}
	if len(findings) > 0 {
		return exitViolation
	}
	return exitClean
}

func parseArgs(args []string, stderr io.Writer) (func(string) ([]finding, error), string, error) {
	flags := flag.NewFlagSet("policycheck", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.String("check", "", "policy check to run")
	root := flags.String("root", "", "module directory")
	if err := flags.Parse(args); err != nil {
		return nil, "", fmt.Errorf("parse flags: %w", err)
	}
	if flags.NArg() != 0 {
		return nil, "", fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	if *root == "" {
		return nil, "", errors.New("-root is required")
	}
	switch *check {
	case "testnames":
		return testNames, *root, nil
	case checkNameArch:
		return arch, *root, nil
	default:
		return nil, "", fmt.Errorf("unknown -check %q (want testnames|arch)", *check)
	}
}

func testNames(root string) ([]finding, error) {
	return checkTestNames(os.DirFS(root))
}

func arch(root string) ([]finding, error) {
	ctx, cancel := context.WithTimeout(context.Background(), archTimeout)
	defer cancel()
	return checkArch(ctx, root, goList)
}
