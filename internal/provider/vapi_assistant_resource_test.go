package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

func TestVAPIAssistantResourceLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	createResponse := mustMarshal(t, vapi.Assistant{
		ID:                        "assistant-1",
		OrgID:                     "org-1",
		Name:                      "assistant",
		FirstMessageMode:          "immediate",
		HipaaEnabled:              true,
		ClientMessages:            []string{"client-1"},
		ServerMessages:            []string{"server-1"},
		BackgroundSound:           "lofi",
		BackgroundDenoising:       true,
		ModelOutputEnabled:        true,
		Language:                  "en",
		ForwardingPhoneNumber:     "+123456789",
		InterruptionsEnabled:      true,
		EndCallFunctionEnabled:    true,
		DialKeypadFunctionEnabled: true,
		FillersEnabled:            false,
		MaxDurationSeconds:        120,
		AnalysisPlan: &vapi.AnalysisPlan{
			SummaryPrompt:           "summary",
			StructuredDataPrompt:    "structured",
			StructuredDataSchema:    &vapi.StructuredDataSchema{Type: "object", Properties: map[string]*vapi.Property{"field": {Type: "string", Description: "desc"}}},
			SuccessEvaluationPrompt: "success?",
			SuccessEvaluationRubric: "rubric",
		},
		MessagePlan: &vapi.MessagePlan{
			IdleMessages: []string{"idle"},
		},
		Model: &vapi.Model{
			Model:        "gpt-4",
			SystemPrompt: "system",
			Provider:     "openai",
			MaxTokens:    4096,
			Temperature:  0.5,
			ToolIDs:      []string{"tool-1"},
			KnowledgeBase: &vapi.KnowledgeBase{
				TopK:     3,
				FileIDs:  []string{"file-1"},
				Provider: "kb-provider",
			},
		},
		Voice: &vapi.Voice{
			Model:           "voice-model",
			Provider:        "voice-provider",
			VoiceID:         "voice-1",
			Stability:       0.9,
			SimilarityBoost: 0.8,
		},
	})

	updateResponse := mustMarshal(t, vapi.Assistant{
		ID:                        "assistant-1",
		OrgID:                     "org-1",
		Name:                      "assistant-updated",
		FirstMessageMode:          "immediate",
		HipaaEnabled:              true,
		ClientMessages:            []string{"client-1"},
		ServerMessages:            []string{"server-1"},
		BackgroundSound:           "lofi",
		BackgroundDenoising:       true,
		ModelOutputEnabled:        true,
		Language:                  "en",
		ForwardingPhoneNumber:     "+123456789",
		InterruptionsEnabled:      true,
		EndCallFunctionEnabled:    true,
		DialKeypadFunctionEnabled: true,
		FillersEnabled:            false,
		MaxDurationSeconds:        120,
		AnalysisPlan: &vapi.AnalysisPlan{
			SummaryPrompt:           "summary",
			StructuredDataPrompt:    "structured",
			StructuredDataSchema:    &vapi.StructuredDataSchema{Type: "object", Properties: map[string]*vapi.Property{"field": {Type: "string", Description: "desc"}}},
			SuccessEvaluationPrompt: "success?",
			SuccessEvaluationRubric: "rubric",
		},
		MessagePlan: &vapi.MessagePlan{
			IdleMessages: []string{"idle"},
		},
		Model: &vapi.Model{
			Model:        "gpt-4",
			SystemPrompt: "system",
			Provider:     "openai",
			MaxTokens:    4096,
			Temperature:  0.5,
			ToolIDs:      []string{"tool-1"},
		},
		Voice: &vapi.Voice{
			Model:           "voice-model",
			Provider:        "voice-provider",
			VoiceID:         "voice-1",
			Stability:       0.9,
			SimilarityBoost: 0.8,
		},
	})

	transport := &queueRoundTripper{
		t: t,
		responses: []queuedResponse{
			{method: http.MethodPost, path: "/assistant", status: 200, body: createResponse},
			{method: http.MethodGet, path: "/assistant/assistant-1", status: 200, body: createResponse},
			{method: http.MethodPatch, path: "/assistant/assistant-1", status: 200, body: updateResponse},
			{method: http.MethodDelete, path: "/assistant/assistant-1", status: 200, body: []byte(`{}`)},
		},
	}

	res := &VAPIAssistantResource{
		client: &vapi.APIClient{
			BaseURL:    "https://api.example.com",
			Token:      "token",
			HTTPClient: &http.Client{Transport: transport},
		},
	}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	createPlan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := createPlan.Set(ctx, assistantTestModel()); diags.HasError() {
		t.Fatalf("plan.Set diagnostics: %v", diags)
	}

	createResp := resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	res.Create(ctx, resource.CreateRequest{Plan: createPlan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", createResp.Diagnostics)
	}

	readResp := resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	res.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", readResp.Diagnostics)
	}

	updateModel := assistantTestModel()
	updateModel.Name = types.StringValue("assistant-updated")
	updatePlan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := updatePlan.Set(ctx, updateModel); diags.HasError() {
		t.Fatalf("update plan diagnostics: %v", diags)
	}

	updateResp := resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	res.Update(ctx, resource.UpdateRequest{
		State: readResp.State,
		Plan:  updatePlan,
	}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", updateResp.Diagnostics)
	}

	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", deleteResp.Diagnostics)
	}

	transport.assertDrained()
}

