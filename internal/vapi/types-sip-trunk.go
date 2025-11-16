package vapi

// ImportSIPTrunkRequest struct represents a request to import a BYO SIP trunk.
type ImportSIPTrunkRequest struct {
	Provider                   string                      `json:"provider"`
	Name                       string                      `json:"name"`
	Gateways                   []SIPGateway                `json:"gateways"`
	OutboundAuthenticationPlan *OutboundAuthenticationPlan `json:"outboundAuthenticationPlan,omitempty"`
	OutboundLeadingPlusEnabled bool                        `json:"outboundLeadingPlusEnabled"`
	TechPrefix                 string                      `json:"techPrefix,omitempty"`
	SIPDiversionHeader         string                      `json:"sipDiversionHeader,omitempty"`
}

// SIPGateway struct represents an individual SIP gateway.
type SIPGateway struct {
	IP string `json:"ip"`
}

// SIPTrunk represents the API response for a SIP trunk.
type SIPTrunk struct {
	ID                         string                      `json:"id"`
	OrgID                      string                      `json:"orgId"`
	Provider                   string                      `json:"provider"`
	Name                       string                      `json:"name"`
	Gateways                   []SIPGateway                `json:"gateways"`
	OutboundAuthenticationPlan *OutboundAuthenticationPlan `json:"outboundAuthenticationPlan,omitempty"`
	OutboundLeadingPlusEnabled bool                        `json:"outboundLeadingPlusEnabled"`
	TechPrefix                 string                      `json:"techPrefix,omitempty"`
	SIPDiversionHeader         string                      `json:"sipDiversionHeader,omitempty"`
	CreatedAt                  string                      `json:"createdAt,omitempty"`
	UpdatedAt                  string                      `json:"updatedAt,omitempty"`
}

// OutboundAuthenticationPlan struct represents outbound SIP authentication details.
type OutboundAuthenticationPlan struct {
	AuthUsername    string           `json:"authUsername"`
	AuthPassword    string           `json:"authPassword"`
	SIPRegisterPlan *SIPRegisterPlan `json:"sipRegisterPlan,omitempty"`
}

// SIPRegisterPlan struct represents the SIP registration plan details.
type SIPRegisterPlan struct {
	Domain   string `json:"domain"`
	Username string `json:"username"`
	Realm    string `json:"realm"`
}
