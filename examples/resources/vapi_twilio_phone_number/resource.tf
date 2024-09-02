resource "vapi_twilio_phone_number" "test-vapi_twilio_phone_number" {
  name                                           = "test twilio phone number"
  number                                         = "+11234567890"
  twilio_account_sid                             = "sid"
  twilio_auth_token                              = "auth-token"
  fallback_destination_type                      = "number"
  fallback_destination_number_e164_check_enabled = bool
  fallback_destination_number                    = "+11234567890"
  fallback_destination_extension                 = "123"
  fallback_destination_message                   = "Message"
  fallback_destination_description               = "Description"
}
