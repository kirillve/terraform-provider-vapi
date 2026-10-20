package vapiv1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

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
	ID                       types.String `tfsdk:"id"`
	OrgID                    types.String `tfsdk:"org_id"`
	Number                   types.String `tfsdk:"number"`
	CreatedAt                types.String `tfsdk:"created_at"`
	UpdatedAt                types.String `tfsdk:"updated_at"`
	TwilioAccountSid         types.String `tfsdk:"twilio_account_sid"`
	TwilioAuthToken          types.String `tfsdk:"twilio_auth_token"`
	Name                     types.String `tfsdk:"name"`
	PhoneProvider            types.String `tfsdk:"phone_provider"`
	FallbackType             types.String `tfsdk:"fallback_destination_type"`
	FallbackE164CheckEnabled types.String `tfsdk:"fallback_destination_number_e164_check_enabled"`
	FallbackNumber           types.String `tfsdk:"fallback_destination_number"`
	FallbackExtension        types.String `tfsdk:"fallback_destination_extension"`
	FallbackMessage          types.String `tfsdk:"fallback_destination_message"`
	FallbackDescription      types.String `tfsdk:"fallback_destination_description"`
	AssistantID              types.String `tfsdk:"assistant_id"`
	SmsEnabled               types.Bool   `tfsdk:"sms_enabled"`
	ServerURL                types.String `tfsdk:"server_url"`
}

// Metadata sets the resource type name.
func (r *VAPITwilioPhoneNumberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_twilio_phone_number"
}

// Schema defines the resource schema.
func (r *VAPITwilioPhoneNumberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
					stringplanmodifier.UseStateForUnknown(),
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
			"fallback_destination_type": schema.StringAttribute{
				MarkdownDescription: "The FallbackDestination Type.",
				Optional:            true,
			},
			"fallback_destination_number_e164_check_enabled": schema.StringAttribute{
				MarkdownDescription: "The FallbackDestination E164 check.",
				Optional:            true,
			},
			"fallback_destination_number": schema.StringAttribute{
				MarkdownDescription: "The FallbackDestination Number.",
				Optional:            true,
			},
			"fallback_destination_extension": schema.StringAttribute{
				MarkdownDescription: "The FallbackDestination Extension.",
				Optional:            true,
			},
			"fallback_destination_message": schema.StringAttribute{
				MarkdownDescription: "The FallbackDestination Message.",
				Optional:            true,
			},
			"fallback_destination_description": schema.StringAttribute{
				MarkdownDescription: "The FallbackDestination Description.",
				Optional:            true,
			},
			"sms_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether VAPI manages SMS for this Twilio number. Set false to disable VAPI SMS and avoid overwriting the Twilio messaging webhook. When omitted, VAPI uses its default on import; removing this attribute stops managing SMS without resetting the remote setting. Disabling SMS does not restore a previously overwritten webhook.",
				Optional:            true,
			},
			"server_url": schema.StringAttribute{
				MarkdownDescription: "URL of the server that receives webhook messages for calls on this phone number (server.url). Takes precedence over the assistant and org server URLs. When omitted, the number has no server of its own; removing this attribute stops managing it without clearing the remote setting.",
				Optional:            true,
			},
			"assistant_id": schema.StringAttribute{
				MarkdownDescription: "This is the assistant that will be used for incoming calls to this phone number.",
				Optional:            true,
			},
		},
	}
}

