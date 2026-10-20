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
// model. An optional attribute that is not computed keeps the null the configuration left when the
// API returns its zero value (see optionalString and the other optional helpers); computed
// attributes and required attributes always take the API value.
func (m *VAPIAssistantResourceModel) flatten(in *vapi.Assistant) {
	m.flattenBasic(in)
	m.Voice = flattenVoice(m.Voice, in.Voice)
	m.Model = flattenModel(m.Model, in.Model)
	m.Transcriber = flattenTranscriber(m.Transcriber, in.Transcriber)
	m.MessagePlan = flattenMessagePlan(m.MessagePlan, in.MessagePlan)
	m.AnalysisPlan = flattenAnalysisPlan(m.AnalysisPlan, in.AnalysisPlan)
	m.StartSpeakingPlan = flattenStartSpeakingPlan(m.StartSpeakingPlan, in.StartSpeakingPlan)
	m.StopSpeakingPlan = flattenStopSpeakingPlan(m.StopSpeakingPlan, in.StopSpeakingPlan)
}

// flattenBasic binds the scalar and list attributes of the assistant.
func (m *VAPIAssistantResourceModel) flattenBasic(in *vapi.Assistant) {
	m.ID = types.StringValue(in.ID)
	m.OrgID = types.StringValue(in.OrgID)
	m.Name = types.StringValue(in.Name)
	m.FirstMessageMode = optionalString(m.FirstMessageMode, in.FirstMessageMode)
	m.HipaaEnabled = optionalBool(m.HipaaEnabled, in.HipaaEnabled)
	m.BackgroundSound = optionalString(m.BackgroundSound, in.BackgroundSound)
	m.BackgroundDenoising = optionalBool(m.BackgroundDenoising, in.BackgroundDenoising)
	m.ModelOutputEnabled = optionalBool(m.ModelOutputEnabled, in.ModelOutputEnabled)
	m.ForwardingPhoneNumber = optionalString(m.ForwardingPhoneNumber, in.ForwardingPhoneNumber)
	m.EndCallFunctionEnabled = optionalBool(m.EndCallFunctionEnabled, in.EndCallFunctionEnabled)
	m.RecordingEnabled = optionalBool(m.RecordingEnabled, in.RecordingEnabled)
	m.FirstMessage = optionalString(m.FirstMessage, in.FirstMessage)
	m.VoicemailMessage = optionalString(m.VoicemailMessage, in.VoicemailMessage)
	m.EndCallPhrases = optionalStringList(m.EndCallPhrases, in.EndCallPhrases)
	m.MaxDurationSeconds = optionalInt64(m.MaxDurationSeconds, in.MaxDurationSeconds)
	if in.SilenceTimeoutSeconds == nil {
		m.SilenceTimeoutSeconds = types.Float64Null()
	} else {
		m.SilenceTimeoutSeconds = optionalFloat64(m.SilenceTimeoutSeconds, *in.SilenceTimeoutSeconds)
	}
	// computed attributes: the API value is always the state value
	m.Language = types.StringValue(in.Language)
	m.InterruptionsEnabled = types.BoolValue(in.InterruptionsEnabled)
	m.DialKeypadFunctionEnabled = types.BoolValue(in.DialKeypadFunctionEnabled)
	m.FillersEnabled = types.BoolValue(in.FillersEnabled)
	m.ResponseDelaySeconds = types.Float64PointerValue(in.ResponseDelaySeconds)
	m.NumWordsToInterruptAssistant = types.Int64PointerValue(in.NumWordsToInterruptAssistant)
	m.LiveTranscriptsEnabled = types.BoolPointerValue(in.LiveTranscriptsEnabled)
	m.Keywords = ListValueFromStrings(in.Keywords)
	m.ClientMessages = ListValueFromStrings(in.ClientMessages)
	m.ServerMessages = ListValueFromStrings(in.ServerMessages)
	m.ParentID = types.StringPointerValue(in.ParentID)
	// phone_number_id is never part of the API response; an unknown planned value would stay unknown after apply
	if m.PhoneNumberID.IsUnknown() {
		m.PhoneNumberID = types.StringNull()
	}
}

// flattenVoice binds the voice block; the stability values are optional.
func flattenVoice(prior *VoiceResourceModel, in *vapi.Voice) *VoiceResourceModel {
	if in == nil || (prior == nil && *in == (vapi.Voice{})) {
		return nil
	}
	if prior == nil {
		prior = &VoiceResourceModel{}
	}
	return &VoiceResourceModel{
		Model:           types.StringValue(in.Model),
		Provider:        types.StringValue(in.Provider),
		VoiceID:         types.StringValue(in.VoiceID),
		Stability:       optionalFloat64(prior.Stability, in.Stability),
		SimilarityBoost: optionalFloat64(prior.SimilarityBoost, in.SimilarityBoost),
	}
}

