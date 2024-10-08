---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "vapi_file Resource - vapi"
subcategory: ""
description: |-
  Manages a file resource in the VAPI system.
---

# vapi_file (Resource)

Manages a file resource in the VAPI system.

## Example Usage

```terraform
resource "vapi_file" "test-vapi_file" {
  content  = file("/tmp/file.txt")
  filename = "file.txt"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `content` (String) The file content to upload.
- `filename` (String) The filename for upload.

### Read-Only

- `bucket` (String) The uploaded file bucket.
- `bytes` (Number) The size of the file in bytes.
- `created_at` (String) The timestamp when the file was created.
- `id` (String) The ID of the file.
- `mimetype` (String) The MIME type of the file.
- `name` (String) The name of the file.
- `org_id` (String) The OrgId of the file.
- `original_name` (String) The original name of the file.
- `path` (String) The path to the file.
- `purpose` (String) The uploaded file purpose.
- `status` (String) The uploaded file status.
- `updated_at` (String) The timestamp when the file was last updated.
- `url` (String) The URL to access the file.
