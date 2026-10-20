package provider

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

type charCase struct {
	Name     string          `json:"name"`
	Op       string          `json:"op"`
	Config   json.RawMessage `json:"config,omitempty"`
	Prior    json.RawMessage `json:"prior_state,omitempty"`
	RawState json.RawMessage `json:"raw_state,omitempty"`
	ImportID string          `json:"import_id,omitempty"`
	API      []stubResponse  `json:"api"`
}

type stubResponse struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Status  int    `json:"status"`
	Body    string `json:"body"`
	Failure string `json:"failure,omitempty"`
}

type charOutcome struct {
	Requests        []recordedRequest `json:"requests"`
	PlannedState    any               `json:"planned_state,omitempty"`
	RequiresReplace []string          `json:"requires_replace,omitempty"`
	PlanApplyDiff   []string          `json:"plan_apply_diff,omitempty"`
	ImportedState   any               `json:"imported_state,omitempty"`
	NewState        any               `json:"new_state"`
	Diagnostics     []recordedDiag    `json:"diagnostics"`
	Unmatched       []recordedRequest `json:"unmatched,omitempty"`
	Problems        []string          `json:"problems,omitempty"`
	Unconsumed      []stubResponse    `json:"unconsumed,omitempty"`
}

type recordedDiag struct {
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
	Detail   string `json:"detail"`
	Path     string `json:"path,omitempty"`
}

type recordedRequest struct {
	Method        string         `json:"method"`
	Path          string         `json:"path"`
	Authorization string         `json:"authorization"`
	ContentType   string         `json:"content_type"`
	Body          string         `json:"body,omitempty"`
	Multipart     []recordedPart `json:"multipart,omitempty"`
}

type recordedPart struct {
	Header   map[string][]string `json:"header"`
	Name     string              `json:"name"`
	Filename string              `json:"filename"`
	Content  string              `json:"content"`
}

type fakeVAPI struct {
	srv *httptest.Server
	mu  sync.Mutex
	// mu guards stubs, requests, unmatched, and problems.
	stubs     []stubResponse
	requests  []recordedRequest
	unmatched []recordedRequest
	problems  []string
}

func newFakeVAPI(t *testing.T, stubs []stubResponse) *fakeVAPI {
	t.Helper()
	api := &fakeVAPI{
		stubs:    slices.Clone(stubs),
		requests: []recordedRequest{},
	}
	api.srv = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.serveHTTP(w, r)
	}))
	api.srv.Config.SetKeepAlivesEnabled(false)
	api.srv.Start()
	t.Cleanup(api.srv.Close)
	return api
}

func (api *fakeVAPI) serveHTTP(w http.ResponseWriter, r *http.Request) {
	stub, request, matched := api.capture(r)
	if !matched {
		w.WriteHeader(http.StatusNotImplemented)
		if _, err := io.WriteString(w, "unmatched request"); err != nil {
			api.recordProblem(request)
		}
		return
	}
	if stub.Failure == "disconnect" {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			api.recordProblem(request)
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			api.recordProblem(request)
			return
		}
		if err := conn.Close(); err != nil {
			api.recordProblem(request)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if stub.Failure == "short_body" {
		w.Header().Set("Content-Length", strconv.Itoa(len(stub.Body)+1))
	}
	w.WriteHeader(stub.Status)
	if _, err := io.WriteString(w, stub.Body); err != nil {
		api.recordProblem(request)
	}
}

func (api *fakeVAPI) capture(r *http.Request) (stubResponse, recordedRequest, bool) {
	request, err := fakeRequest(r)
	api.mu.Lock()
	defer api.mu.Unlock()
	api.requests = append(api.requests, request)
	if err != nil {
		api.problems = append(api.problems, "read request: "+err.Error())
	}
	for i, stub := range api.stubs {
		if stub.Method == r.Method && stub.Path == r.URL.Path {
			api.stubs = slices.Delete(api.stubs, i, i+1)
			return stub, request, true
		}
	}
	api.unmatched = append(api.unmatched, request)
	return stubResponse{}, request, false
}

func fakeRequest(r *http.Request) (recordedRequest, error) {
	request := recordedRequest{
		Method:        r.Method,
		Path:          r.URL.Path,
		Authorization: r.Header.Get("Authorization"),
		ContentType:   r.Header.Get("Content-Type"),
	}
	if r.URL.RawQuery != "" {
		request.Path += "?" + r.URL.RawQuery
	}
	if !strings.HasPrefix(request.ContentType, "multipart/form-data") {
		body, err := io.ReadAll(r.Body)
		request.Body = string(body)
		return request, err
	}
	_, params, err := mime.ParseMediaType(request.ContentType)
	if err != nil {
		return request, err
	}
	request.ContentType = strings.ReplaceAll(request.ContentType, params["boundary"], "{{boundary}}")
	reader := multipart.NewReader(r.Body, params["boundary"])
	for {
		part, err := reader.NextRawPart()
		if errors.Is(err, io.EOF) {
			return request, nil
		}
		if err != nil {
			return request, err
		}
		body, err := io.ReadAll(part)
		_, disposition, headerErr := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if headerErr != nil {
			return request, headerErr
		}
		request.Multipart = append(request.Multipart, recordedPart{
			Header:   http.Header(part.Header).Clone(),
			Name:     disposition["name"],
			Filename: disposition["filename"],
			Content:  string(body),
		})
		if err != nil {
			return request, err
		}
	}
}

func (api *fakeVAPI) recordProblem(request recordedRequest) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.problems = append(api.problems, "response failure for "+request.Method+" "+request.Path)
}

