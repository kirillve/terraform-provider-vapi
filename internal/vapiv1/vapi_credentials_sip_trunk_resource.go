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
	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

var _ resource.Resource = &VAPISIPTrunkResource{}
var _ resource.ResourceWithImportState = &VAPISIPTrunkResource{}

// NewVAPISIPTrunkResource returns a new SIP trunk resource.
func NewVAPISIPTrunkResource() resource.Resource {
	return &VAPISIPTrunkResource{}
}

// VAPISIPTrunkResource manages a SIP trunk credential.
type VAPISIPTrunkResource struct {
	client *vapi.APIClient
}

// VAPISIPTrunkResourceModel maps the schema data.
type VAPISIPTrunkResourceModel struct {
	ID                         types.String                     `tfsdk:"id"`
	SIPProvider                types.String                     `tfsdk:"sip_provider"`
	Name                       types.String                     `tfsdk:"name"`
	Gateways                   []SIPGatewayModel                `tfsdk:"gateways"`
	OutboundAuthenticationPlan *OutboundAuthenticationPlanModel `tfsdk:"outbound_authentication_plan"`
	OutboundLeadingPlusEnabled types.Bool                       `tfsdk:"outbound_leading_plus_enabled"`
	TechPrefix                 types.String                     `tfsdk:"tech_prefix"`
	SIPDiversionHeader         types.String                     `tfsdk:"sip_diversion_header"`
}

// SIPGatewayModel is one SIP gateway of a trunk.
type SIPGatewayModel struct {
	IP types.String `tfsdk:"ip"`
}

// OutboundAuthenticationPlanModel holds the outbound authentication settings of a trunk.
type OutboundAuthenticationPlanModel struct {
	AuthUsername    types.String          `tfsdk:"auth_username"`
	AuthPassword    types.String          `tfsdk:"auth_password"`
	SIPRegisterPlan *SIPRegisterPlanModel `tfsdk:"sip_register_plan"`
}

// SIPRegisterPlanModel holds the SIP registration settings of an outbound authentication plan.
type SIPRegisterPlanModel struct {
	Domain   types.String `tfsdk:"domain"`
	Username types.String `tfsdk:"username"`
	Realm    types.String `tfsdk:"realm"`
}

// Metadata sets the resource type name.
func (r *VAPISIPTrunkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sip_trunk"
}

// Schema defines the resource schema.
func (r *VAPISIPTrunkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SIP trunk in the VAPI system.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sip_provider": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The SIP trunk provider identifier (e.g., byo-sip-trunk).",
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"gateways": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
			"outbound_authentication_plan": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"auth_username": schema.StringAttribute{Required: true},
					"auth_password": schema.StringAttribute{Required: true},
					"sip_register_plan": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"domain":   schema.StringAttribute{Required: true},
							"username": schema.StringAttribute{Required: true},
							"realm":    schema.StringAttribute{Required: true},
						},
					},
				},
			},
			"outbound_leading_plus_enabled": schema.BoolAttribute{
				Required: true,
			},
			"tech_prefix": schema.StringAttribute{
				Optional: true,
			},
			"sip_diversion_header": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

// Configure binds the API client to the resource.
func (r *VAPISIPTrunkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

// Create creates the SIP trunk.
func (r *VAPISIPTrunkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPISIPTrunkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created vapi.SIPTrunk
	if err := r.client.Do(http.MethodPost, "credential", data.expand(), &created); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Unmarshal Error", err.Error())
			return
		}
		resp.Diagnostics.AddError("Create Failed", fmt.Sprintf("Status: %d, Error: %v", vapi.StatusCode(err), err))
		return
	}

	data.flatten(&created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read fetches the current state from the API.
func (r *VAPISIPTrunkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPISIPTrunkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	var fetched vapi.SIPTrunk
	err := r.client.Do(http.MethodGet, "credential/"+id, nil, &fetched)
	if vapi.StatusCode(err) == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddWarning("Unmarshal Warning", err.Error())
			return
		}
		resp.Diagnostics.AddError("Read Failed", fmt.Sprintf("Status: %d, Error: %v", vapi.StatusCode(err), err))
		return
	}

	data.flatten(&fetched)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the SIP trunk in place.
