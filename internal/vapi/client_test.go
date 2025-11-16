package vapi

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestUploadData(t *testing.T) {
	t.Parallel()

	var snapshot multipartSnapshot
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/file" {
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		snapshot = parseMultipartSnapshot(t, body, req.Header.Get("Content-Type"))
		return jsonResponse(http.StatusCreated, `{"id":"abc"}`), nil
	})

	client := &APIClient{
		BaseURL:    "https://api.example.com",
		Token:      "token",
		HTTPClient: &http.Client{Transport: transport},
	}

	body := []byte("content")
	resp, status, err := client.UploadData("file", "example.txt", body)
	if err != nil {
		t.Fatalf("upload error: %v", err)
	}
	if status != http.StatusCreated {
		t.Fatalf("unexpected status %d", status)
	}
	if string(resp) != `{"id":"abc"}` {
		t.Fatalf("unexpected response %s", string(resp))
	}
	if snapshot.filename != "example.txt" || string(snapshot.payload) != string(body) {
		t.Fatalf("unexpected multipart snapshot: %#v", snapshot)
	}
}

func TestSendRequestHandlesErrorStatus(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusInternalServerError, "boom"), nil
	})

	client := &APIClient{
		BaseURL:    "https://api.example.com",
		Token:      "token",
		HTTPClient: &http.Client{Transport: transport},
	}

	_, status, err := client.SendRequest(http.MethodGet, "resource", nil)
	if err == nil {
		t.Fatalf("expected error for non-2xx status")
	}
	if status != http.StatusInternalServerError {
		t.Fatalf("unexpected status %d", status)
	}
}

func TestConvenienceEndpoints(t *testing.T) {
	t.Parallel()

	qt := &queueTransport{t: t}

	qt.enqueue("GET /file/file-1", http.StatusOK, `{"id":"file-1"}`)
	qt.enqueue("DELETE /file/file-1", http.StatusNoContent, ``)
	qt.enqueue("POST /phone-number", http.StatusCreated, `{"id":"pn-1"}`)
	qt.enqueue("DELETE /phone-number/pn", http.StatusOK, ``)
	qt.enqueue("PATCH /phone-number/pn-update", http.StatusOK, `{"id":"pn-update"}`)
	qt.enqueue("POST /tool", http.StatusCreated, `{"id":"tool-1"}`)
	qt.enqueue("GET /tool/tool", http.StatusOK, `{"id":"tool"}`)
	qt.enqueue("PATCH /tool/tool", http.StatusOK, `{"id":"tool"}`)
	qt.enqueue("DELETE /tool/tool", http.StatusOK, ``)
	qt.enqueue("POST /tool", http.StatusOK, `{"id":"tool-2"}`)
	qt.enqueue("PATCH /tool/tool", http.StatusOK, `{"id":"tool"}`)
	qt.enqueue("DELETE /tool/tool", http.StatusOK, ``)
	qt.enqueue("POST /assistant", http.StatusOK, `{"id":"assistant-1"}`)
	qt.enqueue("PATCH /assistant/assistant", http.StatusOK, `{"id":"assistant"}`)
	qt.enqueue("GET /assistant/assistant", http.StatusOK, `{"id":"assistant"}`)
	qt.enqueue("DELETE /assistant/assistant", http.StatusOK, ``)
	qt.enqueue("POST /credential", http.StatusOK, `{"id":"cred-1"}`)
	qt.enqueue("PATCH /credential/trunk", http.StatusOK, `{"id":"trunk"}`)
	qt.enqueue("GET /credential/trunk", http.StatusOK, `{"id":"trunk"}`)
	qt.enqueue("DELETE /credential/trunk", http.StatusOK, ``)
	qt.enqueue("POST /phone-number", http.StatusOK, `{"id":"pn-2"}`)

	client := &APIClient{
		BaseURL:    "https://api.example.com",
		Token:      "token",
		HTTPClient: &http.Client{Transport: qt},
	}

	if _, status, err := client.DeleteFile(""); err != nil || status != http.StatusNotFound {
		t.Fatalf("expected 404 short circuit, got status %d err %v", status, err)
	}

	if _, status, err := client.GetFile("file-1"); err != nil || status < 200 || status >= 300 {
		t.Fatalf("GetFile unexpected status %d err %v", status, err)
	}
	if _, status, err := client.DeleteFile("file-1"); err != nil || status < 200 || status >= 300 {
		t.Fatalf("DeleteFile unexpected status %d err %v", status, err)
	}
	if _, status, err := client.ImportTwilioPhoneNumber(ImportTwilioRequest{}); err != nil || status < 200 || status >= 300 {
		t.Fatalf("ImportTwilioPhoneNumber unexpected status %d err %v", status, err)
	}
	if _, status, err := client.DeletePhoneNumber("pn"); err != nil || status < 200 || status >= 300 {
		t.Fatalf("DeletePhoneNumber unexpected status %d err %v", status, err)
	}
	if _, status, err := client.UpdatePhoneNumber("pn-update", struct{}{}); err != nil || status < 200 || status >= 300 {
		t.Fatalf("UpdatePhoneNumber unexpected status %d err %v", status, err)
	}
	if _, status, err := client.CreateToolFunction(ToolFunctionRequest{}); err != nil || status < 200 || status >= 300 {
		t.Fatalf("CreateToolFunction unexpected status %d err %v", status, err)
	}
	if _, status, err := client.GetToolFunction("tool"); err != nil || status < 200 || status >= 300 {
		t.Fatalf("GetToolFunction unexpected status %d err %v", status, err)
	}
	if _, status, err := client.UpdateToolFunction("tool", ToolFunctionRequest{}); err != nil || status < 200 || status >= 300 {
		t.Fatalf("UpdateToolFunction unexpected status %d err %v", status, err)
	}
	if _, status, err := client.DeleteToolFunction("tool"); err != nil || status < 200 || status >= 300 {
		t.Fatalf("DeleteToolFunction unexpected status %d err %v", status, err)
	}
	if _, status, err := client.CreateToolQueryFunction(ToolQueryFunctionRequest{}); err != nil || status < 200 || status >= 300 {
		t.Fatalf("CreateToolQueryFunction unexpected status %d err %v", status, err)
	}
	if _, status, err := client.UpdateToolQueryFunction("tool", ToolQueryFunctionRequest{}); err != nil || status < 200 || status >= 300 {
		t.Fatalf("UpdateToolQueryFunction unexpected status %d err %v", status, err)
	}
	if _, status, err := client.DeleteToolQueryFunction("tool"); err != nil || status < 200 || status >= 300 {
		t.Fatalf("DeleteToolQueryFunction unexpected status %d err %v", status, err)
	}
	if _, status, err := client.CreateAssistant(CreateAssistantRequest{}); err != nil || status < 200 || status >= 300 {
		t.Fatalf("CreateAssistant unexpected status %d err %v", status, err)
	}
	if _, status, err := client.UpdateAssistant("assistant", CreateAssistantRequest{}); err != nil || status < 200 || status >= 300 {
		t.Fatalf("UpdateAssistant unexpected status %d err %v", status, err)
	}
	if _, status, err := client.GetAssistant("assistant"); err != nil || status < 200 || status >= 300 {
		t.Fatalf("GetAssistant unexpected status %d err %v", status, err)
	}
	if _, status, err := client.DeleteAssistant("assistant"); err != nil || status < 200 || status >= 300 {
		t.Fatalf("DeleteAssistant unexpected status %d err %v", status, err)
	}
	if _, status, err := client.CreateSIPTrunk(ImportSIPTrunkRequest{}); err != nil || status < 200 || status >= 300 {
		t.Fatalf("CreateSIPTrunk unexpected status %d err %v", status, err)
	}
	if _, status, err := client.UpdateSIPTrunk("trunk", ImportSIPTrunkRequest{}); err != nil || status < 200 || status >= 300 {
		t.Fatalf("UpdateSIPTrunk unexpected status %d err %v", status, err)
	}
	if _, status, err := client.GetSIPTrunk("trunk"); err != nil || status < 200 || status >= 300 {
		t.Fatalf("GetSIPTrunk unexpected status %d err %v", status, err)
	}
	if _, status, err := client.DeleteSIPTrunk("trunk"); err != nil || status < 200 || status >= 300 {
		t.Fatalf("DeleteSIPTrunk unexpected status %d err %v", status, err)
	}
	if _, status, err := client.ImportSIPTrunkPhoneNumber(ImportSIPTrunkPhoneNumberRequest{}); err != nil || status < 200 || status >= 300 {
		t.Fatalf("ImportSIPTrunkPhoneNumber unexpected status %d err %v", status, err)
	}

	qt.assertExhausted()
}

