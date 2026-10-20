package vapiv1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

var _ resource.Resource = &VAPIAssistantResource{}

func NewVAPIAssistantResource() resource.Resource {
	return &VAPIAssistantResource{}
}

type VAPIAssistantResource struct {
	client *vapi.APIClient
}

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

type TranscriberResourceModel struct {
	Provider types.String `tfsdk:"provider"`
	Model    types.String `tfsdk:"model"`
	Language types.String `tfsdk:"language"`
}

type ArtifactPlanResourceModel struct {
	RecordingFormat types.String `tfsdk:"recording_format"`
}

type ModelResourceModel struct {
	Model         types.String                `tfsdk:"model"`
	SystemPrompt  types.String                `tfsdk:"system_prompt"`
	Provider      types.String                `tfsdk:"provider"`
	Temperature   types.Float64               `tfsdk:"temperature"`
	MaxTokens     types.Int64                 `tfsdk:"max_tokens"`
	ToolIDs       types.List                  `tfsdk:"tool_ids"`
	KnowledgeBase *KnowledgeBaseResourceModel `tfsdk:"knowledge_base"`
}

type KnowledgeBaseResourceModel struct {
	TopK     types.Int64  `tfsdk:"top_k"`
	FileIDs  types.List   `tfsdk:"file_ids"`
	Provider types.String `tfsdk:"provider"`
}

type VoiceResourceModel struct {
	Model           types.String  `tfsdk:"model"`
	Provider        types.String  `tfsdk:"provider"`
	VoiceID         types.String  `tfsdk:"voice_id"`
	Stability       types.Float64 `tfsdk:"stability"`
	SimilarityBoost types.Float64 `tfsdk:"similarity_boost"`
}

type StartSpeakingPlanResourceModel struct {
	WaitSeconds                  types.Float64                              `tfsdk:"wait_seconds"`
	SmartEndpointingEnabled      types.Bool                                 `tfsdk:"smart_endpointing_enabled"`
	TranscriptionEndpointingPlan *TranscriptionEndpointingPlanResourceModel `tfsdk:"transcription_endpointing_plan"`
}

type TranscriptionEndpointingPlanResourceModel struct {
	OnPunctuationSeconds   types.Float64 `tfsdk:"on_punctuation_seconds"`
	OnNoPunctuationSeconds types.Float64 `tfsdk:"on_no_punctuation_seconds"`
	OnNumberSeconds        types.Float64 `tfsdk:"on_number_seconds"`
}

type StopSpeakingPlanResourceModel struct {
	NumWords       types.Float64 `tfsdk:"num_words"`
	VoiceSeconds   types.Float64 `tfsdk:"voice_seconds"`
	BackoffSeconds types.Float64 `tfsdk:"backoff_seconds"`
}

type AnalysisPlanResourceModel struct {
	SummaryPrompt           types.String                       `tfsdk:"summary_prompt"`
	StructuredDataPrompt    types.String                       `tfsdk:"structured_data_prompt"`
	StructuredDataSchema    *StructuredDataSchemaResourceModel `tfsdk:"structured_data_schema"`
	SuccessEvaluationPrompt types.String                       `tfsdk:"success_evaluation_prompt"`
	SuccessEvaluationRubric types.String                       `tfsdk:"success_evaluation_rubric"`
}

type StructuredDataSchemaResourceModel struct {
	Type       types.String                     `tfsdk:"type"`
	Properties map[string]PropertyResourceModel `tfsdk:"properties"`
}

type PropertyResourceModel struct {
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
}

type MessagePlanResourceModel struct {
	IdleMessages types.List `tfsdk:"idle_messages"`
}

// Metadata sets the resource type name for Terraform.
func (r *VAPIAssistantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_assistant"
}

