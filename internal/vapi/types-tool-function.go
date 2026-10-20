package vapi

import "encoding/json"

// ToolFunctionRequest struct.
type ToolFunctionRequest struct {
	Destinations []Destination `json:"destinations,omitempty"`
	Server       *Server       `json:"server"`
	Function     Function      `json:"function"`
	Type         string        `json:"type"`
	Async        bool          `json:"async"`
}

// Destination is one transfer-call destination.
type Destination struct {
	Type                   string `json:"type"`
	Number                 string `json:"number"`
	Extension              string `json:"extension,omitempty"`
	Message                string `json:"message,omitempty"`
	Description            string `json:"description,omitempty"`
	NumberE164CheckEnabled bool   `json:"numberE164CheckEnabled"`
}

// Function is the function definition of a tool request.
type Function struct {
	Description string `json:"description"`
	Async       bool   `json:"async,omitempty"`
	Name        string `json:"name,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// FunctionParams is the flat function parameters schema of a tool request.
type FunctionParams struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// Property is one entry of the flat function parameters schema.
type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// ToolFunctionResponse struct.
type ToolFunctionResponse struct {
	ID        string           `json:"id"`
	CreatedAt string           `json:"createdAt"`
	UpdatedAt string           `json:"updatedAt"`
	Type      string           `json:"type"`
	Function  ResponseFunction `json:"function"`
	OrgID     string           `json:"orgId"`
	Server    ResponseServer   `json:"server"`
	Async     bool             `json:"async"`
}

// ResponseFunction is the function definition of a tool response.
type ResponseFunction struct {
	Name        string                 `json:"name"`
	Async       bool                   `json:"async"`
	Description string                 `json:"description"`
	Parameters  ResponseFunctionParams `json:"parameters"`
}

// ResponseFunctionParams is the function parameters schema of a tool response.
type ResponseFunctionParams struct {
	Raw        json.RawMessage             `json:"-"`
	Type       string                      `json:"type"`
	Properties map[string]ResponseProperty `json:"properties"`
	Required   []string                    `json:"required"`
}

// ResponseProperty is one entry of the function parameters schema of a tool response.
type ResponseProperty struct {
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Enum        []string `json:"enum,omitempty"`
}

// ResponseServer is the server block of a tool response.
type ResponseServer struct {
	URL            string `json:"url"`
	TimeoutSeconds int64  `json:"timeoutSeconds"`
}

// UnmarshalJSON preserves schemas that cannot be represented by the legacy flat model.
func (p *ResponseFunctionParams) UnmarshalJSON(data []byte) error {
	type flatParams ResponseFunctionParams
	var flat flatParams
	// Arbitrary JSON Schema properties (for example, numeric enums or union types)
	// need not fit the flat fields. Raw remains the source of truth in JSON mode.
	if err := json.Unmarshal(data, &flat); err != nil {
		flat = flatParams{}
	}
	*p = ResponseFunctionParams(flat)
	p.Raw = append(json.RawMessage(nil), data...)
	return nil
}
