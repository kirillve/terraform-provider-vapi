package vapiv1

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

var _ resource.Resource = &VAPIAssistantResource{}

// NewVAPIAssistantResource returns a new assistant resource.
func NewVAPIAssistantResource() resource.Resource {
	return &VAPIAssistantResource{}
}

// VAPIAssistantResource manages a voice assistant in VAPI.
type VAPIAssistantResource struct {
	client *vapi.APIClient
}

// VAPIAssistantResourceModel maps the schema data.
type VAPIAssistantResourceModel struct {
	ID                           types.String                    `tfsdk:"id"`
	OrgID                        types.String                    `tfsdk:"org_id"`
	Name                         types.String                    `tfsdk:"name"`
	FirstMessageMode             types.String                    `tfsdk:"first_message_mode"`
	HipaaEnabled                 types.Bool                      `tfsdk:"hipaa_enabled"`
	ClientMessages               types.List                      `tfsdk:"client_messages"`
	ServerMessages               types.List                      `tfsdk:"server_messages"`
	SilenceTimeoutSeconds        types.Float64                   `tfsdk:"silence_timeout_seconds"`
	MaxDurationSeconds           types.Int64                     `tfsdk:"max_duration_seconds"`
	BackgroundSound              types.String                    `tfsdk:"background_sound"`
	BackgroundDenoising          types.Bool                      `tfsdk:"background_denoising"`
	ModelOutputEnabled           types.Bool                      `tfsdk:"model_output_enabled"`
	FirstMessage                 types.String                    `tfsdk:"first_message"`
	VoicemailMessage             types.String                    `tfsdk:"voicemail_message"`
	EndCallMessage               types.String                    `tfsdk:"end_call_message"`
	ServerURL                    types.String                    `tfsdk:"server_url"`
	ServerURLSecret              types.String                    `tfsdk:"server_url_secret"`
	EndCallPhrases               types.List                      `tfsdk:"end_call_phrases"`
	Transcriber                  *TranscriberResourceModel       `tfsdk:"transcriber"`
	Model                        *ModelResourceModel             `tfsdk:"model"`
	Voice                        *VoiceResourceModel             `tfsdk:"voice"`
	StartSpeakingPlan            *StartSpeakingPlanResourceModel `tfsdk:"start_speaking_plan"`
	StopSpeakingPlan             *StopSpeakingPlanResourceModel  `tfsdk:"stop_speaking_plan"`
	AnalysisPlan                 *AnalysisPlanResourceModel      `tfsdk:"analysis_plan"`
	MessagePlan                  *MessagePlanResourceModel       `tfsdk:"message_plan"`
	ArtifactPlan                 *ArtifactPlanResourceModel      `tfsdk:"artifact_plan"`
	EndCallFunctionEnabled       types.Bool                      `tfsdk:"end_call_function_enabled"`
	RecordingEnabled             types.Bool                      `tfsdk:"recording_enabled"`
	ForwardingPhoneNumber        types.String                    `tfsdk:"forwarding_phone_number"`
	PhoneNumberID                types.String                    `tfsdk:"phone_number_id"`
	Language                     types.String                    `tfsdk:"language"`
	InterruptionsEnabled         types.Bool                      `tfsdk:"interruptions_enabled"`
	DialKeypadFunctionEnabled    types.Bool                      `tfsdk:"dial_keypad_function_enabled"`
	FillersEnabled               types.Bool                      `tfsdk:"fillers_enabled"`
	ResponseDelaySeconds         types.Float64                   `tfsdk:"response_delay_seconds"`
	NumWordsToInterruptAssistant types.Int64                     `tfsdk:"num_words_to_interrupt_assistant"`
	LiveTranscriptsEnabled       types.Bool                      `tfsdk:"live_transcripts_enabled"`
	Keywords                     types.List                      `tfsdk:"keywords"`
	ParentID                     types.String                    `tfsdk:"parent_id"`
}

// TranscriberResourceModel is the transcriber block.
type TranscriberResourceModel struct {
	Provider types.String `tfsdk:"provider"`
	Model    types.String `tfsdk:"model"`
	Language types.String `tfsdk:"language"`
}

// ArtifactPlanResourceModel is the artifact_plan block.
type ArtifactPlanResourceModel struct {
	RecordingFormat types.String `tfsdk:"recording_format"`
}

