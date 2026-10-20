package vapiv1

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ListValueFromStrings converts a slice of strings to a Terraform types.List.
func ListValueFromStrings(strings []string) types.List {
	elements := make([]attr.Value, len(strings))
	for i, v := range strings {
		elements[i] = types.StringValue(v)
	}
	return types.ListValueMust(types.StringType, elements)
}

// ElementsAsString converts a Terraform types.List into a slice of strings.
func ElementsAsString(list types.List) []string {
	var result []string
	for _, v := range list.Elements() {
		if str, ok := v.(types.String); ok {
			result = append(result, str.ValueString())
		}
	}
	return result
}

// The optional helpers bind an API value to an optional attribute that is not computed. When the
// configuration (or the prior state) left the attribute null and the API returns the zero value, the
// attribute stays null: Terraform rejects a state that differs from the planned null. A value the
// configuration set, including an explicit zero value, and a non-zero API value are stored as they are.

// optionalString binds an optional string attribute.
func optionalString(prior types.String, api string) types.String {
	if api == "" && prior.IsNull() {
		return types.StringNull()
	}
	return types.StringValue(api)
}

// optionalBool binds an optional bool attribute.
func optionalBool(prior types.Bool, api bool) types.Bool {
	if !api && prior.IsNull() {
		return types.BoolNull()
	}
	return types.BoolValue(api)
}

// optionalInt64 binds an optional number attribute with an integer API value.
func optionalInt64(prior types.Int64, api int64) types.Int64 {
	if api == 0 && prior.IsNull() {
		return types.Int64Null()
	}
	return types.Int64Value(api)
}

// optionalFloat64 binds an optional number attribute with a floating point API value.
func optionalFloat64(prior types.Float64, api float64) types.Float64 {
	if api == 0 && prior.IsNull() {
		return types.Float64Null()
	}
	return types.Float64Value(api)
}

// optionalStringList binds an optional list-of-strings attribute; an empty API list keeps a null list null.
func optionalStringList(prior types.List, api []string) types.List {
	if len(api) == 0 && prior.IsNull() {
		return types.ListNull(types.StringType)
	}
	return ListValueFromStrings(api)
}
