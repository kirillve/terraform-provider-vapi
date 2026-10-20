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
