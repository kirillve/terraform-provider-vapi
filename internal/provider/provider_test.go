package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	frameworkProvider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

func TestProviderMetadataSchemaAndConfigure(t *testing.T) {
	ctx := context.Background()

	providerFactory := New("1.2.3")
	prov, ok := providerFactory().(*VAPIProvider)
	if !ok {
		t.Fatalf("expected *VAPIProvider")
	}

	metaResp := &frameworkProvider.MetadataResponse{}
	prov.Metadata(ctx, frameworkProvider.MetadataRequest{}, metaResp)
	if metaResp.TypeName != "vapi" {
		t.Fatalf("unexpected type name %s", metaResp.TypeName)
	}
	if metaResp.Version != "1.2.3" {
		t.Fatalf("unexpected version %s", metaResp.Version)
	}

	var schemaResp frameworkProvider.SchemaResponse
	prov.Schema(ctx, frameworkProvider.SchemaRequest{}, &schemaResp)
	if len(schemaResp.Schema.Attributes) != 2 {
		t.Fatalf("expected provider schema to define 2 attributes")
	}

	configValue := buildProviderConfig(t, ctx)

	confResp := &frameworkProvider.ConfigureResponse{}
	prov.Configure(ctx, frameworkProvider.ConfigureRequest{
		Config: tfsdk.Config{
			Raw:    configValue,
			Schema: schemaResp.Schema,
		},
	}, confResp)

	if confResp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %v", confResp.Diagnostics)
	}

	if _, ok := confResp.ResourceData.(*vapi.APIClient); !ok {
		t.Fatalf("expected resource data to be *vapi.APIClient")
	}
	if _, ok := confResp.DataSourceData.(*vapi.APIClient); !ok {
		t.Fatalf("expected data source data to be *vapi.APIClient")
	}

	if len(prov.Resources(ctx)) == 0 {
		t.Fatalf("expected resources to be registered")
	}
	if len(prov.DataSources(ctx)) != 0 {
		t.Fatalf("expected no data sources")
	}
	if len(prov.Functions(ctx)) != 0 {
		t.Fatalf("expected no functions")
	}
}

func buildProviderConfig(t *testing.T, ctx context.Context) tftypes.Value {
	obj, diags := types.ObjectValue(map[string]attr.Type{
		"url":   types.StringType,
		"token": types.StringType,
	}, map[string]attr.Value{
		"url":   types.StringValue("https://api.example.com"),
		"token": types.StringValue("secret"),
	})
	if diags.HasError() {
		t.Fatalf("object value diagnostics: %v", diags)
	}

	value, err := obj.ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("terraform value conversion failed: %v", err)
	}

	return value
}
