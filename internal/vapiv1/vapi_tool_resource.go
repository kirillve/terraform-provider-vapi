package vapiv1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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

var (
	_ resource.Resource                   = &VAPIToolFunctionResource{}
	_ resource.ResourceWithValidateConfig = &VAPIToolFunctionResource{}
	_ resource.ResourceWithImportState    = &VAPIToolFunctionResource{}
)

// NewVAPIToolFunctionResource returns a new tool function resource.
func NewVAPIToolFunctionResource() resource.Resource {
	return &VAPIToolFunctionResource{}
}

// VAPIToolFunctionResource manages a function, DTMF or transfer-call tool in VAPI.
type VAPIToolFunctionResource struct {
	client *vapi.APIClient
}

// Destination is one transfer-call destination.
type Destination struct {
	Type                   types.String `tfsdk:"type"`
	Number                 types.String `tfsdk:"number"`
	Extension              types.String `tfsdk:"extension"`
	Message                types.String `tfsdk:"message"`
	Description            types.String `tfsdk:"description"`
	NumberE164CheckEnabled types.Bool   `tfsdk:"number_e164_check_enabled"`
}

// VAPIToolFunctionResourceModel maps the schema data.
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

// Parameters is the structured form of the function parameters schema.
type Parameters struct {
	Type       types.String        `tfsdk:"type"`
	Async      types.Bool          `tfsdk:"async"`
	Properties map[string]Property `tfsdk:"properties"`
	Required   types.List          `tfsdk:"required"`
}

// Property is one entry of the structured function parameters.
type Property struct {
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
	Enum        types.List   `tfsdk:"enum"`
}

// Metadata sets the resource type name.
func (r *VAPIToolFunctionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool_function"
}

