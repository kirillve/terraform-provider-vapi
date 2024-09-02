package vapi

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// FileResponse represents the structure of the API response.
type FileResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	OriginalName string                 `json:"originalName"`
	Bytes        int64                  `json:"bytes"`
	Mimetype     string                 `json:"mimetype"`
	Path         string                 `json:"path"`
	URL          string                 `json:"url"`
	Metadata     map[string]interface{} `json:"metadata"`
	OrgID        string                 `json:"orgId"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
	Status       string                 `json:"status"`
	Purpose      string                 `json:"purpose"`
	Bucket       string                 `json:"bucket"`
}

// UnmarshalJSON implements custom unmarshaling for FileResponse.
func (fr *FileResponse) UnmarshalJSON(data []byte) error {
	type Alias FileResponse
	temp := &struct {
		Bytes json.RawMessage `json:"bytes"`
		*Alias
	}{
		Alias: (*Alias)(fr),
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	var bytesValue interface{}
	if err := json.Unmarshal(temp.Bytes, &bytesValue); err != nil {
		return err
	}

	switch v := bytesValue.(type) {
	case float64:
		fr.Bytes = int64(v)
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return err
		}
		fr.Bytes = parsed
	case int64:
		fr.Bytes = v
	default:
		return fmt.Errorf("unexpected type for bytes: %T", v)
	}

	return nil
}
