package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"io"
	"math/big"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

func TestVAPIToolFunctionResourceLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	initialResponse := mustMarshal(t, vapi.ToolFunctionResponse{
		ID:        "tool-1",
		OrgID:     "org-1",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
		Type:      "function",
		Async:     true,
		Server: vapi.ResponseServer{
			URL: "https://server.example.com",
		},
		Function: vapi.ResponseFunction{
			Name:        "tool",
			Description: "desc",
			Async:       true,
			Parameters: vapi.ResponseFunctionParams{
				Type:     "object",
				Required: []string{"foo"},
				Properties: map[string]vapi.ResponseProperty{
					"foo": {Description: "A field", Type: "string", Enum: []string{"opt1"}},
				},
			},
		},
	})

	updateResponse := mustMarshal(t, vapi.ToolFunctionResponse{
		ID:        "tool-1",
		OrgID:     "org-1",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-02T00:00:00Z",
		Type:      "function",
		Async:     true,
		Server: vapi.ResponseServer{
			URL: "https://server.example.com/updated",
		},
		Function: vapi.ResponseFunction{
			Name:        "tool-updated",
			Description: "desc",
			Async:       true,
			Parameters: vapi.ResponseFunctionParams{
				Type:     "object",
				Required: []string{"foo"},
				Properties: map[string]vapi.ResponseProperty{
					"foo": {Description: "A field", Type: "string", Enum: []string{"opt1"}},
				},
			},
		},
	})

	transport := &queueRoundTripper{
		t: t,
		responses: []queuedResponse{
			{method: http.MethodPost, path: "/tool", status: 200, body: initialResponse},
			{method: http.MethodGet, path: "/tool/tool-1", status: 200, body: initialResponse},
			{method: http.MethodPatch, path: "/tool/tool-1", status: 200, body: updateResponse},
			{method: http.MethodDelete, path: "/tool/tool-1", status: 200, body: []byte(`{}`)},
		},
	}

	res := &VAPIToolFunctionResource{
		client: &vapi.APIClient{
			BaseURL:    "https://api.example.com",
			Token:      "token",
			HTTPClient: &http.Client{Transport: transport},
		},
	}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	createPlan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := createPlan.Set(ctx, toolFunctionModel("tool", "https://server.example.com")); diags.HasError() {
		t.Fatalf("plan.Set diagnostics: %v", diags)
	}

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Create(ctx, resource.CreateRequest{Plan: createPlan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", createResp.Diagnostics)
	}

	readResp := resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", readResp.Diagnostics)
	}

	var readModel VAPIToolFunctionResourceModel
	if diags := readResp.State.Get(ctx, &readModel); diags.HasError() {
		t.Fatalf("state get diagnostics: %v", diags)
	}
	if readModel.ID.ValueString() != "tool-1" {
		t.Fatalf("expected ID tool-1, got %s", readModel.ID.ValueString())
	}

	updateModel := toolFunctionModel("tool-updated", "https://server.example.com/updated")
	updateModel.ID = types.StringValue("tool-1")
	updateModel.OrgID = types.StringValue("org-1")

	updatePlan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := updatePlan.Set(ctx, updateModel); diags.HasError() {
		t.Fatalf("update plan diagnostics: %v", diags)
	}

	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Update(ctx, resource.UpdateRequest{State: readResp.State, Plan: updatePlan}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", updateResp.Diagnostics)
	}

	var updatedModel VAPIToolFunctionResourceModel
	if diags := updateResp.State.Get(ctx, &updatedModel); diags.HasError() {
		t.Fatalf("updated state diagnostics: %v", diags)
	}
	if updatedModel.Name.ValueString() != "tool-updated" {
		t.Fatalf("expected updated name, got %s", updatedModel.Name.ValueString())
	}
	if updatedModel.ServerURL.ValueString() != "https://server.example.com/updated" {
		t.Fatalf("expected updated server url, got %s", updatedModel.ServerURL.ValueString())
	}

	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", deleteResp.Diagnostics)
	}

	importResp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "tool-1"}, &importResp)

	transport.assertDrained()
}