func (api *fakeVAPI) snapshot() ([]recordedRequest, []recordedRequest, []string, []stubResponse) {
	api.mu.Lock()
	defer api.mu.Unlock()
	return fakeCloneRequests(api.requests), fakeCloneRequests(api.unmatched), slices.Clone(api.problems), slices.Clone(api.stubs)
}

func fakeCloneRequests(requests []recordedRequest) []recordedRequest {
	result := slices.Clone(requests)
	for i := range result {
		result[i].Multipart = slices.Clone(result[i].Multipart)
		for j := range result[i].Multipart {
			result[i].Multipart[j].Header = http.Header(result[i].Multipart[j].Header).Clone()
		}
	}
	return result
}

type harness struct {
	t       *testing.T
	server  tfprotov6.ProviderServer
	api     *fakeVAPI
	schemas map[string]*tfprotov6.Schema
}

func newHarness(t *testing.T, stubs []stubResponse) *harness {
	t.Helper()
	h := &harness{
		t:   t,
		api: newFakeVAPI(t, stubs),
	}
	var err error
	h.server, err = providerserver.NewProtocol6WithError(New("test")())()
	require.NoError(t, err)
	schemas, err := h.server.GetProviderSchema(t.Context(), &tfprotov6.GetProviderSchemaRequest{})
	require.NoError(t, err)
	require.Empty(t, schemas.Diagnostics)
	h.schemas = schemas.ResourceSchemas
	config, err := json.Marshal(map[string]string{"url": h.api.srv.URL, "token": "test-token"})
	require.NoError(t, err)
	configured, err := h.server.ConfigureProvider(t.Context(), &tfprotov6.ConfigureProviderRequest{
		Config: harnessDynamic(t, jsonToValue(t, schemas.Provider.ValueType(), config)),
	})
	require.NoError(t, err)
	require.Empty(t, configured.Diagnostics)
	return h
}

func (h *harness) run(typeName string, c charCase) charOutcome {
	h.t.Helper()
	require.Contains(h.t, h.schemas, typeName)
	out := charOutcome{Diagnostics: []recordedDiag{}}
	switch c.Op {
	case "plan_create", "plan_update":
		h.planCase(typeName, c, &out)
	case "validate":
		h.validateCase(typeName, c, &out)
	case "create", "update":
		h.applyCase(typeName, c, &out)
	case "read":
		h.readCase(typeName, h.caseValue(typeName, c.Prior), nil, &out)
	case "delete":
		h.deleteCase(typeName, c, &out)
	case "import":
		h.importCase(typeName, c, &out)
	case "upgrade":
		h.upgradeCase(typeName, c, &out)
	default:
		h.t.Fatalf("unknown operation %q", c.Op)
	}
	out.Requests, out.Unmatched, out.Problems, out.Unconsumed = h.api.snapshot()
	for i := range out.Requests {
		out.Requests[i].Path = h.normalize(out.Requests[i].Path)
	}
	for i := range out.Unmatched {
		out.Unmatched[i].Path = h.normalize(out.Unmatched[i].Path)
	}
	for i := range out.Problems {
		out.Problems[i] = h.normalize(out.Problems[i])
	}
	return out
}

