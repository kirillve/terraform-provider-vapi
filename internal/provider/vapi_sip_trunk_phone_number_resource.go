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

var _ resource.Resource = &VAPISIPTrunkPhoneNumberResource{}
var _ resource.ResourceWithImportState = &VAPISIPTrunkPhoneNumberResource{}

// NewVAPISIPTrunkPhoneNumberResource returns a new SIP trunk phone number resource.
func NewVAPISIPTrunkPhoneNumberResource() resource.Resource {
	return &VAPISIPTrunkPhoneNumberResource{}
}

// VAPISIPTrunkPhoneNumberResource manages a SIP Trunk phone number resource.
type VAPISIPTrunkPhoneNumberResource struct {
	client *vapi.APIClient
}

// VAPISIPTrunkPhoneNumberResourceModel maps the schema data.
type VAPISIPTrunkPhoneNumberResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	OrgID                  types.String `tfsdk:"org_id"`
	Number                 types.String `tfsdk:"number"`
	Name                   types.String `tfsdk:"name"`
	PhoneProvider          types.String `tfsdk:"phone_provider"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
	CredentialID           types.String `tfsdk:"credential_id"`
	NumberE164CheckEnabled types.Bool   `tfsdk:"number_e164_check_enabled"`
}

// Metadata sets the resource type name.
func (r *VAPISIPTrunkPhoneNumberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sip_trunk_phone_number"
}

// Schema defines the resource schema.
func (r *VAPISIPTrunkPhoneNumberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SIP Trunk (BYO) phone number resource in the VAPI system.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the phone number.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_id": schema.StringAttribute{
				MarkdownDescription: "The OrgID of the phone number.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the phone number.",
				Required:            true,
			},
			"number": schema.StringAttribute{
				MarkdownDescription: "The phone number in E.164 format.",
				Required:            true,
			},
			"credential_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the SIP credential to use.",
				Required:            true,
			},
			"number_e164_check_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether to enforce E.164 validation on the number.",
				Required:            true,
			},
			"phone_provider": schema.StringAttribute{
				MarkdownDescription: "The provider of the phone number.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The creation timestamp.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The last update timestamp.",
				Computed:            true,
			},
		},
	}
}

// Configure binds the API client to the resource.
func (r *VAPISIPTrunkPhoneNumberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*vapi.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *vapi.APIClient, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the SIP trunk phone number.
func (r *VAPISIPTrunkPhoneNumberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPISIPTrunkPhoneNumberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestData := vapi.ImportSIPTrunkPhoneNumberRequest{
		Provider:               "byo-phone-number",
		Name:                   data.Name.ValueString(),
		Number:                 data.Number.ValueString(),
		CredentialID:           data.CredentialID.ValueString(),
		NumberE164CheckEnabled: data.NumberE164CheckEnabled.ValueBool(),
	}

	response, responseCode, err := r.client.ImportSIPTrunkPhoneNumber(requestData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create SIP phone number: %s", err))
		return
	}

	if responseCode < 200 || responseCode >= 300 {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected response code [%d]: %s", responseCode, string(response)))
		return
	}

	var sipResp vapi.ImportSIPTrunkPhoneNumberResponse
	if err := json.Unmarshal(response, &sipResp); err != nil {
		resp.Diagnostics.AddError("Unmarshal Error", fmt.Sprintf("Failed to parse API response: %s", err))
		return
	}

	bindSIPPhoneNumberResponse(&data, &sipResp)
	tflog.Trace(ctx, "created a SIP trunk phone number resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read fetches the current state from the API.
func (r *VAPISIPTrunkPhoneNumberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPISIPTrunkPhoneNumberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, responseCode, err := r.client.GetPhoneNumber(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read SIP phone number: %s", err))
		return
	}

	if responseCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if responseCode < 200 || responseCode >= 300 {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected response code [%d]: %s", responseCode, string(response)))
		return
	}

	var sipResp vapi.ImportSIPTrunkPhoneNumberResponse
	if err := json.Unmarshal(response, &sipResp); err != nil {
		resp.Diagnostics.AddError("Unmarshal Error", fmt.Sprintf("Failed to parse API response: %s", err))
		return
	}

	bindSIPPhoneNumberResponse(&data, &sipResp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update deletes and recreates the SIP trunk phone number.
func (r *VAPISIPTrunkPhoneNumberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VAPISIPTrunkPhoneNumberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeletePhoneNumber(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to delete SIP phone number: %s", err))
		return
	}

	requestData := vapi.ImportSIPTrunkPhoneNumberRequest{
		Provider:               "byo-phone-number",
		Name:                   data.Name.ValueString(),
		Number:                 data.Number.ValueString(),
		CredentialID:           data.CredentialID.ValueString(),
		NumberE164CheckEnabled: data.NumberE164CheckEnabled.ValueBool(),
	}

	response, responseCode, err := r.client.ImportSIPTrunkPhoneNumber(requestData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update SIP phone number: %s", err))
		return
	}

	if responseCode < 200 || responseCode >= 300 {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected response code [%d]: %s", responseCode, string(response)))
		return
	}

	var sipResp vapi.ImportSIPTrunkPhoneNumberResponse
	if err := json.Unmarshal(response, &sipResp); err != nil {
		resp.Diagnostics.AddError("Unmarshal Error", fmt.Sprintf("Failed to parse API response: %s", err))
		return
	}

	bindSIPPhoneNumberResponse(&data, &sipResp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the SIP trunk phone number.
func (r *VAPISIPTrunkPhoneNumberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPISIPTrunkPhoneNumberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeletePhoneNumber(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to delete SIP phone number: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted SIP trunk phone number resource")
}

// ImportState enables `terraform import`.
func (r *VAPISIPTrunkPhoneNumberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper function to bind API response to Terraform model.
func bindSIPPhoneNumberResponse(data *VAPISIPTrunkPhoneNumberResourceModel, resp *vapi.ImportSIPTrunkPhoneNumberResponse) {
	data.ID = types.StringValue(resp.ID)
	data.OrgID = types.StringValue(resp.OrgID)
	data.Number = types.StringValue(resp.Number)
	data.Name = types.StringValue(resp.Name)
	data.PhoneProvider = types.StringValue(resp.Provider)
	data.CreatedAt = types.StringValue(resp.CreatedAt)
	data.UpdatedAt = types.StringValue(resp.UpdatedAt)
	data.CredentialID = types.StringValue(resp.CredentialID)
	data.NumberE164CheckEnabled = types.BoolValue(resp.NumberE164CheckEnabled)
}