func TestVAPIToolFunctionResourceCreateVariants(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		model VAPIToolFunctionResourceModel
	}{
		{
			name: "dtmf",
			model: func() VAPIToolFunctionResourceModel {
				m := toolFunctionModel("dtmf-tool", "")
				m.Type = types.StringValue("dtmf")
				return m
			}(),
		},
		{
			name: "transfer",
			model: func() VAPIToolFunctionResourceModel {
				m := toolFunctionModel("transfer-tool", "")
				m.Type = types.StringValue("transferCall")
				m.Destinations = []Destination{
					{
						Type:                   types.StringValue("number"),
						Number:                 types.StringValue("+123"),
						Message:                types.StringValue("message"),
						Description:            types.StringValue("desc"),
						NumberE164CheckEnabled: types.BoolValue(true),
					},
				}
				return m
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			respBody := mustMarshal(t, vapi.ToolFunctionResponse{
				ID:    "tool-variant",
				OrgID: "org-variant",
				Type:  tc.model.Type.ValueString(),
				Function: vapi.ResponseFunction{
					Name:        tc.model.Name.ValueString(),
					Description: tc.model.Description.ValueString(),
					Parameters:  vapi.ResponseFunctionParams{Type: "object"},
				},
			})

			transport := &queueRoundTripper{
				t: t,
				responses: []queuedResponse{
					{method: http.MethodPost, path: "/tool", status: 200, body: respBody},
				},
			}

			res := &VAPIToolFunctionResource{
				client: &vapi.APIClient{
					BaseURL:    "https://api.example.com",
					Token:      "token",
					HTTPClient: &http.Client{Transport: transport},
				},
			}

			var schemaResp resource.SchemaResponse
			res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

			plan := tfsdk.Plan{Schema: schemaResp.Schema}
			if diags := plan.Set(ctx, tc.model); diags.HasError() {
				t.Fatalf("plan diagnostics: %v", diags)
			}

			createResp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			res.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
			if createResp.Diagnostics.HasError() {
				t.Fatalf("create diagnostics: %v", createResp.Diagnostics)
			}

			transport.assertDrained()
		})
	}
}

func toolFunctionModel(name, serverURL string) VAPIToolFunctionResourceModel {
	return VAPIToolFunctionResourceModel{
		Name:         types.StringValue(name),
		Description:  types.StringValue("desc"),
		Async:        types.BoolValue(true),
		Type:         types.StringValue("function"),
		ServerURL:    types.StringValue(serverURL),
		ServerSecret: types.StringValue("secret"),
		Parameters: &Parameters{
			Type:     types.StringValue("object"),
			Async:    types.BoolValue(true),
			Required: ListValueFromStrings([]string{"foo"}),
			Properties: map[string]Property{
				"foo": {
					Type:        types.StringValue("string"),
					Description: types.StringValue("A field"),
					Enum:        ListValueFromStrings([]string{"opt1"}),
				},
			},
		},
	}
}

// Captured against the unmodified provider before adding parameters_json.
func TestToolFunctionFlatGolden(t *testing.T) {
	model := toolFunctionModel("tool", "https://server.example.com")
	got := string(mustMarshal(t, buildToolFunctionRequest(&model)))
	const want = `{"server":{"url":"https://server.example.com","secret":"secret","timeoutSeconds":20},"function":{"description":"desc","async":true,"name":"tool","parameters":{"type":"object","properties":{"foo":{"type":"string","description":"A field","enum":["opt1"]}},"required":["foo"]}},"type":"function","async":true}`
	if got != want {
		t.Fatalf("flat request changed\nwant %s\ngot  %s", want, got)
	}
}

