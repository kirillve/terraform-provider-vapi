package vapiv1

import (
	"context"
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

var (
	_ resource.Resource                = &VAPISIPTrunkPhoneNumberResource{}
	_ resource.ResourceWithImportState = &VAPISIPTrunkPhoneNumberResource{}
)

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
func (r *VAPISIPTrunkPhoneNumberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sip_trunk_phone_number"
}

// Schema defines the resource schema.
func (r *VAPISIPTrunkPhoneNumberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SIP Trunk (BYO) phone number resource in the VAPI system.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the phone number.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
func (r *VAPISIPTrunkPhoneNumberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

// Create creates the SIP trunk phone number.
func (r *VAPISIPTrunkPhoneNumberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPISIPTrunkPhoneNumberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sipResp vapi.ImportSIPTrunkPhoneNumberResponse
	if err := r.client.Do(ctx, http.MethodPost, "phone-number", data.expand(), &sipResp); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Unmarshal Error", fmt.Sprintf("Failed to parse API response: %s", err))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create SIP phone number: %s", err))
		return
	}

	data.flatten(&sipResp)
	tflog.Trace(ctx, "created a SIP trunk phone number resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read fetches the current state from the API. A phone number that no longer exists (HTTP 404)
// is removed from the state.
func (r *VAPISIPTrunkPhoneNumberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPISIPTrunkPhoneNumberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	var sipResp vapi.ImportSIPTrunkPhoneNumberResponse
	err := r.client.Do(ctx, http.MethodGet, "phone-number/"+id, nil, &sipResp)
	if vapi.StatusCode(err) == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Unmarshal Error", fmt.Sprintf("Failed to parse API response: %s", err))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read SIP phone number: %s", err))
		return
	}

	data.flatten(&sipResp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the SIP trunk phone number in place.
func (r *VAPISIPTrunkPhoneNumberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state VAPISIPTrunkPhoneNumberResourceModel
	var plan VAPISIPTrunkPhoneNumberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("API Error", "Unexpected response code [404]: ")
		return
	}

	var sipResp vapi.ImportSIPTrunkPhoneNumberResponse
	if err := r.client.Do(ctx, http.MethodPatch, "phone-number/"+id, plan.expand(), &sipResp); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Unmarshal Error", fmt.Sprintf("Failed to parse API response: %s", err))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update SIP phone number: %s", err))
		return
	}

	plan.flatten(&sipResp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the SIP trunk phone number. A phone number that no longer exists (HTTP 404)
// counts as deleted.
func (r *VAPISIPTrunkPhoneNumberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPISIPTrunkPhoneNumberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if id := data.ID.ValueString(); id != "" {
		if err := r.client.Do(ctx, http.MethodDelete, "phone-number/"+id, nil, nil); err != nil && vapi.StatusCode(err) != http.StatusNotFound {
			resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to delete SIP phone number: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "deleted SIP trunk phone number resource")
}

// ImportState enables `terraform import`.
func (r *VAPISIPTrunkPhoneNumberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// expand builds the API request from the planned model.
func (m *VAPISIPTrunkPhoneNumberResourceModel) expand() vapi.ImportSIPTrunkPhoneNumberRequest {
	return vapi.ImportSIPTrunkPhoneNumberRequest{
		Provider:               "byo-phone-number",
		Name:                   m.Name.ValueString(),
		Number:                 m.Number.ValueString(),
		CredentialID:           m.CredentialID.ValueString(),
		NumberE164CheckEnabled: m.NumberE164CheckEnabled.ValueBool(),
	}
}

// flatten binds the API response to the model.
func (m *VAPISIPTrunkPhoneNumberResourceModel) flatten(in *vapi.ImportSIPTrunkPhoneNumberResponse) {
	m.ID = types.StringValue(in.ID)
	m.OrgID = types.StringValue(in.OrgID)
	m.Number = types.StringValue(in.Number)
	m.Name = types.StringValue(in.Name)
	m.PhoneProvider = types.StringValue(in.Provider)
	m.CreatedAt = types.StringValue(in.CreatedAt)
	m.UpdatedAt = types.StringValue(in.UpdatedAt)
	m.CredentialID = types.StringValue(in.CredentialID)
	m.NumberE164CheckEnabled = types.BoolValue(in.NumberE164CheckEnabled)
}
