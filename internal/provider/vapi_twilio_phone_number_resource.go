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

var _ resource.Resource = &VAPITwilioPhoneNumberResource{}
var _ resource.ResourceWithImportState = &VAPITwilioPhoneNumberResource{}

// NewVAPIPhoneNumberResource constructor.
func NewVAPIPhoneNumberResource() resource.Resource {
	return &VAPITwilioPhoneNumberResource{}
}

// VAPITwilioPhoneNumberResource struct.
type VAPITwilioPhoneNumberResource struct {
	client *vapi.APIClient
}

// VAPITwilioPhoneNumberResourceModel struct.
type VAPITwilioPhoneNumberResourceModel struct {
	ID               types.String `tfsdk:"id"`
	OrgID            types.String `tfsdk:"org_id"`
	Number           types.String `tfsdk:"number"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	TwilioAccountSid types.String `tfsdk:"twilio_account_sid"`
	TwilioAuthToken  types.String `tfsdk:"twilio_auth_token"`
	Name             types.String `tfsdk:"name"`
	PhoneProvider    types.String `tfsdk:"phone_provider"`
}

func (r *VAPITwilioPhoneNumberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_twilio_phone_number"
}

func (r *VAPITwilioPhoneNumberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a phone number resource in the VAPI system.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the phone number.",
				Required:            true,
			},
			"number": schema.StringAttribute{
				MarkdownDescription: "The phone number.",
				Required:            true,
			},
			"twilio_account_sid": schema.StringAttribute{
				MarkdownDescription: "The Twilio account SID.",
				Required:            true,
				Sensitive:           true,
			},
			"twilio_auth_token": schema.StringAttribute{
				MarkdownDescription: "The Twilio auth token.",
				Required:            true,
				Sensitive:           true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the phone number.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_id": schema.StringAttribute{
				MarkdownDescription: "The OrgID of the phone number.",
				Computed:            true,
			},
			"phone_provider": schema.StringAttribute{
				MarkdownDescription: "The provider of the phone number.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the phone number was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the phone number was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *VAPITwilioPhoneNumberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VAPITwilioPhoneNumberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPITwilioPhoneNumberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestData := vapi.ImportTwilioRequest{
		Provider:         "twilio",
		Name:             data.Name.ValueString(),
		Number:           data.Number.ValueString(),
		TwilioAccountSID: data.TwilioAccountSid.ValueString(),
		TwilioAuthToken:  data.TwilioAuthToken.ValueString(),
	}

	response, _, err := r.client.ImportTwilioPhoneNumber(requestData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create phone number: %s", err))
		return
	}

	var twilioPhoneNumberResp vapi.TwilioPhoneNumber
	if err := json.Unmarshal(response, &twilioPhoneNumberResp); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal response: %s", err))
		return
	}

	updateVAPIPhoneNumberResourceData(&data, &twilioPhoneNumberResp)
	tflog.Trace(ctx, "created a phone number resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPITwilioPhoneNumberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPITwilioPhoneNumberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, responseCode, err := r.client.GetPhoneNumber(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read phone number: %s", err))
		return
	}

	var phoneNumberResp vapi.TwilioPhoneNumber
	if responseCode >= 200 && responseCode < 300 {
		if err := json.Unmarshal(response, &phoneNumberResp); err != nil {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse phone number response: %s", err))
			return
		}
	}

	updateVAPIPhoneNumberResourceData(&data, &phoneNumberResp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPITwilioPhoneNumberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VAPITwilioPhoneNumberResourceModel
	var phoneNumberResp vapi.TwilioPhoneNumber

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, responseCode, err := r.client.GetPhoneNumber(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Update :: Unable to read phone number: %s", err))
		return
	}

	if responseCode >= 200 && responseCode < 300 {
		if err := json.Unmarshal(response, &phoneNumberResp); err != nil {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Update :: Unable to parse phone number response: %s", err))
			return
		}
	}

	if phoneNumberResp.Name != data.Name.String() ||
		phoneNumberResp.Number != data.Number.String() ||
		phoneNumberResp.TwilioAccountSid != data.TwilioAccountSid.String() ||
		phoneNumberResp.TwilioAuthToken != data.TwilioAuthToken.String() {
		_, _, err := r.client.DeletePhoneNumber(data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Update :: Unable to delete phone number: %s", err))
			return
		}

		requestData := vapi.ImportTwilioRequest{
			Provider:         "twilio",
			Name:             data.Name.ValueString(),
			Number:           data.Number.ValueString(),
			TwilioAccountSID: data.TwilioAccountSid.ValueString(),
			TwilioAuthToken:  data.TwilioAuthToken.ValueString(),
		}

		response, responseCode, err := r.client.ImportTwilioPhoneNumber(requestData)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Update :: Unable to create phone number: %s", err))
			return
		}

		if responseCode >= 200 && responseCode < 300 {
			if err := json.Unmarshal(response, &phoneNumberResp); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Update :: Unable to unmarshal response: %s", err))
				return
			}
		}
	}

	updateVAPIPhoneNumberResourceData(&data, &phoneNumberResp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPITwilioPhoneNumberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPITwilioPhoneNumberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeletePhoneNumber(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete phone number: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a phone number resource")
}

func (r *VAPITwilioPhoneNumberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func updateVAPIPhoneNumberResourceData(data *VAPITwilioPhoneNumberResourceModel, phoneNumberResp *vapi.TwilioPhoneNumber) {
	data.ID = types.StringValue(phoneNumberResp.ID)
	data.OrgID = types.StringValue(phoneNumberResp.OrgID)
	data.Name = types.StringValue(phoneNumberResp.Name)
	data.Number = types.StringValue(phoneNumberResp.Number)
	data.CreatedAt = types.StringValue(phoneNumberResp.CreatedAt)
	data.UpdatedAt = types.StringValue(phoneNumberResp.UpdatedAt)
	data.TwilioAccountSid = types.StringValue(phoneNumberResp.TwilioAccountSid)
	data.TwilioAuthToken = types.StringValue(phoneNumberResp.TwilioAuthToken)
	data.PhoneProvider = types.StringValue(phoneNumberResp.Provider)
}
