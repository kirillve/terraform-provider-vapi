package vapi

import (
	"bytes"
	"encoding/json"
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

// Uploads a file using multipart/form-data.
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
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("error reading response body: %v", err)
	}

	// Return the response body and status code
	return responseData, resp.StatusCode, nil
}

// Sends a request to the API.
func (c *APIClient) SendRequest(method, endpoint string, body interface{}) ([]byte, int, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, 500, err
		}
	}

	req, err := http.NewRequest(method, c.BaseURL+"/"+endpoint, &buf)
	if err != nil {
		return nil, 500, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	defer resp.Body.Close()

	// Read the response body
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("error reading response body: %v", err)
	}

	// Return the response body and status code
	return responseData, resp.StatusCode, nil
}

// ImportTwilioPhoneNumber requests the creation of a new phone number.
func (c *APIClient) ImportTwilioPhoneNumber(requestData ImportTwilioRequest) ([]byte, int, error) {
	return c.SendRequest("POST", "phone-number", requestData)
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