type multipartSnapshot struct {
	filename string
	payload  []byte
}

func parseMultipartSnapshot(t *testing.T, body []byte, contentType string) multipartSnapshot {
	reader := multipartReader(t, body, contentType)
	part, err := reader.NextPart()
	if err != nil {
		t.Fatalf("next part: %v", err)
	}
	data, err := io.ReadAll(part)
	if err != nil {
		t.Fatalf("read part: %v", err)
	}
	return multipartSnapshot{filename: part.FileName(), payload: data}
}

func multipartReader(t *testing.T, body []byte, contentType string) *multipart.Reader {
	reader := multipart.NewReader(bytes.NewReader(body), boundaryFromContentType(t, contentType))
	return reader
}

func boundaryFromContentType(t *testing.T, contentType string) string {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("parse media type: %v", err)
	}
	b, ok := params["boundary"]
	if !ok {
		t.Fatalf("missing boundary in content type")
	}
	return b
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type queueTransport struct {
	t     *testing.T
	mu    sync.Mutex
	queue []queueItem
}

type queueItem struct {
	expect string
	status int
	body   string
}

func (qt *queueTransport) enqueue(expect string, status int, body string) {
	qt.queue = append(qt.queue, queueItem{expect: expect, status: status, body: body})
}

func (qt *queueTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	qt.mu.Lock()
	defer qt.mu.Unlock()
	if len(qt.queue) == 0 {
		qt.t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
	}
	item := qt.queue[0]
	qt.queue = qt.queue[1:]
	actual := req.Method + " " + req.URL.Path
	if item.expect != actual {
		qt.t.Fatalf("expected %s got %s", item.expect, actual)
	}
	return jsonResponse(item.status, item.body), nil
}

func (qt *queueTransport) assertExhausted() {
	qt.mu.Lock()
	defer qt.mu.Unlock()
	if len(qt.queue) != 0 {
		qt.t.Fatalf("unhandled requests remaining: %d", len(qt.queue))
	}
}

func jsonResponse(status int, body string) *http.Response {
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
