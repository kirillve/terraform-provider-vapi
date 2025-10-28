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

func TestVAPITwilioPhoneNumberResourceLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	createResp := mustMarshal(t, vapi.TwilioPhoneNumber{
		ID:        "pn-1",
		OrgID:     "org-1",
		Name:      "primary",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
		Provider:  "twilio",
		Fallback: &vapi.FallbackDestination{
			Type:                   "number",
			NumberE164CheckEnabled: true,
			Number:                 "+123",
			Extension:              "101",
			Message:                "fallback",
			Description:            "desc",
		},
	})

	updateResp := mustMarshal(t, vapi.TwilioPhoneNumber{
		ID:        "pn-1",
		OrgID:     "org-1",
		Name:      "primary-updated",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-02T00:00:00Z",
		Provider:  "twilio",
	})

	transport := &queueRoundTripper{
		t: t,
		responses: []queuedResponse{
			{method: http.MethodPost, path: "/phone-number", status: 200, body: createResp},
			{method: http.MethodGet, path: "/phone-number/pn-1", status: 200, body: createResp},
			{method: http.MethodPatch, path: "/phone-number/pn-1", status: 200, body: updateResp},
			{method: http.MethodDelete, path: "/phone-number/pn-1", status: 200, body: []byte(`{}`)},
		},
	}

	res := &VAPITwilioPhoneNumberResource{
		client: &vapi.APIClient{
			BaseURL:    "https://api.example.com",
			Token:      "token",
			HTTPClient: &http.Client{Transport: transport},
		},
	}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, twilioPhoneModel("primary")); diags.HasError() {
		t.Fatalf("plan diagnostics: %v", diags)
	}

	createState := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Create(ctx, resource.CreateRequest{Plan: plan}, &createState)
	if createState.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", createState.Diagnostics)
	}

	readResp := resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Read(ctx, resource.ReadRequest{State: createState.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", readResp.Diagnostics)
	}

	var readModel VAPITwilioPhoneNumberResourceModel
	if diags := readResp.State.Get(ctx, &readModel); diags.HasError() {
		t.Fatalf("state diagnostics: %v", diags)
	}
	if readModel.FallbackE164CheckEnabled.ValueString() != "true" {
		t.Fatalf("expected fallback flag true, got %s", readModel.FallbackE164CheckEnabled.ValueString())
	}

	updatePlan := tfsdk.Plan{Schema: schemaResp.Schema}
	updatedModel := twilioPhoneModel("primary-updated")
	if diags := updatePlan.Set(ctx, updatedModel); diags.HasError() {
		t.Fatalf("update plan diagnostics: %v", diags)
	}

	updateState := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Update(ctx, resource.UpdateRequest{State: readResp.State, Plan: updatePlan}, &updateState)
	if updateState.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", updateState.Diagnostics)
	}

	var updated VAPITwilioPhoneNumberResourceModel
	if diags := updateState.State.Get(ctx, &updated); diags.HasError() {
		t.Fatalf("updated state diagnostics: %v", diags)
	}
	if updated.Name.ValueString() != "primary-updated" {
		t.Fatalf("expected updated name, got %s", updated.Name.ValueString())
	}

	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateState.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", deleteResp.Diagnostics)
	}

	importResp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "pn-1"}, &importResp)

	transport.assertDrained()
}

func twilioPhoneModel(name string) VAPITwilioPhoneNumberResourceModel {
	return VAPITwilioPhoneNumberResourceModel{
		Name:                     types.StringValue(name),
		Number:                   types.StringValue("+123"),
		TwilioAccountSid:         types.StringValue("sid"),
		TwilioAuthToken:          types.StringValue("token"),
		AssistantID:              types.StringValue("assistant-1"),
		FallbackType:             types.StringValue("number"),
		FallbackE164CheckEnabled: types.StringValue("true"),
		FallbackNumber:           types.StringValue("+123"),
		FallbackExtension:        types.StringValue("101"),
		FallbackMessage:          types.StringValue("fallback"),
		FallbackDescription:      types.StringValue("desc"),
	}
}
