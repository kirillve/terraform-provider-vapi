package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

var _ resource.Resource = &VAPISIPTrunkResource{}
var _ resource.ResourceWithImportState = &VAPISIPTrunkResource{}

func NewVAPISIPTrunkResource() resource.Resource {
	return &VAPISIPTrunkResource{}
}

type VAPISIPTrunkResource struct {
	client *vapi.APIClient
}

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

type SIPGatewayModel struct {
	IP types.String `tfsdk:"ip"`
}

type OutboundAuthenticationPlanModel struct {
	AuthUsername    types.String          `tfsdk:"auth_username"`
	AuthPassword    types.String          `tfsdk:"auth_password"`
	SIPRegisterPlan *SIPRegisterPlanModel `tfsdk:"sip_register_plan"`
}

type SIPRegisterPlanModel struct {
	Domain   types.String `tfsdk:"domain"`
	Username types.String `tfsdk:"username"`
	Realm    types.String `tfsdk:"realm"`
}

func (r *VAPISIPTrunkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sip_trunk"
}

func (r *VAPISIPTrunkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SIP trunk in the VAPI system.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
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

func (r *VAPISIPTrunkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*vapi.APIClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *vapi.APIClient, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *VAPISIPTrunkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VAPISIPTrunkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	trunkReq := buildSIPTrunkRequest(&data)

	respBytes, status, err := r.client.CreateSIPTrunk(trunkReq)
	if err != nil || status >= 400 {
		resp.Diagnostics.AddError("Create Failed", fmt.Sprintf("Status: %d, Error: %v", status, err))
		return
	}

	var created vapi.SIPTrunk
	if err := json.Unmarshal(respBytes, &created); err != nil {
		resp.Diagnostics.AddError("Unmarshal Error", err.Error())
		return
	}

	bindVAPISIPTrunkResourceData(&data, &created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPISIPTrunkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VAPISIPTrunkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBytes, status, err := r.client.GetSIPTrunk(data.ID.ValueString())
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil || status >= 400 {
		resp.Diagnostics.AddError("Read Failed", fmt.Sprintf("Status: %d, Error: %v", status, err))
		return
	}

	var fetched vapi.SIPTrunk
	if err := json.Unmarshal(respBytes, &fetched); err != nil {
		resp.Diagnostics.AddWarning("Unmarshal Warning", err.Error())
		return
	}

	bindVAPISIPTrunkResourceData(&data, &fetched)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VAPISIPTrunkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state VAPISIPTrunkResourceModel
	var plan VAPISIPTrunkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	trunkReq := buildSIPTrunkRequest(&plan)

	respBytes, status, err := r.client.UpdateSIPTrunk(state.ID.ValueString(), trunkReq)
	if err != nil || status >= 400 {
		resp.Diagnostics.AddError("Update Failed", fmt.Sprintf("Status: %d, Error: %v", status, err))
		return
	}

	var updated vapi.SIPTrunk
	if err := json.Unmarshal(respBytes, &updated); err != nil {
		resp.Diagnostics.AddError("Unmarshal Error", err.Error())
		return
	}

	bindVAPISIPTrunkResourceData(&plan, &updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VAPISIPTrunkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VAPISIPTrunkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, status, err := r.client.DeleteSIPTrunk(data.ID.ValueString())
	if err != nil || status >= 400 {
		resp.Diagnostics.AddError("Delete Failed", fmt.Sprintf("Status: %d, Error: %v", status, err))
	}
}

func (r *VAPISIPTrunkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildSIPTrunkRequest(model *VAPISIPTrunkResourceModel) vapi.ImportSIPTrunkRequest {
	return vapi.ImportSIPTrunkRequest{
		Provider:                   model.SIPProvider.ValueString(),
		Name:                       model.Name.ValueString(),
		Gateways:                   convertGateways(model.Gateways),
		OutboundAuthenticationPlan: convertAuthPlan(model.OutboundAuthenticationPlan),
		OutboundLeadingPlusEnabled: model.OutboundLeadingPlusEnabled.ValueBool(),
		TechPrefix:                 model.TechPrefix.ValueString(),
		SIPDiversionHeader:         model.SIPDiversionHeader.ValueString(),
	}
}

func bindVAPISIPTrunkResourceData(model *VAPISIPTrunkResourceModel, trunk *vapi.SIPTrunk) {
	if trunk == nil {
		return
	}

	model.ID = types.StringValue(trunk.ID)
	model.SIPProvider = types.StringValue(trunk.Provider)
	model.Name = types.StringValue(trunk.Name)
	model.OutboundLeadingPlusEnabled = types.BoolValue(trunk.OutboundLeadingPlusEnabled)
	model.TechPrefix = types.StringValue(trunk.TechPrefix)
	model.SIPDiversionHeader = types.StringValue(trunk.SIPDiversionHeader)
	model.Gateways = flattenSIPGateways(trunk.Gateways)
	model.OutboundAuthenticationPlan = flattenOutboundAuthenticationPlan(trunk.OutboundAuthenticationPlan)
}

func flattenSIPGateways(gateways []vapi.SIPGateway) []SIPGatewayModel {
	if len(gateways) == 0 {
		return nil
	}

	result := make([]SIPGatewayModel, 0, len(gateways))
	for _, g := range gateways {
		result = append(result, SIPGatewayModel{
			IP: types.StringValue(g.IP),
		})
	}
	return result
}

func flattenOutboundAuthenticationPlan(plan *vapi.OutboundAuthenticationPlan) *OutboundAuthenticationPlanModel {
	if plan == nil {
		return nil
	}

	model := &OutboundAuthenticationPlanModel{
		AuthUsername: types.StringValue(plan.AuthUsername),
		AuthPassword: types.StringValue(plan.AuthPassword),
	}
	model.SIPRegisterPlan = flattenSIPRegisterPlan(plan.SIPRegisterPlan)
	return model
}

func flattenSIPRegisterPlan(plan *vapi.SIPRegisterPlan) *SIPRegisterPlanModel {
	if plan == nil {
		return nil
	}
	return &SIPRegisterPlanModel{
		Domain:   types.StringValue(plan.Domain),
		Username: types.StringValue(plan.Username),
		Realm:    types.StringValue(plan.Realm),
	}
}

// Helpers.
func convertGateways(models []SIPGatewayModel) []vapi.SIPGateway {
	var out []vapi.SIPGateway
	for _, m := range models {
		out = append(out, vapi.SIPGateway{IP: m.IP.ValueString()})
	}
	return out
}

func convertAuthPlan(model *OutboundAuthenticationPlanModel) *vapi.OutboundAuthenticationPlan {
	if model == nil {
		return nil
	}
	var reg *vapi.SIPRegisterPlan
	if model.SIPRegisterPlan != nil {
		reg = &vapi.SIPRegisterPlan{
			Domain:   model.SIPRegisterPlan.Domain.ValueString(),
			Username: model.SIPRegisterPlan.Username.ValueString(),
			Realm:    model.SIPRegisterPlan.Realm.ValueString(),
		}
	}
	return &vapi.OutboundAuthenticationPlan{
		AuthUsername:    model.AuthUsername.ValueString(),
		AuthPassword:    model.AuthPassword.ValueString(),
		SIPRegisterPlan: reg,
	}
}