func harnessDynamic(t *testing.T, value tftypes.Value) *tfprotov6.DynamicValue {
	t.Helper()
	dynamic, err := tfprotov6.NewDynamicValue(value.Type(), value)
	require.NoError(t, err)
	return &dynamic
}

func (h *harness) caseValue(typeName string, raw json.RawMessage) *tfprotov6.DynamicValue {
	return harnessDynamic(h.t, jsonToValue(h.t, h.schemas[typeName].ValueType(), raw))
}

func (h *harness) decode(typeName string, dynamic *tfprotov6.DynamicValue) tftypes.Value {
	if dynamic == nil {
		return tftypes.NewValue(h.schemas[typeName].ValueType(), nil)
	}
	value, err := dynamic.Unmarshal(h.schemas[typeName].ValueType())
	require.NoError(h.t, err)
	return value
}

func (h *harness) normalize(text string) string {
	return strings.ReplaceAll(text, h.api.srv.URL, "{{api}}")
}

func (h *harness) diagnostics(out *charOutcome, diagnostics []*tfprotov6.Diagnostic) bool {
	failed := false
	for _, diag := range diagnostics {
		severity := "warning"
		if diag.Severity == tfprotov6.DiagnosticSeverityError {
			severity, failed = "error", true
		}
		path := ""
		if diag.Attribute != nil {
			path = harnessPath(h.t, diag.Attribute)
		}
		out.Diagnostics = append(out.Diagnostics, recordedDiag{
			Severity: severity,
			Summary:  diag.Summary,
			Detail:   h.normalize(diag.Detail),
			Path:     path,
		})
	}
	return failed
}

func (h *harness) planCase(typeName string, c charCase, out *charOutcome) (*tfprotov6.PlanResourceChangeResponse, bool) {
	schema := h.schemas[typeName]
	prior := jsonToValue(h.t, schema.ValueType(), c.Prior)
	config := jsonToValue(h.t, schema.ValueType(), c.Config)
	response, err := h.server.PlanResourceChange(h.t.Context(), &tfprotov6.PlanResourceChangeRequest{
		TypeName:         typeName,
		PriorState:       harnessDynamic(h.t, prior),
		Config:           harnessDynamic(h.t, config),
		ProposedNewState: harnessDynamic(h.t, proposedNewState(h.t, schema, prior, config)),
	})
	require.NoError(h.t, err)
	failed := h.diagnostics(out, response.Diagnostics)
	out.PlannedState = valueToJSON(h.t, h.decode(typeName, response.PlannedState))
	for _, path := range response.RequiresReplace {
		out.RequiresReplace = append(out.RequiresReplace, harnessPath(h.t, path))
	}
	sort.Strings(out.RequiresReplace)
	return response, failed
}

func (h *harness) validateCase(typeName string, c charCase, out *charOutcome) {
	response, err := h.server.ValidateResourceConfig(h.t.Context(), &tfprotov6.ValidateResourceConfigRequest{
		TypeName: typeName,
		Config:   h.caseValue(typeName, c.Config),
	})
	require.NoError(h.t, err)
	h.diagnostics(out, response.Diagnostics)
}

func (h *harness) applyCase(typeName string, c charCase, out *charOutcome) {
	plan, failed := h.planCase(typeName, c, out)
	if failed {
		return
	}
	response, err := h.server.ApplyResourceChange(h.t.Context(), &tfprotov6.ApplyResourceChangeRequest{
		TypeName:       typeName,
		PriorState:     h.caseValue(typeName, c.Prior),
		Config:         h.caseValue(typeName, c.Config),
		PlannedState:   plan.PlannedState,
		PlannedPrivate: plan.PlannedPrivate,
	})
	require.NoError(h.t, err)
	failed = h.diagnostics(out, response.Diagnostics)
	applied := h.decode(typeName, response.NewState)
	out.NewState = valueToJSON(h.t, applied)
	if !failed {
		out.PlanApplyDiff = harnessKnownDiff(h.t, h.decode(typeName, plan.PlannedState), applied)
	}
}

