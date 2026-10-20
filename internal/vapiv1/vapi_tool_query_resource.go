package vapiv1

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// NewVAPIToolQueryFunctionResource returns a new tool query function resource.
func NewVAPIToolQueryFunctionResource() resource.Resource {
	return &VAPIToolQueryFunctionResource{}
}

// VAPIToolQueryFunctionResource manages a query tool in VAPI.
type VAPIToolQueryFunctionResource struct {
	client *vapi.APIClient
}

// KnowledgeBase is one knowledge base entry of a query tool.
type KnowledgeBase struct {
	Provider    types.String `tfsdk:"provider"`
	Name        types.String `tfsdk:"name"`
	Model       types.String `tfsdk:"model"`
	Description types.String `tfsdk:"description"`
	FileIDs     types.List   `tfsdk:"file_ids"`
}

// VAPIToolQueryFunctionResourceModel maps the schema data.
type VAPIToolQueryFunctionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrgID          types.String `tfsdk:"org_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	KnowledgeBases types.List   `tfsdk:"knowledge_bases"`
}

// Metadata sets the resource type name.
func (r *VAPIToolQueryFunctionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool_query_function"
}

// Schema defines the resource schema.
func (r *VAPIToolQueryFunctionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

// Configure binds the API client to the resource.
func (r *VAPIToolQueryFunctionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

// Create creates the query tool.
func (r *VAPIToolQueryFunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPIToolQueryFunctionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request, diags := data.expand(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	request.Type = "query"

	var res vapi.ToolQueryFunctionResponse
	if err := r.client.Do(http.MethodPost, "tool", request, &res); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Response Error", fmt.Sprintf("Error parsing response: %v", err))
			return
		}
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Error creating resource: %v", err))
		return
	}

	data.ID = types.StringValue(res.ID)
	data.OrgID = types.StringValue(res.OrgID)
	data.flattenDescription(&res)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read fetches the current state from the API.
func (r *VAPIToolQueryFunctionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPIToolQueryFunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	var res vapi.ToolQueryFunctionResponse
	err := r.client.Do(http.MethodGet, "tool/"+id, nil, &res)
	if vapi.StatusCode(err) == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Failed to parse response: %v", err))
			return
		}
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Error reading resource: %v", err))
		return
	}

	data.Name = types.StringValue(res.Function.Name)
	data.flattenDescription(&res)
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

// Update updates the query tool in place.
func (r *VAPIToolQueryFunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state VAPIToolQueryFunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request, diags := data.expand(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("API Error", "HTTP 404: ")
		return
	}

	var res vapi.ToolQueryFunctionResponse
	if err := r.client.Do(http.MethodPatch, "tool/"+id, request, &res); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Response Error", fmt.Sprintf("Error parsing update response: %v", err))
			return
		}
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Error updating resource: %v", err))
		return
	}

	data.flattenDescription(&res)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the query tool.
func (r *VAPIToolQueryFunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPIToolQueryFunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		return
	}

	if err := r.client.Do(http.MethodDelete, "tool/"+id, nil, nil); err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Error deleting resource: %v", err))
	}
}

// ImportState enables `terraform import`.
func (r *VAPIToolQueryFunctionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// expand builds the API request (without Type) from the planned model, reporting a
// wrongly typed knowledge base element as an error diagnostic.
func (m *VAPIToolQueryFunctionResourceModel) expand(ctx context.Context) (vapi.ToolQueryFunctionRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	var kbs []vapi.TQKnowledgeBase
	for _, kbVal := range m.KnowledgeBases.Elements() {
		objVal, ok := kbVal.(types.Object)
		if !ok {
			diags.AddError("Type Assertion Error", "Expected Object type for knowledge_base element")
			return vapi.ToolQueryFunctionRequest{}, diags
		}
		attrs := objVal.Attributes()

		provider, ok := attrs["provider"].(types.String)
		if !ok {
			diags.AddError("Type Assertion Error", "Expected String type for provider")
			return vapi.ToolQueryFunctionRequest{}, diags
		}
		name, ok := attrs["name"].(types.String)
		if !ok {
			diags.AddError("Type Assertion Error", "Expected String type for name")
			return vapi.ToolQueryFunctionRequest{}, diags
		}
		model, ok := attrs["model"].(types.String)
		if !ok {
			diags.AddError("Type Assertion Error", "Expected String type for model")
			return vapi.ToolQueryFunctionRequest{}, diags
		}
		description, ok := attrs["description"].(types.String)
		if !ok {
			description = types.StringValue("") // fallback if optional
		}
		fileIDsList, ok := attrs["file_ids"].(types.List)
		if !ok {
			diags.AddError("Type Assertion Error", "Expected List type for file_ids")
			return vapi.ToolQueryFunctionRequest{}, diags
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

	return vapi.ToolQueryFunctionRequest{
		Function: vapi.Function{
			Name:        m.Name.ValueString(),
			Description: m.Description.ValueString(),
		},
		KnowledgeBases: kbs,
	}, diags
}

// flattenDescription binds the response description to the model.
func (m *VAPIToolQueryFunctionResourceModel) flattenDescription(in *vapi.ToolQueryFunctionResponse) {
	m.Description = types.StringValue(in.Function.Description)
}