// ModelResourceModel is the model block.
type ModelResourceModel struct {
	Model         types.String                `tfsdk:"model"`
	SystemPrompt  types.String                `tfsdk:"system_prompt"`
	Provider      types.String                `tfsdk:"provider"`
	Temperature   types.Float64               `tfsdk:"temperature"`
	MaxTokens     types.Int64                 `tfsdk:"max_tokens"`
	ToolIDs       types.List                  `tfsdk:"tool_ids"`
	KnowledgeBase *KnowledgeBaseResourceModel `tfsdk:"knowledge_base"`
}

// KnowledgeBaseResourceModel is the knowledge_base block of the model.
type KnowledgeBaseResourceModel struct {
	TopK     types.Int64  `tfsdk:"top_k"`
	FileIDs  types.List   `tfsdk:"file_ids"`
	Provider types.String `tfsdk:"provider"`
}

// VoiceResourceModel is the voice block.
type VoiceResourceModel struct {
	Model           types.String  `tfsdk:"model"`
	Provider        types.String  `tfsdk:"provider"`
	VoiceID         types.String  `tfsdk:"voice_id"`
	Stability       types.Float64 `tfsdk:"stability"`
	SimilarityBoost types.Float64 `tfsdk:"similarity_boost"`
}

// StartSpeakingPlanResourceModel is the start_speaking_plan block.
type StartSpeakingPlanResourceModel struct {
	WaitSeconds                  types.Float64                              `tfsdk:"wait_seconds"`
	SmartEndpointingEnabled      types.Bool                                 `tfsdk:"smart_endpointing_enabled"`
	TranscriptionEndpointingPlan *TranscriptionEndpointingPlanResourceModel `tfsdk:"transcription_endpointing_plan"`
}

// TranscriptionEndpointingPlanResourceModel is the transcription_endpointing_plan block.
type TranscriptionEndpointingPlanResourceModel struct {
	OnPunctuationSeconds   types.Float64 `tfsdk:"on_punctuation_seconds"`
	OnNoPunctuationSeconds types.Float64 `tfsdk:"on_no_punctuation_seconds"`
	OnNumberSeconds        types.Float64 `tfsdk:"on_number_seconds"`
}

// StopSpeakingPlanResourceModel is the stop_speaking_plan block.
type StopSpeakingPlanResourceModel struct {
	NumWords       types.Float64 `tfsdk:"num_words"`
	VoiceSeconds   types.Float64 `tfsdk:"voice_seconds"`
	BackoffSeconds types.Float64 `tfsdk:"backoff_seconds"`
}

// AnalysisPlanResourceModel is the analysis_plan block.
type AnalysisPlanResourceModel struct {
	SummaryPrompt           types.String                       `tfsdk:"summary_prompt"`
	StructuredDataPrompt    types.String                       `tfsdk:"structured_data_prompt"`
	StructuredDataSchema    *StructuredDataSchemaResourceModel `tfsdk:"structured_data_schema"`
	SuccessEvaluationPrompt types.String                       `tfsdk:"success_evaluation_prompt"`
	SuccessEvaluationRubric types.String                       `tfsdk:"success_evaluation_rubric"`
}

// StructuredDataSchemaResourceModel is the structured_data_schema block of the analysis plan.
type StructuredDataSchemaResourceModel struct {
	Type       types.String                     `tfsdk:"type"`
	Properties map[string]PropertyResourceModel `tfsdk:"properties"`
}

// PropertyResourceModel is one property of the structured data schema.
type PropertyResourceModel struct {
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
}

// MessagePlanResourceModel is the message_plan block.
type MessagePlanResourceModel struct {
	IdleMessages types.List `tfsdk:"idle_messages"`
}

// Metadata sets the resource type name for Terraform.
func (r *VAPIAssistantResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_assistant"
}

