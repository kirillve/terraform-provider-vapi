//go:build generate

// Package tools pins provider documentation generation.
package tools

//go:generate go tool tfplugindocs generate --provider-dir .. -provider-name vapi
