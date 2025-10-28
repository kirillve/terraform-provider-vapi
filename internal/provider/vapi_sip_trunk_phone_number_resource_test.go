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

func TestVAPISIPTrunkPhoneNumberResourceLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	createResp := mustMarshal(t, vapi.ImportSIPTrunkPhoneNumberResponse{
		ID:                     "sip-pn-1",
		OrgID:                  "org-1",
		Number:                 "+1555",
		CreatedAt:              "2024-01-01T00:00:00Z",
		UpdatedAt:              "2024-01-01T00:00:00Z",
		Provider:               "byo-phone-number",
		Name:                   "sip-number",
		NumberE164CheckEnabled: true,
		CredentialID:           "cred-1",
	})

	updateResp := mustMarshal(t, vapi.ImportSIPTrunkPhoneNumberResponse{
		ID:                     "sip-pn-1",
		OrgID:                  "org-1",
		Number:                 "+1666",
		CreatedAt:              "2024-01-01T00:00:00Z",
		UpdatedAt:              "2024-01-02T00:00:00Z",
		Provider:               "byo-phone-number",
		Name:                   "sip-number-updated",
		NumberE164CheckEnabled: false,
		CredentialID:           "cred-1",
	})

	transport := &queueRoundTripper{
		t: t,
		responses: []queuedResponse{
			{method: http.MethodPost, path: "/phone-number", status: 200, body: createResp},
			{method: http.MethodGet, path: "/phone-number/sip-pn-1", status: 200, body: createResp},
			{method: http.MethodPatch, path: "/phone-number/sip-pn-1", status: 200, body: updateResp},
			{method: http.MethodDelete, path: "/phone-number/sip-pn-1", status: 200, body: []byte(`{}`)},
		},
	}

	res := &VAPISIPTrunkPhoneNumberResource{
		client: &vapi.APIClient{
			BaseURL:    "https://api.example.com",
			Token:      "token",
			HTTPClient: &http.Client{Transport: transport},
		},
	}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, sipPhoneModel("sip-number", true)); diags.HasError() {
		t.Fatalf("plan diagnostics: %v", diags)
	}

	createRespState := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Create(ctx, resource.CreateRequest{Plan: plan}, &createRespState)
	if createRespState.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", createRespState.Diagnostics)
	}

	readResp := resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Read(ctx, resource.ReadRequest{State: createRespState.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", readResp.Diagnostics)
	}

	var readModel VAPISIPTrunkPhoneNumberResourceModel
	if diags := readResp.State.Get(ctx, &readModel); diags.HasError() {
		t.Fatalf("state diagnostics: %v", diags)
	}
	if !readModel.NumberE164CheckEnabled.ValueBool() {
		t.Fatalf("expected e164 check enabled")
	}

	updatePlan := tfsdk.Plan{Schema: schemaResp.Schema}
	updatedModel := sipPhoneModel("sip-number-updated", false)
	if diags := updatePlan.Set(ctx, updatedModel); diags.HasError() {
		t.Fatalf("update plan diagnostics: %v", diags)
	}

	updateRespState := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Update(ctx, resource.UpdateRequest{State: readResp.State, Plan: updatePlan}, &updateRespState)
	if updateRespState.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", updateRespState.Diagnostics)
	}

	var updated VAPISIPTrunkPhoneNumberResourceModel
	if diags := updateRespState.State.Get(ctx, &updated); diags.HasError() {
		t.Fatalf("updated state diagnostics: %v", diags)
	}
	if updated.Name.ValueString() != "sip-number-updated" {
		t.Fatalf("expected updated name, got %s", updated.Name.ValueString())
	}
	if updated.NumberE164CheckEnabled.ValueBool() {
		t.Fatalf("expected e164 flag false after update")
	}

	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateRespState.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", deleteResp.Diagnostics)
	}

	importResp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "sip-pn-1"}, &importResp)

	transport.assertDrained()
}

func sipPhoneModel(name string, e164 bool) VAPISIPTrunkPhoneNumberResourceModel {
	return VAPISIPTrunkPhoneNumberResourceModel{
		Name:                   types.StringValue(name),
		Number:                 types.StringValue("+1555"),
		CredentialID:           types.StringValue("cred-1"),
		NumberE164CheckEnabled: types.BoolValue(e164),
	}
}
