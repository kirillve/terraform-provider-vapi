resource "vapi_tool_function" "test-vapi_tool_function" {
  name        = "function-name"
  description = "function-description"
  async       = false
  type        = "function"

  server_url    = "https://somewhere.com/api/vapi/functions/basic"
  server_secret = "123asd"

  parameters = {
    type     = "object"
    async    = false
    required = []
    properties = {
      property1 = {
        type        = "string"
        description = "property1 description"
      }
      property2 = {
        type        = "string"
        description = "property2 description"
      }
      property3 = {
        type        = "string"
        description = "property3 description"
      }
      propertyN = {
        type        = "string"
        description = "propertyN description"
      }
    }
  }
}

resource "vapi_tool_function" "booking_outgoing" {
  name        = "booking_outgoing"
  description = "Book an appointment for a customer."
  async       = false
  type        = "function"

  server_url             = "https://somewhere.com/api/vapi/functions/booking"
  server_secret          = "replace-with-your-secret"
  server_timeout_seconds = 30

  parameters_json = jsonencode({
    type = "object"
    properties = {
      booking_info = {
        type = "object"
        properties = {
          customer = {
            type = "object"
            properties = {
              first_name = { type = "string" }
              last_name  = { type = "string" }
            }
            required = ["first_name", "last_name"]
          }
        }
        required = ["customer"]
      }
    }
    required = ["booking_info"]
  })
}
