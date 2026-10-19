package vapiv1

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kirillve/terraform-provider-vapi/internal/providerdata"
)

func TestConfigureClient_NilProviderData(t *testing.T) {
	t.Parallel()

	resp := &resource.ConfigureResponse{}

	client := configureClient(resource.ConfigureRequest{ProviderData: nil}, resp)

	assert.Nil(t, client)
	assert.Empty(t, resp.Diagnostics)
}

func TestConfigureClient_WrongType(t *testing.T) {
	t.Parallel()

	tests := map[string]any{
		"pointer to config": &providerdata.Config{URL: "https://api.example.com", Token: "test-token"},
		"string":            "https://api.example.com",
	}

	for name, providerData := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := &resource.ConfigureResponse{}

			client := configureClient(resource.ConfigureRequest{ProviderData: providerData}, resp)

			assert.Nil(t, client)
			require.Equal(t, 1, resp.Diagnostics.ErrorsCount())
			assert.Equal(t, "Unexpected Resource Configure Type", resp.Diagnostics.Errors()[0].Summary())
			assert.NotContains(t, resp.Diagnostics.Errors()[0].Detail(), "test-token")
		})
	}
}

func TestResources_OrderAndNames(t *testing.T) {
	t.Parallel()

	want := []string{
		"vapi_assistant",
		"vapi_file",
		"vapi_twilio_phone_number",
		"vapi_tool_function",
		"vapi_tool_query_function",
		"vapi_sip_trunk",
		"vapi_sip_trunk_phone_number",
	}

	factories := Resources()
	require.Len(t, factories, len(want))

	got := make([]string, 0, len(factories))
	for _, newResource := range factories {
		var resp resource.MetadataResponse
		newResource().Metadata(t.Context(), resource.MetadataRequest{ProviderTypeName: "vapi"}, &resp)
		got = append(got, resp.TypeName)
	}
	assert.Equal(t, want, got)

	factories[0] = nil
	assert.NotNil(t, Resources()[0], "Resources must return a fresh slice on every call")
}
