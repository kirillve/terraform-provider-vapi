package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	ID                     types.String                    `tfsdk:"id"`
	Name                   types.String                    `tfsdk:"name"`
	FirstMessageMode       types.String                    `tfsdk:"first_message_mode"`
	HipaaEnabled           types.Bool                      `tfsdk:"hipaa_enabled"`
	ClientMessages         types.List                      `tfsdk:"client_messages"`
	ServerMessages         types.List                      `tfsdk:"server_messages"`
	SilenceTimeoutSeconds  types.Int64                     `tfsdk:"silence_timeout_seconds"`
	MaxDurationSeconds     types.Int64                     `tfsdk:"max_duration_seconds"`
	BackgroundSound        types.String                    `tfsdk:"background_sound"`
	BackgroundDenoising    types.Bool                      `tfsdk:"background_denoising"`
	ModelOutputEnabled     types.Bool                      `tfsdk:"model_output_enabled"`
	FirstMessage           types.String                    `tfsdk:"first_message"`
	VoicemailMessage       types.String                    `tfsdk:"voicemail_message"`
	EndCallMessage         types.String                    `tfsdk:"end_call_message"`
	ServerURL              types.String                    `tfsdk:"server_url"`
	ServerURLSecret        types.String                    `tfsdk:"server_url_secret"`
	EndCallPhrases         types.List                      `tfsdk:"end_call_phrases"`
	Transcriber            *TranscriberResourceModel       `tfsdk:"transcriber"`
	Model                  *ModelResourceModel             `tfsdk:"model"`
	Voice                  *VoiceResourceModel             `tfsdk:"voice"`
	StartSpeakingPlan      *StartSpeakingPlanResourceModel `tfsdk:"start_speaking_plan"`
	StopSpeakingPlan       *StopSpeakingPlanResourceModel  `tfsdk:"stop_speaking_plan"`
	AnalysisPlan           *AnalysisPlanResourceModel      `tfsdk:"analysis_plan"`
	MessagePlan            *MessagePlanResourceModel       `tfsdk:"message_plan"`
	EndCallFunctionEnabled types.Bool                      `tfsdk:"end_call_function_enabled"`
	RecordingEnabled       types.Bool                      `tfsdk:"recording_enabled"`
	ForwardingPhoneNumber  types.String                    `tfsdk:"forwarding_phone_number"`
	PhoneNumberID          types.String                    `tfsdk:"phone_number_id"`
}

type TranscriberResourceModel struct {
	Provider types.String `tfsdk:"provider"`
	Model    types.String `tfsdk:"model"`
	Keywords types.List   `tfsdk:"keywords"`
}

