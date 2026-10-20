package main

import (
	"os"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		stamped string
		info    *debug.BuildInfo
		want    string
	}{
		{"stamped version wins over the module version", "1.2.3", &debug.BuildInfo{Main: debug.Module{Version: "v0.14.0"}}, "1.2.3"},
		{"module version replaces the dev default", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v0.14.0"}}, "0.14.0"},
		{"pre-release tag is a release", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v0.15.0-rc.1"}}, "0.15.0-rc.1"},
		{"pseudo-version keeps dev", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20261020140230-1e374d1077ed"}}, "dev"},
		{"dirty pseudo-version keeps dev", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20261020140230-1e374d1077ed+dirty"}}, "dev"},
		{"pseudo-version after a tag keeps dev", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v0.14.1-0.20261020140230-1e374d1077ed"}}, "dev"},
		{"dirty tag keeps dev", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v0.14.0+dirty"}}, "dev"},
		{"devel module version keeps dev", "dev", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, "dev"},
		{"empty module version keeps dev", "dev", &debug.BuildInfo{}, "dev"},
		{"missing build info keeps dev", "dev", nil, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, resolveVersion(tt.stamped, tt.info))
		})
	}
}

// TestProviderVersionMatchesBuild checks the version of a build stamped with -ldflags: it runs only when
// VAPI_EXPECT_VERSION names the version the test binary was linked with.
func TestProviderVersionMatchesBuild(t *testing.T) {
	want, ok := os.LookupEnv("VAPI_EXPECT_VERSION")
	if !ok {
		t.Skip("VAPI_EXPECT_VERSION is not set")
	}
	assert.Equal(t, want, providerVersion())
}