// flattenModel binds the model block. provider, max_tokens and tool_ids are computed.
func flattenModel(prior *ModelResourceModel, in *vapi.Model) *ModelResourceModel {
	if in == nil || (prior == nil && emptyModel(in)) {
		return nil
	}
	if prior == nil {
		prior = &ModelResourceModel{}
	}
	return &ModelResourceModel{
		Model:         types.StringValue(in.Model),
		SystemPrompt:  optionalString(prior.SystemPrompt, in.SystemPrompt),
		Provider:      types.StringValue(in.Provider),
		MaxTokens:     types.Int64Value(in.MaxTokens),
		Temperature:   optionalFloat64(prior.Temperature, in.Temperature),
		ToolIDs:       ListValueFromStrings(in.ToolIDs),
		KnowledgeBase: flattenKnowledgeBase(prior.KnowledgeBase, in.KnowledgeBase),
	}
}

// flattenKnowledgeBase binds the knowledge base block; an empty block the configuration did not ask for stays absent.
func flattenKnowledgeBase(prior *KnowledgeBaseResourceModel, in *vapi.KnowledgeBase) *KnowledgeBaseResourceModel {
	if in == nil || (prior == nil && in.TopK == 0 && in.Provider == "" && len(in.FileIDs) == 0) {
		return nil
	}
	if prior == nil {
		prior = &KnowledgeBaseResourceModel{}
	}
	return &KnowledgeBaseResourceModel{
		TopK:     types.Int64Value(in.TopK),
		FileIDs:  optionalStringList(prior.FileIDs, in.FileIDs),
		Provider: types.StringValue(in.Provider),
	}
}

// flattenTranscriber binds the transcriber block; model and language are optional.
func flattenTranscriber(prior *TranscriberResourceModel, in *vapi.Transcriber) *TranscriberResourceModel {
	if in == nil || (prior == nil && *in == (vapi.Transcriber{})) {
		return nil
	}
	if prior == nil {
		prior = &TranscriberResourceModel{}
	}
	return &TranscriberResourceModel{
		Provider: types.StringValue(in.Provider),
		Model:    optionalString(prior.Model, in.Model),
		Language: optionalString(prior.Language, in.Language),
	}
}

// flattenMessagePlan binds the message plan block; an empty block the configuration did not ask for stays absent.
func flattenMessagePlan(prior *MessagePlanResourceModel, in *vapi.MessagePlan) *MessagePlanResourceModel {
	if in == nil || (prior == nil && len(in.IdleMessages) == 0) {
		return nil
	}
	if prior == nil {
		prior = &MessagePlanResourceModel{}
	}
	return &MessagePlanResourceModel{IdleMessages: optionalStringList(prior.IdleMessages, in.IdleMessages)}
}

// flattenAnalysisPlan binds the analysis plan block; an empty block the configuration did not ask for stays absent.
func flattenAnalysisPlan(prior *AnalysisPlanResourceModel, in *vapi.AnalysisPlan) *AnalysisPlanResourceModel {
	if in == nil || (prior == nil && emptyAnalysisPlan(in)) {
		return nil
	}
	if prior == nil {
		prior = &AnalysisPlanResourceModel{}
	}
	return &AnalysisPlanResourceModel{
		SummaryPrompt:           optionalString(prior.SummaryPrompt, in.SummaryPrompt),
		StructuredDataPrompt:    optionalString(prior.StructuredDataPrompt, in.StructuredDataPrompt),
		StructuredDataSchema:    flattenStructuredDataSchema(prior.StructuredDataSchema, in.StructuredDataSchema),
		SuccessEvaluationPrompt: optionalString(prior.SuccessEvaluationPrompt, in.SuccessEvaluationPrompt),
		SuccessEvaluationRubric: optionalString(prior.SuccessEvaluationRubric, in.SuccessEvaluationRubric),
	}
}

