resource "vapi_twilio_phone_number" "test-vapi_twilio_phone_number" {
  name               = "test twilio phone number"
  number             = "+11234567890"
  twilio_account_sid = "sid"
  twilio_auth_token  = "auth-token"
}
