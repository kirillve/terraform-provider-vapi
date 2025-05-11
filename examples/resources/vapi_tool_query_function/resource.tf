resource "vapi_tool_query_function" "test-vapi_tool_query_function" {
  name        = "function-name"
  description = "function-description"

  knowledge_bases = [
    {
      provider    = "google"
      name        = "new_knowledge_base"
      model       = "gemini-1.5-flash"
      description = "kb description"
      file_ids    = ["38d2487b-188d-4214-b14d-9ffb1c026724"]
    }
  ]
}
