package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"reflect"
	"sort"
	"testing"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

// unknownMarker represents unknown values distinctly from null.
const unknownMarker = "<unknown>"

// jsonToValue converts raw JSON into a value of type typ.
func jsonToValue(t *testing.T, typ tftypes.Type, raw json.RawMessage) tftypes.Value {
	t.Helper()
	if len(bytes.TrimSpace(raw)) == 0 {
		return tftypes.NewValue(typ, nil)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	require.NoError(t, decoder.Decode(&decoded))
	var trailing any
	require.ErrorIs(t, decoder.Decode(&trailing), io.EOF)
	if decoded == nil {
		return tftypes.NewValue(typ, nil)
	}
	if decoded == unknownMarker {
		return tftypes.NewValue(typ, tftypes.UnknownValue)
	}
	switch kind := typ.(type) {
	case tftypes.Object:
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &fields))
		for name := range fields {
			if _, ok := kind.AttributeTypes[name]; !ok {
				t.Fatalf("unknown attribute %q", name)
			}
		}
		values := make(map[string]tftypes.Value, len(kind.AttributeTypes))
		for name, child := range kind.AttributeTypes {
			values[name] = jsonToValue(t, child, fields[name])
		}
		return tftypes.NewValue(typ, values)
	case tftypes.Map:
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &fields))
		values := make(map[string]tftypes.Value, len(fields))
		for name, child := range fields {
			values[name] = jsonToValue(t, kind.ElementType, child)
		}
		return tftypes.NewValue(typ, values)
	case tftypes.List:
		return tfJSONSequence(t, typ, kind.ElementType, raw)
	case tftypes.Set:
		return tfJSONSequence(t, typ, kind.ElementType, raw)
	}
	switch {
	case typ.Equal(tftypes.String):
		require.IsType(t, "", decoded)
	case typ.Equal(tftypes.Bool):
		require.IsType(t, false, decoded)
	case typ.Equal(tftypes.Number):
		number, ok := decoded.(json.Number)
		require.True(t, ok, "expected JSON number")
		parsed, _, err := big.ParseFloat(string(number), 10, 512, big.ToNearestEven)
		require.NoError(t, err)
		decoded = parsed
	default:
		t.Fatalf("unsupported type %s", typ)
	}
	return tftypes.NewValue(typ, decoded)
}

func tfJSONSequence(t *testing.T, typ, element tftypes.Type, raw json.RawMessage) tftypes.Value {
	t.Helper()
	var items []json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &items))
	values := make([]tftypes.Value, len(items))
	for i, item := range items {
		values[i] = jsonToValue(t, element, item)
	}
	return tftypes.NewValue(typ, values)
}

// valueToJSON converts a value into a JSON-marshalable Go value.
func valueToJSON(t *testing.T, v tftypes.Value) any {
	t.Helper()
	if v.IsNull() {
		return nil
	}
	if !v.IsKnown() {
		return unknownMarker
	}
	switch typ := v.Type(); {
	case typ.Equal(tftypes.String):
		var value string
		require.NoError(t, v.As(&value))
		return value
	case typ.Equal(tftypes.Bool):
		var value bool
		require.NoError(t, v.As(&value))
		return value
	case typ.Equal(tftypes.Number):
		var value big.Float
		require.NoError(t, v.As(&value))
		return json.Number(value.Text('g', -1))
	case typ.Is(tftypes.List{}), typ.Is(tftypes.Set{}):
		var values []tftypes.Value
		require.NoError(t, v.As(&values))
		result := make([]any, 0, len(values))
		for _, value := range values {
			result = append(result, valueToJSON(t, value))
		}
		if typ.Is(tftypes.Set{}) {
			sort.Slice(result, func(i, j int) bool {
				left, err := json.Marshal(result[i])
				require.NoError(t, err)
				right, err := json.Marshal(result[j])
				require.NoError(t, err)
				return string(left) < string(right)
			})
		}
		return result
	case typ.Is(tftypes.Object{}), typ.Is(tftypes.Map{}):
		var values map[string]tftypes.Value
		require.NoError(t, v.As(&values))
		result := make(map[string]any, len(values))
		for name, value := range values {
			result[name] = valueToJSON(t, value)
		}
		return result
	default:
		t.Fatalf("unsupported type %s", typ)
	}
	return nil
}

