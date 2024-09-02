package vapi

// ImportTwilioRequest struct.
type ImportTwilioRequest struct {
	Provider         string               `json:"provider"`
	Name             string               `json:"name"`
	Number           string               `json:"number"`
	TwilioAccountSID string               `json:"twilioAccountSid"`
	TwilioAuthToken  string               `json:"twilioAuthToken"`
	Fallback         *FallbackDestination `json:"fallbackDestination,omitempty"`
}

// FallbackDestination struct.
type FallbackDestination struct {
	Type                   string `json:"type"`
	NumberE164CheckEnabled bool   `json:"numberE164CheckEnabled"`
	Number                 string `json:"number"`
	Extension              string `json:"extension"`
	Message                string `json:"message"`
	Description            string `json:"description"`
}

// TwilioPhoneNumber represents the structure of a Twilio phone number API response.
type TwilioPhoneNumber struct {
	ID               string `json:"id"`
	OrgID            string `json:"orgId"`
	Number           string `json:"number"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
	TwilioAccountSid string `json:"twilioAccountSid"`
	TwilioAuthToken  string `json:"twilioAuthToken"`
	Name             string `json:"name"`
	Provider         string `json:"provider"`
}
