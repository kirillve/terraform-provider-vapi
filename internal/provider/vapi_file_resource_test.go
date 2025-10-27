package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

func TestVAPIFileResourceLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	fileResponseCreate := vapi.FileResponse{
		ID:           "file-123",
		Name:         "file.txt",
		OriginalName: "file.txt",
		Bytes:        12,
		Mimetype:     "text/plain",
		Path:         "files/file-123.txt",
		URL:          "https://api.example.com/files/file-123",
		OrgID:        "org-1",
		CreatedAt:    "2024-01-01T00:00:00Z",
		UpdatedAt:    "2024-01-01T00:00:00Z",
		Status:       "ready",
		Bucket:       "uploads",
		Purpose:      "knowledge_base",
	}

	fileResponseUpdate := fileResponseCreate
	fileResponseUpdate.ID = "file-123"
	fileResponseUpdate.Name = "file-updated.txt"
	fileResponseUpdate.OriginalName = "file-updated.txt"
	fileResponseUpdate.UpdatedAt = "2024-01-02T00:00:00Z"

	transport := &fileResourceTransport{
		t:              t,
		createResponse: fileResponseCreate,
		updateResponse: fileResponseUpdate,
		latestResponse: fileResponseCreate,
	}

	res := &VAPIFileResource{
		client: &vapi.APIClient{
			BaseURL:    "http://api.example.com",
			Token:      "token",
			HTTPClient: &http.Client{Transport: transport},
		},
	}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}

	createPlan := mustSetPlan(t, schemaResp.Schema, baseFileModel(types.StringValue("content-1"), types.StringValue("file.txt")))

	createResp := resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	res.Create(ctx, resource.CreateRequest{
		Plan: createPlan,
	}, &createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", createResp.Diagnostics)
	}

	createState := mustStateModel(t, ctx, createResp.State)
	if createState.Id.ValueString() != fileResponseCreate.ID {
		t.Fatalf("expected ID %s, got %s", fileResponseCreate.ID, createState.Id.ValueString())
	}
	if createState.URL.ValueString() != fileResponseCreate.URL {
		t.Fatalf("expected URL %s, got %s", fileResponseCreate.URL, createState.URL.ValueString())
	}

	readResp := resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	res.Read(ctx, resource.ReadRequest{
		State: createResp.State,
	}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", readResp.Diagnostics)
	}

	readState := mustStateModel(t, ctx, readResp.State)
	if readState.Id.ValueString() != fileResponseCreate.ID {
		t.Fatalf("expected read ID %s, got %s", fileResponseCreate.ID, readState.Id.ValueString())
	}

	updateModel := createState
	updateModel.Content = types.StringValue("content-2")
	updateModel.Filename = types.StringValue("file-updated.txt")
	updatePlan := mustSetPlan(t, schemaResp.Schema, updateModel)

	updateResp := resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	res.Update(ctx, resource.UpdateRequest{
		Plan: updatePlan,
	}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", updateResp.Diagnostics)
	}

	updateState := mustStateModel(t, ctx, updateResp.State)
	if updateState.Name.ValueString() != fileResponseUpdate.Name {
		t.Fatalf("expected updated name %s, got %s", fileResponseUpdate.Name, updateState.Name.ValueString())
	}
	if updateState.UpdatedAt.ValueString() != fileResponseUpdate.UpdatedAt {
		t.Fatalf("expected updated timestamp %s, got %s", fileResponseUpdate.UpdatedAt, updateState.UpdatedAt.ValueString())
	}

	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{
		State: updateResp.State,
	}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", deleteResp.Diagnostics)
	}

	importResp := resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	importResp.State.RemoveResource(ctx)
	res.ImportState(ctx, resource.ImportStateRequest{
		ID: fileResponseCreate.ID,
	}, &importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %v", importResp.Diagnostics)
	}

	var importState types.String
	importResp.State.GetAttribute(ctx, path.Root("id"), &importState)
	if importState.ValueString() != fileResponseCreate.ID {
		t.Fatalf("expected import ID %s, got %s", fileResponseCreate.ID, importState.ValueString())
	}
}

type fileResourceTransport struct {
	t              *testing.T
	createResponse vapi.FileResponse
	updateResponse vapi.FileResponse
	latestResponse vapi.FileResponse
	postCount      int
}

func (rt *fileResourceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case req.Method == http.MethodPost && req.URL.Path == "/file":
		body, err := io.ReadAll(req.Body)
		if err != nil {
			rt.t.Fatalf("failed to read request body: %v", err)
		}
		_ = req.Body.Close()
		if len(body) == 0 {
			rt.t.Fatalf("expected multipart payload")
		}

		if rt.postCount == 0 {
			rt.latestResponse = rt.createResponse
		} else {
			rt.latestResponse = rt.updateResponse
		}
		rt.postCount++

		return rt.jsonResponse(req, rt.latestResponse, http.StatusOK), nil

	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/file/"):
		id := strings.TrimPrefix(req.URL.Path, "/file/")
		if id != rt.latestResponse.ID {
			return rt.jsonResponse(req, map[string]string{"error": "not found"}, http.StatusNotFound), nil
		}
		return rt.jsonResponse(req, rt.latestResponse, http.StatusOK), nil

	case req.Method == http.MethodDelete && strings.HasPrefix(req.URL.Path, "/file/"):
		id := strings.TrimPrefix(req.URL.Path, "/file/")
		if id != rt.createResponse.ID && id != rt.updateResponse.ID {
			return rt.jsonResponse(req, map[string]string{"error": "not found"}, http.StatusNotFound), nil
		}
		return rt.jsonResponse(req, struct{}{}, http.StatusOK), nil

	default:
		rt.t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		return nil, nil
	}
}

func (rt *fileResourceTransport) jsonResponse(req *http.Request, payload interface{}, status int) *http.Response {
	body, err := json.Marshal(payload)
	if err != nil {
		rt.t.Fatalf("failed to marshal payload: %v", err)
	}

	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    req,
	}
}

func baseFileModel(content, filename types.String) VAPIFileResourceModel {
	return VAPIFileResourceModel{
		Content:      content,
		Filename:     filename,
		Name:         types.StringNull(),
		OriginalName: types.StringNull(),
		Bytes:        types.Int64Null(),
		Mimetype:     types.StringNull(),
		Path:         types.StringNull(),
		URL:          types.StringNull(),
		CreatedAt:    types.StringNull(),
		UpdatedAt:    types.StringNull(),
		Id:           types.StringNull(),
		OrgID:        types.StringNull(),
		Status:       types.StringNull(),
		Bucket:       types.StringNull(),
		Purpose:      types.StringNull(),
	}
}

func mustSetPlan(t *testing.T, schema schema.Schema, model VAPIFileResourceModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), model); diags.HasError() {
		t.Fatalf("plan.Set diagnostics: %v", diags)
	}
	return plan
}

func mustStateModel(t *testing.T, ctx context.Context, state tfsdk.State) VAPIFileResourceModel {
	t.Helper()

	var model VAPIFileResourceModel
	if diags := state.Get(ctx, &model); diags.HasError() {
		t.Fatalf("state.Get diagnostics: %v", diags)
	}
	return model
}