// Schema defines the resource schema for the assistant resource.
func (r *VAPIAssistantResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an assistant resource in the VAPI system.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the assistant resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				MarkdownDescription: "Org identifier for the assistant resource.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the assistant resource.",
				Required:            true,
			},
			"first_message_mode": schema.StringAttribute{
				MarkdownDescription: "Mode of the first message.",
				Optional:            true,
			},
			"hipaa_enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether HIPAA compliance is enabled.",
				Optional:            true,
			},
			"client_messages": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of messages from the client.",
				Optional:            true,
				Computed:            true,
			},
			"server_messages": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of messages from the server.",
				Optional:            true,
				Computed:            true,
			},
			"silence_timeout_seconds": schema.Float64Attribute{
				MarkdownDescription: "Timeout in seconds for silence.",
				Optional:            true,
			},
			"max_duration_seconds": schema.Int64Attribute{
				MarkdownDescription: "Maximum duration of the call in seconds.",
				Optional:            true,
			},
			"background_sound": schema.StringAttribute{
				MarkdownDescription: "Background sound used during the call.",
				Optional:            true,
			},
			"background_denoising": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether background denoising is enabled.",
				Optional:            true,
			},
			"model_output_enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether model output is enabled.",
				Optional:            true,
			},
			"first_message": schema.StringAttribute{
				MarkdownDescription: "Initial message sent to the client.",
				Optional:            true,
			},
			"voicemail_message": schema.StringAttribute{
				MarkdownDescription: "Message to be used for voicemail.",
				Optional:            true,
			},
			"end_call_message": schema.StringAttribute{
				MarkdownDescription: "Message to be used when ending the call.",
				Optional:            true,
			},
			"server_url": schema.StringAttribute{
				MarkdownDescription: "Server URL.",
				Optional:            true,
			},
			"server_url_secret": schema.StringAttribute{
				MarkdownDescription: "Server URL Secret.",
				Optional:            true,
			},
			"end_call_phrases": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of phrases to end the call.",
				Optional:            true,
			},
			"transcriber": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for the transcriber model.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"provider": schema.StringAttribute{
						MarkdownDescription: "Provider for the transcriber service.",
						Required:            true,
					},
					"model": schema.StringAttribute{
						MarkdownDescription: "Model used for transcription.",
						Optional:            true,
					},
					"language": schema.StringAttribute{
						MarkdownDescription: "Language used for transcription.",
						Optional:            true,
					},
				},
			},
			"model": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for the assistant model.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"model": schema.StringAttribute{
						MarkdownDescription: "The assistant model type.",
						Required:            true,
					},
					"system_prompt": schema.StringAttribute{
						MarkdownDescription: "Prompt text used to guide the assistant model.",
						Optional:            true,
					},
					"provider": schema.StringAttribute{
						MarkdownDescription: "Provider for the assistant model: openai, azure-openai, together-ai, anyscale, openrouter, perplexity-ai, deepinfra, custom-llm, runpod, groq, vapi, anthropic, google",
						Optional:            true,
						Computed:            true,
					},
					"max_tokens": schema.Int64Attribute{
						MarkdownDescription: "The maximum number of tokens allowed for the model's response.",
						Optional:            true,
						Computed:            true,
					},
					"temperature": schema.Float64Attribute{
						MarkdownDescription: "Temperature setting for the model's response randomness.",
						Optional:            true,
					},
					"tool_ids": schema.ListAttribute{
						ElementType:         types.StringType,
						MarkdownDescription: "List of tool IDs used by the model.",
						Optional:            true,
						Computed:            true,
					},
					"knowledge_base": schema.SingleNestedAttribute{
						MarkdownDescription: "Knowledge base configuration for the assistant model.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"top_k": schema.Int64Attribute{
								MarkdownDescription: "The maximum number of documents to retrieve from the knowledge base.",
								Optional:            true,
								Computed:            true,
							},
							"file_ids": schema.ListAttribute{
								ElementType:         types.StringType,
								MarkdownDescription: "List of file IDs in the knowledge base.",
								Optional:            true,
							},
							"provider": schema.StringAttribute{
								MarkdownDescription: "Provider for the knowledge base.",
								Optional:            true,
								Computed:            true,
							},
						},
					},
				},
			},
			"voice": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for the voice model.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"model": schema.StringAttribute{
						MarkdownDescription: "Model for the voice model.",
						Required:            true,
					},
					"voice_id": schema.StringAttribute{
						MarkdownDescription: "ID of the voice model.",
						Required:            true,
					},
					"provider": schema.StringAttribute{
						MarkdownDescription: "Provider for the voice model.",
						Required:            true,
					},
					"stability": schema.Float64Attribute{
						MarkdownDescription: "Stability of the voice output.",
						Optional:            true,
					},
					"similarity_boost": schema.Float64Attribute{
						MarkdownDescription: "Boost factor for similarity in voice.",
						Optional:            true,
					},
				},
			},
			"start_speaking_plan": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for starting the speaking plan.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"wait_seconds": schema.Float64Attribute{
						MarkdownDescription: "Seconds to wait before speaking starts.",
						Optional:            true,
					},
					"smart_endpointing_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable smart endpointing for better control.",
						Optional:            true,
					},
					"transcription_endpointing_plan": schema.SingleNestedAttribute{
						MarkdownDescription: "Endpointing plan for transcription.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"on_punctuation_seconds": schema.Float64Attribute{
								MarkdownDescription: "Delay in seconds for punctuation-based endpointing.",
								Optional:            true,
							},
							"on_no_punctuation_seconds": schema.Float64Attribute{
								MarkdownDescription: "Delay in seconds for no punctuation.",
								Optional:            true,
							},
							"on_number_seconds": schema.Float64Attribute{
								MarkdownDescription: "Delay in seconds for number-based endpointing.",
								Optional:            true,
							},
						},
					},
				},
			},
			"stop_speaking_plan": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for stopping the speaking plan.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"num_words": schema.Float64Attribute{
						MarkdownDescription: "Number of words required to stop speaking.",
						Optional:            true,
					},
					"voice_seconds": schema.Float64Attribute{
						MarkdownDescription: "Duration in seconds to stop speaking.",
						Optional:            true,
						Computed:            true,
					},
					"backoff_seconds": schema.Float64Attribute{
						MarkdownDescription: "Backoff period in seconds before stopping.",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"analysis_plan": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for the analysis plan.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"summary_prompt": schema.StringAttribute{
						MarkdownDescription: "Prompt for generating a summary.",
						Optional:            true,
					},
					"structured_data_prompt": schema.StringAttribute{
						MarkdownDescription: "Prompt for structured data generation.",
						Optional:            true,
					},
					"structured_data_schema": schema.SingleNestedAttribute{
						MarkdownDescription: "Schema for structured data.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								MarkdownDescription: "Type of structured data.",
								Optional:            true,
							},
							"properties": schema.MapNestedAttribute{
								MarkdownDescription: "Properties of structured data fields.",
								Optional:            true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											MarkdownDescription: "Type of the property.",
											Optional:            true,
										},
										"description": schema.StringAttribute{
											MarkdownDescription: "Description of the property.",
											Optional:            true,
										},
									},
								},
							},
						},
					},
					"success_evaluation_prompt": schema.StringAttribute{
						MarkdownDescription: "Prompt for success evaluation.",
						Optional:            true,
					},
					"success_evaluation_rubric": schema.StringAttribute{
						MarkdownDescription: "Rubric for evaluating success.",
						Optional:            true,
					},
				},
			},

			"message_plan": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for message plan.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"idle_messages": schema.ListAttribute{
						ElementType:         types.StringType,
						MarkdownDescription: "List of idle messages.",
						Optional:            true,
					},
				},
			},

			"artifact_plan": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for artifact plan.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"recording_format": schema.StringAttribute{
						MarkdownDescription: "Recording format wav or mp3.",
						Optional:            true,
					},
				},
			},

			"end_call_function_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enables the end call function.",
				Optional:            true,
			},

			"recording_enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if call recording is enabled.",
				Optional:            true,
			},

			"forwarding_phone_number": schema.StringAttribute{
				MarkdownDescription: "Phone number to which calls are forwarded.",
				Optional:            true,
			},

			"phone_number_id": schema.StringAttribute{
				MarkdownDescription: "ID of the phone number associated with the assistant.",
				Optional:            true,
				Computed:            true,
			},

			"response_delay_seconds": schema.Float64Attribute{
				MarkdownDescription: "Response Delay Seconds",
				Optional:            true,
				Computed:            true,
			},
			"dial_keypad_function_enabled": schema.BoolAttribute{
				MarkdownDescription: "Dial Keypad Function enabled",
				Optional:            true,
				Computed:            true,
			},
			"num_words_to_interrupt_assistant": schema.Int64Attribute{
				MarkdownDescription: "Num words to interrupt assistant",
				Optional:            true,
				Computed:            true,
			},
			"interruptions_enabled": schema.BoolAttribute{
				MarkdownDescription: "Interruptions enabled",
				Optional:            true,
				Computed:            true,
			},
			"keywords": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Keywords.",
				Optional:            true,
				Computed:            true,
			},
			"fillers_enabled": schema.BoolAttribute{
				MarkdownDescription: "Fillers enabled",
				Optional:            true,
				Computed:            true,
			},
			"language": schema.StringAttribute{
				MarkdownDescription: "Language",
				Optional:            true,
				Computed:            true,
			},
			"live_transcripts_enabled": schema.BoolAttribute{
				MarkdownDescription: "Parent ID.",
				Optional:            true,
				Computed:            true,
			},
			"parent_id": schema.StringAttribute{
				MarkdownDescription: "Parent ID.",
				Computed:            true,
			},
		},
	}
}

