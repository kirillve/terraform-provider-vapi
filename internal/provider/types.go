package provider

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
		result = append(result, v.(types.String).ValueString())
	}
	return result
}
