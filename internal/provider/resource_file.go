package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"os"
	"strconv"
)

// FileResponse represents the structure of the API response
type FileResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	OriginalName string                 `json:"originalName"`
	Bytes        int64                  `json:"bytes"`
	Mimetype     string                 `json:"mimetype"`
	Path         string                 `json:"path"`
	URL          string                 `json:"url"`
	Metadata     map[string]interface{} `json:"metadata"`
	OrgID        string                 `json:"orgId"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
}

// UnmarshalJSON implements custom unmarshaling for FileResponse
func (fr *FileResponse) UnmarshalJSON(data []byte) error {
	type Alias FileResponse
	temp := &struct {
		Bytes json.RawMessage `json:"bytes"`
		*Alias
	}{
		Alias: (*Alias)(fr),
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	var bytesValue interface{}
	if err := json.Unmarshal(temp.Bytes, &bytesValue); err != nil {
		return err
	}

	switch v := bytesValue.(type) {
	case float64:
		fr.Bytes = int64(v)
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return err
		}
		fr.Bytes = parsed
	case int64:
		fr.Bytes = v
	default:
		return fmt.Errorf("unexpected type for bytes: %T", v)
	}

	return nil
}

func resourceVAPIFile() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFileCreate,
		ReadContext:   resourceFileRead,
		UpdateContext: resourceFileUpdate,
		DeleteContext: resourceFileDelete,

		Schema: map[string]*schema.Schema{
			"file_path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The local path of the file to upload.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the file.",
			},
			"original_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The original name of the file.",
			},
			"bytes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the file in bytes.",
			},
			"mimetype": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The MIME type of the file.",
			},
			"path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The path to the file.",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The URL to access the file.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp when the file was created.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp when the file was last updated.",
			},
			"checksum": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The SHA-256 checksum of the file.",
			},
		},
	}
}

func resourceFileCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*APIClient)

	filePath := d.Get("file_path").(string)
	checksum, err := computeChecksum(filePath)
	if err != nil {
		return diag.FromErr(err)
	}

	// Upload the file using multipart/form-data
	response, err := client.uploadFile("file", filePath)
	if err != nil {
		return diag.FromErr(err)
	}

	// Parse and set the response data
	return setFileResponseData(d, response, checksum)
}

func resourceFileRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*APIClient)

	response, err := client.sendRequest("GET", "file/"+d.Id(), nil)
	if err != nil {
		return diag.FromErr(err)
	}

	var fileResponse FileResponse
	if err := json.Unmarshal(response, &fileResponse); err != nil {
		return diag.FromErr(err)
	}

	// Update the Terraform state with the fetched data
	updateResourceData(d, &fileResponse)

	// Check if file needs re-upload based on checksum
	filePath := d.Get("file_path").(string)
	localChecksum, err := computeChecksum(filePath)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("checksum", localChecksum)

	return nil
}

func resourceFileUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*APIClient)

	filePath := d.Get("file_path").(string)
	localChecksum, err := computeChecksum(filePath)
	if err != nil {
		return diag.FromErr(err)
	}

	// Check the existing checksum from the Terraform state
	storedChecksum := d.Get("checksum").(string)

	// Compare the local checksum with the stored checksum
	if localChecksum != storedChecksum {
		// The file has been changed; need to re-upload it
		if err := deleteExistingFile(client, d.Id()); err != nil {
			return diag.FromErr(err)
		}

		// Upload the new file
		response, err := client.uploadFile("file", filePath)
		if err != nil {
			return diag.FromErr(err)
		}

		// Parse and set the response data
		return setFileResponseData(d, response, localChecksum)
	}

	// File has not changed, no need to re-upload
	d.Set("checksum", localChecksum)

	return nil
}

func resourceFileDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*APIClient)

	if err := deleteExistingFile(client, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}

// Helper method to compute the checksum of a file
func computeChecksum(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %s", filePath, err)
	}
	checksum := sha256.Sum256(content)
	return hex.EncodeToString(checksum[:]), nil
}

// Helper method to set the response data into Terraform state
func setFileResponseData(d *schema.ResourceData, response []byte, checksum string) diag.Diagnostics {
	var fileResponse FileResponse
	if err := json.Unmarshal(response, &fileResponse); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fileResponse.ID)
	updateResourceData(d, &fileResponse)
	d.Set("checksum", checksum)

	return nil
}

// Helper method to update the Terraform resource data from the FileResponse
func updateResourceData(d *schema.ResourceData, fr *FileResponse) {
	d.Set("name", fr.Name)
	d.Set("original_name", fr.OriginalName)
	d.Set("bytes", fr.Bytes)
	d.Set("mimetype", fr.Mimetype)
	d.Set("path", fr.Path)
	d.Set("url", fr.URL)
	d.Set("created_at", fr.CreatedAt)
	d.Set("updated_at", fr.UpdatedAt)
}

// Helper method to delete an existing file
func deleteExistingFile(client *APIClient, fileID string) error {
	_, err := client.sendRequest("DELETE", "file/"+fileID, nil)
	return err
}