// Configure assigns the configured API client to the resource.
func (r *VAPIAssistantResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

// Create implements the create operation for the assistant resource.
func (r *VAPIAssistantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPIAssistantResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var assistantResponse vapi.Assistant
	if err := r.client.Do(http.MethodPost, "assistant", data.expand(), &assistantResponse); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create assistant: %s", err))
		return
	}

	data.flatten(&assistantResponse)
	tflog.Trace(ctx, "created an assistant resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read implements the read operation for the assistant resource. An assistant that no longer
// exists (HTTP 404) is removed from the state.
func (r *VAPIAssistantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPIAssistantResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Client Error", "Unable to read assistant: ID cannot be empty")
		return
	}

	var assistantResponse vapi.Assistant
	err := r.client.Do(http.MethodGet, "assistant/"+id, nil, &assistantResponse)
	if vapi.StatusCode(err) == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read assistant: %s", err))
		return
	}

	data.flatten(&assistantResponse)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update implements the update operation for the assistant resource.
func (r *VAPIAssistantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, data VAPIAssistantResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Failed to read state", fmt.Sprintf("Errors: %v", resp.Diagnostics.Errors()))
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Failed to read plan", fmt.Sprintf("Errors: %v", resp.Diagnostics.Errors()))
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Client Error", "Unable to update assistant: ID cannot be empty")
		return
	}

	var assistantResponse vapi.Assistant
	if err := r.client.Do(http.MethodPatch, "assistant/"+id, data.expand(), &assistantResponse); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update assistant: %s", err))
		return
	}

	data.flatten(&assistantResponse)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements the delete operation for the assistant resource. An assistant that no longer
