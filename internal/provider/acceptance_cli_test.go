package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type assistantCLITools struct {
	terraform string
	config    string
}

type assistantCLI struct {
	tools assistantCLITools
	dir   string
	env   []string
	api   *assistantCLIServer
}

type assistantCLIServer struct {
	srv *httptest.Server
	// mu guards assistant, requests, problems and disagree.
	mu        sync.Mutex
	assistant map[string]any
	requests  []recordedRequest
	problems  []string
	disagree  bool
}

func runAssistantReasoningAcceptance(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("set TF_ACC=1 to run Terraform CLI acceptance")
	}
	tools := buildAssistantCLITools(t)
	cli := newAssistantCLI(t, tools)
	for _, step := range []struct {
		name   string
		effort string
		action string
		method string
	}{
		{"fresh_unset", "", "create", http.MethodPost},
		{"set_none", "none", "update", http.MethodPatch},
		{"high", "high", "update", http.MethodPatch},
	} {
		t.Run(step.name, func(t *testing.T) {
			cli.configure(t, step.effort, true)
			cli.plan(t, step.action, step.effort)
			cli.mustRun(t, "apply", "-auto-approve", "-no-color", "change.tfplan")
			cli.api.assertWrite(t, step.method, step.effort)
			cli.assertState(t, step.effort, true)
			cli.plan(t, "no-op", step.effort)
		})
	}
	runAssistantLegacyAcceptance(t, tools)
	t.Run("invalid", func(t *testing.T) { checkAssistantInvalid(t, tools) })
	t.Run("apply_disagreement", func(t *testing.T) { checkAssistantDisagreement(t, tools) })
	t.Run("remove", func(t *testing.T) {
		cli.configure(t, "", true)
		cli.plan(t, "update", "")
		cli.mustRun(t, "apply", "-auto-approve", "-no-color", "change.tfplan")
		cli.api.assertWrite(t, http.MethodPatch, "")
		require.Equal(t, "high", cli.api.storedEffort(t))
		cli.assertState(t, "", true)
		cli.plan(t, "no-op", "")
	})
}

func buildAssistantCLITools(t *testing.T) assistantCLITools {
	t.Helper()
	terraform, err := exec.LookPath("terraform")
	require.NoError(t, err, "TF_ACC=1 requires Terraform on PATH")
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	binDir := t.TempDir()
	cmd := exec.CommandContext(t.Context(), "go", "build")
	cmd.Args = append(cmd.Args, "-o", filepath.Join(binDir, "terraform-provider-vapi"), ".")
	cmd.Dir = root
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "provider build failed: %s", output)
	config := filepath.Join(t.TempDir(), "terraform.rc")
	writeAssistantFile(t, config, []byte(fmt.Sprintf(`provider_installation {
  dev_overrides { "registry.terraform.io/kirillve/vapi" = %q }
  direct { exclude = ["*/*"] }
}
`, binDir)))
	return assistantCLITools{terraform: terraform, config: config}
}

func newAssistantCLI(t *testing.T, tools assistantCLITools) *assistantCLI {
	t.Helper()
	cli := &assistantCLI{tools: tools, dir: t.TempDir(), api: newAssistantCLIServer(t)}
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "TF_") && !strings.HasPrefix(entry, "CHECKPOINT_DISABLE=") {
			cli.env = append(cli.env, entry)
		}
	}
	cli.env = append(cli.env, "TF_CLI_CONFIG_FILE="+tools.config, "TF_DATA_DIR="+t.TempDir(),
		"TF_IN_AUTOMATION=1", "TF_INPUT=0", "CHECKPOINT_DISABLE=1")
	return cli
}