// Schema defines the resource schema for the assistant resource.
func (r *VAPIAssistantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an assistant resource in the VAPI system.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the assistant resource.",
				Computed:            true,
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
func (r *VAPIAssistantResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

// Create implements the create operation for the assistant resource.
func (r *VAPIAssistantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPIAssistantResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := data.expand()

	response, responseCode, err := r.client.CreateAssistant(requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create assistant: %s", err))
		return
	}

	var assistantResponse vapi.Assistant
	if responseCode >= 200 && responseCode < 300 {
		if err := json.Unmarshal(response, &assistantResponse); err != nil {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
			return
		}
	} else {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response [%d]: %s", responseCode, string(response)))
		return
	}

	mapResponseObject(&data, &assistantResponse)
	tflog.Trace(ctx, "created an assistant resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read implements the read operation for the assistant resource.
func (r *VAPIAssistantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPIAssistantResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, responseCode, err := r.client.GetAssistant(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read assistant: %s", err))
		return
	}

	if responseCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	var assistantResponse vapi.Assistant
	if responseCode >= 200 && responseCode < 300 {
		if err := json.Unmarshal(response, &assistantResponse); err != nil {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
			return
		}
	} else {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response [%d]: %s", responseCode, string(response)))
		return
	}

	mapResponseObject(&data, &assistantResponse)
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

	requestBody := data.expand()

	response, responseCode, err := r.client.UpdateAssistant(state.ID.ValueString(), requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update assistant: %s", err))
		return
	}

	var assistantResponse vapi.Assistant
	if responseCode >= 200 && responseCode < 300 {
		if err := json.Unmarshal(response, &assistantResponse); err != nil {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
			return
		}
	} else {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response [%d]: %s", responseCode, string(response)))
		return
	}

	mapResponseObject(&data, &assistantResponse)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements the delete operation for the assistant resource.
func (r *VAPIAssistantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPIAssistantResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeleteAssistant(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete assistant: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted an assistant resource")
}

func mapResponseObject(data *VAPIAssistantResourceModel, assistantResponse *vapi.Assistant) {
	// Basic fields
	data.ID = types.StringValue(assistantResponse.ID)
	data.OrgID = types.StringValue(assistantResponse.OrgID)
	data.Name = types.StringValue(assistantResponse.Name)
	data.FirstMessageMode = types.StringValue(assistantResponse.FirstMessageMode)
	data.HipaaEnabled = types.BoolValue(assistantResponse.HipaaEnabled)
	data.BackgroundSound = types.StringValue(assistantResponse.BackgroundSound)
	data.BackgroundDenoising = types.BoolValue(assistantResponse.BackgroundDenoising)
	data.ModelOutputEnabled = types.BoolValue(assistantResponse.ModelOutputEnabled)
	data.Language = types.StringValue(assistantResponse.Language)
	data.ForwardingPhoneNumber = types.StringValue(assistantResponse.ForwardingPhoneNumber)
	data.InterruptionsEnabled = types.BoolValue(assistantResponse.InterruptionsEnabled)
	data.EndCallFunctionEnabled = types.BoolValue(assistantResponse.EndCallFunctionEnabled)
	data.DialKeypadFunctionEnabled = types.BoolValue(assistantResponse.DialKeypadFunctionEnabled)
	data.FillersEnabled = types.BoolValue(assistantResponse.FillersEnabled)
	data.SilenceTimeoutSeconds = types.Float64PointerValue(assistantResponse.SilenceTimeoutSeconds)
	data.ResponseDelaySeconds = types.Float64PointerValue(assistantResponse.ResponseDelaySeconds)
	data.NumWordsToInterruptAssistant = types.Int64PointerValue(assistantResponse.NumWordsToInterruptAssistant)
	data.LiveTranscriptsEnabled = types.BoolPointerValue(assistantResponse.LiveTranscriptsEnabled)
	data.Keywords = ListValueFromStrings(assistantResponse.Keywords)
	data.ParentID = types.StringPointerValue(assistantResponse.ParentID)

	// Handle optional Voice struct
	if assistantResponse.Voice != nil {
		data.Voice = &VoiceResourceModel{
			Model:           types.StringValue(assistantResponse.Voice.Model),
			Provider:        types.StringValue(assistantResponse.Voice.Provider),
			VoiceID:         types.StringValue(assistantResponse.Voice.VoiceID),
			Stability:       types.Float64Value(assistantResponse.Voice.Stability),
			SimilarityBoost: types.Float64Value(assistantResponse.Voice.SimilarityBoost),
		}
	} else {
		data.Voice = nil
	}

	// Handle optional Model struct
	if assistantResponse.Model != nil {
		data.Model = &ModelResourceModel{
			Model:        types.StringValue(assistantResponse.Model.Model),
			SystemPrompt: types.StringValue(assistantResponse.Model.SystemPrompt),
			Provider:     types.StringValue(assistantResponse.Model.Provider),
			MaxTokens:    types.Int64Value(assistantResponse.Model.MaxTokens),
			Temperature:  types.Float64Value(assistantResponse.Model.Temperature),
			ToolIDs:      ListValueFromStrings(assistantResponse.Model.ToolIDs),
			KnowledgeBase: func() *KnowledgeBaseResourceModel {
				if assistantResponse.Model.KnowledgeBase != nil {
					return &KnowledgeBaseResourceModel{
						TopK:     types.Int64Value(assistantResponse.Model.KnowledgeBase.TopK),
						FileIDs:  ListValueFromStrings(assistantResponse.Model.KnowledgeBase.FileIDs),
						Provider: types.StringValue(assistantResponse.Model.KnowledgeBase.Provider),
					}
				}
				return nil
			}(),
		}
	} else {
		data.Model = nil
	}

	data.RecordingEnabled = types.BoolValue(assistantResponse.RecordingEnabled)
	data.FirstMessage = types.StringValue(assistantResponse.FirstMessage)
	data.VoicemailMessage = types.StringValue(assistantResponse.VoicemailMessage)
	data.EndCallFunctionEnabled = types.BoolValue(assistantResponse.EndCallFunctionEnabled)

	// Handle optional Transcriber struct
	if assistantResponse.Transcriber != nil {
		data.Transcriber = &TranscriberResourceModel{
			Provider: types.StringValue(assistantResponse.Transcriber.Provider),
			Model:    types.StringValue(assistantResponse.Transcriber.Model),
			Language: types.StringValue(assistantResponse.Transcriber.Language),
		}
	} else {
		data.Transcriber = nil
	}

	//data.ServerURL = types.StringValue(assistantResponse.ServerURL)
	//data.ServerURLSecret = types.StringValue(assistantResponse.ServerURLSecret)

	data.ClientMessages = ListValueFromStrings(assistantResponse.ClientMessages)
	data.ServerMessages = ListValueFromStrings(assistantResponse.ServerMessages)
	data.EndCallPhrases = ListValueFromStrings(assistantResponse.EndCallPhrases)

	data.MaxDurationSeconds = types.Int64Value(assistantResponse.MaxDurationSeconds)

	if assistantResponse.MessagePlan != nil {
		data.MessagePlan = &MessagePlanResourceModel{
			IdleMessages: ListValueFromStrings(assistantResponse.MessagePlan.IdleMessages),
		}
	} else {
		data.MessagePlan = nil
	}

	// Handle optional AnalysisPlan struct
	if assistantResponse.AnalysisPlan != nil {
		data.AnalysisPlan = &AnalysisPlanResourceModel{
			SummaryPrompt:           types.StringValue(assistantResponse.AnalysisPlan.SummaryPrompt),
			StructuredDataPrompt:    types.StringValue(assistantResponse.AnalysisPlan.StructuredDataPrompt),
			StructuredDataSchema:    mapStructuredDataSchemaRequestToResourceModel(assistantResponse.AnalysisPlan.StructuredDataSchema),
			SuccessEvaluationPrompt: types.StringValue(assistantResponse.AnalysisPlan.SuccessEvaluationPrompt),
			SuccessEvaluationRubric: types.StringValue(assistantResponse.AnalysisPlan.SuccessEvaluationRubric),
		}
	} else {
		data.AnalysisPlan = nil
	}

	// Handle optional StartSpeakingPlan struct
	if assistantResponse.StartSpeakingPlan != nil {
		data.StartSpeakingPlan = &StartSpeakingPlanResourceModel{
			WaitSeconds:             types.Float64Value(assistantResponse.StartSpeakingPlan.WaitSeconds),
			SmartEndpointingEnabled: types.BoolValue(assistantResponse.StartSpeakingPlan.SmartEndpointingEnabled),
			TranscriptionEndpointingPlan: func() *TranscriptionEndpointingPlanResourceModel {
				if assistantResponse.StartSpeakingPlan.TranscriptionEndpointingPlan != nil {
					return &TranscriptionEndpointingPlanResourceModel{
						OnPunctuationSeconds:   types.Float64Value(assistantResponse.StartSpeakingPlan.TranscriptionEndpointingPlan.OnPunctuationSeconds),
						OnNoPunctuationSeconds: types.Float64Value(assistantResponse.StartSpeakingPlan.TranscriptionEndpointingPlan.OnNoPunctuationSeconds),
						OnNumberSeconds:        types.Float64Value(assistantResponse.StartSpeakingPlan.TranscriptionEndpointingPlan.OnNumberSeconds),
					}
				}
				return nil
			}(),
		}
	} else {
		data.StartSpeakingPlan = nil
	}

	// Handle optional StopSpeakingPlan struct
	if assistantResponse.StopSpeakingPlan != nil {
		data.StopSpeakingPlan = &StopSpeakingPlanResourceModel{
			NumWords:       types.Float64Value(assistantResponse.StopSpeakingPlan.NumWords),
			VoiceSeconds:   types.Float64Value(assistantResponse.StopSpeakingPlan.VoiceSeconds),
			BackoffSeconds: types.Float64Value(assistantResponse.StopSpeakingPlan.BackoffSeconds),
		}
	} else {
		data.StopSpeakingPlan = nil
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
