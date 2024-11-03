package vapi

// ImportTwilioRequest struct.
type ImportTwilioRequest struct {
	Provider         string               `json:"provider"`
	Name             string               `json:"name"`
	Number           string               `json:"number"`
	TwilioAccountSID string               `json:"twilioAccountSid"`
	TwilioAuthToken  string               `json:"twilioAuthToken"`
	AssistantID      string               `json:"assistantID,omitempty"`
	Fallback         *FallbackDestination `json:"fallbackDestination,omitempty"`
}

// FallbackDestination struct.
type FallbackDestination struct {
	Type                   string `json:"type,omitempty"`
	NumberE164CheckEnabled bool   `json:"numberE164CheckEnabled,omitempty"`
	Number                 string `json:"number,omitempty"`
	Extension              string `json:"extension,omitempty"`
	Message                string `json:"message,omitempty"`
	Description            string `json:"description,omitempty"`
}

// TwilioPhoneNumber represents the structure of a Twilio phone number API response.
type TwilioPhoneNumber struct {
	ID               string               `json:"id"`
	OrgID            string               `json:"orgId"`
	Number           string               `json:"number"`
	CreatedAt        string               `json:"createdAt"`
	UpdatedAt        string               `json:"updatedAt"`
	TwilioAccountSid string               `json:"twilioAccountSid"`
	TwilioAuthToken  string               `json:"twilioAuthToken"`
	Name             string               `json:"name"`
	Provider         string               `json:"provider"`
	AssistantID      string               `json:"assistantID"`
	Fallback         *FallbackDestination `json:"fallbackDestination,omitempty"`
}