func (cli *assistantCLI) configure(t *testing.T, effort string, model bool) {
	t.Helper()
	modelConfig := ""
	if model {
		attribute := ""
		if effort != "" {
			attribute = fmt.Sprintf("reasoning_effort = %q", effort)
		}
		modelConfig = fmt.Sprintf(`model = {
    model = "gpt-6-luna"
    provider = "openai"
    system_prompt = "legacy prompt"
    temperature = 0.3
    max_tokens = 512
    tool_ids = []
    %s
  }`, attribute)
	}
	config := fmt.Sprintf(`terraform {
  required_providers {
    vapi = { source = "kirillve/vapi" }
  }
}
provider "vapi" {
  url = %q
  token = "test-token"
}
resource "vapi_assistant" "reasoning" {
  name = "reasoning-assistant"
  %s
}
`, cli.api.srv.URL, modelConfig)
	writeAssistantFile(t, filepath.Join(cli.dir, "main.tf"), []byte(config))
}

func (cli *assistantCLI) run(t *testing.T, args ...string) (string, int) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "terraform")
	cmd.Path = cli.tools.terraform
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = cli.dir
	cmd.Env = cli.env
	output, err := cmd.CombinedOutput()
	if err == nil {
		return string(output), 0
	}
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit, "Terraform could not start: %s", output)
	return string(output), exit.ExitCode()
}

func (cli *assistantCLI) mustRun(t *testing.T, args ...string) string {
	t.Helper()
	output, code := cli.run(t, args...)
	require.Equal(t, 0, code, "%s: %s", args, output)
	return output
}

func (cli *assistantCLI) plan(t *testing.T, action, effort string) {
	t.Helper()
	output, code := cli.run(t, "plan", "-no-color", "-detailed-exitcode", "-out=change.tfplan")
	wantExit := 2
	if action == "no-op" {
		wantExit = 0
	}
	require.Equal(t, wantExit, code, "Terraform plan: %s", output)
	var plan struct {
		ResourceChanges []struct {
			Address string `json:"address"`
			Change  struct {
				Actions []string       `json:"actions"`
				After   map[string]any `json:"after"`
			} `json:"change"`
		} `json:"resource_changes"`
	}
	require.NoError(t, json.Unmarshal([]byte(cli.mustRun(t, "show", "-json", "change.tfplan")), &plan))
	for _, change := range plan.ResourceChanges {
		if change.Address == "vapi_assistant.reasoning" {
			require.Equal(t, []string{action}, change.Change.Actions)
			assertAssistantEffort(t, change.Change.After, effort)
			return
		}
	}
	t.Fatal("saved plan has no vapi_assistant.reasoning change")
}

func assertAssistantEffort(t *testing.T, attributes map[string]any, effort string) {
	t.Helper()
	if attributes["model"] == nil {
		require.Empty(t, effort)
		return
	}
	model := assistantObject(t, attributes["model"])
	require.Contains(t, model, "reasoning_effort")
	if effort == "" {
		require.Nil(t, model["reasoning_effort"])
	} else {
		require.Equal(t, effort, model["reasoning_effort"])
	}
}

func (cli *assistantCLI) assertState(t *testing.T, effort string, model bool) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(cli.dir, "terraform.tfstate"))
	require.NoError(t, err)
	var state struct {
		Resources []struct {
			Instances []struct {
				Attributes map[string]any `json:"attributes"`
			} `json:"instances"`
		} `json:"resources"`
	}
	require.NoError(t, json.Unmarshal(raw, &state))
	require.Len(t, state.Resources, 1)
	require.Len(t, state.Resources[0].Instances, 1)
	attributes := state.Resources[0].Instances[0].Attributes
	require.Equal(t, "reasoning-assistant", attributes["name"])
	if model {
		got := assistantObject(t, attributes["model"])
		require.Equal(t, "gpt-6-luna", got["model"])
		require.Equal(t, "legacy prompt", got["system_prompt"])
		require.Equal(t, "openai", got["provider"])
		require.Equal(t, 0.3, got["temperature"])
		require.Equal(t, float64(512), got["max_tokens"])
		require.Equal(t, []any{}, got["tool_ids"])
	} else {
		require.Nil(t, attributes["model"])
	}
	assertAssistantEffort(t, attributes, effort)
}

