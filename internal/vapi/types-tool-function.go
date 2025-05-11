package vapi

// ToolFunctionRequest struct.
type ToolFunctionRequest struct {
	Server   Server   `json:"server"`
	Function Function `json:"function"`
	Type     string   `json:"type"`
	Async    bool     `json:"async"`
}

type Server struct {
	URL    string `json:"url,omitempty"`
	Secret string `json:"secret,omitempty"`
}

type Function struct {
	Description string          `json:"description"`
	Async       bool            `json:"async,omitempty"`
	Name        string          `json:"name,omitempty"`
	Parameters  *FunctionParams `json:"parameters,omitempty"`
}

type FunctionParams struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
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

type ResponseFunction struct {
	Name        string                 `json:"name"`
	Async       bool                   `json:"async"`
	Description string                 `json:"description"`
	Parameters  ResponseFunctionParams `json:"parameters"`
}

type ResponseFunctionParams struct {
	Type       string                      `json:"type"`
	Properties map[string]ResponseProperty `json:"properties"`
	Required   []string                    `json:"required"`
}

type ResponseProperty struct {
	Description string `json:"description"`
	Type        string `json:"type"`
}

type ResponseServer struct {
	URL string `json:"url"`
}
