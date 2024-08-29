package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
)

// APIClient handles communication with the remote provider.
type APIClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// Uploads a file using multipart/form-data
func (c *APIClient) uploadFile(fieldName, filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreatePart(
		textproto.MIMEHeader{
			"Content-Disposition": []string{fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, filepath.Base(filePath))},
			"Content-Type":        []string{getMimeType(filePath)},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/file", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer res.Body.Close()

	// Check for 400 Bad Request with an empty response body
	if res.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("received 400 Bad Request: please verify the input file and parameters")
	}

	// Handle other non-success status codes
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("received unexpected status code %d: %s", res.StatusCode, http.StatusText(res.StatusCode))
	}

	return io.ReadAll(res.Body)
}

// Sends a request to the API
func (c *APIClient) sendRequest(method, endpoint string, body interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, c.BaseURL+"/"+endpoint, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return io.ReadAll(res.Body)
}

func getMimeType(filename string) string {
	ext := filepath.Ext(filename)
	return mime.TypeByExtension(ext)
}