// exists (HTTP 404) counts as deleted.
func (r *VAPIAssistantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPIAssistantResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Client Error", "Unable to delete assistant: ID cannot be empty")
		return
	}

	if err := r.client.Do(http.MethodDelete, "assistant/"+id, nil, nil); err != nil && vapi.StatusCode(err) != http.StatusNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete assistant: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted an assistant resource")
}

// flatten binds the API response to the model. An absent optional block clears the block in the
// model.
func (m *VAPIAssistantResourceModel) flatten(in *vapi.Assistant) {
	// Basic fields
	m.ID = types.StringValue(in.ID)
	m.OrgID = types.StringValue(in.OrgID)
	m.Name = types.StringValue(in.Name)
	m.FirstMessageMode = types.StringValue(in.FirstMessageMode)
	m.HipaaEnabled = types.BoolValue(in.HipaaEnabled)
	m.BackgroundSound = types.StringValue(in.BackgroundSound)
	m.BackgroundDenoising = types.BoolValue(in.BackgroundDenoising)
	m.ModelOutputEnabled = types.BoolValue(in.ModelOutputEnabled)
	m.Language = types.StringValue(in.Language)
	m.ForwardingPhoneNumber = types.StringValue(in.ForwardingPhoneNumber)
	m.InterruptionsEnabled = types.BoolValue(in.InterruptionsEnabled)
	m.EndCallFunctionEnabled = types.BoolValue(in.EndCallFunctionEnabled)
	m.DialKeypadFunctionEnabled = types.BoolValue(in.DialKeypadFunctionEnabled)
	m.FillersEnabled = types.BoolValue(in.FillersEnabled)
	m.SilenceTimeoutSeconds = types.Float64PointerValue(in.SilenceTimeoutSeconds)
	m.ResponseDelaySeconds = types.Float64PointerValue(in.ResponseDelaySeconds)
	m.NumWordsToInterruptAssistant = types.Int64PointerValue(in.NumWordsToInterruptAssistant)
	m.LiveTranscriptsEnabled = types.BoolPointerValue(in.LiveTranscriptsEnabled)
	m.Keywords = ListValueFromStrings(in.Keywords)
	m.ParentID = types.StringPointerValue(in.ParentID)

	// Handle optional Voice struct
	if in.Voice != nil {
		m.Voice = &VoiceResourceModel{
			Model:           types.StringValue(in.Voice.Model),
			Provider:        types.StringValue(in.Voice.Provider),
			VoiceID:         types.StringValue(in.Voice.VoiceID),
			Stability:       types.Float64Value(in.Voice.Stability),
			SimilarityBoost: types.Float64Value(in.Voice.SimilarityBoost),
		}
	} else {
		m.Voice = nil
	}

	// Handle optional Model struct
	if in.Model != nil {
		m.Model = &ModelResourceModel{
			Model:        types.StringValue(in.Model.Model),
			SystemPrompt: types.StringValue(in.Model.SystemPrompt),
			Provider:     types.StringValue(in.Model.Provider),
			MaxTokens:    types.Int64Value(in.Model.MaxTokens),
			Temperature:  types.Float64Value(in.Model.Temperature),
			ToolIDs:      ListValueFromStrings(in.Model.ToolIDs),
		}
		if in.Model.KnowledgeBase != nil {
			m.Model.KnowledgeBase = &KnowledgeBaseResourceModel{
				TopK:     types.Int64Value(in.Model.KnowledgeBase.TopK),
				FileIDs:  ListValueFromStrings(in.Model.KnowledgeBase.FileIDs),
				Provider: types.StringValue(in.Model.KnowledgeBase.Provider),
			}
		}
	} else {
		m.Model = nil
	}

	m.RecordingEnabled = types.BoolValue(in.RecordingEnabled)
	m.FirstMessage = types.StringValue(in.FirstMessage)
	m.VoicemailMessage = types.StringValue(in.VoicemailMessage)
	m.EndCallFunctionEnabled = types.BoolValue(in.EndCallFunctionEnabled)

	// Handle optional Transcriber struct
	if in.Transcriber != nil {
		m.Transcriber = &TranscriberResourceModel{
			Provider: types.StringValue(in.Transcriber.Provider),
			Model:    types.StringValue(in.Transcriber.Model),
			Language: types.StringValue(in.Transcriber.Language),
		}
	} else {
		m.Transcriber = nil
	}

	m.ClientMessages = ListValueFromStrings(in.ClientMessages)
	m.ServerMessages = ListValueFromStrings(in.ServerMessages)
	m.EndCallPhrases = ListValueFromStrings(in.EndCallPhrases)

	m.MaxDurationSeconds = types.Int64Value(in.MaxDurationSeconds)

	if in.MessagePlan != nil {
		m.MessagePlan = &MessagePlanResourceModel{
			IdleMessages: ListValueFromStrings(in.MessagePlan.IdleMessages),
		}
	} else {
		m.MessagePlan = nil
	}

	// Handle optional AnalysisPlan struct
	if in.AnalysisPlan != nil {
		m.AnalysisPlan = &AnalysisPlanResourceModel{
			SummaryPrompt:           types.StringValue(in.AnalysisPlan.SummaryPrompt),
			StructuredDataPrompt:    types.StringValue(in.AnalysisPlan.StructuredDataPrompt),
			StructuredDataSchema:    mapStructuredDataSchemaRequestToResourceModel(in.AnalysisPlan.StructuredDataSchema),
			SuccessEvaluationPrompt: types.StringValue(in.AnalysisPlan.SuccessEvaluationPrompt),
			SuccessEvaluationRubric: types.StringValue(in.AnalysisPlan.SuccessEvaluationRubric),
		}
	} else {
		m.AnalysisPlan = nil
	}

	// Handle optional StartSpeakingPlan struct
	if in.StartSpeakingPlan != nil {
		m.StartSpeakingPlan = &StartSpeakingPlanResourceModel{
			WaitSeconds:             types.Float64Value(in.StartSpeakingPlan.WaitSeconds),
			SmartEndpointingEnabled: types.BoolValue(in.StartSpeakingPlan.SmartEndpointingEnabled),
		}
		if in.StartSpeakingPlan.TranscriptionEndpointingPlan != nil {
			m.StartSpeakingPlan.TranscriptionEndpointingPlan = &TranscriptionEndpointingPlanResourceModel{
				OnPunctuationSeconds:   types.Float64Value(in.StartSpeakingPlan.TranscriptionEndpointingPlan.OnPunctuationSeconds),
				OnNoPunctuationSeconds: types.Float64Value(in.StartSpeakingPlan.TranscriptionEndpointingPlan.OnNoPunctuationSeconds),
				OnNumberSeconds:        types.Float64Value(in.StartSpeakingPlan.TranscriptionEndpointingPlan.OnNumberSeconds),
			}
		}
	} else {
		m.StartSpeakingPlan = nil
	}

	// Handle optional StopSpeakingPlan struct
	if in.StopSpeakingPlan != nil {
		m.StopSpeakingPlan = &StopSpeakingPlanResourceModel{
			NumWords:       types.Float64Value(in.StopSpeakingPlan.NumWords),
			VoiceSeconds:   types.Float64Value(in.StopSpeakingPlan.VoiceSeconds),
			BackoffSeconds: types.Float64Value(in.StopSpeakingPlan.BackoffSeconds),
		}
	} else {
		m.StopSpeakingPlan = nil
	}
}