// Schema defines the resource schema.
func (r *VAPIToolFunctionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

// Configure binds the API client to the resource.
func (r *VAPIToolFunctionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

// Create creates the tool.
func (r *VAPIToolFunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPIToolFunctionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var functionResponse vapi.ToolFunctionResponse
	if err := r.client.Do(ctx, http.MethodPost, "tool", data.expand(), &functionResponse); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tool function: %s", err))
		return
	}

	data.flatten(ctx, &functionResponse)

	tflog.Trace(ctx, "created a tool function resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read fetches the current state from the API. A tool that no longer exists (HTTP 404) is removed
// from the state.
func (r *VAPIToolFunctionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPIToolFunctionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	var functionResponse vapi.ToolFunctionResponse
	err := r.client.Do(ctx, http.MethodGet, "tool/"+id, nil, &functionResponse)
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
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tool function: %s", err))
		return
	}

	data.flatten(ctx, &functionResponse)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the tool in place.
func (r *VAPIToolFunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state VAPIToolFunctionResourceModel
	var plan VAPIToolFunctionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Client Error", "Unable to parse response [404]: ")
		return
	}

	var functionResponse vapi.ToolFunctionResponse
	if err := r.client.Do(ctx, http.MethodPatch, "tool/"+id, plan.expand(), &functionResponse); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update tool function: %s", err))
		return
	}

	plan.flatten(ctx, &functionResponse)

	tflog.Trace(ctx, "updated a tool function resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the tool. A tool that no longer exists (HTTP 404) counts as deleted.
func (r *VAPIToolFunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPIToolFunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if id := data.ID.ValueString(); id != "" {
		if err := r.client.Do(ctx, http.MethodDelete, "tool/"+id, nil, nil); err != nil && vapi.StatusCode(err) != http.StatusNotFound {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete file: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "deleted a file resource")
}

// ImportState enables `terraform import`.
func (r *VAPIToolFunctionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// expand builds the API request from the planned model.
func (m *VAPIToolFunctionResourceModel) expand() vapi.ToolFunctionRequest {
	var parameters any
	if !m.ParametersJSON.IsNull() && !m.ParametersJSON.IsUnknown() {
		parameters = json.RawMessage(m.ParametersJSON.ValueString())
	} else if m.Parameters != nil {
		properties := make(map[string]vapi.Property, len(m.Parameters.Properties))
		for key, prop := range m.Parameters.Properties {
			properties[key] = vapi.Property{Type: prop.Type.ValueString(), Description: prop.Description.ValueString(), Enum: ElementsAsString(prop.Enum)}
		}
		parameters = &vapi.FunctionParams{Type: m.Parameters.Type.ValueString(), Properties: properties, Required: ElementsAsString(m.Parameters.Required)}
	}
	request := vapi.ToolFunctionRequest{
		Type: m.Type.ValueString(), Async: m.Async.ValueBool(),
		Function: vapi.Function{Name: m.Name.ValueString(), Description: m.Description.ValueString(), Async: m.Async.ValueBool(), Parameters: parameters},
	}

	switch request.Type {
	case "dtmf":
		return request
	case "transferCall":
		var destinations []vapi.Destination
		for i := range m.Destinations {
			dst := &m.Destinations[i]
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
			URL:            m.ServerURL.ValueString(),
			Secret:         m.ServerSecret.ValueString(),
			TimeoutSeconds: 20,
		}
		if !m.ServerTimeoutSeconds.IsNull() && !m.ServerTimeoutSeconds.IsUnknown() {
			request.Server.TimeoutSeconds = m.ServerTimeoutSeconds.ValueInt64()
		}
		return request
	}
}

// flatten binds the API response to the model, preserving unset optional values.
func (m *VAPIToolFunctionResourceModel) flatten(_ context.Context, in *vapi.ToolFunctionResponse) {
	m.ID = types.StringValue(in.ID)
	m.OrgID = types.StringValue(in.OrgID)
	m.CreatedAt = types.StringValue(in.CreatedAt)
	m.UpdatedAt = types.StringValue(in.UpdatedAt)
	m.Type = types.StringValue(in.Type)
	m.Async = types.BoolValue(in.Async)
	if in.Server.URL != "" {
		m.ServerURL = types.StringValue(in.Server.URL)
	}

	m.Name = types.StringValue(in.Function.Name)
	m.Description = types.StringValue(in.Function.Description)
	m.Async = types.BoolValue(in.Function.Async)

	// Null means the legacy implicit timeout; preserve it through refresh and import.
	if !m.ServerTimeoutSeconds.IsNull() && !m.ServerTimeoutSeconds.IsUnknown() {
		m.ServerTimeoutSeconds = types.Int64Value(in.Server.TimeoutSeconds)
	}
	m.flattenParameters(in)
}

// flattenParameters preserves the selected representation and leaves an unset empty block null.
func (m *VAPIToolFunctionResourceModel) flattenParameters(in *vapi.ToolFunctionResponse) {
	raw := in.Function.Parameters.Raw
	jsonMode := !m.ParametersJSON.IsNull()
	importing := m.Parameters == nil && !jsonMode
	// Explicit JSON null also leaves both unset parameter representations null.
	if importing && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return
	}
	// Responses constructed in Go may have no raw capture. Their typed fields
	// represent only flat parameters, so import them into the legacy model.
	if jsonMode || (importing && len(raw) > 0 && !flatToolParameters(raw)) {
		m.ParametersJSON = jsontypes.NewNormalizedValue(string(raw))
		m.Parameters = nil
		return
	}
	m.ParametersJSON = jsontypes.NewNormalizedNull()
	parameters := in.Function.Parameters
	if m.Parameters == nil && parameters.Type == "" && len(parameters.Properties) == 0 && len(parameters.Required) == 0 {
		return
	}
	priorProperties := map[string]Property(nil)
	if m.Parameters != nil {
		priorProperties = m.Parameters.Properties
	}
	m.Parameters = &Parameters{
		Type:       types.StringValue(in.Function.Parameters.Type),
		Async:      types.BoolValue(in.Function.Async),
		Required:   ListValueFromStrings(in.Function.Parameters.Required),
		Properties: make(map[string]Property),
	}

	for key, prop := range in.Function.Parameters.Properties {
		m.Parameters.Properties[key] = Property{
			Type:        types.StringValue(prop.Type),
			Description: optionalString(priorProperties[key].Description, prop.Description),
			Enum:        optionalStringList(priorProperties[key].Enum, prop.Enum),
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
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil || top == nil {
		return false
	}
	for key := range top {
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
	if err := json.Unmarshal(top["properties"], &properties); err != nil && len(top["properties"]) > 0 {
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