const nestedToolSchema = `{"type":"object","properties":{"booking_info":{"type":"object","properties":{"customer":{"type":"object","properties":{"first_name":{"type":"string"},"tags":{"type":"array","items":{"type":"string"}}}}}}},"required":["booking_info"]}`

// Exercise the real framework RPC path without live VAPI credentials or a new test dependency.
type toolProtocolHarness struct {
	t        *testing.T
	server   tfprotov6.ProviderServer
	typ      tftypes.Type
	remote   map[string]any
	requests []map[string]any
	methods  []string
}

func newToolProtocolHarness(t *testing.T) *toolProtocolHarness {
	t.Helper()
	h := &toolProtocolHarness{t: t}
	transport := toolRoundTripper(func(r *http.Request) (*http.Response, error) {
		h.methods = append(h.methods, r.Method)
		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			h.requests = append(h.requests, body)
			h.remote = body
			h.remote["id"] = "tool-1"
			h.remote["orgId"] = "org-1"
			h.remote["createdAt"] = "2024-01-01"
			h.remote["updatedAt"] = "2024-01-01"
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(mustMarshal(t, h.remote))), Header: make(http.Header)}, nil
	})
	h.server = providerserver.NewProtocol6(&toolTestProvider{client: &vapi.APIClient{BaseURL: "https://api.example.com", Token: "test", HTTPClient: &http.Client{Transport: transport}}})()
	ctx := context.Background()
	schemas, err := h.server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	checkToolDiagnostics(t, schemas.Diagnostics)
	h.typ = schemas.ResourceSchemas["vapi_tool_function"].ValueType()
	cfg := &tfprotov6.DynamicValue{JSON: mustMarshal(t, map[string]any{"url": "https://api.example.com", "token": "test"})}
	configured, err := h.server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	checkToolDiagnostics(t, configured.Diagnostics)
	return h
}

func checkToolDiagnostics(t *testing.T, diagnostics []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, d := range diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", d.Summary, d.Detail)
		}
	}
}

func flatToolConfig(kind string) map[string]any {
	cfg := map[string]any{"name": "tool", "description": "desc", "async": true, "type": kind,
		"server_url": "https://server.example.com", "server_secret": "secret",
		"parameters": map[string]any{"type": "object", "async": true, "required": []string{"foo"},
			"properties": map[string]any{"foo": map[string]any{"type": "string", "description": "A field", "enum": []string{"opt1"}}}}}
	if kind == "transferCall" {
		cfg["destinations"] = []any{map[string]any{"type": "number", "number": "+17805551234", "extension": nil, "message": "Transfer", "description": "Office", "number_e164_check_enabled": true}}
	}
	return cfg
}

func (h *toolProtocolHarness) value(cfg map[string]any) *tfprotov6.DynamicValue {
	h.t.Helper()
	filled := map[string]any{}
	for key := range toolTestAs[tftypes.Object](h.t, h.typ).AttributeTypes {
		filled[key] = nil
	}
	for key, value := range cfg {
		if _, ok := filled[key]; !ok {
			h.t.Fatalf("missing schema attribute %s", key)
		}
		filled[key] = value
	}
	return &tfprotov6.DynamicValue{JSON: mustMarshal(h.t, filled)}
}

func (h *toolProtocolHarness) stateMap(state *tfprotov6.DynamicValue) map[string]any {
	h.t.Helper()
	value, err := state.Unmarshal(h.typ)
	if err != nil {
		h.t.Fatal(err)
	}
	var fields map[string]tftypes.Value
	if err := value.As(&fields); err != nil {
		h.t.Fatal(err)
	}
	result := map[string]any{}
	for key, val := range fields {
		result[key] = toolTestValue(h.t, val)
	}
	return result
}

