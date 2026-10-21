package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	maxListBytes   = 64 << 20
	maxStderrBytes = 1 << 20
	waitDelay      = 10 * time.Second
)

const listTemplate = `P {{.ImportPath}}{{"\x00"}}N {{.Name}}{{"\x00"}}D {{.Dir}}{{"\x00"}}{{if .DepOnly}}F dep{{"\x00"}}{{end}}{{if .Standard}}F std{{"\x00"}}{{end}}{{if .Module}}{{if .Module.Main}}F main{{"\x00"}}{{range .Deps}}I {{.}}{{"\x00"}}{{end}}{{end}}{{if .Module.Replace}}{{if not .Module.Replace.Version}}F localreplace{{"\x00"}}{{end}}{{end}}{{end}}{{range .GoFiles}}G {{.}}{{"\x00"}}{{end}}{{range .CgoFiles}}G {{.}}{{"\x00"}}{{end}}E{{"\x00"}}`

var errOutputTooLarge = errors.New("command output over the size limit")

type target struct {
	name string
	env  []string
}

// hostTarget is the build golangci-lint sees: its listing is the set of files that get linted.
const hostTarget = "host"

// releaseTargets mirrors the goos/goarch matrix in .goreleaser.yml (darwin/386 is ignored there
// too); goreleaser builds with CGO_ENABLED=0. Change both together. windows/arm is left out:
// the toolchain cannot list it without cgo ("default PIE binary requires external (cgo)
// linking"); its source set equals windows/386 and windows/arm64 unless a file is GOARCH=arm only.
var releaseTargets = []struct{ goos, goarch string }{
	{"freebsd", "amd64"},
	{"freebsd", "386"},
	{"freebsd", "arm"},
	{"freebsd", "arm64"},
	{"windows", "amd64"},
	{"windows", "386"},
	{"windows", "arm64"},
	{"linux", "amd64"},
	{"linux", "386"},
	{"linux", "arm"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
}

var buildTargets = newBuildTargets()

func newBuildTargets() []target {
	targets := []target{{name: hostTarget}}
	for _, r := range releaseTargets {
		targets = append(targets, target{
			name: r.goos + "/" + r.goarch,
			env:  []string{"GOOS=" + r.goos, "GOARCH=" + r.goarch, "CGO_ENABLED=0"},
		})
	}
	return targets
}

type listedPackage struct {
	importPath, name, dir                       string
	depOnly, standard, mainModule, localReplace bool
	goFiles                                     []string
	deps                                        []string
}

type lister func(ctx context.Context, root string, t target) ([]byte, error)

func goList(ctx context.Context, root string, t target) ([]byte, error) {
	cmd := exec.CommandContext(
		ctx, "go", "list", "-deps", "-test", "-f", listTemplate, "./...",
	)
	cmd.Dir = root
	cmd.WaitDelay = waitDelay
	cmd.Env = append(cmd.Environ(), t.env...)
	stdout, stderr := cappedBuffer{limit: maxListBytes}, cappedBuffer{limit: maxStderrBytes}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list (%s): %w: %s", t.name, err, stderr.buf.String())
	}
	return stdout.buf.Bytes(), nil
}

func parseList(out []byte) ([]listedPackage, error) {
	var packages []listedPackage
	var cur *listedPackage
	for tok := range strings.SplitSeq(string(out), "\x00") {
		tok = strings.TrimPrefix(tok, "\n")
		if tok == "" {
			continue
		}
		next, done, err := step(cur, tok)
		if err != nil {
			return nil, err
		}
		if done {
			packages = append(packages, *cur)
		}
		cur = next
	}
	if cur != nil {
		return nil, errors.New("unterminated go list record")
	}
	return packages, nil
}

func step(cur *listedPackage, tok string) (next *listedPackage, done bool, err error) {
	if strings.HasPrefix(tok, "P ") {
		if cur != nil {
			return nil, false, errors.New("nested go list record")
		}
		return &listedPackage{importPath: strings.TrimPrefix(tok, "P ")}, false, nil
	}
	if cur == nil {
		return nil, false, errors.New("go list token outside a record")
	}
	if tok == "E" {
		return nil, true, nil
	}
	if err := applyToken(cur, tok); err != nil {
		return nil, false, err
	}
	return cur, false, nil
}

func applyToken(p *listedPackage, tok string) error {
	key, val, ok := strings.Cut(tok, " ")
	if !ok {
		return fmt.Errorf("malformed go list token %q", tok)
	}
	switch key {
	case "N":
		p.name = val
	case "D":
		p.dir = val
	case "G":
		p.goFiles = append(p.goFiles, val)
	case "I":
		p.deps = append(p.deps, val)
	case "F":
		return applyFlag(p, val)
	default:
		return fmt.Errorf("unknown go list token %q", key)
	}
	return nil
}

func applyFlag(p *listedPackage, val string) error {
	switch val {
	case "dep":
		p.depOnly = true
	case "main":
		p.mainModule = true
	case "std":
		p.standard = true
	case "localreplace":
		p.localReplace = true
	default:
		return fmt.Errorf("unknown go list flag %q", val)
	}
	return nil
}

type cappedBuffer struct {
	buf   bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.buf.Len() {
		return 0, errOutputTooLarge
	}
	return b.buf.Write(p)
}
