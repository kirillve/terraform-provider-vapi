package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"os"
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
	FilePath     types.String `tfsdk:"file_path"`
	Name         types.String `tfsdk:"name"`
	OriginalName types.String `tfsdk:"original_name"`
	Bytes        types.Int64  `tfsdk:"bytes"`
	Mimetype     types.String `tfsdk:"mimetype"`
	Path         types.String `tfsdk:"path"`
	URL          types.String `tfsdk:"url"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
	Checksum     types.String `tfsdk:"checksum"`
	Id           types.String `tfsdk:"id"`
}

func (r *VAPIFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

func (r *VAPIFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a file resource in the VAPI system.",

		Attributes: map[string]schema.Attribute{
			"file_path": schema.StringAttribute{
				MarkdownDescription: "The local path of the file to upload.",
				Required:            true,
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
			"checksum": schema.StringAttribute{
				MarkdownDescription: "The SHA-256 checksum of the file.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the file.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

	checksum, err := computeChecksum(data.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("File Error", fmt.Sprintf("Unable to compute checksum: %s", err))
		return
	}

	response, err := r.client.UploadFile("file", data.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upload file: %s", err))
		return
	}

	var fileResponse vapi.FileResponse
	if err := json.Unmarshal(response, &fileResponse); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal response: %s", err))
		return
	}

	data.Id = types.StringValue(fileResponse.ID)
	updateVAPIFileResourceData(&data, &fileResponse)
	data.Checksum = types.StringValue(checksum)

	tflog.Trace(ctx, "created a file resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPIFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPIFileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.GetFile(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read file: %s", err))
		return
	}

	var fileResponse vapi.FileResponse
	if len(response) > 0 {
		if err := json.Unmarshal(response, &fileResponse); err != nil {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse file response: %s", err))
			return
		}
	}

	updateVAPIFileResourceData(&data, &fileResponse)

	checksum, err := computeChecksum(data.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("File Error", fmt.Sprintf("Unable to compute checksum: %s", err))
		return
	}
	data.Checksum = types.StringValue(checksum)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPIFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VAPIFileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	localChecksum, err := computeChecksum(data.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("File Error", fmt.Sprintf("Unable to compute checksum: %s", err))
		return
	}

	if localChecksum != data.Checksum.ValueString() {
		_, err := r.client.DeleteFile(data.Id.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete old file: %s", err))
			return
		}

		response, err := r.client.UploadFile("file", data.FilePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upload file: %s", err))
			return
		}

		var fileResponse vapi.FileResponse
		if err := json.Unmarshal(response, &fileResponse); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal response: %s", err))
			return
		}

		data.Id = types.StringValue(fileResponse.ID)
		updateVAPIFileResourceData(&data, &fileResponse)
		data.Checksum = types.StringValue(localChecksum)
	}

	data.Checksum = types.StringValue(localChecksum)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPIFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPIFileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.SendRequest("DELETE", "file/"+data.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete file: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a file resource")
}

func (r *VAPIFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func computeChecksum(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %s", filePath, err)
	}
	checksum := sha256.Sum256(content)
	return hex.EncodeToString(checksum[:]), nil
}

func updateVAPIFileResourceData(data *VAPIFileResourceModel, fileResponse *vapi.FileResponse) {
	data.Name = types.StringValue(fileResponse.Name)
	data.OriginalName = types.StringValue(fileResponse.OriginalName)
	data.Bytes = types.Int64Value(fileResponse.Bytes)
	data.Mimetype = types.StringValue(fileResponse.Mimetype)
	data.Path = types.StringValue(fileResponse.Path)
	data.URL = types.StringValue(fileResponse.URL)
	data.CreatedAt = types.StringValue(fileResponse.CreatedAt)
	data.UpdatedAt = types.StringValue(fileResponse.UpdatedAt)
}
