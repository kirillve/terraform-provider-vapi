// Package vapi is the frozen v1 HTTP client for the VAPI.AI REST API and its JSON DTOs.
//
// In scope: bearer-authenticated JSON requests (SendRequest, Do), multipart upload
// (UploadData), typed non-2xx/body-read errors (APIError, StatusCode), typed decode errors
// (DecodeError), and the v1 payload/response structs whose JSON encoding is a frozen wire
// contract pinned by the internal/provider goldens.
// Out of scope: Terraform types, retries, context propagation, logging of bodies.
// v1 only: the v2 resource set uses the official VAPI SDK and MUST NOT import this package.
// Tier 1.
package vapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
)

// APIClient handles communication with the remote provider.
type APIClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// UploadData Uploads a file using multipart/form-data.
func (c *APIClient) UploadData(fieldName, filename string, content []byte) ([]byte, int, error) {
	// Create a buffer to write our multipart data into
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create the file field in the multipart form
	part, err := writer.CreatePart(
		textproto.MIMEHeader{
			"Content-Disposition": []string{fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, filepath.Base(filename))},
			"Content-Type":        []string{getMimeType(filename)},
		},
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error creating form file: %v", err)
	}

	// Write the file content to the form field
	_, err = part.Write(content)
	if err != nil {
		return nil, 0, fmt.Errorf("error writing file content: %v", err)
	}

	// Close the multipart writer to finalize the request body
	err = writer.Close()
	if err != nil {
		return nil, 0, fmt.Errorf("error closing writer: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", c.BaseURL+"/file", body)
	if err != nil {
		return nil, 0, fmt.Errorf("error creating request: %v", err)
	}

	// Set the headers
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error sending request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read the response body
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("error reading response body: %v", err)
	}

	// Return the response body and status code
	return responseData, resp.StatusCode, nil
}

// SendRequest sends one JSON request and returns the response body and HTTP status. A non-2xx
// status and a body-read failure are returned as *APIError; transport, encode and request-build
// failures are plain wrapped errors with status 0.
func (c *APIClient) SendRequest(method, endpoint string, body interface{}) ([]byte, int, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, 0, fmt.Errorf("failed to encode request body: %w", err)
		}
	}

	req, err := http.NewRequest(method, c.BaseURL+"/"+endpoint, &buf)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	responseData, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, &APIError{
			StatusCode: resp.StatusCode,
			Err:        fmt.Errorf("error reading response body: %w", readErr),
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseData, resp.StatusCode, &APIError{StatusCode: resp.StatusCode, Body: string(responseData)}
	}

	return responseData, resp.StatusCode, nil
}

// Do sends one JSON request via SendRequest. On a 2xx status with out != nil it unmarshals the
// response body into out; a decode failure is returned as *DecodeError and out keeps the partial
// value. With out == nil the response body is discarded.
func (c *APIClient) Do(method, endpoint string, in, out any) error {
	body, _, err := c.SendRequest(method, endpoint, in)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return &DecodeError{Err: err}
	}
	return nil
}

// APIError is returned for a non-2xx response and for a failure while reading the response body.
type APIError struct {
	StatusCode int
	Body       string
	Err        error // set only for a body-read failure
}

