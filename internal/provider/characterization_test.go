package provider

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "record characterization goldens")

// only narrows -update to the named cases: every other entry of an outcome golden is kept byte for byte and
// the schema goldens are never written. It has no effect without -update.
var only = flag.String("only", "", "comma-separated case names; with -update, record only these cases")

// v1Types is the explicit, ordered list of pinned types.
var v1Types = []string{"vapi_assistant", "vapi_file", "vapi_twilio_phone_number",
	"vapi_sip_trunk_phone_number", "vapi_tool_function", "vapi_tool_query_function", "vapi_sip_trunk"}

func TestV1SchemaGolden(t *testing.T) {
	t.Parallel()
	for _, typeName := range v1Types {
		t.Run(typeName, func(t *testing.T) {
			t.Parallel()
			dir := characterizationDirectory(t, typeName)
			p := New("test")()
			server, err := providerserver.NewProtocol6WithError(p)()
			require.NoError(t, err)
			response, err := server.GetProviderSchema(t.Context(), &tfprotov6.GetProviderSchemaRequest{})
			require.NoError(t, err)
			require.Empty(t, response.Diagnostics)
			schema := response.ResourceSchemas[typeName]
			require.NotNil(t, schema)
			require.NotNil(t, schema.Block)
			require.Empty(t, schema.Block.BlockTypes, "nested blocks are unsupported")
			public := characterizationSchema(t, typeName, p.Resources(t.Context()))
			got := map[string]any{
				"schema_version": schema.Version,
				"attributes":     characterizationAttributes(schema.Block.Attributes, ""),
				"fingerprints":   schemaFingerprint(t.Context(), t, public.Schema),
			}
			compareGoldenJSON(t, filepath.Join(dir, "schema.golden.json"), got)
		})
	}
}

func characterizationDirectory(t *testing.T, typeName string) string {
	t.Helper()
	dir := filepath.Join("testdata", "v1", typeName)
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		t.Skipf("missing characterization directory %s", dir)
	}
	require.NoError(t, err)
	require.True(t, info.IsDir(), "%s must be a directory", dir)
	return dir
}

func characterizationSchema(t *testing.T, typeName string, constructors []func() resource.Resource) resource.SchemaResponse {
	t.Helper()
	for _, constructor := range constructors {
		r := constructor()
		var metadata resource.MetadataResponse
		r.Metadata(t.Context(), resource.MetadataRequest{ProviderTypeName: "vapi"}, &metadata)
		if metadata.TypeName == typeName {
			var response resource.SchemaResponse
			r.Schema(t.Context(), resource.SchemaRequest{}, &response)
			require.Empty(t, response.Diagnostics)
			return response
		}
	}
	t.Fatalf("resource %q was not registered", typeName)
	return resource.SchemaResponse{}
}

func characterizationAttributes(attributes []*tfprotov6.SchemaAttribute, prefix string) map[string]any {
	result := make(map[string]any)
	for _, attribute := range attributes {
		path := prefix + attribute.Name
		nesting := ""
		if attribute.NestedType != nil {
			nesting = attribute.NestedType.Nesting.String()
			for child, value := range characterizationAttributes(attribute.NestedType.Attributes, path+".") {
				result[child] = value
			}
		}
		result[path] = map[string]any{
			"type": attribute.ValueType(), "nesting": nesting,
			"required": attribute.Required, "optional": attribute.Optional,
			"computed": attribute.Computed, "sensitive": attribute.Sensitive,
			"description": attribute.Description, "description_kind": attribute.DescriptionKind,
			"deprecated": attribute.Deprecated, "deprecation_message": attribute.DeprecationMessage,
		}
	}
	return result
}

func TestV1Characterization(t *testing.T) {
	t.Parallel()
	for _, typeName := range v1Types {
		t.Run(typeName, func(t *testing.T) {
			t.Parallel()
			dir := characterizationDirectory(t, typeName)
			paths, err := filepath.Glob(filepath.Join(dir, "*.cases.json"))
			require.NoError(t, err)
			require.NotEmpty(t, paths, "no case files in %s", dir)
			for _, path := range paths {
				group := strings.TrimSuffix(filepath.Base(path), ".cases.json")
				t.Run(group, func(t *testing.T) {
					cases := loadCases(t, path)
					got := make(map[string]charOutcome, len(cases))
					for _, c := range cases {
						t.Run(c.Name, func(t *testing.T) {
							h := newHarness(t, c.API)
							got[c.Name] = h.run(typeName, c)
						})
					}
					if t.Failed() {
						return
					}
					compareGolden(t, filepath.Join(dir, group+".golden.json"), got)
				})
			}
		})
	}
}

