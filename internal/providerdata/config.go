// Package providerdata defines the generation-neutral provider configuration that the
// vapi provider hands to every resource set's Configure method.
//
// In scope: the Config value (base URL and bearer token) as read from the provider block.
// Out of scope: HTTP clients, validation of url/token, defaults, environment lookup, any
// resource-set specific type.
//
// Tier 0: imports nothing. Producer: internal/provider. Consumers: internal/vapiv1 (later internal/vapiv2).
package providerdata

// Config is passed as ProviderData to each resource's Configure. Token is Sensitive: never
// logged, never formatted with %v/%+v; Config deliberately has no String method.
type Config struct {
	URL   string // provider attribute "url", verbatim
	Token string // provider attribute "token"
}