func checkAssistantInvalid(t *testing.T, tools assistantCLITools) {
	t.Helper()
	cli := newAssistantCLI(t, tools)
	cli.configure(t, "ultra", true)
	output, code := cli.run(t, "plan", "-no-color")
	require.NotZero(t, code, "%s", output)
	require.Contains(t, strings.Join(strings.Fields(output), " "), "model.reasoning_effort")
	cli.api.mu.Lock()
	defer cli.api.mu.Unlock()
	require.Empty(t, cli.api.requests, "invalid configuration must not reach VAPI")
}

func checkAssistantDisagreement(t *testing.T, tools assistantCLITools) {
	t.Helper()
	cli := newAssistantCLI(t, tools)
	cli.api.mu.Lock()
	cli.api.disagree = true
	cli.api.mu.Unlock()
	cli.configure(t, "none", true)
	output, code := cli.run(t, "apply", "-auto-approve", "-no-color")
	require.NotZero(t, code, "%s", output)
	require.Contains(t, strings.ToLower(strings.Join(strings.Fields(output), " ")), "inconsistent result after apply")
	cli.api.assertWrite(t, http.MethodPost, "none")
}

func runAssistantLegacyAcceptance(t *testing.T, tools assistantCLITools) {
	t.Helper()
	cases := loadCases(t, "testdata/v1/vapi_assistant/reasoning.cases.json")
	for _, version := range []string{"v0_13_0", "v0_15_1"} {
		t.Run("legacy_"+version, func(t *testing.T) {
			for _, variant := range []string{"", "_no_model"} {
				t.Run("state"+variant, func(t *testing.T) {
					name := "upgrade_" + version + "_state" + variant
					index := slices.IndexFunc(cases, func(c charCase) bool { return c.Name == name })
					require.NotEqual(t, -1, index, "missing historical fixture %s", name)
					cli := newAssistantCLI(t, tools)
					cli.seedLegacy(t, cases[index].RawState)
					cli.configure(t, "", variant == "")
					cli.plan(t, "no-op", "")
					cli.mustRun(t, "apply", "-auto-approve", "-no-color", "change.tfplan")
					cli.assertState(t, "", variant == "")
					cli.plan(t, "no-op", "")
				})
			}
		})
	}
}

func (cli *assistantCLI) seedLegacy(t *testing.T, raw json.RawMessage) {
	t.Helper()
	var attributes map[string]any
	require.NoError(t, json.Unmarshal(raw, &attributes))
	if attributes["model"] != nil {
		require.NotContains(t, assistantObject(t, attributes["model"]), "reasoning_effort")
	}
	state := map[string]any{
		"version": 4, "terraform_version": "1.0.0", "serial": 1,
		"lineage": "11111111-1111-4111-8111-abcdefabcdef",
		"outputs": map[string]any{},
		"resources": []any{map[string]any{
			"mode": "managed", "type": "vapi_assistant", "name": "reasoning",
			"provider":  `provider["registry.terraform.io/kirillve/vapi"]`,
			"instances": []any{map[string]any{"schema_version": 0, "attributes": raw}},
		}},
	}
	encoded, err := json.Marshal(state)
	require.NoError(t, err)
	writeAssistantFile(t, filepath.Join(cli.dir, "terraform.tfstate"), encoded)
	assistant := map[string]any{"id": attributes["id"], "orgId": attributes["org_id"], "name": attributes["name"]}
	if attributes["model"] != nil {
		model := assistantObject(t, attributes["model"])
		assistant["model"] = map[string]any{
			"model": model["model"], "provider": model["provider"], "systemPrompt": model["system_prompt"],
			"maxTokens": model["max_tokens"], "temperature": model["temperature"], "toolIds": model["tool_ids"],
		}
	}
	cli.api.mu.Lock()
	cli.api.assistant = assistant
	cli.api.mu.Unlock()
}

func writeAssistantFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, contents, 0o600))
}

func assistantObject(t *testing.T, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	require.True(t, ok, "expected JSON object, got %T", value)
	return object
}

func newAssistantCLIServer(t *testing.T) *assistantCLIServer {
	t.Helper()
	api := &assistantCLIServer{}
	api.srv = httptest.NewUnstartedServer(http.HandlerFunc(api.serveHTTP))
	api.srv.Config.SetKeepAlivesEnabled(false)
	api.srv.Start()
	t.Cleanup(func() {
		api.srv.Close()
		api.mu.Lock()
		defer api.mu.Unlock()
		require.Empty(t, api.problems)
	})
	return api
}