func (h *toolProtocolHarness) plan(cfg map[string]any, prior *tfprotov6.DynamicValue) *tfprotov6.PlanResourceChangeResponse {
	h.t.Helper()
	proposed := map[string]any{}
	if prior != nil {
		previous := h.stateMap(prior)
		for _, key := range []string{"id", "org_id", "created_at", "updated_at"} {
			proposed[key] = previous[key]
		}
	}
	for k, v := range cfg {
		proposed[k] = v
	}
	if prior == nil {
		prior = &tfprotov6.DynamicValue{JSON: []byte("null")}
	}
	result, err := h.server.PlanResourceChange(context.Background(), &tfprotov6.PlanResourceChangeRequest{
		TypeName: "vapi_tool_function", Config: h.value(cfg), PriorState: prior, ProposedNewState: h.value(proposed)})
	if err != nil {
		h.t.Fatal(err)
	}
	checkToolDiagnostics(h.t, result.Diagnostics)
	return result
}

func (h *toolProtocolHarness) apply(cfg map[string]any, prior *tfprotov6.DynamicValue) *tfprotov6.DynamicValue {
	h.t.Helper()
	plan := h.plan(cfg, prior)
	if prior == nil {
		prior = &tfprotov6.DynamicValue{JSON: []byte("null")}
	}
	applied, err := h.server.ApplyResourceChange(context.Background(), &tfprotov6.ApplyResourceChangeRequest{
		TypeName: "vapi_tool_function", Config: h.value(cfg), PriorState: prior, PlannedState: plan.PlannedState})
	if err != nil {
		h.t.Fatal(err)
	}
	checkToolDiagnostics(h.t, applied.Diagnostics)
	return applied.NewState
}

func (h *toolProtocolHarness) refresh(state *tfprotov6.DynamicValue) *tfprotov6.DynamicValue {
	h.t.Helper()
	read, err := h.server.ReadResource(context.Background(), &tfprotov6.ReadResourceRequest{TypeName: "vapi_tool_function", CurrentState: state})
	if err != nil {
		h.t.Fatal(err)
	}
	checkToolDiagnostics(h.t, read.Diagnostics)
	return read.NewState
}

func (h *toolProtocolHarness) assertEmptyPlan(cfg map[string]any, state *tfprotov6.DynamicValue) {
	h.t.Helper()
	plan := h.plan(cfg, state)
	before, err := state.Unmarshal(h.typ)
	if err != nil {
		h.t.Fatal(err)
	}
	after, err := plan.PlannedState.Unmarshal(h.typ)
	if err != nil {
		h.t.Fatal(err)
	}
	if !before.Equal(after) || len(plan.RequiresReplace) != 0 {
		h.t.Fatalf("nonempty plan\nbefore: %s\nafter: %s\nreplacement: %v", before, after, plan.RequiresReplace)
	}
}

func TestToolFunctionFlatEmptyPlan(t *testing.T) {
	for _, kind := range []string{"function", "dtmf", "transferCall"} {
		t.Run(kind, func(t *testing.T) {
			h := newToolProtocolHarness(t)
			cfg := flatToolConfig(kind)
			state := h.apply(cfg, nil)
			// Legacy state snapshots have neither of the newly added attributes.
			legacy := h.stateMap(state)
			delete(legacy, "parameters_json")
			delete(legacy, "server_timeout_seconds")
			upgraded, err := h.server.UpgradeResourceState(context.Background(), &tfprotov6.UpgradeResourceStateRequest{
				TypeName: "vapi_tool_function", Version: 0, RawState: &tfprotov6.RawState{JSON: mustMarshal(t, legacy)},
			})
			if err != nil {
				t.Fatal(err)
			}
			checkToolDiagnostics(t, upgraded.Diagnostics)
			state = h.refresh(upgraded.UpgradedState)
			h.assertEmptyPlan(cfg, state)
			if h.stateMap(state)["server_timeout_seconds"] != nil {
				t.Fatal("legacy timeout must remain null")
			}
		})
	}
}

