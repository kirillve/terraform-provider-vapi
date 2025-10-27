package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

func TestListValueConversions(t *testing.T) {
	list := ListValueFromStrings([]string{"a", "b"})
	if list.ElementType(ctxForTest()) != types.StringType {
		t.Fatalf("expected string list, got %v", list.ElementType(ctxForTest()))
	}

	out := ElementsAsString(list)
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Fatalf("unexpected conversion result: %v", out)
	}

	empty := ElementsAsString(types.ListNull(types.StringType))
	if len(empty) != 0 {
		t.Fatalf("expected empty slice, got %v", empty)
	}
}

func TestMapVAPIAssistantRequestAndResponse(t *testing.T) {
	t.Helper()

	listFrom := func(values ...string) types.List {
		return ListValueFromStrings(values)
	}

	model := VAPIAssistantResourceModel{
		Name:                         types.StringValue("assistant"),
		FirstMessageMode:             types.StringValue("immediate"),
		HipaaEnabled:                 types.BoolValue(true),
		ClientMessages:               listFrom("client-1"),
		ServerMessages:               listFrom("server-1"),
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
		EndCallPhrases:               listFrom("bye"),
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
		Keywords:                     listFrom("sales", "demo"),
		ParentID:                     types.StringValue("parent-1"),
		MessagePlan: &MessagePlanResourceModel{
			IdleMessages: listFrom("idle"),
		},
		ArtifactPlan: &ArtifactPlanResourceModel{
			RecordingFormat: types.StringNull(),
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
			ToolIDs:      listFrom("tool-1"),
			KnowledgeBase: &KnowledgeBaseResourceModel{
				TopK:     types.Int64Value(3),
				FileIDs:  listFrom("file-1"),
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
	}

	request := mapVAPIAssistantRequest(&model)
	if request.Name != "assistant" || request.Model == nil || request.Model.KnowledgeBase == nil {
		t.Fatalf("unexpected request mapping: %#v", request)
	}
	if request.ArtifactPlan == nil || request.ArtifactPlan.RecordingFormat != "mp3" {
		t.Fatalf("expected default recording format, got %#v", request.ArtifactPlan)
	}
	if request.Server == nil || request.Server.URL != "https://hook.example.com" {
		t.Fatalf("expected server mapping, got %#v", request.Server)
	}

	parent := "parent-1"
	silence := 3.5
	respDelay := 0.75
	numWords := int64(4)
	live := true

	resp := &vapi.Assistant{
		ID:                           "assistant-1",
		OrgID:                        "org-1",
		Name:                         request.Name,
		FirstMessageMode:             request.FirstMessageMode,
		HipaaEnabled:                 request.HipaaEnabled,
		ClientMessages:               request.ClientMessages,
		ServerMessages:               request.ServerMessages,
		BackgroundSound:              request.BackgroundSound,
		BackgroundDenoising:          request.BackgroundDenoising,
		ModelOutputEnabled:           request.ModelOutputEnabled,
		Language:                     request.Language,
		ForwardingPhoneNumber:        request.ForwardingPhoneNumber,
		InterruptionsEnabled:         request.InterruptionsEnabled,
		EndCallFunctionEnabled:       request.EndCallFunctionEnabled,
		DialKeypadFunctionEnabled:    request.DialKeypadFunctionEnabled,
		FillersEnabled:               request.FillersEnabled,
		SilenceTimeoutSeconds:        &silence,
		ResponseDelaySeconds:         &respDelay,
		NumWordsToInterruptAssistant: &numWords,
		LiveTranscriptsEnabled:       &live,
		Keywords:                     request.Keywords,
		ParentID:                     &parent,
		Voice: &vapi.Voice{
			Model:           request.Voice.Model,
			VoiceID:         request.Voice.VoiceID,
			Provider:        request.Voice.Provider,
			Stability:       request.Voice.Stability,
			SimilarityBoost: request.Voice.SimilarityBoost,
		},
		Model: &vapi.Model{
			Model:        request.Model.Model,
			SystemPrompt: request.Model.SystemPrompt,
			Provider:     request.Model.Provider,
			Temperature:  request.Model.Temperature,
			MaxTokens:    request.Model.MaxTokens,
			ToolIDs:      request.Model.ToolIDs,
			KnowledgeBase: &vapi.KnowledgeBase{
				TopK:     request.Model.KnowledgeBase.TopK,
				FileIDs:  request.Model.KnowledgeBase.FileIDs,
				Provider: request.Model.KnowledgeBase.Provider,
			},
		},
		RecordingEnabled: request.RecordingEnabled,
		FirstMessage:     request.FirstMessage,
		VoicemailMessage: request.VoicemailMessage,
		EndCallMessage:   request.EndCallMessage,
		Transcriber: &vapi.Transcriber{
			Provider: request.Transcriber.Provider,
			Model:    request.Transcriber.Model,
			Language: request.Transcriber.Language,
		},
		EndCallPhrases:     request.EndCallPhrases,
		MaxDurationSeconds: request.MaxDurationSeconds,
		AnalysisPlan: &vapi.AnalysisPlan{
			SummaryPrompt:           request.AnalysisPlan.SummaryPrompt,
			StructuredDataPrompt:    request.AnalysisPlan.StructuredDataPrompt,
			StructuredDataSchema:    request.AnalysisPlan.StructuredDataSchema,
			SuccessEvaluationPrompt: request.AnalysisPlan.SuccessEvaluationPrompt,
			SuccessEvaluationRubric: request.AnalysisPlan.SuccessEvaluationRubric,
		},
		MessagePlan: &vapi.MessagePlan{
			IdleMessages: request.MessagePlan.IdleMessages,
		},
		StartSpeakingPlan: &vapi.StartSpeakingPlan{
			WaitSeconds:             request.StartSpeakingPlan.WaitSeconds,
			SmartEndpointingEnabled: request.StartSpeakingPlan.SmartEndpointingEnabled,
			TranscriptionEndpointingPlan: &vapi.TranscriptionEndpointingPlan{
				OnPunctuationSeconds:   request.StartSpeakingPlan.TranscriptionEndpointingPlan.OnPunctuationSeconds,
				OnNoPunctuationSeconds: request.StartSpeakingPlan.TranscriptionEndpointingPlan.OnNoPunctuationSeconds,
				OnNumberSeconds:        request.StartSpeakingPlan.TranscriptionEndpointingPlan.OnNumberSeconds,
			},
		},
		StopSpeakingPlan: &vapi.StopSpeakingPlan{
			NumWords:       request.StopSpeakingPlan.NumWords,
			VoiceSeconds:   request.StopSpeakingPlan.VoiceSeconds,
			BackoffSeconds: request.StopSpeakingPlan.BackoffSeconds,
		},
		ArtifactPlan: &vapi.ArtifactPlan{
			RecordingFormat: "mp3",
		},
	}

	var mapped VAPIAssistantResourceModel
	mapResponseObject(&mapped, resp)

	if mapped.Name.ValueString() != "assistant" {
		t.Fatalf("unexpected name mapping: %#v", mapped.Name)
	}
	if mapped.Model == nil || mapped.Model.KnowledgeBase == nil {
		t.Fatalf("expected knowledge base mapping")
	}
	if mapped.StartSpeakingPlan == nil || mapped.StopSpeakingPlan == nil {
		t.Fatalf("expected speaking plans mapped: %#v", mapped)
	}
	if mapped.AnalysisPlan == nil || mapped.AnalysisPlan.StructuredDataSchema == nil {
		t.Fatalf("expected analysis plan mapping: %#v", mapped.AnalysisPlan)
	}
}

func TestMapVAPIAssistantRequestNilBranches(t *testing.T) {
	minimal := VAPIAssistantResourceModel{
		Name:             types.StringValue("assistant"),
		FirstMessageMode: types.StringValue("immediate"),
		HipaaEnabled:     types.BoolValue(false),
		ClientMessages:   ListValueFromStrings(nil),
		ServerMessages:   ListValueFromStrings(nil),
		EndCallPhrases:   ListValueFromStrings(nil),
		Keywords:         ListValueFromStrings(nil),
	}

	req := mapVAPIAssistantRequest(&minimal)
	if req.Model != nil || req.Voice != nil || req.MessagePlan != nil {
		t.Fatalf("expected optional blocks to be nil")
	}

	resp := &vapi.Assistant{
		ID: "id",
	}
	var mapped VAPIAssistantResourceModel
	mapResponseObject(&mapped, resp)
	if mapped.Model != nil || mapped.Voice != nil || mapped.MessagePlan != nil {
		t.Fatalf("expected optional fields cleared on nil response")
	}
}

func TestCreateRequestObjectHelpers(t *testing.T) {
	if createRequestObject(nil) != nil {
		t.Fatalf("expected nil input to return nil")
	}
	schemaModel := &StructuredDataSchemaResourceModel{
		Type: types.StringValue("object"),
		Properties: map[string]PropertyResourceModel{
			"field": {
				Type:        types.StringValue("string"),
				Description: types.StringValue("description"),
			},
		},
	}
	req := createRequestObject(schemaModel)
	if req == nil || req.Properties["field"].Description != "description" {
		t.Fatalf("unexpected request object: %#v", req)
	}

	back := mapStructuredDataSchemaRequestToResourceModel(req)
	if back == nil || back.Properties["field"].Description.ValueString() != "description" {
		t.Fatalf("unexpected mapped resource: %#v", back)
	}
	if mapStructuredDataSchemaRequestToResourceModel(nil) != nil {
		t.Fatalf("expected nil schema to return nil")
	}
}

func TestConvertHelpers(t *testing.T) {
	gateways := convertGateways([]SIPGatewayModel{
		{IP: types.StringValue("10.0.0.1")},
		{IP: types.StringValue("10.0.0.2")},
	})
	if len(gateways) != 2 || gateways[1].IP != "10.0.0.2" {
		t.Fatalf("unexpected gateways: %#v", gateways)
	}

	if convertAuthPlan(nil) != nil {
		t.Fatalf("expected nil auth plan")
	}

	plan := convertAuthPlan(&OutboundAuthenticationPlanModel{
		AuthUsername: types.StringValue("user"),
		AuthPassword: types.StringValue("pass"),
		SIPRegisterPlan: &SIPRegisterPlanModel{
			Domain:   types.StringValue("example.com"),
			Username: types.StringValue("sip-user"),
			Realm:    types.StringValue("realm"),
		},
	})
	if plan == nil || plan.SIPRegisterPlan == nil || plan.SIPRegisterPlan.Username != "sip-user" {
		t.Fatalf("unexpected auth plan: %#v", plan)
	}
}

func TestBindVAPIToolFunctionResourceData(t *testing.T) {
	resp := &vapi.ToolFunctionResponse{
		ID:        "tool-1",
		OrgID:     "org-1",
		CreatedAt: "now",
		UpdatedAt: "later",
		Type:      "function",
		Async:     true,
		Server: vapi.ResponseServer{
			URL: "https://server.example.com",
		},
		Function: vapi.ResponseFunction{
			Name:        "name",
			Description: "desc",
			Async:       true,
			Parameters: vapi.ResponseFunctionParams{
				Type:     "object",
				Required: []string{"foo", "bar"},
				Properties: map[string]vapi.ResponseProperty{
					"foo": {
						Type:        "string",
						Description: "a",
						Enum:        []string{"x", "y"},
					},
				},
			},
		},
	}

	var model VAPIToolFunctionResourceModel
	bindVAPIToolFunctionResourceData(&model, resp)

	if model.ID.ValueString() != "tool-1" || model.ServerURL.ValueString() != "https://server.example.com" {
		t.Fatalf("unexpected ID or server mapping: %#v", model)
	}

	if len(model.Parameters.Properties) != 1 {
		t.Fatalf("expected property mapping, got %#v", model.Parameters.Properties)
	}

	enumVals := ElementsAsString(model.Parameters.Properties["foo"].Enum)
	if len(enumVals) != 2 || enumVals[0] != "x" {
		t.Fatalf("unexpected enum mapping: %#v", enumVals)
	}
}

type queuedResponse struct {
	method string
	path   string
	status int
	body   []byte
}

type queueRoundTripper struct {
	t         *testing.T
	responses []queuedResponse
}

func (rt *queueRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(rt.responses) == 0 {
		rt.t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
	}

	next := rt.responses[0]
	rt.responses = rt.responses[1:]

	if req.Method != next.method || req.URL.Path != next.path {
		rt.t.Fatalf("unexpected request, got %s %s want %s %s", req.Method, req.URL.Path, next.method, next.path)
	}

	return &http.Response{
		StatusCode: next.status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(next.body)),
		Request:    req,
	}, nil
}

func (rt *queueRoundTripper) assertDrained() {
	if len(rt.responses) != 0 {
		rt.t.Fatalf("unconsumed responses: %d", len(rt.responses))
	}
}

// ctxForTest returns a reusable background context.
func ctxForTest() context.Context {
	return context.Background()
}