// expand builds the API request from the planned model.
func (m *VAPIAssistantResourceModel) expand() vapi.CreateAssistantRequest {
	out := vapi.CreateAssistantRequest{
		Name:                         m.Name.ValueString(),
		FirstMessageMode:             m.FirstMessageMode.ValueString(),
		HipaaEnabled:                 m.HipaaEnabled.ValueBool(),
		BackgroundSound:              m.BackgroundSound.ValueString(),
		BackgroundDenoising:          m.BackgroundDenoising.ValueBool(),
		ModelOutputEnabled:           m.ModelOutputEnabled.ValueBool(),
		Language:                     m.Language.ValueString(),
		ForwardingPhoneNumber:        m.ForwardingPhoneNumber.ValueString(),
		InterruptionsEnabled:         m.InterruptionsEnabled.ValueBool(),
		EndCallFunctionEnabled:       m.EndCallFunctionEnabled.ValueBool(),
		DialKeypadFunctionEnabled:    m.DialKeypadFunctionEnabled.ValueBool(),
		FillersEnabled:               m.FillersEnabled.ValueBool(),
		SilenceTimeoutSeconds:        m.SilenceTimeoutSeconds.ValueFloat64Pointer(),
		ResponseDelaySeconds:         m.ResponseDelaySeconds.ValueFloat64Pointer(),
		NumWordsToInterruptAssistant: m.NumWordsToInterruptAssistant.ValueInt64Pointer(),
		LiveTranscriptsEnabled:       m.LiveTranscriptsEnabled.ValueBoolPointer(),
		Keywords:                     ElementsAsString(m.Keywords),
		ServerMessages:               ElementsAsString(m.ServerMessages),
		RecordingEnabled:             m.RecordingEnabled.ValueBool(),
		FirstMessage:                 m.FirstMessage.ValueString(),
		VoicemailMessage:             m.VoicemailMessage.ValueString(),
		EndCallMessage:               m.EndCallMessage.ValueString(),
		Server: &vapi.Server{
			URL:            m.ServerURL.ValueString(),
			Secret:         m.ServerURLSecret.ValueString(),
			TimeoutSeconds: 20,
		},
		ClientMessages:     ElementsAsString(m.ClientMessages),
		EndCallPhrases:     ElementsAsString(m.EndCallPhrases),
		MaxDurationSeconds: m.MaxDurationSeconds.ValueInt64(),
	}

	if m.MessagePlan != nil {
		out.MessagePlan = &vapi.MessagePlan{
			IdleMessages: ElementsAsString(m.MessagePlan.IdleMessages),
		}
	}
	if m.Voice != nil {
		out.Voice = &vapi.Voice{
			Model:           m.Voice.Model.ValueString(),
			VoiceID:         m.Voice.VoiceID.ValueString(),
			Provider:        m.Voice.Provider.ValueString(),
			Stability:       m.Voice.Stability.ValueFloat64(),
			SimilarityBoost: m.Voice.SimilarityBoost.ValueFloat64(),
		}
	}
	if m.Model != nil {
		out.Model = &vapi.Model{
			Model:        m.Model.Model.ValueString(),
			SystemPrompt: m.Model.SystemPrompt.ValueString(),
			Provider:     m.Model.Provider.ValueString(),
			MaxTokens:    m.Model.MaxTokens.ValueInt64(),
			Temperature:  m.Model.Temperature.ValueFloat64(),
			ToolIDs:      ElementsAsString(m.Model.ToolIDs),
		}
		if m.Model.KnowledgeBase != nil {
			out.Model.KnowledgeBase = &vapi.KnowledgeBase{
				TopK:     m.Model.KnowledgeBase.TopK.ValueInt64(),
				FileIDs:  ElementsAsString(m.Model.KnowledgeBase.FileIDs),
				Provider: m.Model.KnowledgeBase.Provider.ValueString(),
			}
		}
	}
	if m.Transcriber != nil {
		out.Transcriber = &vapi.Transcriber{
			Provider: m.Transcriber.Provider.ValueString(),
			Model:    m.Transcriber.Model.ValueString(),
			Language: m.Transcriber.Language.ValueString(),
		}
	}
	if m.AnalysisPlan != nil {
		out.AnalysisPlan = &vapi.AnalysisPlan{
			SummaryPrompt:           m.AnalysisPlan.SummaryPrompt.ValueString(),
			StructuredDataPrompt:    m.AnalysisPlan.StructuredDataPrompt.ValueString(),
			StructuredDataSchema:    createRequestObject(m.AnalysisPlan.StructuredDataSchema),
			SuccessEvaluationPrompt: m.AnalysisPlan.SuccessEvaluationPrompt.ValueString(),
			SuccessEvaluationRubric: m.AnalysisPlan.SuccessEvaluationRubric.ValueString(),
		}
	}
	if m.StartSpeakingPlan != nil {
		out.StartSpeakingPlan = &vapi.StartSpeakingPlan{
			WaitSeconds:             m.StartSpeakingPlan.WaitSeconds.ValueFloat64(),
			SmartEndpointingEnabled: m.StartSpeakingPlan.SmartEndpointingEnabled.ValueBool(),
		}
		if m.StartSpeakingPlan.TranscriptionEndpointingPlan != nil {
			out.StartSpeakingPlan.TranscriptionEndpointingPlan = &vapi.TranscriptionEndpointingPlan{
				OnPunctuationSeconds:   m.StartSpeakingPlan.TranscriptionEndpointingPlan.OnPunctuationSeconds.ValueFloat64(),
				OnNoPunctuationSeconds: m.StartSpeakingPlan.TranscriptionEndpointingPlan.OnNoPunctuationSeconds.ValueFloat64(),
				OnNumberSeconds:        m.StartSpeakingPlan.TranscriptionEndpointingPlan.OnNumberSeconds.ValueFloat64(),
			}
		}
	}
	if m.StopSpeakingPlan != nil {
		out.StopSpeakingPlan = &vapi.StopSpeakingPlan{
			NumWords:       m.StopSpeakingPlan.NumWords.ValueFloat64(),
			VoiceSeconds:   m.StopSpeakingPlan.VoiceSeconds.ValueFloat64(),
			BackoffSeconds: m.StopSpeakingPlan.BackoffSeconds.ValueFloat64(),
		}
	}
	if m.ArtifactPlan != nil {
		out.ArtifactPlan = &vapi.ArtifactPlan{RecordingFormat: m.ArtifactPlan.RecordingFormat.ValueString()}
		if out.ArtifactPlan.RecordingFormat == "" {
			out.ArtifactPlan.RecordingFormat = "mp3"
		}
	}
	return out
}

func createRequestObject(data *StructuredDataSchemaResourceModel) *vapi.StructuredDataSchema {
	if data == nil {
		return nil
	}

	properties := make(map[string]*vapi.Property)
	for key, prop := range data.Properties {
		properties[key] = &vapi.Property{
			Type:        prop.Type.ValueString(),
			Description: prop.Description.ValueString(),
		}
	}

	return &vapi.StructuredDataSchema{
		Type:       data.Type.ValueString(),
		Properties: properties,
	}
}

func mapStructuredDataSchemaRequestToResourceModel(data *vapi.StructuredDataSchema) *StructuredDataSchemaResourceModel {
	if data == nil {
		return nil
	}

	properties := make(map[string]PropertyResourceModel)
	for key, prop := range data.Properties {
		properties[key] = PropertyResourceModel{
			Type:        types.StringValue(prop.Type),
			Description: types.StringValue(prop.Description),
		}
	}

	return &StructuredDataSchemaResourceModel{
		Type:       types.StringValue(data.Type),
		Properties: properties,
	}
}
