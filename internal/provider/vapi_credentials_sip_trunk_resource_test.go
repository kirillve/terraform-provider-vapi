package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

func TestVAPISIPTrunkResourceLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	createPayload := mustMarshal(t, vapi.TwilioPhoneNumber{ID: "trunk-1"})
	updatePayload := mustMarshal(t, vapi.TwilioPhoneNumber{ID: "trunk-1"})

	transport := &queueRoundTripper{
		t: t,
		responses: []queuedResponse{
			{method: http.MethodPost, path: "/credential", status: 200, body: createPayload},
			{method: http.MethodGet, path: "/credential/trunk-1", status: 200, body: createPayload},
			{method: http.MethodPatch, path: "/credential/trunk-1", status: 200, body: updatePayload},
			{method: http.MethodDelete, path: "/credential/trunk-1", status: 200, body: []byte(`{}`)},
		},
	}

	res := &VAPISIPTrunkResource{
		client: &vapi.APIClient{
			BaseURL:    "https://api.example.com",
			Token:      "token",
			HTTPClient: &http.Client{Transport: transport},
		},
	}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, sipTrunkModel("trunk")); diags.HasError() {
		t.Fatalf("plan diagnostics: %v", diags)
	}

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", createResp.Diagnostics)
	}

	readResp := resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", readResp.Diagnostics)
	}

	updatePlan := tfsdk.Plan{Schema: schemaResp.Schema}
	updated := sipTrunkModel("trunk-updated")
	updated.ID = types.StringValue("trunk-1")
	if diags := updatePlan.Set(ctx, updated); diags.HasError() {
		t.Fatalf("update plan diagnostics: %v", diags)
	}

	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Update(ctx, resource.UpdateRequest{State: readResp.State, Plan: updatePlan}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", updateResp.Diagnostics)
	}

	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", deleteResp.Diagnostics)
	}

	importResp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "trunk-1"}, &importResp)

	transport.assertDrained()
}

func sipTrunkModel(name string) VAPISIPTrunkResourceModel {
	return VAPISIPTrunkResourceModel{
		SIPProvider: types.StringValue("byo"),
		Name:        types.StringValue(name),
		Gateways: []SIPGatewayModel{
			{IP: types.StringValue("1.1.1.1")},
		},
		OutboundAuthenticationPlan: &OutboundAuthenticationPlanModel{
			AuthUsername: types.StringValue("user"),
			AuthPassword: types.StringValue("pass"),
			SIPRegisterPlan: &SIPRegisterPlanModel{
				Domain:   types.StringValue("example.com"),
				Username: types.StringValue("sip-user"),
				Realm:    types.StringValue("realm"),
			},
		},
		OutboundLeadingPlusEnabled: types.BoolValue(true),
		TechPrefix:                 types.StringValue("*123"),
		SIPDiversionHeader:         types.StringValue("Diversion"),
	}
}
