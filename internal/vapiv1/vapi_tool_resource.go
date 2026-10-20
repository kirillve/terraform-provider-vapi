package vapiv1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

var _ resource.Resource = &VAPIToolFunctionResource{}
var _ resource.ResourceWithValidateConfig = &VAPIToolFunctionResource{}
var _ resource.ResourceWithImportState = &VAPIToolFunctionResource{}

func NewVAPIToolFunctionResource() resource.Resource {
	return &VAPIToolFunctionResource{}
}

type VAPIToolFunctionResource struct {
	client *vapi.APIClient
}

type Destination struct {
	Type                   types.String `tfsdk:"type"`
	Number                 types.String `tfsdk:"number"`
	Extension              types.String `tfsdk:"extension"`
	Message                types.String `tfsdk:"message"`
	Description            types.String `tfsdk:"description"`
	NumberE164CheckEnabled types.Bool   `tfsdk:"number_e164_check_enabled"`
}

type VAPIToolFunctionResourceModel struct {
	ID                   types.String         `tfsdk:"id"`
	OrgID                types.String         `tfsdk:"org_id"`
	Name                 types.String         `tfsdk:"name"`
	Description          types.String         `tfsdk:"description"`
	Async                types.Bool           `tfsdk:"async"`
	Type                 types.String         `tfsdk:"type"`
	ServerURL            types.String         `tfsdk:"server_url"`
	ServerSecret         types.String         `tfsdk:"server_secret"`
	Parameters           *Parameters          `tfsdk:"parameters"`
	ParametersJSON       jsontypes.Normalized `tfsdk:"parameters_json"`
	ServerTimeoutSeconds types.Int64          `tfsdk:"server_timeout_seconds"`
	Destinations         []Destination        `tfsdk:"destinations"`
	CreatedAt            types.String         `tfsdk:"created_at"`
	UpdatedAt            types.String         `tfsdk:"updated_at"`
}

type Parameters struct {
	Type       types.String        `tfsdk:"type"`
	Async      types.Bool          `tfsdk:"async"`
	Properties map[string]Property `tfsdk:"properties"`
	Required   types.List          `tfsdk:"required"`
}

type Property struct {
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
	Enum        types.List   `tfsdk:"enum"`
}

func (r *VAPIToolFunctionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool_function"
}

func (r *VAPIToolFunctionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a function tool resource in the VAPI system. For function tools, set exactly one of parameters or parameters_json. Import selects parameters for flat schemas containing only type, properties, and required, with property type, description, and enum; other schemas use parameters_json.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the tool function.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the tool function.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the function.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The description of the function.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"async": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Indicates whether the function is asynchronous.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of the tool (function).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"server_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The URL of the server where the function is hosted.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"server_secret": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The secret used to authenticate with the server.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parameters_json": schema.StringAttribute{
				Optional:            true,
				CustomType:          jsontypes.NormalizedType{},
				MarkdownDescription: "Complete JSON Schema object for function.parameters, including nested objects and arrays. JSON whitespace and key order are ignored. Mutually exclusive with parameters for function tools.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"server_timeout_seconds": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Server timeout in seconds for function tools. When omitted, requests use 20 and state remains null. Changes update the tool in place.",
			},
			"parameters": schema.SingleNestedAttribute{
				MarkdownDescription: "Function parameters including type, async, and properties.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The type of parameters (object).",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"async": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Indicates whether the function parameters are async.",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
					},
					"required": schema.ListAttribute{
						Required:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "List of required fields.",
						PlanModifiers: []planmodifier.List{
							listplanmodifier.RequiresReplace(),
						},
					},
					"properties": schema.MapNestedAttribute{
						MarkdownDescription: "The properties for the function parameters.",
						Required:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "The type of the property.",
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.RequiresReplace(),
									},
								},
								"description": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "A description of the property.",
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.RequiresReplace(),
									},
								},
								"enum": schema.ListAttribute{
									Optional:            true,
									ElementType:         types.StringType,
									MarkdownDescription: "List of possible values for the property.",
									PlanModifiers: []planmodifier.List{
										listplanmodifier.RequiresReplace(),
									},
								},
							},
						},
					},
				},
			},
			"destinations": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "List of destinations to forward calls.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The type of the destination (e.g., number).",
						},
						"number": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The phone number to forward to.",
						},
						"extension": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The phone number extension to forward to.",
						},
						"message": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Message to play before forwarding.",
						},
						"description": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Description for the destination.",
						},
						"number_e164_check_enabled": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "Indicates whether to check the number for E.164 format.",
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the tool function was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the tool function was last updated.",
			},
		},
	}
}