func loadCases(t testing.TB, path string) []charCase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var cases []charCase
	require.NoError(t, decoder.Decode(&cases))
	var trailing any
	require.ErrorIs(t, decoder.Decode(&trailing), io.EOF)
	require.NotNil(t, cases, "cases must be a JSON array")
	names := make(map[string]bool, len(cases))
	for _, c := range cases {
		require.False(t, names[c.Name], "duplicate case name %q in %s", c.Name, path)
		names[c.Name] = true
	}
	return cases
}

func compareGolden(t testing.TB, path string, got map[string]charOutcome) {
	t.Helper()
	if *update && *only != "" {
		mergeGolden(t, path, got)
		return
	}
	compareGoldenJSON(t, path, got)
}

// mergeGolden replaces only the entries named by -only in the existing golden and writes the file back.
func mergeGolden(t testing.TB, path string, got map[string]charOutcome) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)
	var merged map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &merged))
	for _, name := range strings.Split(*only, ",") {
		if outcome, ok := got[name]; ok {
			merged[name] = json.RawMessage(bytes.TrimSpace(encodeGolden(t, outcome)))
		}
	}
	require.NoError(t, os.WriteFile(filepath.Clean(path), encodeGolden(t, merged), 0o600))
}

func encodeGolden(t testing.TB, value any) []byte {
	t.Helper()
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	require.NoError(t, encoder.Encode(value))
	return buffer.Bytes()
}

func compareGoldenJSON(t testing.TB, path string, got any) {
	t.Helper()
	encoded := encodeGolden(t, got)
	if *update && *only == "" {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(filepath.Clean(path), encoded, 0o600))
		return
	}
	want, err := os.ReadFile(filepath.Clean(path))
	if os.IsNotExist(err) {
		t.Fatalf("missing golden %s; run with -update to record it", path)
	}
	require.NoError(t, err)
	if !bytes.Equal(want, encoded) {
		t.Errorf("golden %s differs for keys %q; run with -update to record it", path, goldenDifferingKeys(t, want, encoded))
	}
}

func goldenDifferingKeys(t testing.TB, want, got []byte) []string {
	t.Helper()
	var before, after map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(want, &before))
	require.NoError(t, json.Unmarshal(got, &after))
	keys := make(map[string]bool, len(before)+len(after))
	for key, value := range before {
		if !bytes.Equal(value, after[key]) {
			keys[key] = true
		}
	}
	for key, value := range after {
		if !bytes.Equal(value, before[key]) {
			keys[key] = true
		}
	}
	result := make([]string, 0, len(keys))
	for key := range keys {
		result = append(result, key)
	}
	sort.Strings(result)
	if len(result) == 0 {
		return []string{"formatting"}
	}
	return result
}

type recordingTB struct {
	testing.TB
	failed   bool
	messages []string
}

func (*recordingTB) Helper() {}

func (tb *recordingTB) Errorf(format string, args ...any) {
	tb.failed = true
	tb.messages = append(tb.messages, fmt.Sprintf(format, args...))
}

func (tb *recordingTB) Fatalf(format string, args ...any) {
	tb.Errorf(format, args...)
	runtime.Goexit()
}

func (tb *recordingTB) FailNow() {
	tb.failed = true
	runtime.Goexit()
}

func runRecorded(t *testing.T, fn func(tb testing.TB)) *recordingTB {
	t.Helper()
	tb := &recordingTB{TB: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(tb)
	}()
	<-done
	return tb
}

func TestCompareGolden_WhenOutcomeDiffers_FailsWithKeyedDiffAndNeverWritesWithoutUpdate(t *testing.T) {
	t.Parallel()
	if *update {
		t.Skip("golden recording is enabled")
	}
	path := filepath.Join(t.TempDir(), "outcome.golden.json")
	before, err := json.MarshalIndent(map[string]charOutcome{
		"a": {NewState: "before"},
		"b": {NewState: "unchanged"},
	}, "", "  ")
	require.NoError(t, err)
	before = append(before, '\n')
	require.NoError(t, os.WriteFile(path, before, 0o600))
	recorded := runRecorded(t, func(tb testing.TB) {
		compareGolden(tb, path, map[string]charOutcome{
			"a": {NewState: "after"},
			"b": {NewState: "unchanged"},
		})
	})
	require.True(t, recorded.failed)
	require.Contains(t, strings.Join(recorded.messages, "\n"), `"a"`)
	require.NotContains(t, strings.Join(recorded.messages, "\n"), `"b"`)
	after, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestLoadCases_WhenDuplicateCaseName_FailsTest(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "duplicate.cases.json")
	require.NoError(t, os.WriteFile(path, []byte(`[{"name":"validate_duplicate"},{"name":"validate_duplicate"}]`), 0o600))
	recorded := runRecorded(t, func(tb testing.TB) {
		loadCases(tb, path)
	})
	require.True(t, recorded.failed)
	require.Contains(t, strings.Join(recorded.messages, "\n"), "validate_duplicate")
}