func (h *harness) readCase(typeName string, state *tfprotov6.DynamicValue, private []byte, out *charOutcome) {
	response, err := h.server.ReadResource(h.t.Context(), &tfprotov6.ReadResourceRequest{
		TypeName:     typeName,
		CurrentState: state,
		Private:      private,
	})
	require.NoError(h.t, err)
	h.diagnostics(out, response.Diagnostics)
	out.NewState = valueToJSON(h.t, h.decode(typeName, response.NewState))
}

func (h *harness) deleteCase(typeName string, c charCase, out *charOutcome) {
	response, err := h.server.ApplyResourceChange(h.t.Context(), &tfprotov6.ApplyResourceChangeRequest{
		TypeName:     typeName,
		PriorState:   h.caseValue(typeName, c.Prior),
		Config:       h.caseValue(typeName, nil),
		PlannedState: h.caseValue(typeName, nil),
	})
	require.NoError(h.t, err)
	h.diagnostics(out, response.Diagnostics)
	out.NewState = valueToJSON(h.t, h.decode(typeName, response.NewState))
}

func (h *harness) importCase(typeName string, c charCase, out *charOutcome) {
	response, err := h.server.ImportResourceState(h.t.Context(), &tfprotov6.ImportResourceStateRequest{
		TypeName: typeName,
		ID:       c.ImportID,
	})
	require.NoError(h.t, err)
	if h.diagnostics(out, response.Diagnostics) || len(response.ImportedResources) == 0 {
		return
	}
	require.Len(h.t, response.ImportedResources, 1)
	imported := response.ImportedResources[0]
	out.ImportedState = valueToJSON(h.t, h.decode(imported.TypeName, imported.State))
	h.readCase(imported.TypeName, imported.State, imported.Private, out)
}

func (h *harness) upgradeCase(typeName string, c charCase, out *charOutcome) {
	response, err := h.server.UpgradeResourceState(h.t.Context(), &tfprotov6.UpgradeResourceStateRequest{
		TypeName: typeName,
		Version:  0,
		RawState: &tfprotov6.RawState{JSON: c.RawState},
	})
	require.NoError(h.t, err)
	h.diagnostics(out, response.Diagnostics)
	out.NewState = valueToJSON(h.t, h.decode(typeName, response.UpgradedState))
}

func harnessKnownDiff(t *testing.T, planned, applied tftypes.Value) []string {
	t.Helper()
	diffs, err := planned.Diff(applied)
	require.NoError(t, err)
	paths := []string{}
	for _, diff := range diffs {
		if diff.Value1 != nil && diff.Value1.IsFullyKnown() {
			paths = append(paths, harnessPath(t, diff.Path))
		}
	}
	sort.Strings(paths)
	return slices.Compact(paths)
}

func harnessPath(t *testing.T, path *tftypes.AttributePath) string {
	t.Helper()
	parts := []string{}
	for _, step := range path.Steps() {
		switch step := step.(type) {
		case tftypes.AttributeName:
			parts = append(parts, string(step))
		case tftypes.ElementKeyString:
			parts = append(parts, string(step))
		case tftypes.ElementKeyInt:
			parts = append(parts, strconv.FormatInt(int64(step), 10))
		case tftypes.ElementKeyValue:
			encoded, err := json.Marshal(valueToJSON(t, tftypes.Value(step)))
			require.NoError(t, err)
			parts = append(parts, string(encoded))
		default:
			t.Fatalf("unsupported path step %T", step)
		}
	}
	return strings.Join(parts, ".")
}

func fakeClient(t *testing.T) *http.Client {
	t.Helper()
	transport := &http.Transport{}
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{Transport: transport}
}

func fakeDo(t *testing.T, client *http.Client, method, url, body string) (*http.Response, error) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), method, url, strings.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Content-Type", "application/json")
	return client.Do(request)
}

func fakeRead(t *testing.T, response *http.Response) ([]byte, error) {
	t.Helper()
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
	return io.ReadAll(response.Body)
}

