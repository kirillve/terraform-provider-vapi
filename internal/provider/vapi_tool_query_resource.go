package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

var _ resource.Resource = &VAPIToolQueryFunctionResource{}
var _ resource.ResourceWithImportState = &VAPIToolQueryFunctionResource{}

func NewVAPIToolQueryFunctionResource() resource.Resource {
	return &VAPIToolQueryFunctionResource{}
}

type VAPIToolQueryFunctionResource struct {
	client *vapi.APIClient
}

type KnowledgeBase struct {
	Provider    types.String `tfsdk:"provider"`
	Name        types.String `tfsdk:"name"`
	Model       types.String `tfsdk:"model"`
	Description types.String `tfsdk:"description"`
	FileIDs     types.List   `tfsdk:"file_ids"`
}

type VAPIToolQueryFunctionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrgID          types.String `tfsdk:"org_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	KnowledgeBases types.List   `tfsdk:"knowledge_bases"`
}

func (r *VAPIToolQueryFunctionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool_query_function"
}

func (r *VAPIToolQueryFunctionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a tool query function resource in VAPI.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the resource.",
			},
			"org_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Organization ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the tool query function.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Description of the function.",
			},
			"knowledge_bases": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "List of knowledge bases.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"provider": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Provider name.",
						},
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Knowledge base name.",
						},
						"model": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Model used by the knowledge base.",
						},
						"description": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Description.",
						},
						"file_ids": schema.ListAttribute{
							ElementType:         types.StringType,
							Required:            true,
							MarkdownDescription: "List of file IDs.",
						},
					},
				},
			},
		},
	}
}

func (r *VAPIToolQueryFunctionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*vapi.APIClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *vapi.APIClient.")
		return
	}
	r.client = client
}

func (r *VAPIToolQueryFunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPIToolQueryFunctionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var kbs []vapi.TQKnowledgeBase
	for _, kbVal := range data.KnowledgeBases.Elements() {
		objVal, ok := kbVal.(types.Object)
		if !ok {
			resp.Diagnostics.AddError("Type Assertion Error", "Expected Object type for knowledge_base element")
			return
		}
		attrs := objVal.Attributes()

		provider, ok := attrs["provider"].(types.String)
		if !ok {
			resp.Diagnostics.AddError("Type Assertion Error", "Expected String type for provider")
			return
		}
		name, ok := attrs["name"].(types.String)
		if !ok {
			resp.Diagnostics.AddError("Type Assertion Error", "Expected String type for name")
			return
		}
		model, ok := attrs["model"].(types.String)
		if !ok {
			resp.Diagnostics.AddError("Type Assertion Error", "Expected String type for model")
			return
		}
		description, ok := attrs["description"].(types.String)
		if !ok {
			description = types.StringValue("") // fallback if optional
		}
		fileIDsList, ok := attrs["file_ids"].(types.List)
		if !ok {
			resp.Diagnostics.AddError("Type Assertion Error", "Expected List type for file_ids")
			return
		}

		var fileIDs []string
		fileIDsList.ElementsAs(ctx, &fileIDs, false)

		kbs = append(kbs, vapi.TQKnowledgeBase{
			Provider:    provider.ValueString(),
			Name:        name.ValueString(),
			Model:       model.ValueString(),
			Description: description.ValueString(),
			FileIDs:     fileIDs,
		})
	}

	request := vapi.ToolQueryFunctionRequest{
		Type: "query",
		Function: vapi.Function{
			Name:        data.Name.ValueString(),
			Description: data.Description.ValueString(),
		},
		KnowledgeBases: kbs,
	}

	resBody, status, err := r.client.CreateToolQueryFunction(request)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Error creating resource: %v", err))
		return
	}
	if status >= 300 {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("HTTP %d: %s", status, string(resBody)))
		return
	}

	var res vapi.ToolQueryFunctionResponse
	if err := json.Unmarshal(resBody, &res); err != nil {
		resp.Diagnostics.AddError("Response Error", fmt.Sprintf("Error parsing response: %v", err))
		return
	}

	data.ID = types.StringValue(res.ID)
	data.OrgID = types.StringValue(res.OrgID)
	data.Description = types.StringValue(res.Function.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPIToolQueryFunctionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPIToolQueryFunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resBody, status, err := r.client.GetToolQueryFunction(data.ID.ValueString())
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil || status >= 300 {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Error reading resource: %v", err))
		return
	}

	var res vapi.ToolQueryFunctionResponse
	if err := json.Unmarshal(resBody, &res); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Failed to parse response: %v", err))
		return
	}

	data.Name = types.StringValue(res.Function.Name)
	data.Description = types.StringValue(res.Function.Description)
	data.OrgID = types.StringValue(res.OrgID)

	var kbElements []attr.Value
	for _, kb := range res.KnowledgeBases {
		fileIDVals := make([]attr.Value, len(kb.FileIDs))
		for i, fid := range kb.FileIDs {
			fileIDVals[i] = types.StringValue(fid)
		}
		fileIDList, _ := types.ListValue(types.StringType, fileIDVals)

		kbObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"provider":    types.StringType,
				"name":        types.StringType,
				"model":       types.StringType,
				"description": types.StringType,
				"file_ids":    types.ListType{ElemType: types.StringType},
			},
			map[string]attr.Value{
				"provider":    types.StringValue(kb.Provider),
				"name":        types.StringValue(kb.Name),
				"model":       types.StringValue(kb.Model),
				"description": types.StringValue(kb.Description),
				"file_ids":    fileIDList,
			},
		)
		kbElements = append(kbElements, kbObj)
	}

	data.KnowledgeBases, _ = types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"provider":    types.StringType,
		"name":        types.StringType,
		"model":       types.StringType,
		"description": types.StringType,
		"file_ids":    types.ListType{ElemType: types.StringType},
	}}, kbElements)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPIToolQueryFunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state VAPIToolQueryFunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var kbs []vapi.TQKnowledgeBase
	for _, kbVal := range data.KnowledgeBases.Elements() {
		objVal, ok := kbVal.(types.Object)
		if !ok {
			resp.Diagnostics.AddError("Type Assertion Error", "Expected Object type for knowledge_base element")
			return
		}
		attrs := objVal.Attributes()

		provider, ok := attrs["provider"].(types.String)
		if !ok {
			resp.Diagnostics.AddError("Type Assertion Error", "Expected String type for provider")
			return
		}
		name, ok := attrs["name"].(types.String)
		if !ok {
			resp.Diagnostics.AddError("Type Assertion Error", "Expected String type for name")
			return
		}
		model, ok := attrs["model"].(types.String)
		if !ok {
			resp.Diagnostics.AddError("Type Assertion Error", "Expected String type for model")
			return
		}
		description, ok := attrs["description"].(types.String)
		if !ok {
			description = types.StringValue("")
		}
		fileIDsList, ok := attrs["file_ids"].(types.List)
		if !ok {
			resp.Diagnostics.AddError("Type Assertion Error", "Expected List type for file_ids")
			return
		}

		var fileIDs []string
		fileIDsList.ElementsAs(ctx, &fileIDs, false)

		kbs = append(kbs, vapi.TQKnowledgeBase{
			Provider:    provider.ValueString(),
			Name:        name.ValueString(),
			Model:       model.ValueString(),
			Description: description.ValueString(),
			FileIDs:     fileIDs,
		})
	}

	request := vapi.ToolQueryFunctionRequest{
		Function: vapi.Function{
			Name:        data.Name.ValueString(),
			Description: data.Description.ValueString(),
		},
		KnowledgeBases: kbs,
	}

	resBody, status, err := r.client.UpdateToolQueryFunction(state.ID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Error updating resource: %v", err))
		return
	}
	if status >= 300 {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("HTTP %d: %s", status, string(resBody)))
		return
	}

	var res vapi.ToolQueryFunctionResponse
	if err := json.Unmarshal(resBody, &res); err != nil {
		resp.Diagnostics.AddError("Response Error", fmt.Sprintf("Error parsing update response: %v", err))
		return
	}

	data.Description = types.StringValue(res.Function.Description)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPIToolQueryFunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPIToolQueryFunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeleteToolQueryFunction(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Error deleting resource: %v", err))
	}
}

func (r *VAPIToolQueryFunctionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