func (r *VAPISIPTrunkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state VAPISIPTrunkResourceModel
	var plan VAPISIPTrunkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Update Failed", "Status: 404, Error: ID cannot be empty")
		return
	}

	var updated vapi.SIPTrunk
	if err := r.client.Do(http.MethodPatch, "credential/"+id, plan.expand(), &updated); err != nil {
		var decErr *vapi.DecodeError
		if errors.As(err, &decErr) {
			resp.Diagnostics.AddError("Unmarshal Error", err.Error())
			return
		}
		resp.Diagnostics.AddError("Update Failed", fmt.Sprintf("Status: %d, Error: %v", vapi.StatusCode(err), err))
		return
	}

	plan.flatten(&updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the SIP trunk. A trunk that no longer exists (HTTP 404) counts as deleted.
func (r *VAPISIPTrunkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPISIPTrunkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Delete Failed", "Status: 404, Error: ID cannot be empty")
		return
	}

	if err := r.client.Do(http.MethodDelete, "credential/"+id, nil, nil); err != nil && vapi.StatusCode(err) != http.StatusNotFound {
		resp.Diagnostics.AddError("Delete Failed", fmt.Sprintf("Status: %d, Error: %v", vapi.StatusCode(err), err))
	}
}

// ImportState enables `terraform import`.
func (r *VAPISIPTrunkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// expand builds the API request from the planned model.
func (m *VAPISIPTrunkResourceModel) expand() vapi.ImportSIPTrunkRequest {
	out := vapi.ImportSIPTrunkRequest{
		Provider:                   m.SIPProvider.ValueString(),
		Name:                       m.Name.ValueString(),
		OutboundLeadingPlusEnabled: m.OutboundLeadingPlusEnabled.ValueBool(),
		TechPrefix:                 m.TechPrefix.ValueString(),
		SIPDiversionHeader:         m.SIPDiversionHeader.ValueString(),
	}
	for _, g := range m.Gateways {
		out.Gateways = append(out.Gateways, vapi.SIPGateway{IP: g.IP.ValueString()})
	}
	if plan := m.OutboundAuthenticationPlan; plan != nil {
		out.OutboundAuthenticationPlan = &vapi.OutboundAuthenticationPlan{
			AuthUsername: plan.AuthUsername.ValueString(),
			AuthPassword: plan.AuthPassword.ValueString(),
		}
		if reg := plan.SIPRegisterPlan; reg != nil {
			out.OutboundAuthenticationPlan.SIPRegisterPlan = &vapi.SIPRegisterPlan{
				Domain:   reg.Domain.ValueString(),
				Username: reg.Username.ValueString(),
				Realm:    reg.Realm.ValueString(),
			}
		}
	}
	return out
}

// flatten binds the API response to the model.
func (m *VAPISIPTrunkResourceModel) flatten(in *vapi.SIPTrunk) {
	m.ID = types.StringValue(in.ID)
	m.SIPProvider = types.StringValue(in.Provider)
	m.Name = types.StringValue(in.Name)
	m.OutboundLeadingPlusEnabled = types.BoolValue(in.OutboundLeadingPlusEnabled)
	m.TechPrefix = optionalString(m.TechPrefix, in.TechPrefix)
	m.SIPDiversionHeader = optionalString(m.SIPDiversionHeader, in.SIPDiversionHeader)

	m.Gateways = nil
	for _, g := range in.Gateways {
		m.Gateways = append(m.Gateways, SIPGatewayModel{IP: types.StringValue(g.IP)})
	}

	prior := m.OutboundAuthenticationPlan
	m.OutboundAuthenticationPlan = nil
	if plan := in.OutboundAuthenticationPlan; plan != nil {
		var priorRegister *SIPRegisterPlanModel
		if prior != nil {
			priorRegister = prior.SIPRegisterPlan
		}
		register := flattenSIPRegisterPlan(priorRegister, plan.SIPRegisterPlan)
		if prior == nil && plan.AuthUsername == "" && plan.AuthPassword == "" && register == nil {
			return
		}
		m.OutboundAuthenticationPlan = &OutboundAuthenticationPlanModel{
			AuthUsername:    types.StringValue(plan.AuthUsername),
			AuthPassword:    types.StringValue(plan.AuthPassword),
			SIPRegisterPlan: register,
		}
	}
}

// flattenSIPRegisterPlan keeps an empty registration block absent when it was unset.
func flattenSIPRegisterPlan(prior *SIPRegisterPlanModel, in *vapi.SIPRegisterPlan) *SIPRegisterPlanModel {
	if in == nil || (prior == nil && *in == (vapi.SIPRegisterPlan{})) {
		return nil
	}
	return &SIPRegisterPlanModel{
		Domain:   types.StringValue(in.Domain),
		Username: types.StringValue(in.Username),
		Realm:    types.StringValue(in.Realm),
	}
}