// Configure binds the API client to the resource.
func (r *VAPITwilioPhoneNumberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

// Create imports the Twilio phone number.
func (r *VAPITwilioPhoneNumberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPITwilioPhoneNumberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var twilioPhoneNumberResp vapi.TwilioPhoneNumber
	if err := r.client.Do(http.MethodPost, "phone-number", data.expand(), &twilioPhoneNumberResp); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal response: %s", err))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create phone number: %s", err))
		return
	}

	data.flatten(&twilioPhoneNumberResp)
	tflog.Trace(ctx, "created a phone number resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read fetches the current state from the API.
func (r *VAPITwilioPhoneNumberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPITwilioPhoneNumberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	var phoneNumberResp vapi.TwilioPhoneNumber
	if err := r.client.Do(http.MethodGet, "phone-number/"+id, nil, &phoneNumberResp); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse phone number response: %s", err))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read phone number: %s", err))
		return
	}

	data.flatten(&phoneNumberResp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the Twilio phone number in place.
func (r *VAPITwilioPhoneNumberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state VAPITwilioPhoneNumberResourceModel
	var plan VAPITwilioPhoneNumberResourceModel

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

	var phoneNumberResp vapi.TwilioPhoneNumber
	if err := r.client.Do(http.MethodPatch, "phone-number/"+id, plan.expand(), &phoneNumberResp); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Update :: Unable to unmarshal response: %s", err))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update phone number: %s", err))
		return
	}

	plan.flatten(&phoneNumberResp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the Twilio phone number.
func (r *VAPITwilioPhoneNumberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPITwilioPhoneNumberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if id := data.ID.ValueString(); id != "" {
		if err := r.client.Do(http.MethodDelete, "phone-number/"+id, nil, nil); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete phone number: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "deleted a phone number resource")
}

// ImportState enables `terraform import`.
func (r *VAPITwilioPhoneNumberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// expand builds the API request from the planned model. The fallback destination is always
// sent; server is omitted when server_url is unmanaged.
func (m *VAPITwilioPhoneNumberResourceModel) expand() vapi.ImportTwilioRequest {
	out := vapi.ImportTwilioRequest{
		Provider:         "twilio",
		SmsEnabled:       m.SmsEnabled.ValueBoolPointer(),
		Name:             m.Name.ValueString(),
		Number:           m.Number.ValueString(),
		TwilioAccountSID: m.TwilioAccountSid.ValueString(),
		TwilioAuthToken:  m.TwilioAuthToken.ValueString(),
		AssistantID:      m.AssistantID.ValueString(),
		Fallback: &vapi.FallbackDestination{
			Type:                   m.FallbackType.ValueString(),
			NumberE164CheckEnabled: m.FallbackE164CheckEnabled.ValueString() == "true",
			Number:                 m.FallbackNumber.ValueString(),
			Extension:              m.FallbackExtension.ValueString(),
			Message:                m.FallbackMessage.ValueString(),
			Description:            m.FallbackDescription.ValueString(),
		},
	}
	if !m.ServerURL.IsNull() && !m.ServerURL.IsUnknown() {
		out.Server = &vapi.Server{URL: m.ServerURL.ValueString()}
	}
	return out
}

// flatten binds the API response to the model. sms_enabled and server_url are refreshed only
// when they are managed and the API returned a value.
func (m *VAPITwilioPhoneNumberResourceModel) flatten(in *vapi.TwilioPhoneNumber) {
	if !m.SmsEnabled.IsNull() && in.SmsEnabled != nil {
		m.SmsEnabled = types.BoolValue(*in.SmsEnabled)
	}
	if !m.ServerURL.IsNull() && in.Server != nil && in.Server.URL != "" {
		m.ServerURL = types.StringValue(in.Server.URL)
	}
	m.ID = types.StringValue(in.ID)
	m.OrgID = types.StringValue(in.OrgID)
	m.Name = types.StringValue(in.Name)
	m.CreatedAt = types.StringValue(in.CreatedAt)
	m.UpdatedAt = types.StringValue(in.UpdatedAt)
	m.PhoneProvider = types.StringValue(in.Provider)

	if fb := in.Fallback; fb != nil {
		m.FallbackType = types.StringValue(fb.Type)
		m.FallbackE164CheckEnabled = types.StringValue(strconv.FormatBool(fb.NumberE164CheckEnabled))
		m.FallbackNumber = types.StringValue(fb.Number)
		m.FallbackExtension = types.StringValue(fb.Extension)
		m.FallbackMessage = types.StringValue(fb.Message)
		m.FallbackDescription = types.StringValue(fb.Description)
	}
}