func (api *assistantCLIServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()
	body, err := io.ReadAll(r.Body)
	api.requests = append(api.requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Body: string(body)})
	if err != nil {
		api.problems = append(api.problems, err.Error())
		http.Error(w, "body read failed", http.StatusBadRequest)
		return
	}
	if r.Header.Get("Authorization") != "Bearer test-token" {
		api.problems = append(api.problems, "unexpected authorization")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	status := api.route(r.Method, r.URL.Path, body)
	if status != http.StatusOK {
		http.Error(w, http.StatusText(status), status)
		return
	}
	response := api.response()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		api.problems = append(api.problems, err.Error())
	}
}

// route runs with mu held and merges only members included in a write.
func (api *assistantCLIServer) route(method, path string, body []byte) int {
	if method == http.MethodPost && path == "/assistant" {
		api.assistant = map[string]any{"id": "asst-1", "orgId": "org-1"}
		return api.merge(body)
	}
	id, ok := api.assistant["id"].(string)
	if !ok || path != "/assistant/"+id {
		return http.StatusNotFound
	}
	switch method {
	case http.MethodGet:
		return http.StatusOK
	case http.MethodPatch:
		return api.merge(body)
	case http.MethodDelete:
		api.assistant = nil
		return http.StatusOK
	default:
		api.problems = append(api.problems, "unexpected method: "+method)
		return http.StatusMethodNotAllowed
	}
}

// merge runs with mu held; omitting reasoningEffort preserves its stored value.
func (api *assistantCLIServer) merge(body []byte) int {
	var patch map[string]any
	if err := json.Unmarshal(body, &patch); err != nil {
		api.problems = append(api.problems, err.Error())
		return http.StatusBadRequest
	}
	for key, value := range patch {
		incoming, model := value.(map[string]any)
		if key == "model" && model {
			stored, ok := api.assistant[key].(map[string]any)
			if !ok {
				stored = map[string]any{}
			}
			for member, field := range incoming {
				stored[member] = field
			}
			value = stored
		}
		api.assistant[key] = value
	}
	return http.StatusOK
}

// response runs with mu held; defaults and disagreement affect only the response.
func (api *assistantCLIServer) response() map[string]any {
	response := make(map[string]any, len(api.assistant))
	for key, value := range api.assistant {
		response[key] = value
	}
	stored, ok := api.assistant["model"].(map[string]any)
	if !ok {
		stored = map[string]any{}
	}
	model := make(map[string]any, len(stored)+1)
	for key, value := range stored {
		model[key] = value
	}
	if _, exists := model["reasoningEffort"]; !exists {
		model["reasoningEffort"] = "none"
	}
	if api.disagree {
		model["reasoningEffort"] = "low"
	}
	response["model"] = model
	return response
}

func (api *assistantCLIServer) assertWrite(t *testing.T, method, effort string) {
	t.Helper()
	api.mu.Lock()
	requests := slices.Clone(api.requests)
	api.mu.Unlock()
	slices.Reverse(requests)
	index := slices.IndexFunc(requests, func(r recordedRequest) bool { return r.Method == method })
	require.NotEqual(t, -1, index, "missing %s request", method)
	request := requests[index]
	path := "/assistant/asst-1"
	if method == http.MethodPost {
		path = "/assistant"
	}
	require.Equal(t, path, request.Path)
	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(request.Body), &body))
	model := assistantObject(t, body["model"])
	if effort == "" {
		require.NotContains(t, model, "reasoningEffort")
	} else {
		require.Equal(t, effort, model["reasoningEffort"])
	}
	require.Equal(t, "gpt-6-luna", model["model"])
	require.Equal(t, "legacy prompt", model["systemPrompt"])
}

func (api *assistantCLIServer) storedEffort(t *testing.T) any {
	t.Helper()
	api.mu.Lock()
	defer api.mu.Unlock()
	return assistantObject(t, api.assistant["model"])["reasoningEffort"]
}