// Error returns Err's text when Err is set, else "HTTP <status>: <body>".
func (e *APIError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// Unwrap returns the body-read error, or nil.
func (e *APIError) Unwrap() error { return e.Err }

// DecodeError is returned when a 2xx response body is not valid JSON for the target value.
type DecodeError struct {
	Err error
}

// Error returns the decoder's text verbatim.
func (e *DecodeError) Error() string { return e.Err.Error() }

// Unwrap returns the decoder error.
func (e *DecodeError) Unwrap() error { return e.Err }

// StatusCode returns the HTTP status carried by an *APIError in err's chain, or 0 when there is
// none (nil, transport failure, decode failure).
func StatusCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

// GetFile retrieves the details of a specific phone number by ID.
func (c *APIClient) GetFile(id string) ([]byte, int, error) {
	if len(id) == 0 {
		return []byte{}, 404, nil
	}
	endpoint := fmt.Sprintf("file/%s", id)
	return c.SendRequest("GET", endpoint, nil)
}

// DeleteFile deletes a specific phone number by ID.
func (c *APIClient) DeleteFile(id string) ([]byte, int, error) {
	if len(id) == 0 {
		return []byte{}, 404, nil
	}
	endpoint := fmt.Sprintf("file/%s", id)
	return c.SendRequest("DELETE", endpoint, nil)
}

// ImportTwilioPhoneNumber requests the creation of a new phone number.
func (c *APIClient) ImportTwilioPhoneNumber(requestData ImportTwilioRequest) ([]byte, int, error) {
	return c.SendRequest("POST", "phone-number", requestData)
}

// GetPhoneNumber retrieves the details of a specific phone number by ID.
func (c *APIClient) GetPhoneNumber(id string) ([]byte, int, error) {
	if len(id) == 0 {
		return []byte{}, 404, nil
	}
	endpoint := fmt.Sprintf("phone-number/%s", id)
	return c.SendRequest("GET", endpoint, nil)
}

// DeletePhoneNumber deletes a specific phone number by ID.
func (c *APIClient) DeletePhoneNumber(id string) ([]byte, int, error) {
	if len(id) == 0 {
		return []byte{}, 404, nil
	}
	endpoint := fmt.Sprintf("phone-number/%s", id)
	return c.SendRequest("DELETE", endpoint, nil)
}

// UpdatePhoneNumber updates a phone number by ID.
func (c *APIClient) UpdatePhoneNumber(id string, requestData interface{}) ([]byte, int, error) {
	if len(id) == 0 {
		return []byte{}, 404, nil
	}
	endpoint := fmt.Sprintf("phone-number/%s", id)
	return c.SendRequest("PATCH", endpoint, requestData)
}

// ImportSIPTrunkPhoneNumber requests the creation of a new phone number.
func (c *APIClient) ImportSIPTrunkPhoneNumber(requestData ImportSIPTrunkPhoneNumberRequest) ([]byte, int, error) {
	return c.SendRequest("POST", "phone-number", requestData)
}

// CreateToolQueryFunction method.
func (c *APIClient) CreateToolQueryFunction(requestData ToolQueryFunctionRequest) ([]byte, int, error) {
	return c.SendRequest("POST", "tool", requestData)
}

// UpdateToolQueryFunction method.
func (c *APIClient) UpdateToolQueryFunction(id string, requestData ToolQueryFunctionRequest) ([]byte, int, error) {
	if len(id) == 0 {
		return []byte{}, 404, nil
	}
	endpoint := fmt.Sprintf("tool/%s", id)
	return c.SendRequest("PATCH", endpoint, requestData)
}

// GetToolQueryFunction retrieves the details of a specific tool by ID.
func (c *APIClient) GetToolQueryFunction(id string) ([]byte, int, error) {
	if len(id) == 0 {
		return []byte{}, 404, nil
	}
	endpoint := fmt.Sprintf("tool/%s", id)
	return c.SendRequest("GET", endpoint, nil)
}

// DeleteToolQueryFunction deletes a specific tool by ID.
func (c *APIClient) DeleteToolQueryFunction(id string) ([]byte, int, error) {
	if len(id) == 0 {
		return []byte{}, 404, nil
	}
	endpoint := fmt.Sprintf("tool/%s", id)
	return c.SendRequest("DELETE", endpoint, nil)
}

// CreateToolFunction method.
func (c *APIClient) CreateToolFunction(requestData ToolFunctionRequest) ([]byte, int, error) {
	return c.SendRequest("POST", "tool", requestData)
}

// UpdateToolFunction updates an existing tool function by ID.
func (c *APIClient) UpdateToolFunction(id string, requestData ToolFunctionRequest) ([]byte, int, error) {
	if len(id) == 0 {
		return []byte{}, 404, nil
	}
	endpoint := fmt.Sprintf("tool/%s", id)
	return c.SendRequest("PATCH", endpoint, requestData)
}

// GetToolFunction retrieves the details of a specific tool by ID.
func (c *APIClient) GetToolFunction(id string) ([]byte, int, error) {
	if len(id) == 0 {
		return []byte{}, 404, nil
	}
	endpoint := fmt.Sprintf("tool/%s", id)
	return c.SendRequest("GET", endpoint, nil)
}

// DeleteToolFunction deletes a specific tool by ID.
func (c *APIClient) DeleteToolFunction(id string) ([]byte, int, error) {
	if len(id) == 0 {
		return []byte{}, 404, nil
	}
	endpoint := fmt.Sprintf("tool/%s", id)
	return c.SendRequest("DELETE", endpoint, nil)
}

// CreateAssistant creates a new assistant.
func (c *APIClient) CreateAssistant(requestData CreateAssistantRequest) ([]byte, int, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(requestData); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to encode request data: %v", err)
	}

	return c.SendRequest("POST", "assistant", requestData)
}