// proposedNewState derives the proposed new state from prior state and config.
func proposedNewState(t *testing.T, s *tfprotov6.Schema, prior, config tftypes.Value) tftypes.Value {
	t.Helper()
	require.NotNil(t, s.Block)
	require.Empty(t, s.Block.BlockTypes, "blocks are unsupported")
	if prior.IsNull() {
		return config
	}
	return tfProposedAttributes(t, s.ValueType(), s.Block.Attributes, prior, config)
}

func tfProposedAttributes(t *testing.T, typ tftypes.Type, attrs []*tfprotov6.SchemaAttribute, prior, config tftypes.Value) tftypes.Value {
	t.Helper()
	var old, proposed map[string]tftypes.Value
	require.NoError(t, prior.As(&old))
	require.NoError(t, config.As(&proposed))
	for _, attr := range attrs {
		before, after := old[attr.Name], proposed[attr.Name]
		switch {
		case after.IsNull() && attr.Computed:
			proposed[attr.Name] = before
		case attr.NestedType != nil && attr.NestedType.Nesting == tfprotov6.SchemaObjectNestingModeSingle && !before.IsNull() && !after.IsNull() && before.IsKnown() && after.IsKnown():
			proposed[attr.Name] = tfProposedAttributes(t, attr.ValueType(), attr.NestedType.Attributes, before, after)
		}
	}
	return tftypes.NewValue(typ, proposed)
}

// schemaFingerprint returns ordered default and plan modifier descriptions by attribute path.
func schemaFingerprint(ctx context.Context, t *testing.T, s rschema.Schema) map[string][]string {
	t.Helper()
	result := make(map[string][]string)
	tfFingerprintWalk(ctx, t, reflect.ValueOf(s), "", result)
	return result
}

func tfFingerprintWalk(ctx context.Context, t *testing.T, value reflect.Value, path string, result map[string][]string) {
	t.Helper()
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return
	}
	defaultValue := value.FieldByName("Default")
	if defaultValue.IsValid() && !defaultValue.IsNil() {
		result[path] = append(result[path], "default:"+tfFingerprintDescription(ctx, t, defaultValue))
	}
	modifiers := value.FieldByName("PlanModifiers")
	if modifiers.IsValid() {
		for i := 0; i < modifiers.Len(); i++ {
			result[path] = append(result[path], "plan_modifier:"+tfFingerprintDescription(ctx, t, modifiers.Index(i)))
		}
	}
	attrs := value.FieldByName("Attributes")
	if attrs.IsValid() {
		for _, key := range attrs.MapKeys() {
			child := key.String()
			if path != "" {
				child = path + "." + child
			}
			tfFingerprintWalk(ctx, t, attrs.MapIndex(key), child, result)
		}
	}
	nested := value.FieldByName("NestedObject")
	if nested.IsValid() {
		tfFingerprintWalk(ctx, t, nested, path, result)
	}
}

func tfFingerprintDescription(ctx context.Context, t *testing.T, value reflect.Value) string {
	t.Helper()
	describer, ok := value.Interface().(interface{ Description(context.Context) string })
	require.True(t, ok, "expected public Description method")
	return describer.Description(ctx)
}

func TestTFValue_RoundTrip(t *testing.T) {
	t.Parallel()
	object := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"name": tftypes.String, "missing": tftypes.Bool}}
	cases := []struct {
		name, raw string
		typ       tftypes.Type
		expected  any
	}{
		{"null_list_vs_empty_list", `[]`, tftypes.List{ElementType: tftypes.String}, []any{}},
		{"null_vs_false", `false`, tftypes.Bool, false},
		{"unknown_marker", `"<unknown>"`, object, unknownMarker},
		{"big_float_text", `9007199254740993`, tftypes.Number, json.Number("9.007199254740993e+15")},
		{"nested_object", `{"nested":{"name":"test"}}`, tftypes.Object{AttributeTypes: map[string]tftypes.Type{"nested": object}}, map[string]any{"nested": map[string]any{"name": "test", "missing": nil}}},
		{"set", `["z","a"]`, tftypes.Set{ElementType: tftypes.String}, []any{"a", "z"}},
		{"map", `{"yes":true,"absent":null}`, tftypes.Map{ElementType: tftypes.Bool}, map[string]any{"yes": true, "absent": nil}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := jsonToValue(t, tc.typ, json.RawMessage(tc.raw))
			converted := valueToJSON(t, value)
			require.Equal(t, tc.expected, converted)
			raw, err := json.Marshal(converted)
			require.NoError(t, err)
			require.True(t, value.Equal(jsonToValue(t, tc.typ, raw)))
			if tc.name == "null_list_vs_empty_list" || tc.name == "null_vs_false" {
				for _, input := range []json.RawMessage{nil, json.RawMessage(`null`)} {
					null := jsonToValue(t, tc.typ, input)
					require.Nil(t, valueToJSON(t, null))
					require.True(t, null.Equal(jsonToValue(t, tc.typ, json.RawMessage(`null`))))
					require.False(t, null.Equal(value))
				}
			}
			if tc.name == "big_float_text" {
				large := jsonToValue(t, tc.typ, json.RawMessage(`1e30`))
				encoded, err := json.Marshal(valueToJSON(t, large))
				require.NoError(t, err)
				require.True(t, large.Equal(jsonToValue(t, tc.typ, encoded)))
				require.Equal(t, json.Number("1e+30"), valueToJSON(t, large))
			}
		})
	}
}