func TestToolFunctionNestedRequestAndRoundTrip(t *testing.T) {
	for _, timeout := range []int{0, 30} {
		t.Run(fmt.Sprint(timeout), func(t *testing.T) {
			h := newToolProtocolHarness(t)
			cfg := flatToolConfig("function")
			delete(cfg, "parameters")
			cfg["parameters_json"] = nestedToolSchema
			if timeout != 0 {
				cfg["server_timeout_seconds"] = timeout
			}
			state := h.apply(cfg, nil)
			body := h.requests[0]
			parameters := toolTestAs[map[string]any](t, body["function"])["parameters"]
			var want any
			if err := json.Unmarshal([]byte(nestedToolSchema), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(parameters, want) {
				t.Fatalf("nested schema lost: %v", parameters)
			}
			server := toolTestAs[map[string]any](t, body["server"])
			expectedTimeout := timeout
			if expectedTimeout == 0 {
				expectedTimeout = 20
			}
			if server["url"] != cfg["server_url"] || server["secret"] != cfg["server_secret"] || server["timeoutSeconds"] != float64(expectedTimeout) {
				t.Fatalf("server: %v", server)
			}
			// MarshalIndent sorts keys and adds whitespace, unlike the original input.
			reordered, err := json.MarshalIndent(parameters, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			toolTestAs[map[string]any](t, h.remote["function"])["parameters"] = json.RawMessage(reordered)
			state = h.refresh(state)
			h.assertEmptyPlan(cfg, state)
			if h.stateMap(state)["parameters"] != nil {
				t.Fatal("nested mode populated flat parameters")
			}
			toolTestAs[map[string]any](t, h.remote["function"])["parameters"] = json.RawMessage(strings.ReplaceAll(nestedToolSchema, `"first_name":{"type":"string"}`, `"first_name":{"type":"number"}`))
			drift := h.refresh(state)
			plan := h.plan(cfg, drift)
			before, _ := drift.Unmarshal(h.typ)
			after, _ := plan.PlannedState.Unmarshal(h.typ)
			if before.Equal(after) {
				t.Fatal("nested remote drift hidden")
			}
		})
	}
}

func TestToolFunctionValidation(t *testing.T) {
	cases := []struct {
		name, kind string
		flat       bool
		raw        any
		wantError  bool
	}{
		{"flat", "function", true, nil, false}, {"nested", "function", false, nestedToolSchema, false},
		{"both", "function", true, nestedToolSchema, true}, {"neither", "function", false, nil, true},
		{"array", "function", false, "[]", true}, {"number", "function", false, "1", true},
		{"invalid", "function", false, "{", true}, {"null", "function", false, "null", true},
		{"dtmf", "dtmf", true, nil, false}, {"dtmf_without", "dtmf", false, nil, false},
		{"transfer", "transferCall", true, nil, false}, {"transfer_nested", "transferCall", false, nestedToolSchema, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newToolProtocolHarness(t)
			cfg := flatToolConfig(tc.kind)
			if !tc.flat {
				delete(cfg, "parameters")
			}
			if tc.raw != nil {
				cfg["parameters_json"] = tc.raw
			}
			result, err := h.server.ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{TypeName: "vapi_tool_function", Config: h.value(cfg)})
			if err != nil {
				t.Fatal(err)
			}
			hasError := false
			for _, d := range result.Diagnostics {
				hasError = hasError || d.Severity == tfprotov6.DiagnosticSeverityError
			}
			if hasError != tc.wantError {
				t.Fatalf("want error %v, diagnostics %v", tc.wantError, result.Diagnostics)
			}
		})
	}
}

