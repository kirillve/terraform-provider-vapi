package vapiv1

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kirillve/terraform-provider-vapi/internal/providerdata"
	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
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

func TestFlattenModel_ReasoningEffort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		owner    types.String
		body     string
		want     types.String
		nilPrior bool
		wantNil  bool
	}{
		{name: "unset omitted", body: `{}`, want: types.StringNull()},
		{name: "unset null", body: `{"reasoningEffort":null}`, want: types.StringNull()},
		{name: "unset empty", body: `{"reasoningEffort":""}`, want: types.StringNull()},
		{name: "unset server default", body: `{"reasoningEffort":"none"}`, want: types.StringNull()},
		{name: "unset high", body: `{"reasoningEffort":"high"}`, want: types.StringNull()},
		{name: "nil prior substantive model", body: `{"model":"gpt-6-luna","reasoningEffort":"none"}`, want: types.StringNull(), nilPrior: true},
		{name: "managed echo", owner: types.StringValue("none"), body: `{"reasoningEffort":"none"}`, want: types.StringValue("none")},
		{name: "managed drift", owner: types.StringValue("none"), body: `{"reasoningEffort":"low"}`, want: types.StringValue("low")},
		{name: "managed outside enum", owner: types.StringValue("none"), body: `{"reasoningEffort":"max"}`, want: types.StringValue("max")},
		{name: "managed omitted", owner: types.StringValue("none"), body: `{}`, want: types.StringNull()},
		{name: "managed null", owner: types.StringValue("none"), body: `{"reasoningEffort":null}`, want: types.StringNull()},
		{name: "managed empty", owner: types.StringValue("none"), body: `{"reasoningEffort":""}`, want: types.StringNull()},
		{name: "unknown returned", owner: types.StringUnknown(), body: `{"reasoningEffort":"high"}`, want: types.StringValue("high")},
		{name: "unknown empty", owner: types.StringUnknown(), body: `{"reasoningEffort":""}`, want: types.StringNull()},
		{name: "unknown omitted", owner: types.StringUnknown(), body: `{}`, want: types.StringNull()},
		{name: "unknown null", owner: types.StringUnknown(), body: `{"reasoningEffort":null}`, want: types.StringNull()},
		{name: "nil API model", owner: types.StringValue("none"), body: `null`, wantNil: true},
		{name: "nil prior effort only", body: `{"reasoningEffort":"none"}`, nilPrior: true, wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var in *vapi.Model
			require.NoError(t, json.Unmarshal([]byte(tt.body), &in))
			prior := &ModelResourceModel{ReasoningEffort: tt.owner}
			if tt.nilPrior {
				prior = nil
			}
			got := flattenModel(prior, in)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.ReasoningEffort)
			again := flattenModel(got, in)
			require.NotNil(t, again)
			assert.Equal(t, tt.want, again.ReasoningEffort)
		})
	}
}

func TestExpandModel_ReasoningEffort(t *testing.T) {
	t.Parallel()

	tests := map[string]types.String{
		"minimal": types.StringValue("minimal"),
		"none":    types.StringValue("none"),
		"low":     types.StringValue("low"),
		"medium":  types.StringValue("medium"),
		"high":    types.StringValue("high"),
		"xhigh":   types.StringValue("xhigh"),
		"unset":   types.StringNull(),
		"unknown": types.StringUnknown(),
	}
	for name, effort := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			model := reasoningModelFixture(t, effort)
			var out vapi.CreateAssistantRequest
			model.expandVoiceAndModel(&out)
			require.NotNil(t, out.Model)
			assert.Equal(t, &vapi.Model{
				Model: "gpt-6-luna", SystemPrompt: "System prompt", Provider: "openai",
				MaxTokens: 512, Temperature: 0.3, ToolIDs: []string{"tool-1"},
				KnowledgeBase:   &vapi.KnowledgeBase{TopK: 2, FileIDs: []string{"file-1"}, Provider: "canonical"},
				ReasoningEffort: effort.ValueString(),
			}, out.Model)
			assertReasoningRequest(t, &out, effort.ValueString())
		})
	}
	t.Run("absent model", func(t *testing.T) {
		t.Parallel()
		var model VAPIAssistantResourceModel
		var out vapi.CreateAssistantRequest
		model.expandVoiceAndModel(&out)
		assert.Nil(t, out.Model)
		assertReasoningRequest(t, &out, "")
	})
}

func reasoningModelFixture(t *testing.T, effort types.String) *VAPIAssistantResourceModel {
	t.Helper()
	toolIDs, diags := types.ListValueFrom(t.Context(), types.StringType, []string{"tool-1"})
	require.Empty(t, diags)
	fileIDs, diags := types.ListValueFrom(t.Context(), types.StringType, []string{"file-1"})
	require.Empty(t, diags)
	return &VAPIAssistantResourceModel{Model: &ModelResourceModel{
		Model: types.StringValue("gpt-6-luna"), SystemPrompt: types.StringValue("System prompt"),
		Provider: types.StringValue("openai"), MaxTokens: types.Int64Value(512),
		Temperature: types.Float64Value(0.3), ToolIDs: toolIDs, ReasoningEffort: effort,
		KnowledgeBase: &KnowledgeBaseResourceModel{
			TopK: types.Int64Value(2), FileIDs: fileIDs, Provider: types.StringValue("canonical"),
		},
	}}
}

func assertReasoningRequest(t *testing.T, out *vapi.CreateAssistantRequest, want string) {
	t.Helper()
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	var request map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &request))
	if out.Model == nil {
		assert.NotContains(t, request, "model")
		return
	}
	var model map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(request["model"], &model))
	if want == "" {
		assert.NotContains(t, model, "reasoningEffort")
		return
	}
	require.Contains(t, model, "reasoningEffort")
	assert.JSONEq(t, `"`+want+`"`, string(model["reasoningEffort"]))
}
