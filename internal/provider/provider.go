package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

// Provider function
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The base URL of the remote API.",
			},
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "The Bearer token used for API authentication.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"vapi_file": resourceVAPIFile(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

// Provider configuration function
func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	remoteURL := d.Get("url").(string)
	token := d.Get("token").(string)

	client := &APIClient{
		BaseURL:    remoteURL,
		Token:      token,
		HTTPClient: &http.Client{},
	}

	return client, nil
}
