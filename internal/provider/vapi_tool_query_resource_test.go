package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

func TestVAPIToolQueryFunctionResourceLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	initial := mustMarshal(t, vapi.ToolQueryFunctionResponse{
		ID:       "tool-query-1",
		OrgID:    "org-1",
		Type:     "query",
		Function: vapi.Function{Name: "query", Description: "desc"},
		KnowledgeBases: []vapi.TQKnowledgeBase{
			{Provider: "vectara", Name: "kb", Model: "gpt", Description: "kb-desc", FileIDs: []string{"file-1"}},
		},
	})

	updated := mustMarshal(t, vapi.ToolQueryFunctionResponse{
		ID:       "tool-query-1",
		OrgID:    "org-1",
		Type:     "query",
		Function: vapi.Function{Name: "query", Description: "desc-updated"},
		KnowledgeBases: []vapi.TQKnowledgeBase{
			{Provider: "vectara", Name: "kb", Model: "gpt", Description: "kb-desc", FileIDs: []string{"file-1"}},
		},
	})

	transport := &queueRoundTripper{
		t: t,
		responses: []queuedResponse{
			{method: http.MethodPost, path: "/tool", status: 200, body: initial},
			{method: http.MethodGet, path: "/tool/tool-query-1", status: 200, body: initial},
			{method: http.MethodPatch, path: "/tool/tool-query-1", status: 200, body: updated},
			{method: http.MethodDelete, path: "/tool/tool-query-1", status: 200, body: []byte(`{}`)},
		},
	}

	res := &VAPIToolQueryFunctionResource{
		client: &vapi.APIClient{
			BaseURL:    "https://api.example.com",
			Token:      "token",
			HTTPClient: &http.Client{Transport: transport},
		},
	}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, toolQueryModel("desc")); diags.HasError() {
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

	var readModel VAPIToolQueryFunctionResourceModel
	if diags := readResp.State.Get(ctx, &readModel); diags.HasError() {
		t.Fatalf("state diagnostics: %v", diags)
	}
	if readModel.ID.ValueString() != "tool-query-1" {
		t.Fatalf("expected ID tool-query-1, got %s", readModel.ID.ValueString())
	}

	updatePlan := tfsdk.Plan{Schema: schemaResp.Schema}
	updatedModel := toolQueryModel("desc-updated")
	updatedModel.ID = types.StringValue("tool-query-1")
	updatedModel.OrgID = types.StringValue("org-1")
	if diags := updatePlan.Set(ctx, updatedModel); diags.HasError() {
		t.Fatalf("update plan diagnostics: %v", diags)
	}

	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Update(ctx, resource.UpdateRequest{State: readResp.State, Plan: updatePlan}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", updateResp.Diagnostics)
	}

	var updatedState VAPIToolQueryFunctionResourceModel
	if diags := updateResp.State.Get(ctx, &updatedState); diags.HasError() {
		t.Fatalf("updated state diagnostics: %v", diags)
	}
	if updatedState.Description.ValueString() != "desc-updated" {
		t.Fatalf("expected updated description, got %s", updatedState.Description.ValueString())
	}

	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", deleteResp.Diagnostics)
	}

	importResp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "tool-query-1"}, &importResp)

	transport.assertDrained()
}

type knowledgeBaseInput struct {
	Provider    string
	Name        string
	Model       string
	Description string
	FileIDs     []string
}

func toolQueryModel(description string) VAPIToolQueryFunctionResourceModel {
	kbs := knowledgeBasesValue([]knowledgeBaseInput{{
		Provider:    "vectara",
		Name:        "kb",
		Model:       "gpt",
		Description: "kb-desc",
		FileIDs:     []string{"file-1"},
	}})

	return VAPIToolQueryFunctionResourceModel{
		Name:           types.StringValue("query"),
		Description:    types.StringValue(description),
		KnowledgeBases: kbs,
	}
}

func knowledgeBasesValue(entries []knowledgeBaseInput) types.List {
	objType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"provider":    types.StringType,
		"name":        types.StringType,
		"model":       types.StringType,
		"description": types.StringType,
		"file_ids":    types.ListType{ElemType: types.StringType},
	}}

	var elements []attr.Value
	for _, entry := range entries {
		obj, diags := types.ObjectValue(objType.AttrTypes, map[string]attr.Value{
			"provider":    types.StringValue(entry.Provider),
			"name":        types.StringValue(entry.Name),
			"model":       types.StringValue(entry.Model),
			"description": types.StringValue(entry.Description),
			"file_ids":    ListValueFromStrings(entry.FileIDs),
		})
		if diags.HasError() {
			panic(diags)
		}
		elements = append(elements, obj)
	}

	list, diags := types.ListValue(objType, elements)
	if diags.HasError() {
		panic(diags)
	}
	return list
}