func (r *VAPIToolFunctionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *VAPIToolFunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPIToolFunctionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := buildToolFunctionRequest(&data)
	response, responseCode, err := r.client.CreateToolFunction(requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tool function: %s", err))
		return
	}

	var functionResponse vapi.ToolFunctionResponse
	if responseCode >= 200 && responseCode < 300 {
		if err := json.Unmarshal(response, &functionResponse); err != nil {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
			return
		}
	} else {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response [%d]: %s", responseCode, string(response)))
		return
	}

	bindVAPIToolFunctionResourceData(&data, &functionResponse)

	tflog.Trace(ctx, "created a tool function resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPIToolFunctionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPIToolFunctionResourceModel
	var functionResponse vapi.ToolFunctionResponse

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, responseCode, err := r.client.GetToolFunction(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tool function: %s", err))
		return
	}

	if responseCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if responseCode >= 200 && responseCode < 300 {
		if err := json.Unmarshal(response, &functionResponse); err != nil {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
			return
		}
	} else {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response [%d]: %s", responseCode, string(response)))
		return
	}

	bindVAPIToolFunctionResourceData(&data, &functionResponse)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPIToolFunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state VAPIToolFunctionResourceModel
	var plan VAPIToolFunctionResourceModel
	var functionResponse vapi.ToolFunctionResponse

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := buildToolFunctionRequest(&plan)
	response, responseCode, err := r.client.UpdateToolFunction(state.ID.ValueString(), requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update tool function: %s", err))
		return
	}

	if responseCode >= 200 && responseCode < 300 {
		if err := json.Unmarshal(response, &functionResponse); err != nil {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
			return
		}
	} else {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response [%d]: %s", responseCode, string(response)))
		return
	}

	bindVAPIToolFunctionResourceData(&plan, &functionResponse)

	tflog.Trace(ctx, "updated a tool function resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VAPIToolFunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPIToolFunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeleteToolFunction(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete file: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a file resource")
}

func (r *VAPIToolFunctionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildToolFunctionRequest(data *VAPIToolFunctionResourceModel) vapi.ToolFunctionRequest {
	var parameters any
	if !data.ParametersJSON.IsNull() && !data.ParametersJSON.IsUnknown() {
		parameters = json.RawMessage(data.ParametersJSON.ValueString())
	} else if data.Parameters != nil {
		properties := make(map[string]vapi.Property, len(data.Parameters.Properties))
		for key, prop := range data.Parameters.Properties {
			properties[key] = vapi.Property{Type: prop.Type.ValueString(), Description: prop.Description.ValueString(), Enum: ElementsAsString(prop.Enum)}
		}
		parameters = &vapi.FunctionParams{Type: data.Parameters.Type.ValueString(), Properties: properties, Required: ElementsAsString(data.Parameters.Required)}
	}
	request := vapi.ToolFunctionRequest{
		Type: data.Type.ValueString(), Async: data.Async.ValueBool(),
		Function: vapi.Function{Name: data.Name.ValueString(), Description: data.Description.ValueString(), Async: data.Async.ValueBool(), Parameters: parameters},
	}

	switch request.Type {
	case "dtmf":
		return request
	case "transferCall":
		var destinations []vapi.Destination
		for _, dst := range data.Destinations {
			destinations = append(destinations, vapi.Destination{
				Type:                   dst.Type.ValueString(),
				Number:                 dst.Number.ValueString(),
				Extension:              dst.Extension.ValueString(),
				Message:                dst.Message.ValueString(),
				Description:            dst.Description.ValueString(),
				NumberE164CheckEnabled: dst.NumberE164CheckEnabled.ValueBool(),
			})
		}
		request.Destinations = destinations
		return request
	default:
		request.Type = "function"
		request.Server = &vapi.Server{
			URL:            data.ServerURL.ValueString(),
			Secret:         data.ServerSecret.ValueString(),
			TimeoutSeconds: 20,
		}
		if !data.ServerTimeoutSeconds.IsNull() && !data.ServerTimeoutSeconds.IsUnknown() {
			request.Server.TimeoutSeconds = data.ServerTimeoutSeconds.ValueInt64()
		}
		return request
	}
}

func bindVAPIToolFunctionResourceData(data *VAPIToolFunctionResourceModel, functionResponse *vapi.ToolFunctionResponse) {
	data.ID = types.StringValue(functionResponse.ID)
	data.OrgID = types.StringValue(functionResponse.OrgID)
	data.CreatedAt = types.StringValue(functionResponse.CreatedAt)
	data.UpdatedAt = types.StringValue(functionResponse.UpdatedAt)
	data.Type = types.StringValue(functionResponse.Type)
	data.Async = types.BoolValue(functionResponse.Async)
	if len(functionResponse.Server.URL) > 0 {
		data.ServerURL = types.StringValue(functionResponse.Server.URL)
	}

	data.Name = types.StringValue(functionResponse.Function.Name)
	data.Description = types.StringValue(functionResponse.Function.Description)
	data.Async = types.BoolValue(functionResponse.Function.Async)

	// Null means the legacy implicit timeout; preserve it through refresh and import.
	if !data.ServerTimeoutSeconds.IsNull() && !data.ServerTimeoutSeconds.IsUnknown() {
		data.ServerTimeoutSeconds = types.Int64Value(functionResponse.Server.TimeoutSeconds)
	}
	raw := functionResponse.Function.Parameters.Raw
	jsonMode := !data.ParametersJSON.IsNull()
	importing := data.Parameters == nil && !jsonMode
	// Responses constructed in Go may have no raw capture. Their typed fields
	// represent only flat parameters, so import them into the legacy model.
	if jsonMode || (importing && len(raw) > 0 && !flatToolParameters(raw)) {
		data.ParametersJSON = jsontypes.NewNormalizedValue(string(raw))
		data.Parameters = nil
		return
	}
	data.ParametersJSON = jsontypes.NewNormalizedNull()
	data.Parameters = &Parameters{
		Type:       types.StringValue(functionResponse.Function.Parameters.Type),
		Async:      types.BoolValue(functionResponse.Function.Async),
		Required:   ListValueFromStrings(functionResponse.Function.Parameters.Required),
		Properties: make(map[string]Property),
	}

	for key, prop := range functionResponse.Function.Parameters.Properties {
		enumList, diag := types.ListValueFrom(context.Background(), types.StringType, prop.Enum)
		if diag.HasError() {
			// Optionally log or handle diagnostics
			continue
		}

		data.Parameters.Properties[key] = Property{
			Type:        types.StringValue(prop.Type),
			Description: types.StringValue(prop.Description),
			Enum:        enumList,
		}
	}
}

// ValidateConfig defers unknown values until Terraform can resolve them.
func (r *VAPIToolFunctionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var kind types.String
	var parameters types.Object
	var raw jsontypes.Normalized
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("type"), &kind)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("parameters"), &parameters)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("parameters_json"), &raw)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !raw.IsNull() && !raw.IsUnknown() {
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw.ValueString()), &object); err != nil || object == nil {
			resp.Diagnostics.AddAttributeError(path.Root("parameters_json"), "Invalid parameters_json", "parameters_json must be a valid JSON object containing the complete function parameters schema.")
		}
	}
	if kind.IsUnknown() || kind.ValueString() != "function" || parameters.IsUnknown() || raw.IsUnknown() {
		return
	}
	if parameters.IsNull() == raw.IsNull() {
		resp.Diagnostics.AddError("Invalid function parameters", "For type = function, exactly one of parameters and parameters_json must be set.")
	}
}

// Import into the legacy model only when doing so preserves the entire schema.
func flatToolParameters(raw json.RawMessage) bool {
	var schema map[string]json.RawMessage
	if err := json.Unmarshal(raw, &schema); err != nil || schema == nil {
		return false
	}
	for key := range schema {
		switch key {
		case "type", "properties", "required":
		default:
			return false
		}
	}
	var flat vapi.FunctionParams
	if err := json.Unmarshal(raw, &flat); err != nil {
		return false
	}
	var properties map[string]map[string]json.RawMessage
	if err := json.Unmarshal(schema["properties"], &properties); err != nil && len(schema["properties"]) > 0 {
		return false
	}
	for _, property := range properties {
		for key := range property {
			switch key {
			case "type", "description", "enum":
			default:
				return false
			}
		}
	}
	return true
}
