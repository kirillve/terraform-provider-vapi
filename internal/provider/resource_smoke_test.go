package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

func TestResourceSchemaAndMetadata(t *testing.T) {
	ctx := context.Background()

	resources := []resource.Resource{
		NewVAPIFileResource(),
		NewVAPIAssistantResource(),
		NewVAPIToolFunctionResource(),
		NewVAPIToolQueryFunctionResource(),
		NewVAPISIPTrunkResource(),
		NewVAPISIPTrunkPhoneNumberResource(),
		NewVAPIPhoneNumberResource(),
	}

	for _, res := range resources {
		t.Run(fmt.Sprintf("%T", res), func(t *testing.T) {
			metaResp := &resource.MetadataResponse{}
			res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "vapi"}, metaResp)
			if metaResp.TypeName == "" {
				t.Fatalf("expected type name for %T", res)
			}

			var schemaResp resource.SchemaResponse
			res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			if schemaResp.Schema.Attributes == nil && schemaResp.Schema.Blocks == nil {
				t.Fatalf("expected schema for %T", res)
			}

			if configurable, ok := res.(resource.ResourceWithConfigure); ok {
				configurable.Configure(ctx, resource.ConfigureRequest{
					ProviderData: &vapi.APIClient{
						BaseURL: "https://example.com",
						Token:   "token",
					},
				}, &resource.ConfigureResponse{})
			}

		})
	}
}
