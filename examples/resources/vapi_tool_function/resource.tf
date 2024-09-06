resource "vapi_tool_function" "test-vapi_tool_function" {
  name        = "function-name"
  description = "function-description"
  async       = false
  type        = "function"

  server_url    = "https://somewhere.com/api/vapi/functions/basic"
  server_secret = "123asd"

  parameters = {
    type  = "object"
    async = false
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
