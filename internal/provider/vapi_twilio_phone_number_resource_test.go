package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"net/http"
	"net/http/httptest"
	"reflect"
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

// Exercise framework planning and CRUD over a local API, including replacement decisions.
type twilioHarness struct {
	t        *testing.T
	server   tfprotov6.ProviderServer
	typ      tftypes.Type
	remote   map[string]any
	methods  []string
	requests []map[string]any
}

func newTwilioHarness(t *testing.T) *twilioHarness {
	t.Helper()
	h := &twilioHarness{t: t, remote: map[string]any{"id": "pn-1", "orgId": "org-1", "name": "primary", "provider": "twilio"}}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.methods = append(h.methods, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			h.requests = append(h.requests, body)
			for _, key := range []string{"smsEnabled", "server"} {
				if v, ok := body[key]; ok {
					h.remote[key] = v
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(h.remote); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(api.Close)
	h.server = providerserver.NewProtocol6(&VAPIProvider{version: "test"})()
	schemas, err := h.server.GetProviderSchema(t.Context(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	checkToolDiagnostics(t, schemas.Diagnostics)
	s := schemas.ResourceSchemas["vapi_twilio_phone_number"]
	h.typ = s.ValueType()
	var schemaResp resource.SchemaResponse
	(&VAPITwilioPhoneNumberResource{}).Schema(t.Context(), resource.SchemaRequest{}, &schemaResp)
	attr, ok := schemaResp.Schema.Attributes["sms_enabled"].(schema.BoolAttribute)
	if !ok || !attr.Optional || attr.Computed || len(attr.PlanModifiers) != 0 {
		t.Fatal("sms_enabled must be an optional, non-computed bool without replacement modifiers")
	}
	configured, err := h.server.ConfigureProvider(t.Context(), &tfprotov6.ConfigureProviderRequest{Config: &tfprotov6.DynamicValue{JSON: mustMarshal(t, map[string]any{"url": api.URL, "token": "test"})}})
	if err != nil {
		t.Fatal(err)
	}
	checkToolDiagnostics(t, configured.Diagnostics)
	return h
}

func (h *twilioHarness) config(attrs map[string]any) map[string]any {
	config := map[string]any{"name": "primary", "number": "+15555550123", "twilio_account_sid": "test-sid", "twilio_auth_token": "test-token"}
	for key, value := range attrs {
		config[key] = value
	}
	return config
}

func (h *twilioHarness) value(fields map[string]any) *tfprotov6.DynamicValue {
	h.t.Helper()
	filled := map[string]any{}
	object, ok := h.typ.(tftypes.Object)
	if !ok {
		h.t.Fatal("expected object schema")
	}
	for key := range object.AttributeTypes {
		filled[key] = nil
	}
	for key, value := range fields {
		filled[key] = value
	}
	return &tfprotov6.DynamicValue{JSON: mustMarshal(h.t, filled)}
}

func (h *twilioHarness) fields(state *tfprotov6.DynamicValue) map[string]any {
	h.t.Helper()
	v, err := state.Unmarshal(h.typ)
	if err != nil {
		h.t.Fatal(err)
	}
	var fields map[string]tftypes.Value
	if err := v.As(&fields); err != nil {
		h.t.Fatal(err)
	}
	out := map[string]any{}
	for key, value := range fields {
		if !value.IsKnown() {
			out[key] = tftypes.UnknownValue
		} else {
			out[key] = toolTestValue(h.t, value)
		}
	}
	return out
}

func (h *twilioHarness) plan(attrs map[string]any, prior *tfprotov6.DynamicValue) *tfprotov6.PlanResourceChangeResponse {
	h.t.Helper()
	proposed := h.config(attrs)
	if prior == nil {
		prior = &tfprotov6.DynamicValue{JSON: []byte("null")}
	} else {
		fields := h.fields(prior)
		for _, key := range []string{"id", "org_id", "phone_provider", "created_at", "updated_at"} {
			proposed[key] = fields[key]
		}
	}
	plan, err := h.server.PlanResourceChange(h.t.Context(), &tfprotov6.PlanResourceChangeRequest{TypeName: "vapi_twilio_phone_number", Config: h.value(h.config(attrs)), PriorState: prior, ProposedNewState: h.value(proposed)})
	if err != nil {
		h.t.Fatal(err)
	}
	checkToolDiagnostics(h.t, plan.Diagnostics)
	if len(plan.RequiresReplace) != 0 {
		h.t.Fatalf("unexpected replacement: %v", plan.RequiresReplace)
	}
	if !bytes.Equal(prior.JSON, []byte("null")) && h.fields(plan.PlannedState)["id"] != "pn-1" {
		h.t.Fatal("plan did not preserve id")
	}
	return plan
}

func (h *twilioHarness) apply(attrs map[string]any, prior *tfprotov6.DynamicValue) *tfprotov6.DynamicValue {
	h.t.Helper()
	plan := h.plan(attrs, prior)
	if prior == nil {
		prior = &tfprotov6.DynamicValue{JSON: []byte("null")}
	}
	applied, err := h.server.ApplyResourceChange(h.t.Context(), &tfprotov6.ApplyResourceChangeRequest{TypeName: "vapi_twilio_phone_number", Config: h.value(h.config(attrs)), PriorState: prior, PlannedState: plan.PlannedState})
	if err != nil {
		h.t.Fatal(err)
	}
	checkToolDiagnostics(h.t, applied.Diagnostics)
	if h.fields(applied.NewState)["id"] != "pn-1" {
		h.t.Fatal("apply changed id")
	}
	return applied.NewState
}

func (h *twilioHarness) refresh(state *tfprotov6.DynamicValue) *tfprotov6.DynamicValue {
	h.t.Helper()
	read, err := h.server.ReadResource(h.t.Context(), &tfprotov6.ReadResourceRequest{TypeName: "vapi_twilio_phone_number", CurrentState: state})
	if err != nil {
		h.t.Fatal(err)
	}
	checkToolDiagnostics(h.t, read.Diagnostics)
	return read.NewState
}

func TestTwilioSMSEnabledLifecycle(t *testing.T) {
	h := newTwilioHarness(t)
	state := h.apply(map[string]any{"sms_enabled": false}, nil)
	if h.requests[0]["smsEnabled"] != false {
		t.Fatal("create must send explicit false")
	}
	state = h.apply(map[string]any{"sms_enabled": true}, state)
	if h.requests[1]["smsEnabled"] != true {
		t.Fatal("update must send true")
	}
	if !reflect.DeepEqual(h.methods, []string{"POST /phone-number", "PATCH /phone-number/pn-1"}) {
		t.Fatalf("unexpected API operations: %v", h.methods)
	}
	state = h.apply(nil, state)
	if _, ok := h.requests[2]["smsEnabled"]; ok {
		t.Fatal("removing attribute must omit smsEnabled")
	}
	if h.fields(h.refresh(state))["sms_enabled"] != nil {
		t.Fatal("removing attribute must remain null after refresh")
	}
	plan := h.plan(nil, state)
	before, err := state.Unmarshal(h.typ)
	if err != nil {
		t.Fatal(err)
	}
	after, err := plan.PlannedState.Unmarshal(h.typ)
	if err != nil {
		t.Fatal(err)
	}
	if !before.Equal(after) {
		t.Fatal("removing attribute must converge to an empty plan")
	}
}

func TestTwilioSMSEnabledRead(t *testing.T) {
	for _, prior := range []any{nil, false, true} {
		for _, response := range []any{nil, false, true} {
			t.Run(fmt.Sprintf("prior=%v/response=%v", prior, response), func(t *testing.T) {
				h := newTwilioHarness(t)
				state := h.apply(map[string]any{"sms_enabled": prior}, nil)
				if response == nil {
					delete(h.remote, "smsEnabled")
				} else {
					h.remote["smsEnabled"] = response
				}
				want := prior
				if prior != nil && response != nil {
					want = response
				}
				if got := h.fields(h.refresh(state))["sms_enabled"]; got != want {
					t.Fatalf("read sms_enabled = %v, want %v", got, want)
				}
			})
		}
	}
}

func TestTwilioServerURLLifecycle(t *testing.T) {
	h := newTwilioHarness(t)
	state := h.apply(map[string]any{"server_url": "https://example.com/a"}, nil)
	if !reflect.DeepEqual(h.requests[0]["server"], map[string]any{"url": "https://example.com/a"}) {
		t.Fatalf("create must send server.url: %v", h.requests[0])
	}
	state = h.apply(map[string]any{"server_url": "https://example.com/b"}, state)
	if !reflect.DeepEqual(h.requests[1]["server"], map[string]any{"url": "https://example.com/b"}) {
		t.Fatalf("update must send server.url: %v", h.requests[1])
	}
	if !reflect.DeepEqual(h.methods, []string{"POST /phone-number", "PATCH /phone-number/pn-1"}) {
		t.Fatalf("unexpected API operations: %v", h.methods)
	}
	h.remote["server"] = map[string]any{"url": "https://example.com/drift"}
	if got := h.fields(h.refresh(state))["server_url"]; got != "https://example.com/drift" {
		t.Fatalf("read server_url = %v, want remote drift", got)
	}
	state = h.apply(nil, state)
	if _, ok := h.requests[2]["server"]; ok {
		t.Fatal("removing attribute must omit server")
	}
	if h.fields(h.refresh(state))["server_url"] != nil {
		t.Fatal("removing attribute must remain null after refresh")
	}
}

func TestAccTwilioSMSEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("acceptance test requires live VAPI credentials")
	}
	t.Skip("acceptance stub: configure VAPI and Twilio credentials via environment to verify SMS settings on a live number")
}