func TestToolFunctionImportModes(t *testing.T) {
	cases := []struct {
		name, raw string
		nested    bool
	}{
		{"flat", `{"type":"object","properties":{"foo":{"type":"string","description":"A field","enum":["opt1"]}},"required":["foo"]}`, false},
		{"nested", nestedToolSchema, true},
		{"array", `{"type":"object","properties":{"foo":{"type":"array","items":{"type":"string"}}}}`, true},
		{"top_level_keyword", `{"type":"object","properties":{},"additionalProperties":false}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newToolProtocolHarness(t)
			h.remote = map[string]any{"id": "tool-1", "type": "function", "function": map[string]any{"name": "tool", "description": "desc", "parameters": json.RawMessage(tc.raw)}}
			imported, err := h.server.ImportResourceState(context.Background(), &tfprotov6.ImportResourceStateRequest{TypeName: "vapi_tool_function", ID: "tool-1"})
			if err != nil {
				t.Fatal(err)
			}
			checkToolDiagnostics(t, imported.Diagnostics)
			state := h.stateMap(h.refresh(imported.ImportedResources[0].State))
			if (state["parameters_json"] != nil) != tc.nested || (state["parameters"] == nil) != tc.nested {
				t.Fatalf("wrong import mode: %v", state)
			}
		})
	}
}

func TestToolFunctionTimeoutInPlace(t *testing.T) {
	h := newToolProtocolHarness(t)
	cfg := flatToolConfig("function")
	cfg["server_timeout_seconds"] = 30
	state := h.apply(cfg, nil)
	h.assertEmptyPlan(cfg, h.refresh(state))
	cfg["server_timeout_seconds"] = 25
	plan := h.plan(cfg, state)
	if len(plan.RequiresReplace) != 0 {
		t.Fatalf("timeout change requires replacement: %v", plan.RequiresReplace)
	}
	state = h.apply(cfg, state)
	if h.methods[len(h.methods)-1] != http.MethodPatch {
		t.Fatalf("expected PATCH, got %v", h.methods)
	}
	if toolTestAs[map[string]any](t, h.requests[len(h.requests)-1]["server"])["timeoutSeconds"] != float64(25) {
		t.Fatal("timeout update not sent")
	}
	h.assertEmptyPlan(cfg, h.refresh(state))
}

func TestAccToolFunctionNested(t *testing.T) {
	if testing.Short() {
		t.Skip("acceptance test requires live VAPI credentials")
	}
	t.Skip("acceptance stub: configure VAPI credentials via environment before exercising live nested tools")
}

func toolTestValue(t *testing.T, value tftypes.Value) any {
	t.Helper()
	if value.IsNull() {
		return nil
	}
	switch typ := value.Type(); {
	case typ.Is(tftypes.String):
		var v string
		if err := value.As(&v); err != nil {
			t.Fatal(err)
		}
		return v
	case typ.Is(tftypes.Bool):
		var v bool
		if err := value.As(&v); err != nil {
			t.Fatal(err)
		}
		return v
	case typ.Is(tftypes.Number):
		var v *big.Float
		if err := value.As(&v); err != nil {
			t.Fatal(err)
		}
		n, _ := v.Float64()
		return n
	case typ.Is(tftypes.Object{}), typ.Is(tftypes.Map{}):
		var v map[string]tftypes.Value
		if err := value.As(&v); err != nil {
			t.Fatal(err)
		}
		out := map[string]any{}
		for k, x := range v {
			out[k] = toolTestValue(t, x)
		}
		return out
	default:
		var v []tftypes.Value
		if err := value.As(&v); err != nil {
			t.Fatal(err)
		}
		out := []any{}
		for _, x := range v {
			out = append(out, toolTestValue(t, x))
		}
		return out
	}
}

type toolRoundTripper func(*http.Request) (*http.Response, error)

func (f toolRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type toolTestProvider struct {
	VAPIProvider
	client *vapi.APIClient
}

func (p *toolTestProvider) Configure(_ context.Context, _ frameworkprovider.ConfigureRequest, resp *frameworkprovider.ConfigureResponse) {
	resp.ResourceData = p.client
}

func toolTestAs[T any](t *testing.T, value any) T {
	t.Helper()
	typed, ok := value.(T)
	if !ok {
		t.Fatalf("unexpected type %T, want %T", value, typed)
	}
	return typed
}