// UpdateAssistant updates assistant.
func (c *APIClient) UpdateAssistant(id string, requestData CreateAssistantRequest) ([]byte, int, error) {

	if id == "" {
		return nil, http.StatusNotFound, fmt.Errorf("ID cannot be empty")
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(requestData); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to encode request data: %v", err)
	}

	endpoint := fmt.Sprintf("assistant/%s", id)
	return c.SendRequest("PATCH", endpoint, requestData)
}

// GetAssistant retrieves the details of a specific assistant by ID.
func (c *APIClient) GetAssistant(id string) ([]byte, int, error) {
	if id == "" {
		return nil, http.StatusNotFound, fmt.Errorf("ID cannot be empty")
	}

	endpoint := fmt.Sprintf("assistant/%s", id)
	return c.SendRequest("GET", endpoint, nil)
}

// DeleteAssistant deletes an existing assistant by ID.
func (c *APIClient) DeleteAssistant(id string) ([]byte, int, error) {
	if id == "" {
		return nil, http.StatusNotFound, fmt.Errorf("ID cannot be empty")
	}

	endpoint := fmt.Sprintf("assistant/%s", id)
	return c.SendRequest("DELETE", endpoint, nil)
}

// CreateSIPTrunk creates a new assistant.
func (c *APIClient) CreateSIPTrunk(requestData ImportSIPTrunkRequest) ([]byte, int, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(requestData); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to encode request data: %v", err)
	}

	return c.SendRequest("POST", "credential", requestData)
}

// UpdateSIPTrunk updates assistant.
func (c *APIClient) UpdateSIPTrunk(id string, requestData ImportSIPTrunkRequest) ([]byte, int, error) {

	if id == "" {
		return nil, http.StatusNotFound, fmt.Errorf("ID cannot be empty")
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(requestData); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to encode request data: %v", err)
	}

	endpoint := fmt.Sprintf("credential/%s", id)
	return c.SendRequest("PATCH", endpoint, requestData)
}

// GetSIPTrunk retrieves the details of a specific assistant by ID.
func (c *APIClient) GetSIPTrunk(id string) ([]byte, int, error) {
	if id == "" {
		return nil, http.StatusNotFound, fmt.Errorf("ID cannot be empty")
	}

	endpoint := fmt.Sprintf("credential/%s", id)
	return c.SendRequest("GET", endpoint, nil)
}

// DeleteSIPTrunk deletes an existing assistant by ID.
func (c *APIClient) DeleteSIPTrunk(id string) ([]byte, int, error) {
	if id == "" {
		return nil, http.StatusNotFound, fmt.Errorf("ID cannot be empty")
	}

	endpoint := fmt.Sprintf("credential/%s", id)
	return c.SendRequest("DELETE", endpoint, nil)
}
