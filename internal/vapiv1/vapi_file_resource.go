package vapiv1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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

// NewVAPIFileResource returns a new file resource.
func NewVAPIFileResource() resource.Resource {
	return &VAPIFileResource{}
}

// VAPIFileResource manages an uploaded file in VAPI.
type VAPIFileResource struct {
	client *vapi.APIClient
}

// VAPIFileResourceModel maps the schema data.
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
	ID           types.String `tfsdk:"id"`
	OrgID        types.String `tfsdk:"org_id"`
	Status       types.String `tfsdk:"status"`
	Bucket       types.String `tfsdk:"bucket"`
	Purpose      types.String `tfsdk:"purpose"`
}

// Metadata sets the resource type name.
func (r *VAPIFileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

// Schema defines the resource schema.
func (r *VAPIFileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

// Configure binds the API client to the resource.
func (r *VAPIFileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

// Create uploads the file.
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

	data.flatten(&fileResponse)

	tflog.Trace(ctx, "created a file resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read fetches the current state from the API. A response that cannot be decoded is reported
// as a warning and the partially decoded value is still bound.
func (r *VAPIFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPIFileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	var fileResponse vapi.FileResponse
	if err := r.client.Do(http.MethodGet, "file/"+id, nil, &fileResponse); err != nil {
		var decErr *vapi.DecodeError
		if !errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read file: %s", err))
			return
		}
		resp.Diagnostics.AddWarning("Parse Error", fmt.Sprintf("Unable to parse file response: %s", err))
	}

	data.flatten(&fileResponse)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update deletes the old file, resets the computed attributes and uploads the new content.
func (r *VAPIFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VAPIFileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if id := data.ID.ValueString(); id != "" {
		if err := r.client.Do(http.MethodDelete, "file/"+id, nil, nil); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete file: %s", err))
			return
		}
	}
	data.flatten(&vapi.FileResponse{})

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

	data.flatten(&fileResponse)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the file.
func (r *VAPIFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPIFileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if id := data.ID.ValueString(); id != "" {
		if err := r.client.Do(http.MethodDelete, "file/"+id, nil, nil); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete file: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "deleted a file resource")
}

// ImportState enables `terraform import`.
func (r *VAPIFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// flatten binds the API response to the model. An empty response resets the computed attributes.
func (m *VAPIFileResourceModel) flatten(in *vapi.FileResponse) {
	m.ID = types.StringValue(in.ID)
	m.OrgID = types.StringValue(in.OrgID)
	m.Name = types.StringValue(in.Name)
	m.OriginalName = types.StringValue(in.OriginalName)
	m.Bytes = types.Int64Value(in.Bytes)
	m.Mimetype = types.StringValue(in.Mimetype)
	m.Path = types.StringValue(in.Path)
	m.URL = types.StringValue(in.URL)
	m.CreatedAt = types.StringValue(in.CreatedAt)
	m.UpdatedAt = types.StringValue(in.UpdatedAt)
	m.Status = types.StringValue(in.Status)
	m.Bucket = types.StringValue(in.Bucket)
	m.Purpose = types.StringValue(in.Purpose)
}