func assistantTestModel() VAPIAssistantResourceModel {
	list := func(values ...string) types.List {
		return ListValueFromStrings(values)
	}

	return VAPIAssistantResourceModel{
		Name:                         types.StringValue("assistant"),
		FirstMessageMode:             types.StringValue("immediate"),
		HipaaEnabled:                 types.BoolValue(true),
		ClientMessages:               list("client-1"),
		ServerMessages:               list("server-1"),
		SilenceTimeoutSeconds:        types.Float64Value(3.5),
		MaxDurationSeconds:           types.Int64Value(120),
		BackgroundSound:              types.StringValue("lofi"),
		BackgroundDenoising:          types.BoolValue(true),
		ModelOutputEnabled:           types.BoolValue(true),
		FirstMessage:                 types.StringValue("welcome"),
		VoicemailMessage:             types.StringValue("leave a message"),
		EndCallMessage:               types.StringValue("goodbye"),
		ServerURL:                    types.StringValue("https://hook.example.com"),
		ServerURLSecret:              types.StringValue("secret"),
		EndCallPhrases:               list("bye"),
		RecordingEnabled:             types.BoolValue(true),
		ForwardingPhoneNumber:        types.StringValue("+123456789"),
		PhoneNumberID:                types.StringValue("phone-1"),
		Language:                     types.StringValue("en"),
		InterruptionsEnabled:         types.BoolValue(true),
		DialKeypadFunctionEnabled:    types.BoolValue(true),
		FillersEnabled:               types.BoolValue(false),
		ResponseDelaySeconds:         types.Float64Value(0.75),
		NumWordsToInterruptAssistant: types.Int64Value(4),
		LiveTranscriptsEnabled:       types.BoolValue(true),
		Keywords:                     list("sales", "demo"),
		ParentID:                     types.StringValue("parent-1"),
		MessagePlan: &MessagePlanResourceModel{
			IdleMessages: list("idle"),
		},
		Transcriber: &TranscriberResourceModel{
			Provider: types.StringValue("provider"),
			Model:    types.StringValue("model"),
			Language: types.StringValue("en"),
		},
		Model: &ModelResourceModel{
			Model:        types.StringValue("gpt-4"),
			SystemPrompt: types.StringValue("system"),
			Provider:     types.StringValue("openai"),
			Temperature:  types.Float64Value(0.5),
			MaxTokens:    types.Int64Value(4096),
			ToolIDs:      list("tool-1"),
			KnowledgeBase: &KnowledgeBaseResourceModel{
				TopK:     types.Int64Value(3),
				FileIDs:  list("file-1"),
				Provider: types.StringValue("kb-provider"),
			},
		},
		Voice: &VoiceResourceModel{
			Model:           types.StringValue("voice-model"),
			Provider:        types.StringValue("voice-provider"),
			VoiceID:         types.StringValue("voice-1"),
			Stability:       types.Float64Value(0.9),
			SimilarityBoost: types.Float64Value(0.8),
		},
		AnalysisPlan: &AnalysisPlanResourceModel{
			SummaryPrompt:        types.StringValue("summary"),
			StructuredDataPrompt: types.StringValue("structured"),
			StructuredDataSchema: &StructuredDataSchemaResourceModel{
				Type: types.StringValue("object"),
				Properties: map[string]PropertyResourceModel{
					"field": {
						Type:        types.StringValue("string"),
						Description: types.StringValue("desc"),
					},
				},
			},
			SuccessEvaluationPrompt: types.StringValue("success?"),
			SuccessEvaluationRubric: types.StringValue("rubric"),
		},
		StartSpeakingPlan: &StartSpeakingPlanResourceModel{
			WaitSeconds:             types.Float64Value(1.2),
			SmartEndpointingEnabled: types.BoolValue(true),
			TranscriptionEndpointingPlan: &TranscriptionEndpointingPlanResourceModel{
				OnPunctuationSeconds:   types.Float64Value(0.3),
				OnNoPunctuationSeconds: types.Float64Value(0.6),
				OnNumberSeconds:        types.Float64Value(0.7),
			},
		},
		StopSpeakingPlan: &StopSpeakingPlanResourceModel{
			NumWords:       types.Float64Value(2),
			VoiceSeconds:   types.Float64Value(1.5),
			BackoffSeconds: types.Float64Value(0.5),
		},
	}
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	return data
}
