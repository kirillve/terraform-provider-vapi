// Package main serves the vapi Terraform provider plugin.
package main

import (
	"context"
	"flag"
	"log"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/kirillve/terraform-provider-vapi/internal/provider"
)

var (
	// version and commit are set by the goreleaser configuration through
	// -X main.version and -X main.commit; an unstamped build reports "dev".
	version = "dev"
	commit  string
)

// releaseVersion matches the module version of a tagged release, with an optional pre-release suffix and no
// build metadata, for example v0.14.0 or v0.15.0-rc.1.
var releaseVersion = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// pseudoVersion matches the pseudo-version that a build from an untagged commit records.
var pseudoVersion = regexp.MustCompile(`(^|[-.])\d{14}-[0-9a-f]{12}$`)

// resolveVersion returns the stamped version unless it is still the "dev" default; then it falls back
// to the module version recorded in the build info when that is a tagged release, which is the case for
// `go install <module>@<tag>`. A local build records "(devel)", an empty value, a pseudo-version or a
// "+dirty" version, and keeps "dev". The leading "v" is dropped to match the goreleaser version.
func resolveVersion(stamped string, info *debug.BuildInfo) string {
	if stamped != "dev" || info == nil {
		return stamped
	}
	if v := info.Main.Version; releaseVersion.MatchString(v) && !pseudoVersion.MatchString(v) {
		return strings.TrimPrefix(v, "v")
	}
	return stamped
}

// providerVersion is the version the provider reports to Terraform.
func providerVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return resolveVersion(version, nil)
	}
	return resolveVersion(version, info)
}

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	v := providerVersion()
	if debugMode {
		log.Printf("vapi provider %s (commit %q)", v, commit)
	}

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/kirillve/vapi",
		Debug:   debugMode,
	}

	err := providerserver.Serve(context.Background(), provider.New(v), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