type ModelResourceModel struct {
	Model        types.String  `tfsdk:"model"`
	SystemPrompt types.String  `tfsdk:"system_prompt"`
	Provider     types.String  `tfsdk:"provider"`
	Temperature  types.Float64 `tfsdk:"temperature"`
	ToolIds      types.List    `tfsdk:"tool_ids"`
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the assistant resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"first_message_mode": schema.StringAttribute{
				MarkdownDescription: "Mode of the first message.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hipaa_enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether HIPAA compliance is enabled.",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"client_messages": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of messages from the client.",
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"server_messages": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of messages from the server.",
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"silence_timeout_seconds": schema.Int64Attribute{
				MarkdownDescription: "Timeout in seconds for silence.",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"max_duration_seconds": schema.Int64Attribute{
				MarkdownDescription: "Maximum duration of the call in seconds.",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"background_sound": schema.StringAttribute{
				MarkdownDescription: "Background sound used during the call.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"background_denoising": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether background denoising is enabled.",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"model_output_enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether model output is enabled.",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"first_message": schema.StringAttribute{
				MarkdownDescription: "Initial message sent to the client.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"voicemail_message": schema.StringAttribute{
				MarkdownDescription: "Message to be used for voicemail.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"end_call_message": schema.StringAttribute{
				MarkdownDescription: "Message to be used when ending the call.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"server_url": schema.StringAttribute{
				MarkdownDescription: "Server URL.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"server_url_secret": schema.StringAttribute{
				MarkdownDescription: "Server URL Secret.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"end_call_phrases": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of phrases to end the call.",
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"transcriber": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for the transcriber model.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"provider": schema.StringAttribute{
						MarkdownDescription: "Provider for the transcriber service.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"model": schema.StringAttribute{
						MarkdownDescription: "Model used for transcription.",
						Optional:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"keywords": schema.ListAttribute{
						ElementType:         types.StringType,
						MarkdownDescription: "List of keywords to focus on during transcription.",
						Optional:            true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.RequiresReplace(),
						},
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
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"system_prompt": schema.StringAttribute{
						MarkdownDescription: "Prompt text used to guide the assistant model.",
						Optional:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"provider": schema.StringAttribute{
						MarkdownDescription: "Provider for the assistant model: openai, azure-openai, together-ai, anyscale, openrouter, perplexity-ai, deepinfra, custom-llm, runpod, groq, vapi, anthropic, google",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"temperature": schema.Float64Attribute{
						MarkdownDescription: "Temperature setting for the model's response randomness.",
						Optional:            true,
						PlanModifiers: []planmodifier.Float64{
							float64planmodifier.RequiresReplace(),
						},
					},
					"tool_ids": schema.ListAttribute{
						ElementType:         types.StringType,
						MarkdownDescription: "List of tool IDs used by the model.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.RequiresReplace(),
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
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"provider": schema.StringAttribute{
						MarkdownDescription: "Provider for the voice model.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"voice_id": schema.StringAttribute{
						MarkdownDescription: "ID of the voice model.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"stability": schema.Float64Attribute{
						MarkdownDescription: "Stability of the voice output.",
						Optional:            true,
						PlanModifiers: []planmodifier.Float64{
							float64planmodifier.RequiresReplace(),
						},
					},
					"similarity_boost": schema.Float64Attribute{
						MarkdownDescription: "Boost factor for similarity in voice.",
						Optional:            true,
						PlanModifiers: []planmodifier.Float64{
							float64planmodifier.RequiresReplace(),
						},
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
						PlanModifiers: []planmodifier.Float64{
							float64planmodifier.RequiresReplace(),
						},
					},
					"smart_endpointing_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable smart endpointing for better control.",
						Optional:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
					},
					"transcription_endpointing_plan": schema.SingleNestedAttribute{
						MarkdownDescription: "Endpointing plan for transcription.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"on_punctuation_seconds": schema.Float64Attribute{
								MarkdownDescription: "Delay in seconds for punctuation-based endpointing.",
								Optional:            true,
								PlanModifiers: []planmodifier.Float64{
									float64planmodifier.RequiresReplace(),
								},
							},
							"on_no_punctuation_seconds": schema.Float64Attribute{
								MarkdownDescription: "Delay in seconds for no punctuation.",
								Optional:            true,
								PlanModifiers: []planmodifier.Float64{
									float64planmodifier.RequiresReplace(),
								},
							},
							"on_number_seconds": schema.Float64Attribute{
								MarkdownDescription: "Delay in seconds for number-based endpointing.",
								Optional:            true,
								PlanModifiers: []planmodifier.Float64{
									float64planmodifier.RequiresReplace(),
								},
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
						PlanModifiers: []planmodifier.Float64{
							float64planmodifier.RequiresReplace(),
						},
					},
					"voice_seconds": schema.Float64Attribute{
						MarkdownDescription: "Duration in seconds to stop speaking.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Float64{
							float64planmodifier.RequiresReplace(),
						},
					},
					"backoff_seconds": schema.Float64Attribute{
						MarkdownDescription: "Backoff period in seconds before stopping.",
						Optional:            true,
						PlanModifiers: []planmodifier.Float64{
							float64planmodifier.RequiresReplace(),
						},
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
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"structured_data_prompt": schema.StringAttribute{
						MarkdownDescription: "Prompt for structured data generation.",
						Optional:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"structured_data_schema": schema.SingleNestedAttribute{
						MarkdownDescription: "Schema for structured data.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								MarkdownDescription: "Type of structured data.",
								Optional:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.RequiresReplace(),
								},
							},
							"properties": schema.MapNestedAttribute{
								MarkdownDescription: "Properties of structured data fields.",
								Optional:            true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											MarkdownDescription: "Type of the property.",
											Optional:            true,
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.RequiresReplace(),
											},
										},
										"description": schema.StringAttribute{
											MarkdownDescription: "Description of the property.",
											Optional:            true,
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.RequiresReplace(),
											},
										},
									},
								},
							},
						},
					},
					"success_evaluation_prompt": schema.StringAttribute{
						MarkdownDescription: "Prompt for success evaluation.",
						Optional:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"success_evaluation_rubric": schema.StringAttribute{
						MarkdownDescription: "Rubric for evaluating success.",
						Optional:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
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
						PlanModifiers: []planmodifier.List{
							listplanmodifier.RequiresReplace(),
						},
					},
				},
			},

			"end_call_function_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enables the end call function.",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},

			"recording_enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if call recording is enabled.",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},

			"forwarding_phone_number": schema.StringAttribute{
				MarkdownDescription: "Phone number to which calls are forwarded.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"phone_number_id": schema.StringAttribute{
				MarkdownDescription: "ID of the phone number associated with the assistant.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure assigns the configured API client to the resource.
func (r *VAPIAssistantResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*vapi.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *vapi.APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create implements the create operation for the assistant resource.
func (r *VAPIAssistantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPIAssistantResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := mapVAPIAssistantRequest(&data)

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

	bindVAPIAssistantResourceData(&data, &assistantResponse)
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

	bindVAPIAssistantResourceData(&data, &assistantResponse)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update implements the update operation for the assistant resource.
func (r *VAPIAssistantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VAPIAssistantResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeleteAssistant(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete assistant: %s", err))
		return
	}

	requestBody := mapVAPIAssistantRequest(&data)

	response, responseCode, err := r.client.CreateAssistant(requestBody)
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

	bindVAPIAssistantResourceData(&data, &assistantResponse)
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

func bindVAPIAssistantResourceData(data *VAPIAssistantResourceModel, assistantResponse *vapi.Assistant) {
	// Basic fields
	data.ID = types.StringValue(assistantResponse.ID)
	data.Name = types.StringValue(assistantResponse.Name)
	data.FirstMessageMode = types.StringValue(assistantResponse.FirstMessageMode)
	data.HipaaEnabled = types.BoolValue(assistantResponse.HipaaEnabled)
	data.SilenceTimeoutSeconds = types.Int64Value(int64(assistantResponse.SilenceTimeoutSeconds))
	data.MaxDurationSeconds = types.Int64Value(int64(assistantResponse.MaxDurationSeconds))

	// Lists and nested attributes
	data.ClientMessages = ListValueFromStrings(assistantResponse.ClientMessages)
	data.ServerMessages = ListValueFromStrings(assistantResponse.ServerMessages)
	data.EndCallPhrases = ListValueFromStrings(assistantResponse.EndCallPhrases)

	// Background and Voice
	data.BackgroundSound = types.StringValue(assistantResponse.BackgroundSound)
	data.BackgroundDenoising = types.BoolValue(assistantResponse.BackgroundDenoising)
	data.ModelOutputEnabled = types.BoolValue(assistantResponse.ModelOutputEnabled)

	// Transcriber model
	if assistantResponse.Transcriber != nil {
		data.Transcriber = &TranscriberResourceModel{
			Provider: types.StringValue(assistantResponse.Transcriber.Provider),
			Model:    types.StringValue(assistantResponse.Transcriber.Model),
			Keywords: ListValueFromStrings(assistantResponse.Transcriber.Keywords),
		}
	}

	// Model configuration
	if assistantResponse.Model != nil {
		data.Model = &ModelResourceModel{
			Provider:     types.StringValue(assistantResponse.Model.Provider),
			Model:        types.StringValue(assistantResponse.Model.Model),
			SystemPrompt: types.StringValue(assistantResponse.Model.SystemPrompt),
			Temperature:  types.Float64Value(assistantResponse.Model.Temperature),
			ToolIds:      ListValueFromStrings(assistantResponse.Model.ToolIDs),
		}
	}

	// Voice configuration
	if assistantResponse.Voice != nil {
		data.Voice = &VoiceResourceModel{
			Model:           types.StringValue(assistantResponse.Voice.Model),
			Provider:        types.StringValue(assistantResponse.Voice.Provider),
			VoiceID:         types.StringValue(assistantResponse.Voice.VoiceID),
			Stability:       types.Float64Value(assistantResponse.Voice.Stability),
			SimilarityBoost: types.Float64Value(assistantResponse.Voice.SimilarityBoost),
		}
	}

	// Speaking Plans
	data.StartSpeakingPlan = mapStartSpeakingPlan(assistantResponse.StartSpeakingPlan)
	data.StopSpeakingPlan = mapStopSpeakingPlan(assistantResponse.StopSpeakingPlan)
}

func mapStartSpeakingPlan(plan *vapi.StartSpeakingPlan) *StartSpeakingPlanResourceModel {
	if plan == nil {
		return nil
	}
	return &StartSpeakingPlanResourceModel{
		WaitSeconds:             types.Float64Value(plan.WaitSeconds),
		SmartEndpointingEnabled: types.BoolValue(plan.SmartEndpointingEnabled),
		TranscriptionEndpointingPlan: &TranscriptionEndpointingPlanResourceModel{
			OnPunctuationSeconds:   types.Float64Value(plan.TranscriptionEndpointingPlan.OnPunctuationSeconds),
			OnNoPunctuationSeconds: types.Float64Value(plan.TranscriptionEndpointingPlan.OnNoPunctuationSeconds),
			OnNumberSeconds:        types.Float64Value(plan.TranscriptionEndpointingPlan.OnNumberSeconds),
		},
	}
}

func mapStopSpeakingPlan(plan *vapi.StopSpeakingPlan) *StopSpeakingPlanResourceModel {
	if plan == nil {
		return nil
	}
	return &StopSpeakingPlanResourceModel{
		NumWords:       types.Float64Value(plan.NumWords),
		VoiceSeconds:   types.Float64Value(plan.VoiceSeconds),
		BackoffSeconds: types.Float64PointerValue(plan.BackoffSeconds),
	}
}

func mapVAPIAssistantRequest(data *VAPIAssistantResourceModel) vapi.CreateAssistantRequest {
	return vapi.CreateAssistantRequest{
		Name:                  data.Name.ValueString(),
		FirstMessageMode:      data.FirstMessageMode.ValueString(),
		HipaaEnabled:          data.HipaaEnabled.ValueBool(),
		ClientMessages:        ElementsAsString(data.ClientMessages),
		ServerMessages:        ElementsAsString(data.ServerMessages),
		SilenceTimeoutSeconds: int(data.SilenceTimeoutSeconds.ValueInt64()),
		MaxDurationSeconds:    int(data.MaxDurationSeconds.ValueInt64()),
		BackgroundSound:       data.BackgroundSound.ValueString(),
		BackgroundDenoising:   data.BackgroundDenoising.ValueBool(),
		ModelOutputEnabled:    data.ModelOutputEnabled.ValueBool(),
		FirstMessage:          data.FirstMessage.ValueString(),
		VoicemailMessage:      data.VoicemailMessage.ValueString(),
		EndCallMessage:        data.EndCallMessage.ValueString(),
		EndCallPhrases:        ElementsAsString(data.EndCallPhrases),
		ServerURL:             data.ServerURL.ValueString(),
		ServerURLSecret:       data.ServerURLSecret.ValueString(),

		Transcriber: vapi.Transcriber{
			Provider: data.Transcriber.Provider.ValueString(),
			Model:    data.Transcriber.Model.ValueString(),
			Keywords: ElementsAsString(data.Transcriber.Keywords),
		},
		Model: vapi.Model{
			Model:        data.Model.Model.ValueString(),
			SystemPrompt: data.Model.SystemPrompt.ValueString(),
			Temperature:  data.Model.Temperature.ValueFloat64(),
			ToolIDs:      ElementsAsString(data.Model.ToolIds),
			Provider:     data.Model.Provider.ValueString(),
		},
		Voice: vapi.Voice{
			Model:           data.Voice.Model.ValueString(),
			Provider:        data.Voice.Provider.ValueString(),
			VoiceID:         data.Voice.VoiceID.ValueString(),
			Stability:       data.Voice.Stability.ValueFloat64(),
			SimilarityBoost: data.Voice.SimilarityBoost.ValueFloat64(),
		},
		StartSpeakingPlan: vapi.StartSpeakingPlan{
			WaitSeconds:             data.StartSpeakingPlan.WaitSeconds.ValueFloat64(),
			SmartEndpointingEnabled: data.StartSpeakingPlan.SmartEndpointingEnabled.ValueBool(),
			TranscriptionEndpointingPlan: vapi.TranscriptionEndpointingPlan{
				OnPunctuationSeconds:   data.StartSpeakingPlan.TranscriptionEndpointingPlan.OnPunctuationSeconds.ValueFloat64(),
				OnNoPunctuationSeconds: data.StartSpeakingPlan.TranscriptionEndpointingPlan.OnNoPunctuationSeconds.ValueFloat64(),
				OnNumberSeconds:        data.StartSpeakingPlan.TranscriptionEndpointingPlan.OnNumberSeconds.ValueFloat64(),
			},
		},
		StopSpeakingPlan: vapi.StopSpeakingPlan{
			NumWords:       data.StopSpeakingPlan.NumWords.ValueFloat64(),
			VoiceSeconds:   data.StopSpeakingPlan.VoiceSeconds.ValueFloat64(),
			BackoffSeconds: data.StopSpeakingPlan.BackoffSeconds.ValueFloat64Pointer(),
		},

		AnalysisPlan: &vapi.AnalysisPlan{
			SummaryPrompt:           data.AnalysisPlan.SummaryPrompt.ValueString(),
			StructuredDataPrompt:    data.AnalysisPlan.StructuredDataPrompt.ValueString(),
			StructuredDataSchema:    mapStructuredDataSchemaResourceModelToRequest(data.AnalysisPlan.StructuredDataSchema),
			SuccessEvaluationPrompt: data.AnalysisPlan.SuccessEvaluationPrompt.ValueString(),
			SuccessEvaluationRubric: data.AnalysisPlan.SuccessEvaluationRubric.ValueString(),
		},
	}
}

func mapStructuredDataSchemaResourceModelToRequest(data *StructuredDataSchemaResourceModel) *vapi.StructuredDataSchema {
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