// flattenStartSpeakingPlan binds the start speaking plan block; every attribute of it is optional.
func flattenStartSpeakingPlan(prior *StartSpeakingPlanResourceModel, in *vapi.StartSpeakingPlan) *StartSpeakingPlanResourceModel {
	if in == nil || (prior == nil && emptyStartSpeakingPlan(in)) {
		return nil
	}
	if prior == nil {
		prior = &StartSpeakingPlanResourceModel{}
	}
	out := &StartSpeakingPlanResourceModel{
		WaitSeconds:             optionalFloat64(prior.WaitSeconds, in.WaitSeconds),
		SmartEndpointingEnabled: optionalBool(prior.SmartEndpointingEnabled, in.SmartEndpointingEnabled),
	}
	if plan := in.TranscriptionEndpointingPlan; plan != nil && (prior.TranscriptionEndpointingPlan != nil || *plan != (vapi.TranscriptionEndpointingPlan{})) {
		priorPlan := prior.TranscriptionEndpointingPlan
		if priorPlan == nil {
			priorPlan = &TranscriptionEndpointingPlanResourceModel{}
		}
		out.TranscriptionEndpointingPlan = &TranscriptionEndpointingPlanResourceModel{
			OnPunctuationSeconds:   optionalFloat64(priorPlan.OnPunctuationSeconds, plan.OnPunctuationSeconds),
			OnNoPunctuationSeconds: optionalFloat64(priorPlan.OnNoPunctuationSeconds, plan.OnNoPunctuationSeconds),
			OnNumberSeconds:        optionalFloat64(priorPlan.OnNumberSeconds, plan.OnNumberSeconds),
		}
	}
	return out
}

// flattenStopSpeakingPlan binds the stop speaking plan block; voice_seconds and backoff_seconds are computed.
func flattenStopSpeakingPlan(prior *StopSpeakingPlanResourceModel, in *vapi.StopSpeakingPlan) *StopSpeakingPlanResourceModel {
	if in == nil || (prior == nil && *in == (vapi.StopSpeakingPlan{})) {
		return nil
	}
	if prior == nil {
		prior = &StopSpeakingPlanResourceModel{}
	}
	return &StopSpeakingPlanResourceModel{
		NumWords:       optionalFloat64(prior.NumWords, in.NumWords),
		VoiceSeconds:   types.Float64Value(in.VoiceSeconds),
		BackoffSeconds: types.Float64Value(in.BackoffSeconds),
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

// emptyModel reports whether the API returned a model without any content, including an empty knowledge base.
func emptyModel(in *vapi.Model) bool {
	return in.Model == "" && in.SystemPrompt == "" && in.Provider == "" && in.MaxTokens == 0 && in.Temperature == 0 &&
		len(in.ToolIDs) == 0 && len(in.Messages) == 0 &&
		(in.KnowledgeBase == nil || (in.KnowledgeBase.TopK == 0 && in.KnowledgeBase.Provider == "" && len(in.KnowledgeBase.FileIDs) == 0))
}

// emptyAnalysisPlan reports whether the API returned an analysis plan without any content.
func emptyAnalysisPlan(in *vapi.AnalysisPlan) bool {
	return in.SummaryPrompt == "" && in.StructuredDataPrompt == "" && in.SuccessEvaluationPrompt == "" && in.SuccessEvaluationRubric == "" &&
		(in.StructuredDataSchema == nil || (in.StructuredDataSchema.Type == "" && len(in.StructuredDataSchema.Properties) == 0))
}

// emptyStartSpeakingPlan reports whether the API returned a start speaking plan without any content.
func emptyStartSpeakingPlan(in *vapi.StartSpeakingPlan) bool {
	return in.WaitSeconds == 0 && !in.SmartEndpointingEnabled &&
		(in.TranscriptionEndpointingPlan == nil || *in.TranscriptionEndpointingPlan == (vapi.TranscriptionEndpointingPlan{}))
}

// flattenStructuredDataSchema binds the structured data schema; an empty schema the configuration did not ask for stays absent.
func flattenStructuredDataSchema(prior *StructuredDataSchemaResourceModel, in *vapi.StructuredDataSchema) *StructuredDataSchemaResourceModel {
	if in == nil || (prior == nil && in.Type == "" && len(in.Properties) == 0) {
		return nil
	}
	if prior == nil {
		prior = &StructuredDataSchemaResourceModel{}
	}
	var properties map[string]PropertyResourceModel
	if len(in.Properties) > 0 || prior.Properties != nil {
		properties = make(map[string]PropertyResourceModel, len(in.Properties))
	}
	for key, prop := range in.Properties {
		priorProp := prior.Properties[key]
		properties[key] = PropertyResourceModel{
			Type:        optionalString(priorProp.Type, prop.Type),
			Description: optionalString(priorProp.Description, prop.Description),
		}
	}
	return &StructuredDataSchemaResourceModel{
		Type:       optionalString(prior.Type, in.Type),
		Properties: properties,
	}
}
