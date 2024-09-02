resource "vapi_file" "test-vapi_file" {
  content = file("/tmp/file.txt")
  filename = "file.txt"
}
