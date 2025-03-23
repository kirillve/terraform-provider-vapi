package provider

import (
	"context"
	"encoding/json"
	"fmt"
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
var _ resource.ResourceWithImportState = &VAPIToolFunctionResource{}

func NewVAPIToolFunctionResource() resource.Resource {
	return &VAPIToolFunctionResource{}
}

type VAPIToolFunctionResource struct {
	client *vapi.APIClient
}

type VAPIToolFunctionResourceModel struct {
	ID           types.String `tfsdk:"id"`
	OrgID        types.String `tfsdk:"org_id"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Async        types.Bool   `tfsdk:"async"`
	Type         types.String `tfsdk:"type"`
	ServerURL    types.String `tfsdk:"server_url"`
	ServerSecret types.String `tfsdk:"server_secret"`
	Parameters   Parameters   `tfsdk:"parameters"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
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
}

func (r *VAPIToolFunctionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool_function"
}

func (r *VAPIToolFunctionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a function tool resource in the VAPI system.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the tool function.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the tool function.",
				PlanModifiers: []planmodifier.String{
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
			"parameters": schema.SingleNestedAttribute{
				MarkdownDescription: "Function parameters including type, async, and properties.",
				Required:            true,
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
							},
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

func (r *VAPIToolFunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPIToolFunctionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	properties := make(map[string]vapi.Property)

	for key, prop := range data.Parameters.Properties {
		properties[key] = vapi.Property{
			Type:        prop.Type.ValueString(),
			Description: prop.Description.ValueString(),
		}
	}

	var requestBody vapi.FunctionRequest
	if data.Type.ValueString() == "dtmf" {
		requestBody = vapi.FunctionRequest{
			Type:  data.Type.ValueString(),
			Async: data.Async.ValueBool(),
			Function: vapi.Function{
				Name:        data.Name.ValueString(),
				Description: data.Description.ValueString(),
				Async:       data.Async.ValueBool(),
				Parameters: vapi.FunctionParams{
					Type:       data.Parameters.Type.ValueString(),
					Properties: properties,
					Required:   ElementsAsString(data.Parameters.Required),
				},
			},
		}
	} else {
		requestBody = vapi.FunctionRequest{
			Type:  "function",
			Async: data.Async.ValueBool(),
			Server: vapi.Server{
				URL:    data.ServerURL.ValueString(),
				Secret: data.ServerSecret.ValueString(),
			},
			Function: vapi.Function{
				Name:        data.Name.ValueString(),
				Description: data.Description.ValueString(),
				Async:       data.Async.ValueBool(),
				Parameters: vapi.FunctionParams{
					Type:       data.Parameters.Type.ValueString(),
					Properties: properties,
					Required:   ElementsAsString(data.Parameters.Required),
				},
			},
		}
	}

	response, responseCode, err := r.client.CreateToolFunction(requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tool function: %s", err))
		return
	}

	var functionResponse vapi.FunctionResponse
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
	var functionResponse vapi.FunctionResponse

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
	var data VAPIToolFunctionResourceModel
	var functionResponse vapi.FunctionResponse

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeleteFile(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete file: %s", err))
		return
	}
	bindVAPIToolFunctionResourceData(&data, &vapi.FunctionResponse{})

	tflog.Trace(ctx, "deleted a file resource")

	properties := make(map[string]vapi.Property)

	for key, prop := range data.Parameters.Properties {
		properties[key] = vapi.Property{
			Type:        prop.Type.ValueString(),
			Description: prop.Description.ValueString(),
		}
	}

	var requestBody vapi.FunctionRequest
	if data.Type.ValueString() == "dtmf" {
		requestBody = vapi.FunctionRequest{
			Type:  data.Type.ValueString(),
			Async: data.Async.ValueBool(),
			Function: vapi.Function{
				Name:        data.Name.ValueString(),
				Description: data.Description.ValueString(),
				Async:       data.Async.ValueBool(),
				Parameters: vapi.FunctionParams{
					Type:       data.Parameters.Type.ValueString(),
					Properties: properties,
					Required:   ElementsAsString(data.Parameters.Required),
				},
			},
		}
	} else {
		requestBody = vapi.FunctionRequest{
			Type:  "function",
			Async: data.Async.ValueBool(),
			Server: vapi.Server{
				URL:    data.ServerURL.ValueString(),
				Secret: data.ServerSecret.ValueString(),
			},
			Function: vapi.Function{
				Name:        data.Name.ValueString(),
				Description: data.Description.ValueString(),
				Async:       data.Async.ValueBool(),
				Parameters: vapi.FunctionParams{
					Type:       data.Parameters.Type.ValueString(),
					Properties: properties,
					Required:   ElementsAsString(data.Parameters.Required),
				},
			},
		}
	}

	response, responseCode, err := r.client.CreateToolFunction(requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tool function: %s", err))
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

	tflog.Trace(ctx, "created a tool function resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

func bindVAPIToolFunctionResourceData(data *VAPIToolFunctionResourceModel, functionResponse *vapi.FunctionResponse) {
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

	data.Parameters = Parameters{
		Type:       types.StringValue(functionResponse.Function.Parameters.Type),
		Async:      types.BoolValue(functionResponse.Function.Async),
		Required:   ListValueFromStrings(functionResponse.Function.Parameters.Required),
		Properties: make(map[string]Property),
	}

	for key, prop := range functionResponse.Function.Parameters.Properties {
		data.Parameters.Properties[key] = Property{
			Type:        types.StringValue(prop.Type),
			Description: types.StringValue(prop.Description),
		}
	}
}
