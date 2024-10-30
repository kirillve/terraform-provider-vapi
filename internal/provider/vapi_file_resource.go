package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

var _ resource.Resource = &VAPIFileResource{}
var _ resource.ResourceWithImportState = &VAPIFileResource{}

func NewVAPIFileResource() resource.Resource {
	return &VAPIFileResource{}
}

type VAPIFileResource struct {
	client *vapi.APIClient
}

type VAPIFileResourceModel struct {
	Content      types.String `tfsdk:"content"`
	Filename     types.String `tfsdk:"filename"`
	Name         types.String `tfsdk:"name"`
	OriginalName types.String `tfsdk:"original_name"`
	Bytes        types.Int64  `tfsdk:"bytes"`
	Mimetype     types.String `tfsdk:"mimetype"`
	Path         types.String `tfsdk:"path"`
	URL          types.String `tfsdk:"url"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
	Id           types.String `tfsdk:"id"`
	OrgID        types.String `tfsdk:"org_id"`
	Status       types.String `tfsdk:"status"`
	Bucket       types.String `tfsdk:"bucket"`
	Purpose      types.String `tfsdk:"purpose"`
}

func (r *VAPIFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

func (r *VAPIFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a file resource in the VAPI system.",

		Attributes: map[string]schema.Attribute{
			"content": schema.StringAttribute{
				MarkdownDescription: "The file content to upload.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"filename": schema.StringAttribute{
				MarkdownDescription: "The filename for upload.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the file.",
				Computed:            true,
			},
			"original_name": schema.StringAttribute{
				MarkdownDescription: "The original name of the file.",
				Computed:            true,
			},
			"bytes": schema.Int64Attribute{
				MarkdownDescription: "The size of the file in bytes.",
				Computed:            true,
			},
			"mimetype": schema.StringAttribute{
				MarkdownDescription: "The MIME type of the file.",
				Computed:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "The path to the file.",
				Computed:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL to access the file.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the file was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the file was last updated.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The uploaded file status.",
				Computed:            true,
			},
			"bucket": schema.StringAttribute{
				MarkdownDescription: "The uploaded file bucket.",
				Computed:            true,
			},
			"purpose": schema.StringAttribute{
				MarkdownDescription: "The uploaded file purpose.",
				Computed:            true,
			},
			"org_id": schema.StringAttribute{
				MarkdownDescription: "The OrgId of the file.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the file.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *VAPIFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VAPIFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPIFileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, responseCode, err := r.client.UploadData("file", data.Filename.ValueString(), []byte(data.Content.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upload file: %s", err))
		return
	}

	var fileResponse vapi.FileResponse
	if responseCode >= 200 && responseCode < 300 {
		if err := json.Unmarshal(response, &fileResponse); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal response: %s", err))
			return
		}
	} else {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response [%d]: %s", responseCode, string(response)))
		return
	}

	bindVAPIFileResourceData(&data, &fileResponse)

	tflog.Trace(ctx, "created a file resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPIFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPIFileResourceModel
	// Retrieve the current state
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Attempt to fetch the file details from the remote API
	response, responseCode, err := r.client.GetFile(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read file: %s", err))
		return
	}

	// Check if the file was not found (404 or similar status code indicating missing resource)
	if responseCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	// Handle successful responses (e.g., 200 OK)
	var fileResponse vapi.FileResponse
	if responseCode >= 200 && responseCode < 300 {
		// Parse the response
		if err := json.Unmarshal(response, &fileResponse); err != nil {
			resp.Diagnostics.AddWarning("Parse Error", fmt.Sprintf("Unable to parse file response: %s", err))
		}
		// Bind the file response data to the resource model
		bindVAPIFileResourceData(&data, &fileResponse)
	} else {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response [%d]: %s", responseCode, string(response)))
		return
	}

	// Update the state with the latest data
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPIFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VAPIFileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeleteFile(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete file: %s", err))
		return
	}
	bindVAPIFileResourceData(&data, &vapi.FileResponse{})

	response, responseCode, err := r.client.UploadData("file", data.Filename.ValueString(), []byte(data.Content.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upload file: %s", err))
		return
	}

	var fileResponse vapi.FileResponse
	if responseCode >= 200 && responseCode < 300 {
		if err := json.Unmarshal(response, &fileResponse); err != nil {
			resp.Diagnostics.AddWarning("Parse Error", fmt.Sprintf("Unable to parse file response: %s", err))
		}
	} else {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response [%d]: %s", responseCode, string(response)))
		return
	}

	bindVAPIFileResourceData(&data, &fileResponse)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPIFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPIFileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeleteFile(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete file: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a file resource")
}

func (r *VAPIFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func bindVAPIFileResourceData(data *VAPIFileResourceModel, fileResponse *vapi.FileResponse) {
	data.Id = types.StringValue(fileResponse.ID)
	data.OrgID = types.StringValue(fileResponse.OrgID)
	data.Name = types.StringValue(fileResponse.Name)
	data.OriginalName = types.StringValue(fileResponse.OriginalName)
	data.Bytes = types.Int64Value(fileResponse.Bytes)
	data.Mimetype = types.StringValue(fileResponse.Mimetype)
	data.Path = types.StringValue(fileResponse.Path)
	data.URL = types.StringValue(fileResponse.URL)
	data.CreatedAt = types.StringValue(fileResponse.CreatedAt)
	data.UpdatedAt = types.StringValue(fileResponse.UpdatedAt)
	data.Status = types.StringValue(fileResponse.Status)
	data.Bucket = types.StringValue(fileResponse.Bucket)
	data.Purpose = types.StringValue(fileResponse.Purpose)
}
