package vapi

// ToolQueryFunctionRequest struct.
type ToolQueryFunctionRequest struct {
	Function       Function          `json:"function"`
	KnowledgeBases []TQKnowledgeBase `json:"knowledgeBases"`
	Type           string            `json:"type,omitempty"`
}

type ToolQueryFunctionResponse struct {
	ID             string            `json:"id"`
	CreatedAt      string            `json:"createdAt"`
	UpdatedAt      string            `json:"updatedAt"`
	Type           string            `json:"type"`
	OrgID          string            `json:"orgId"`
	Function       Function          `json:"function"`
	KnowledgeBases []TQKnowledgeBase `json:"knowledgeBases"`
}

type TQKnowledgeBase struct {
	Provider    string   `json:"provider"`
	Name        string   `json:"name"`
	Model       string   `json:"model"`
	Description string   `json:"description"`
	FileIDs     []string `json:"fileIds"`
}