func TestTFValue_ProposedState(t *testing.T) {
	t.Parallel()
	s := &tfprotov6.Schema{Block: &tfprotov6.SchemaBlock{Attributes: []*tfprotov6.SchemaAttribute{
		{Name: "computed", Type: tftypes.String, Optional: true, Computed: true},
		{Name: "optional", Type: tftypes.String, Optional: true},
	}}}
	cases := []struct{ name, prior, config, expected string }{
		{"computed_null_takes_prior", `{"computed":"old"}`, `{}`, `{"computed":"old"}`},
		{"config_wins_over_prior", `{"computed":"old","optional":"old"}`, `{"computed":"new","optional":"new"}`, `{"computed":"new","optional":"new"}`},
		{"create_computed_stays_null", `null`, `{"optional":"new"}`, `{"optional":"new"}`},
		{"optional_removed_goes_null", `{"optional":"old"}`, `{}`, `{}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prior := jsonToValue(t, s.ValueType(), json.RawMessage(tc.prior))
			config := jsonToValue(t, s.ValueType(), json.RawMessage(tc.config))
			expected := jsonToValue(t, s.ValueType(), json.RawMessage(tc.expected))
			actual := proposedNewState(t, s, prior, config)
			require.True(t, s.ValueType().Equal(actual.Type()))
			require.Equal(t, valueToJSON(t, expected), valueToJSON(t, actual))
		})
	}
}

func TestTFValue_SchemaFingerprint(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"modifier_order_is_declared_order", "default_listed_before_modifiers", "nested_path_uses_dots", "attribute_without_modifiers_is_omitted"} {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			replace, state := stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()
			attr := rschema.StringAttribute{Optional: true, PlanModifiers: []planmodifier.String{replace, state}}
			expected := []string{"plan_modifier:" + replace.Description(ctx), "plan_modifier:" + state.Description(ctx)}
			s := rschema.Schema{Attributes: map[string]rschema.Attribute{"value": attr, "plain": rschema.StringAttribute{Optional: true}}}
			switch name {
			case "modifier_order_is_declared_order":
				require.Equal(t, expected, schemaFingerprint(ctx, t, s)["value"])
				attr.PlanModifiers = []planmodifier.String{state, replace}
				s.Attributes["value"] = attr
				reversed := schemaFingerprint(ctx, t, s)["value"]
				require.Equal(t, []string{expected[1], expected[0]}, reversed)
				require.NotEqual(t, expected, reversed)
			case "default_listed_before_modifiers":
				attr.Default = stringdefault.StaticString("test")
				s.Attributes["value"] = attr
				require.Equal(t, append([]string{"default:" + attr.Default.Description(ctx)}, expected...), schemaFingerprint(ctx, t, s)["value"])
			case "nested_path_uses_dots":
				s.Attributes["nested"] = rschema.SingleNestedAttribute{Optional: true, Attributes: map[string]rschema.Attribute{"child": attr}}
				s.Attributes["list"] = rschema.ListNestedAttribute{Optional: true, NestedObject: rschema.NestedAttributeObject{Attributes: map[string]rschema.Attribute{"child": attr}}}
				result := schemaFingerprint(ctx, t, s)
				require.Equal(t, expected, result["nested.child"])
				require.Equal(t, expected, result["list.child"])
			case "attribute_without_modifiers_is_omitted":
				require.Equal(t, map[string][]string{"value": expected}, schemaFingerprint(ctx, t, s))
			}
		})
	}
}
