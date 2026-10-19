package provider

import "testing"

func TestAccTwilioSMSEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("acceptance test requires live VAPI credentials")
	}
	t.Skip("acceptance stub: configure VAPI and Twilio credentials via environment to verify SMS settings on a live number")
}

func TestAccToolFunctionNested(t *testing.T) {
	if testing.Short() {
		t.Skip("acceptance test requires live VAPI credentials")
	}
	t.Skip("acceptance stub: configure VAPI credentials via environment before exercising live nested tools")
}
