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
			{method: http.MethodDelete, path: "/file/tool-1", status: 200, body: []byte(`{}`)},
			{method: http.MethodPost, path: "/tool", status: 200, body: updateResponse},
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
		tc := tc
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
		Parameters: Parameters{
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
