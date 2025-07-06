package vapi

// ImportSIPTrunkPhoneNumberRequest represents the payload to import a SIP trunk phone number.
type ImportSIPTrunkPhoneNumberRequest struct {
	Provider               string `json:"provider"`               // always "byo-phone-number"
	Number                 string `json:"number"`                 // e.g., "+14031234567"
	NumberE164CheckEnabled bool   `json:"numberE164CheckEnabled"` // true/false
	CredentialID           string `json:"credentialId"`           // e.g., UUID for SIP credentials
	Name                   string `json:"name"`                   // descriptive name
}

// ImportSIPTrunkPhoneNumberResponse represents the response structure after import.
type ImportSIPTrunkPhoneNumberResponse struct {
	ID                     string `json:"id"`                     // system-generated phone number ID
	OrgID                  string `json:"orgId"`                  // owning organization ID
	Number                 string `json:"number"`                 // phone number
	CreatedAt              string `json:"createdAt"`              // RFC3339 timestamp
	UpdatedAt              string `json:"updatedAt"`              // RFC3339 timestamp
	Provider               string `json:"provider"`               // "byo-phone-number"
	Name                   string `json:"name"`                   // same as request
	NumberE164CheckEnabled bool   `json:"numberE164CheckEnabled"` // same as request
	CredentialID           string `json:"credentialId"`           // same as request
}
