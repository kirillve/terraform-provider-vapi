package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tfpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requiredCases lists the case names required for every resource type.
var requiredCases = []string{
	"plan_create",
	"plan_update",
	"create",
	"update",
	"read",
	"read_partial",
	"delete",
	"create_500",
	"update_500",
	"read_500",
	"read_404",
	"delete_404",
	"delete_500",
	"delete_403",
	"create_malformed",
	"update_malformed",
	"read_malformed",
	"read_body_error",
	"create_transport_error",
	"read_transport_error",
	"update_transport_error",
	"delete_transport_error",
	"read_empty_id",
	"update_empty_id",
	"delete_empty_id",
}

// requiredCasesByType lists the additional case names required for each resource type.
var requiredCasesByType = map[string][]string{
	"vapi_assistant": {
		"import_unsupported",
		"create_unencodable",
		"update_unencodable",
		"create_full",
		"create_nil_nested_child",
		"create_artifact_default_mp3",
		"create_unicode_false_zero",
		"read_nil_nested_child",
		"create_minimal_zero_values",
		"create_minimal_explicit_zeros",
		"create_minimal_empty_blocks",
		"update_minimal_rename",
		"read_minimal_zero_values",
		"read_explicit_zero_kept",
		"upgrade_v0_13_0_state",
		"upgrade_v0_13_0_state_no_model",
		"upgrade_v0_15_1_state",
		"upgrade_v0_15_1_state_no_model",
		"validate_reasoning_effort_valid_minimal",
		"validate_reasoning_effort_valid_none",
		"validate_reasoning_effort_valid_low",
		"validate_reasoning_effort_valid_medium",
		"validate_reasoning_effort_valid_high",
		"validate_reasoning_effort_valid_xhigh",
		"validate_reasoning_effort_null",
		"validate_reasoning_effort_unknown",
		"validate_reasoning_effort_invalid",
		"validate_reasoning_effort_case",
		"validate_reasoning_effort_empty",
		"validate_reasoning_effort_max",
		"validate_reasoning_effort_whitespace",
		"read_unmanaged_default",
		"read_unmanaged_empty",
		"read_effort_only_unmanaged_model",
		"plan_update_reasoning_effort_absent_noop",
		"plan_update_reasoning_effort_absent_other_change",
		"create_none",
		"update_high",
		"create_unset",
		"read_managed_echo",
		"read_managed_drift",
		"read_managed_missing",
		"create_managed_missing",
		"update_managed_different",
		"update_remove",
		"read_bad_effort_type",
	},
	"vapi_file": {
		"import",
		"import_read_404",
		"create_dir_filename",
		"create_bytes_as_string",
		"update_upload_fails_after_delete",
		"update_delete_404",
		"update_delete_403",
	},
	"vapi_sip_trunk_phone_number": {
		"import",
		"import_read_404",
		"create_bool_false",
	},
	"vapi_sip_trunk": {
		"create_optional_strings_omitted",
		"create_unset_explicit_zeros",
		"create_unset_empty_blocks",
		"read_unset_empty_blocks",
		"update_unset_empty_blocks",
		"import",
		"import_read_404",
		"create_auth_nil",
		"create_register_nil",
		"create_gateways_nil",
		"create_gateways_empty",
		"create_bool_false",
	},
	"vapi_tool_query_function": {
		"read_description_omitted",
		"read_unset_explicit_zeros",
		"import",
		"import_read_404",
		"create_descriptions_optional",
		"create_file_ids_empty",
	},
	"vapi_twilio_phone_number": {
		"create_fallback_check_empty_string",
		"create_unset_explicit_zeros",
		"create_unset_empty_blocks",
		"import",
		"import_read_404",
		"create_sms_omitted",
		"create_sms_false",
		"update_remove_to_null",
	},
	"vapi_tool_function": {
		"create_parameters_api_null",
		"create_property_description_omitted",
		"create_unset_explicit_zeros",
		"create_unset_empty_blocks",
		"import",
		"import_read_404",
		"validate_none",
		"validate_both",
		"validate_non_object",
		"validate_unknown",
		"upgrade_v0",
		"import_nested",
		"create_dtmf",
		"create_transfer_call",
		"create_default_kind",
		"create_params_nested",
		"create_enum_empty",
		"create_timeout_explicit",
		"read_timeout_drift",
	},
}

func TestV1CaseCoverage(t *testing.T) {
	for _, typeName := range v1Types {
		t.Run(typeName, func(t *testing.T) {
			t.Parallel()
			checkTypeCoverage(t, typeName)
		})
	}
}

func checkTypeCoverage(t *testing.T, typeName string) {
	t.Helper()
	dir := filepath.Join("testdata", "v1", typeName)
	require.DirExists(t, dir)
	require.FileExists(t, filepath.Join(dir, "schema.golden.json"))
	paths, err := filepath.Glob(filepath.Join(dir, "*.cases.json"))
	require.NoError(t, err)
	require.NotEmpty(t, paths, "no case files for %s", typeName)
	groups := make(map[string]string)
	for _, path := range paths {
		group := strings.TrimSuffix(filepath.Base(path), ".cases.json")
		names := caseNames(t, path)
		keys := goldenKeys(t, filepath.Join(dir, group+".golden.json"))
		assert.ElementsMatch(t, names, keys, "case and golden key sets differ for group %q", group)
		for _, name := range names {
			previous, exists := groups[name]
			assert.False(t, exists, "case %q is in both %s and %s", name, previous, group)
			groups[name] = group
		}
	}
	required := append([]string(nil), requiredCases...)
	required = append(required, requiredCasesByType[typeName]...)
	for _, name := range required {
		assert.Contains(t, groups, name, "required case %q is missing", name)
	}
}

func caseNames(t *testing.T, path string) []string {
	t.Helper()
	cases := loadCases(t, path)
	names := make([]string, 0, len(cases))
	for i := range cases {
		names = append(names, cases[i].Name)
	}
	return names
}

func goldenKeys(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)
	var golden map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &golden))
	keys := make([]string, 0, len(golden))
	for key := range golden {
		keys = append(keys, key)
	}
	return keys
}

func TestAssistantExampleReasoningEffortInEnum(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../examples/resources/vapi_assistant/resource.tf")
	require.NoError(t, err)
	literals := regexp.MustCompile(`(?m)^\s*reasoning_effort\s*=\s*"([^"\r\n]*)"`).FindAllStringSubmatch(string(raw), -1)
	require.NotEmpty(t, literals, "example must configure reasoning_effort")
	p := New("test")()
	response := characterizationSchema(t, "vapi_assistant", p.Resources(t.Context()))
	model, ok := response.Schema.Attributes["model"].(schema.SingleNestedAttribute)
	require.True(t, ok, "model must be a single nested attribute")
	effort, ok := model.Attributes["reasoning_effort"].(schema.StringAttribute)
	require.True(t, ok, "reasoning_effort must be a string attribute")
	for _, literal := range literals {
		for _, check := range effort.Validators {
			var result validator.StringResponse
			check.ValidateString(t.Context(), validator.StringRequest{
				Path:        tfpath.Root("model").AtName("reasoning_effort"),
				ConfigValue: types.StringValue(literal[1]),
			}, &result)
			assert.False(t, result.Diagnostics.HasError(), "example effort %q: %v", literal[1], result.Diagnostics)
		}
	}
}