func TestHarness_RequestAndFailure(t *testing.T) {
	t.Parallel()
	t.Run("records_method_path_headers_and_raw_body", func(t *testing.T) {
		t.Parallel()
		api := newFakeVAPI(t, []stubResponse{{
			Method: "POST",
			Path:   "/file",
			Status: 201,
			Body:   `{}`,
		}})
		body := " {\"raw\":true}\n"
		response, err := fakeDo(t, fakeClient(t), "POST", api.srv.URL+"/file?x=1&x=2", body)
		require.NoError(t, err)
		_, err = fakeRead(t, response)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode)
		requests, unmatched, problems, remaining := api.snapshot()
		require.Equal(t, []recordedRequest{{
			Method:        "POST",
			Path:          "/file?x=1&x=2",
			Authorization: "Bearer test-token",
			ContentType:   "application/json",
			Body:          body,
		}}, requests)
		require.Empty(t, unmatched)
		require.Empty(t, problems)
		require.Empty(t, remaining)
	})
	t.Run("disconnect_records_request_then_client_error", func(t *testing.T) {
		t.Parallel()
		api := newFakeVAPI(t, []stubResponse{{
			Method:  "POST",
			Path:    "/file",
			Failure: "disconnect",
		}})
		response, err := fakeDo(t, fakeClient(t), "POST", api.srv.URL+"/file", "payload")
		require.Error(t, err)
		require.Nil(t, response)
		requests, unmatched, problems, remaining := api.snapshot()
		require.Len(t, requests, 1)
		require.Equal(t, "payload", requests[0].Body)
		require.Empty(t, unmatched)
		require.Empty(t, problems)
		require.Empty(t, remaining)
	})
	t.Run("short_body_returns_status_then_read_error", func(t *testing.T) {
		t.Parallel()
		api := newFakeVAPI(t, []stubResponse{{
			Method:  "GET",
			Path:    "/file",
			Status:  202,
			Body:    "partial",
			Failure: "short_body",
		}})
		response, err := fakeDo(t, fakeClient(t), "GET", api.srv.URL+"/file", "")
		require.NoError(t, err)
		require.Equal(t, 202, response.StatusCode)
		body, err := fakeRead(t, response)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
		require.Equal(t, "partial", string(body))
		requests, unmatched, problems, remaining := api.snapshot()
		require.Len(t, requests, 1)
		require.Empty(t, unmatched)
		require.Empty(t, problems)
		require.Empty(t, remaining)
	})
	t.Run("get_after_prior_200_records_exactly_one_request", func(t *testing.T) {
		t.Parallel()
		api := newFakeVAPI(t, []stubResponse{
			{
				Method: "GET",
				Path:   "/prime",
				Status: 200,
				Body:   `{}`,
			},
			{
				Method:  "GET",
				Path:    "/file",
				Failure: "disconnect",
			},
		})
		client := fakeClient(t)
		response, err := fakeDo(t, client, "GET", api.srv.URL+"/prime", "")
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode)
		_, err = fakeRead(t, response)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		response, err = fakeDo(t, client, "GET", api.srv.URL+"/file", "")
		if response != nil {
			_, readErr := fakeRead(t, response)
			require.NoError(t, readErr)
		}
		require.Error(t, err)
		requests, unmatched, problems, remaining := api.snapshot()
		require.Equal(t, []string{"/prime", "/file"}, fakeRequestPaths(requests))
		require.Empty(t, unmatched)
		require.Empty(t, problems)
		require.Empty(t, remaining)
	})
}

func fakeRequestPaths(requests []recordedRequest) []string {
	paths := make([]string, len(requests))
	for i, request := range requests {
		paths[i] = request.Path
	}
	return paths
}

func TestHarness_FakeVAPI_WhenNoStubMatches_ReturnsFailureStatusAndRecordsUnmatched(t *testing.T) {
	t.Parallel()
	api := newFakeVAPI(t, []stubResponse{{
		Method: "POST",
		Path:   "/file",
		Status: 200,
	}})
	response, err := fakeDo(t, fakeClient(t), "GET", api.srv.URL+"/missing?x=1", "")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotImplemented, response.StatusCode)
	body, err := fakeRead(t, response)
	require.NoError(t, err)
	require.Equal(t, "unmatched request", string(body))
	requests, unmatched, problems, remaining := api.snapshot()
	require.Equal(t, []recordedRequest{{
		Method:        "GET",
		Path:          "/missing?x=1",
		Authorization: "Bearer test-token",
		ContentType:   "application/json",
	}}, requests)
	require.Equal(t, requests, unmatched)
	require.Empty(t, problems)
	require.Equal(t, []stubResponse{{
		Method: "POST",
		Path:   "/file",
		Status: 200,
	}}, remaining)
}
